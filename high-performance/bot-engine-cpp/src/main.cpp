#include "bot_engine/bot_pool.hpp"
#include "bot_engine/metrics_reporter.hpp"
#include <iostream>
#include <thread>
#include <chrono>
#include <csignal>
#include <atomic>

static std::atomic<bool> g_running{true};

void signal_handler(int) {
    g_running = false;
}

int main() {
    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);

    std::cout << "=== IICPC Bot Engine ===" << std::endl;

    // Load configuration from environment.
    auto config = bot_engine::BotConfig::from_env();

    std::cout << "Target:    " << config.target_url << std::endl;
    std::cout << "Bots:      " << config.bot_count << std::endl;
    std::cout << "Duration:  " << config.duration_seconds << "s" << std::endl;
    std::cout << "OPS:       " << config.orders_per_second << std::endl;

    // Create and start bot pool.
    bot_engine::BotPool pool(config);
    bot_engine::MetricsReporter reporter(config.metrics_endpoint, config.benchmark_id);

    pool.start();

    // Report metrics every second until done.
    while (g_running && pool.is_running()) {
        std::this_thread::sleep_for(std::chrono::seconds(1));

        auto metrics = pool.get_metrics();
        reporter.report(metrics);

        // Check if duration elapsed.
        if (metrics.total_orders_sent > 0 &&
            metrics.current_tps > 0 &&
            static_cast<double>(metrics.total_orders_sent) / metrics.current_tps >= config.duration_seconds) {
            break;
        }
    }

    pool.stop();

    // Print final summary.
    auto final_metrics = pool.get_metrics();
    std::cout << "\n=== Benchmark Complete ===" << std::endl;
    std::cout << "Total Orders Sent:         " << final_metrics.total_orders_sent << std::endl;
    std::cout << "Total Orders Acknowledged: " << final_metrics.total_orders_acked << std::endl;
    std::cout << "Total Errors:              " << final_metrics.total_orders_failed << std::endl;
    std::cout << "Average TPS:               " << final_metrics.current_tps << std::endl;
    std::cout << "Average Latency:           " << final_metrics.avg_latency_us << " µs" << std::endl;

    return 0;
}
