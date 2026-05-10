#include "telemetry/sliding_window.hpp"

namespace telemetry {

SlidingWindow::SlidingWindow(std::chrono::seconds window_size)
    : window_size_(window_size)
{}

void SlidingWindow::record() {
    record_at(std::chrono::steady_clock::now());
}

void SlidingWindow::record_at(std::chrono::steady_clock::time_point tp) {
    std::lock_guard<std::mutex> lock(mutex_);
    events_.push_back(tp);
}

double SlidingWindow::rate() const {
    std::lock_guard<std::mutex> lock(mutex_);
    auto now = std::chrono::steady_clock::now();
    evict_old(now);

    double window_secs = std::chrono::duration<double>(window_size_).count();
    if (window_secs <= 0) return 0.0;

    return static_cast<double>(events_.size()) / window_secs;
}

int64_t SlidingWindow::count() const {
    std::lock_guard<std::mutex> lock(mutex_);
    auto now = std::chrono::steady_clock::now();
    evict_old(now);
    return static_cast<int64_t>(events_.size());
}

void SlidingWindow::reset() {
    std::lock_guard<std::mutex> lock(mutex_);
    events_.clear();
}

void SlidingWindow::evict_old(std::chrono::steady_clock::time_point now) const {
    auto cutoff = now - window_size_;
    auto& mutable_events = const_cast<std::deque<std::chrono::steady_clock::time_point>&>(events_);
    while (!mutable_events.empty() && mutable_events.front() < cutoff) {
        mutable_events.pop_front();
    }
}

} // namespace telemetry
