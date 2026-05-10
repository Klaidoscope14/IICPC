#include "validation/validation_engine.hpp"

namespace validation {

double CorrectnessScorer::score(const ValidationResult& result) {
    if (result.total_trades == 0) return 100.0;

    double valid_ratio = static_cast<double>(result.valid_trades) /
                         static_cast<double>(result.total_trades);

    // Penalty for violations: each violation costs proportionally.
    double violation_penalty = 0.0;
    if (!result.violations.empty()) {
        violation_penalty = static_cast<double>(result.violations.size()) /
                           static_cast<double>(result.total_trades) * 10.0;
    }

    double score = (valid_ratio * 100.0) - violation_penalty;
    if (score < 0.0) score = 0.0;
    if (score > 100.0) score = 100.0;

    return score;
}

} // namespace validation
