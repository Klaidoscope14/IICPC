#include "bot_engine/bot_pool.hpp"
#include "bot_engine/http_client.hpp"
#include <iostream>

namespace bot_engine {

BotPool::BotPool(const BotConfig& config)
    : config_(config)
{}

BotPool::~BotPool() {
    stop();
}

void BotPool::start() {
    if (running_) return;
    running_ = true;
    start_time_ = std::chrono::steady_clock::now();

    auto submit_fn = create_submit_fn();

    workers_.reserve(config_.bot_count);
    threads_.reserve(config_.bot_count);

    for (int i = 0; i < config_.bot_count; ++i) {
        auto worker = std::make_unique<BotWorker>(i, config_, submit_fn);
        auto* worker_ptr = worker.get();
        workers_.push_back(std::move(worker));
        threads_.emplace_back([worker_ptr]() { worker_ptr->run(); });
    }

    std::cout << "[BotPool] Started " << config_.bot_count << " workers targeting "
              << config_.target_url << std::endl;
}

void BotPool::stop() {
    if (!running_) return;
    running_ = false;

    // Signal all workers to stop.
    for (auto& worker : workers_) {
        worker->stop();
    }

    // Join all threads.
    for (auto& thread : threads_) {
        if (thread.joinable()) {
            thread.join();
        }
    }

    std::cout << "[BotPool] All workers stopped" << std::endl;
}

PoolMetrics BotPool::get_metrics() const {
    PoolMetrics metrics;

    for (const auto& worker : workers_) {
        metrics.total_orders_sent += worker->orders_sent();
        metrics.total_orders_acked += worker->orders_acked();
        metrics.total_orders_failed += worker->orders_failed();

        int64_t latency = worker->total_latency_us();
        int64_t count = worker->orders_sent();
        if (count > 0) {
            metrics.avg_latency_us += static_cast<double>(latency) / count;
        }
    }

    // Average the per-worker averages.
    if (!workers_.empty()) {
        metrics.avg_latency_us /= static_cast<double>(workers_.size());
    }

    // Compute current TPS.
    auto elapsed = std::chrono::steady_clock::now() - start_time_;
    double elapsed_seconds = std::chrono::duration<double>(elapsed).count();
    if (elapsed_seconds > 0) {
        metrics.current_tps = static_cast<double>(metrics.total_orders_sent) / elapsed_seconds;
    }

    return metrics;
}

BotWorker::SubmitFunc BotPool::create_submit_fn() {
    auto client = std::make_shared<HttpClient>(config_.target_url);

    return [client](const Order& order) -> OrderResult {
        auto response = client->post_order(order);
        return OrderResult{
            .success = response.success,
            .latency = response.latency,
            .error_message = response.success ? "" : response.body,
        };
    };
}

} // namespace bot_engine
