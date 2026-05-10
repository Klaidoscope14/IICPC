#pragma once

#include "telemetry/types.hpp"
#include <vector>
#include <cstdint>
#include <algorithm>
#include <cmath>
#include <numeric>

namespace telemetry {

/// Lock-free histogram for computing latency percentiles.
///
/// Uses a fixed-bucket approach with microsecond resolution.
/// Optimized for low-overhead insertion in the hot path.
class Histogram {
public:
    /// Create a histogram with the given max value (in microseconds).
    explicit Histogram(int64_t max_value_us = 10'000'000); // 10 seconds max

    /// Record a single latency observation.
    void record(int64_t value_us);

    /// Get the value at a given percentile (0.0 - 1.0).
    double percentile(double p) const;

    /// Get the arithmetic mean.
    double mean() const;

    /// Get the minimum recorded value.
    int64_t min() const { return min_; }

    /// Get the maximum recorded value.
    int64_t max() const { return max_; }

    /// Get the total count of recorded values.
    int64_t count() const { return count_; }

    /// Reset all counters.
    void reset();

private:
    static constexpr int BUCKET_COUNT = 1000;

    std::vector<int64_t> buckets_;
    int64_t max_value_;
    int64_t bucket_width_;
    int64_t count_{0};
    int64_t sum_{0};
    int64_t min_{std::numeric_limits<int64_t>::max()};
    int64_t max_{0};
};

} // namespace telemetry
