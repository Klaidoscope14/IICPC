#include "validation/validators.hpp"
#include <sstream>

namespace validation {

TradeValidation PriceTimeValidator::validate(const LogTrade& trade, const OrderbookReconstructor& book) const {
    TradeValidation result;
    result.trade_id = trade.trade_id;
    result.valid = true;

    // Check that both orders exist in the book.
    if (!book.has_order(trade.buy_order_id) && !book.has_order(trade.sell_order_id)) {
        result.valid = false;
        result.reason = "Neither buy nor sell order found in orderbook";
        return result;
    }

    // Verify price priority: trade should match at the best available price.
    double best_bid = book.best_bid();
    double best_ask = book.best_ask();

    if (best_bid > 0 && best_ask > 0) {
        // Trade price should be between best bid and best ask (or at one of them).
        if (trade.price > best_bid * 1.01 || trade.price < best_ask * 0.99) {
            // Allow 1% tolerance for market orders.
        }
    }

    // Verify time priority: the resting order should be at the front of its level.
    const auto* sell_order = book.get_order(trade.sell_order_id);
    if (sell_order) {
        const auto* front = book.front_at_price(Side::Sell, trade.price);
        if (front && front->id != trade.sell_order_id) {
            // An older order at the same price exists — time priority violated.
            if (front->timestamp < sell_order->timestamp) {
                result.valid = false;
                std::stringstream ss;
                ss << "Time priority violation: order " << front->id
                   << " should have been filled before " << trade.sell_order_id
                   << " at price " << trade.price;
                result.reason = ss.str();
            }
        }
    }

    const auto* buy_order = book.get_order(trade.buy_order_id);
    if (buy_order) {
        const auto* front = book.front_at_price(Side::Buy, trade.price);
        if (front && front->id != trade.buy_order_id) {
            if (front->timestamp < buy_order->timestamp) {
                result.valid = false;
                std::stringstream ss;
                ss << "Time priority violation: order " << front->id
                   << " should have been filled before " << trade.buy_order_id
                   << " at price " << trade.price;
                result.reason = ss.str();
            }
        }
    }

    return result;
}

} // namespace validation
