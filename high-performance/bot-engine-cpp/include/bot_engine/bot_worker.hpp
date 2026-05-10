#pragma once

#include <atomic>
#include <functional>
#include <chrono>
#include <string>
#include "bot_engine/config.hpp"
#include "bot_engine/order_generator.hpp"

namespace bot_engine {

/// Result of a single order submission attempt.
struct OrderResult {
    bool success;
    std::chrono::microseconds latency;
    std::string error_message;
};

/// A single trading bot that generates and submits orders in a loop.
///
/// Each BotWorker runs on its own thread, generating orders at a configured
/// rate and submitting them to the target exchange endpoint.
class BotWorker {
public:
    using SubmitFunc = std::function<OrderResult(const Order&)>;

    /// Create a worker with a unique ID and order submission function.
    BotWorker(int worker_id, const BotConfig& config, SubmitFunc submit_fn);

    /// Run the worker for the configured duration. Blocks until done or stopped.
    void run();

    /// Signal the worker to stop gracefully.
    void stop();

    /// Get the number of orders sent by this worker.
    int64_t orders_sent() const { return orders_sent_.load(); }

    /// Get the number of successful orders.
    int64_t orders_acked() const { return orders_acked_.load(); }

    /// Get the number of failed orders.
    int64_t orders_failed() const { return orders_failed_.load(); }

    /// Get total latency in microseconds (for computing averages).
    int64_t total_latency_us() const { return total_latency_us_.load(); }

private:
    int worker_id_;
    BotConfig config_;
    SubmitFunc submit_fn_;
    OrderGenerator generator_;
    std::atomic<bool> running_{false};
    std::atomic<int64_t> orders_sent_{0};
    std::atomic<int64_t> orders_acked_{0};
    std::atomic<int64_t> orders_failed_{0};
    std::atomic<int64_t> total_latency_us_{0};
};

} // namespace bot_engine
