#pragma once

#include "telemetry/types.hpp"
#include <deque>
#include <chrono>
#include <mutex>

namespace telemetry {

/// Thread-safe sliding window counter for TPS calculation.
///
/// Maintains a time-ordered deque of event timestamps and computes
/// the count per second over a configurable window.
class SlidingWindow {
public:
    /// Create a sliding window with the given duration.
    explicit SlidingWindow(std::chrono::seconds window_size = std::chrono::seconds(1));

    /// Record an event at the current time.
    void record();

    /// Record an event at a specific time.
    void record_at(std::chrono::steady_clock::time_point tp);

    /// Get the current rate (events per second).
    double rate() const;

    /// Get the total count within the current window.
    int64_t count() const;

    /// Reset the window.
    void reset();

private:
    std::chrono::seconds window_size_;
    mutable std::mutex mutex_;
    std::deque<std::chrono::steady_clock::time_point> events_;

    void evict_old(std::chrono::steady_clock::time_point now) const;
};

} // namespace telemetry
