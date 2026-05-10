#pragma once

#include <string>
#include <chrono>
#include <cstdint>
#include <vector>

namespace validation {

enum class Side { Buy, Sell };

/// A single order from the trade log.
struct LogOrder {
    std::string id;
    std::string symbol;
    Side side;
    double price;
    int32_t quantity;
    int32_t remaining_quantity;
    std::chrono::system_clock::time_point timestamp;
    std::string type; // "limit", "market"
    bool is_cancel{false};
};

/// A single trade (fill) from the trade log.
struct LogTrade {
    std::string trade_id;
    std::string buy_order_id;
    std::string sell_order_id;
    std::string symbol;
    double price;
    int32_t quantity;
    std::chrono::system_clock::time_point timestamp;
};

/// The result of validating a single trade.
struct TradeValidation {
    std::string trade_id;
    bool valid;
    std::string reason; // Empty if valid.
};

/// Summary of the full validation run.
struct ValidationResult {
    int total_orders;
    int total_trades;
    int valid_trades;
    int invalid_trades;
    double correctness_score; // 0.0 - 100.0

    std::vector<TradeValidation> trade_validations;
    std::vector<std::string> violations; // Human-readable violation descriptions.
};

} // namespace validation
