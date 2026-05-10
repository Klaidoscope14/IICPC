#pragma once

#include "validation/types.hpp"
#include "validation/orderbook_reconstructor.hpp"

namespace validation {

/// Validates that trades respect price-time priority.
///
/// For each trade, verifies:
/// 1. The trade price matches the resting order's price (price priority).
/// 2. The resting order was at the front of its price level (time priority).
/// 3. No better-priced resting orders were skipped.
class PriceTimeValidator {
public:
    /// Validate a single trade against the current orderbook state.
    TradeValidation validate(const LogTrade& trade, const OrderbookReconstructor& book) const;
};

/// Validates fill correctness.
///
/// For each trade, verifies:
/// 1. Fill quantity does not exceed the resting order's remaining quantity.
/// 2. Fill price is within valid range (between best bid and best ask).
/// 3. Both sides of the trade exist in the orderbook.
class FillValidator {
public:
    /// Validate a single trade's fill quantities and prices.
    TradeValidation validate(const LogTrade& trade, const OrderbookReconstructor& book) const;
};

} // namespace validation
