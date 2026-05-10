#include "telemetry/metrics_ingester.hpp"
#include "telemetry/metrics_aggregator.hpp"
#include "telemetry/metrics_store.hpp"
#include "telemetry/stream_server.hpp"
#include "telemetry/types.hpp"
#include <iostream>
#include <thread>
#include <chrono>
#include <csignal>
#include <atomic>
#include <cstdlib>

static std::atomic<bool> g_running{true};

void signal_handler(int) {
    g_running = false;
}

int main() {
    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);

    std::cout << "=== IICPC Telemetry Core ===" << std::endl;

    // Configuration.
    telemetry::TelemetryConfig config = telemetry::TelemetryConfig::from_env();

    std::cout << "Ingestion Port:  " << config.ingestion_port << std::endl;
    std::cout << "Streaming Port:  " << config.streaming_port << std::endl;
    std::cout << "Benchmark ID:    " << config.benchmark_id << std::endl;

    // Pipeline components.
    telemetry::MetricsAggregator aggregator(config.benchmark_id);
    telemetry::MetricsStore store(config.max_snapshots);

    // Ingestion server — receives raw metrics from bot fleet.
    telemetry::MetricsIngester ingester(config.ingestion_port,
        [&aggregator](const telemetry::RawMetric& metric) {
            aggregator.ingest(metric);
        });

    // Streaming server — pushes aggregated snapshots to clients.
    telemetry::StreamServer streamer(config.streaming_port, store);

    ingester.start();
    streamer.start();

    std::cout << "[Pipeline] Running — Ctrl+C to stop" << std::endl;

    // Aggregation loop: produce a snapshot every second.
    while (g_running) {
        std::this_thread::sleep_for(std::chrono::seconds(config.window_seconds));

        if (aggregator.current_count() > 0) {
            auto snapshot = aggregator.snapshot();
            store.append(snapshot);

            std::cout << "[Snapshot] tps=" << snapshot.current_tps
                      << " sent=" << snapshot.total_orders_sent
                      << " acked=" << snapshot.total_orders_acknowledged
                      << " p99=" << snapshot.p99_latency_us << "µs"
                      << " clients=" << streamer.connected_clients()
                      << std::endl;
        }
    }

    std::cout << "\n[Pipeline] Shutting down..." << std::endl;
    ingester.stop();
    streamer.stop();

    std::cout << "Total metrics ingested: " << ingester.metrics_ingested() << std::endl;
    std::cout << "Snapshots stored: " << store.size() << std::endl;
    std::cout << "=== Telemetry Core Stopped ===" << std::endl;

    return 0;
}
