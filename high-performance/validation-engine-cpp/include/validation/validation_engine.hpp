#pragma once

#include "validation/types.hpp"
#include "validation/orderbook_reconstructor.hpp"
#include "validation/validators.hpp"
#include <unordered_map>
#include <memory>

namespace validation {

/// Main validation engine that processes trade logs and produces a ValidationResult.
///
/// Usage:
///   ValidationEngine engine;
///   engine.load_orders(orders);
///   engine.load_trades(trades);
///   auto result = engine.validate();
class ValidationEngine {
public:
    /// Load orders from the trade log.
    void load_orders(const std::vector<LogOrder>& orders);

    /// Load trades from the trade log.
    void load_trades(const std::vector<LogTrade>& trades);

    /// Run validation and return results.
    ValidationResult validate();

    /// Get the reconstructed orderbook for a symbol.
    const OrderbookReconstructor* get_book(const std::string& symbol) const;

private:
    std::vector<LogOrder> orders_;
    std::vector<LogTrade> trades_;
    std::unordered_map<std::string, std::unique_ptr<OrderbookReconstructor>> books_;

    PriceTimeValidator price_time_validator_;
    FillValidator fill_validator_;

    void ensure_book(const std::string& symbol);
};

/// Computes a correctness score from validation results.
class CorrectnessScorer {
public:
    /// Compute a 0-100 score from validation results.
    static double score(const ValidationResult& result);
};

} // namespace validation
