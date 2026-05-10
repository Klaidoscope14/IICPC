#include "bot_engine/bot_worker.hpp"
#include <thread>
#include <chrono>

namespace bot_engine {

BotWorker::BotWorker(int worker_id, const BotConfig& config, SubmitFunc submit_fn)
    : worker_id_(worker_id)
    , config_(config)
    , submit_fn_(std::move(submit_fn))
    , generator_(config)
{}

void BotWorker::run() {
    running_ = true;

    // Calculate delay between orders for this worker.
    // Total OPS is split across all workers.
    int ops_per_worker = std::max(1, config_.orders_per_second / config_.bot_count);
    auto delay = std::chrono::microseconds(1'000'000 / ops_per_worker);

    auto end_time = std::chrono::steady_clock::now() +
                    std::chrono::seconds(config_.duration_seconds);

    while (running_ && std::chrono::steady_clock::now() < end_time) {
        auto order = generator_.generate();
        auto start = std::chrono::steady_clock::now();

        auto result = submit_fn_(order);

        orders_sent_++;
        if (result.success) {
            orders_acked_++;
        } else {
            orders_failed_++;
        }
        total_latency_us_ += result.latency.count();

        // Rate limiting: sleep to achieve target OPS.
        auto elapsed = std::chrono::steady_clock::now() - start;
        if (elapsed < delay) {
            std::this_thread::sleep_for(delay - elapsed);
        }
    }

    running_ = false;
}

void BotWorker::stop() {
    running_ = false;
}

} // namespace bot_engine
