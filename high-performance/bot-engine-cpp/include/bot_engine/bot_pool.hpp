#pragma once

#include <vector>
#include <thread>
#include <memory>
#include <atomic>
#include "bot_engine/config.hpp"
#include "bot_engine/bot_worker.hpp"

namespace bot_engine {

/// Aggregate metrics from all bot workers.
struct PoolMetrics {
    int64_t total_orders_sent = 0;
    int64_t total_orders_acked = 0;
    int64_t total_orders_failed = 0;
    double avg_latency_us = 0.0;
    double current_tps = 0.0;
};

/// Manages a pool of BotWorker threads for distributed load generation.
///
/// Usage:
///   BotPool pool(config);
///   pool.start();
///   // ... wait for duration ...
///   pool.stop();
///   auto metrics = pool.get_metrics();
class BotPool {
public:
    explicit BotPool(const BotConfig& config);
    ~BotPool();

    /// Start all bot workers in their own threads.
    void start();

    /// Stop all workers and join threads.
    void stop();

    /// Get aggregate metrics from all workers.
    PoolMetrics get_metrics() const;

    /// Check if the pool is currently running.
    bool is_running() const { return running_.load(); }

private:
    BotConfig config_;
    std::vector<std::unique_ptr<BotWorker>> workers_;
    std::vector<std::thread> threads_;
    std::atomic<bool> running_{false};
    std::chrono::steady_clock::time_point start_time_;

    /// Create the HTTP submit function for workers.
    BotWorker::SubmitFunc create_submit_fn();
};

} // namespace bot_engine
