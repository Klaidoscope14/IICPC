#include "telemetry/metrics_store.hpp"

namespace telemetry {

MetricsStore::MetricsStore(int max_snapshots)
    : max_snapshots_(max_snapshots)
{}

void MetricsStore::append(const AggregatedSnapshot& snapshot) {
    std::lock_guard<std::mutex> lock(mutex_);
    snapshots_.push_back(snapshot);
    while (static_cast<int>(snapshots_.size()) > max_snapshots_) {
        snapshots_.pop_front();
    }
}

const AggregatedSnapshot* MetricsStore::latest() const {
    std::lock_guard<std::mutex> lock(mutex_);
    if (snapshots_.empty()) return nullptr;
    return &snapshots_.back();
}

std::vector<AggregatedSnapshot> MetricsStore::last_n(int n) const {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<AggregatedSnapshot> result;
    int start = std::max(0, static_cast<int>(snapshots_.size()) - n);
    for (int i = static_cast<int>(snapshots_.size()) - 1; i >= start; --i) {
        result.push_back(snapshots_[i]);
    }
    return result;
}

std::vector<AggregatedSnapshot> MetricsStore::for_benchmark(const std::string& benchmark_id) const {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<AggregatedSnapshot> result;
    for (const auto& snap : snapshots_) {
        if (snap.benchmark_id == benchmark_id) {
            result.push_back(snap);
        }
    }
    return result;
}

size_t MetricsStore::size() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return snapshots_.size();
}

void MetricsStore::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    snapshots_.clear();
}

} // namespace telemetry
