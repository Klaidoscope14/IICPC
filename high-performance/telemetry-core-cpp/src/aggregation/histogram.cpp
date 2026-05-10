#include "telemetry/histogram.hpp"

namespace telemetry {

Histogram::Histogram(int64_t max_value_us)
    : buckets_(BUCKET_COUNT, 0)
    , max_value_(max_value_us)
    , bucket_width_(max_value_us / BUCKET_COUNT)
{}

void Histogram::record(int64_t value_us) {
    int idx = static_cast<int>(value_us / bucket_width_);
    if (idx >= BUCKET_COUNT) idx = BUCKET_COUNT - 1;
    if (idx < 0) idx = 0;

    buckets_[idx]++;
    count_++;
    sum_ += value_us;

    if (value_us < min_) min_ = value_us;
    if (value_us > max_) max_ = value_us;
}

double Histogram::percentile(double p) const {
    if (count_ == 0) return 0.0;

    int64_t threshold = static_cast<int64_t>(p * count_);
    int64_t cumulative = 0;

    for (int i = 0; i < BUCKET_COUNT; ++i) {
        cumulative += buckets_[i];
        if (cumulative >= threshold) {
            // Return the midpoint of the bucket.
            return static_cast<double>(i * bucket_width_ + bucket_width_ / 2);
        }
    }

    return static_cast<double>(max_value_);
}

double Histogram::mean() const {
    if (count_ == 0) return 0.0;
    return static_cast<double>(sum_) / static_cast<double>(count_);
}

void Histogram::reset() {
    std::fill(buckets_.begin(), buckets_.end(), 0);
    count_ = 0;
    sum_ = 0;
    min_ = std::numeric_limits<int64_t>::max();
    max_ = 0;
}

} // namespace telemetry
