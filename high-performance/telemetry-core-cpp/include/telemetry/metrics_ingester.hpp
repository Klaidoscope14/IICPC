#pragma once

#include "telemetry/types.hpp"
#include "telemetry/metrics_aggregator.hpp"
#include <functional>
#include <thread>
#include <atomic>

namespace telemetry {

/// Ingestion server that accepts raw metric data points.
///
/// Listens on a TCP port and parses incoming JSON metrics,
/// forwarding them to the MetricsAggregator.
class MetricsIngester {
public:
    using MetricCallback = std::function<void(const RawMetric&)>;

    MetricsIngester(int port, MetricCallback callback);
    ~MetricsIngester();

    /// Start the ingestion server in a background thread.
    void start();

    /// Stop the server gracefully.
    void stop();

    /// Check if the server is running.
    bool is_running() const { return running_.load(); }

    /// Get the total number of metrics ingested.
    int64_t metrics_ingested() const { return metrics_count_.load(); }

private:
    int port_;
    MetricCallback callback_;
    std::atomic<bool> running_{false};
    std::atomic<int64_t> metrics_count_{0};
    std::thread server_thread_;
    int server_fd_{-1};

    void run();
    void handle_client(int client_fd);
    RawMetric parse_metric(const std::string& json);
};

} // namespace telemetry
