#pragma once

#include "telemetry/types.hpp"
#include <vector>
#include <deque>
#include <mutex>

namespace telemetry {

/// In-memory metrics store with bounded retention.
///
/// Stores aggregated snapshots in a ring buffer and provides
/// query access for the streaming server.
class MetricsStore {
public:
    explicit MetricsStore(int max_snapshots = 3600);

    /// Append a new snapshot.
    void append(const AggregatedSnapshot& snapshot);

    /// Get the latest snapshot, or nullptr if empty.
    const AggregatedSnapshot* latest() const;

    /// Get the last N snapshots (most recent first).
    std::vector<AggregatedSnapshot> last_n(int n) const;

    /// Get all snapshots for a benchmark.
    std::vector<AggregatedSnapshot> for_benchmark(const std::string& benchmark_id) const;

    /// Get the total number of stored snapshots.
    size_t size() const;

    /// Clear all snapshots.
    void clear();

private:
    int max_snapshots_;
    mutable std::mutex mutex_;
    std::deque<AggregatedSnapshot> snapshots_;
};

} // namespace telemetry
