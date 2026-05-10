#pragma once

#include "telemetry/types.hpp"
#include "telemetry/metrics_store.hpp"
#include <thread>
#include <atomic>
#include <vector>
#include <functional>

namespace telemetry {

/// Simple TCP server that streams aggregated snapshots to connected clients.
///
/// Clients connect via TCP and receive JSON-encoded snapshots at 1-second intervals.
class StreamServer {
public:
    StreamServer(int port, const MetricsStore& store);
    ~StreamServer();

    /// Start the streaming server in a background thread.
    void start();

    /// Stop the server and disconnect all clients.
    void stop();

    /// Check if the server is running.
    bool is_running() const { return running_.load(); }

    /// Get the number of connected clients.
    int connected_clients() const { return client_count_.load(); }

private:
    int port_;
    const MetricsStore& store_;
    std::atomic<bool> running_{false};
    std::atomic<int> client_count_{0};
    std::thread accept_thread_;
    std::vector<std::thread> client_threads_;
    int server_fd_{-1};

    void run();
    void handle_client(int client_fd);
    std::string snapshot_to_json(const AggregatedSnapshot& snap) const;
};

} // namespace telemetry
