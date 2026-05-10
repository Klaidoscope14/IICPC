#include "validation/validators.hpp"
#include <sstream>

namespace validation {

TradeValidation FillValidator::validate(const LogTrade& trade, const OrderbookReconstructor& book) const {
    TradeValidation result;
    result.trade_id = trade.trade_id;
    result.valid = true;

    if (trade.quantity <= 0) {
        result.valid = false;
        result.reason = "Fill quantity must be positive";
        return result;
    }

    if (trade.price <= 0) {
        result.valid = false;
        result.reason = "Fill price must be positive";
        return result;
    }

    // Check buy order exists and has sufficient remaining quantity.
    const auto* buy_order = book.get_order(trade.buy_order_id);
    if (buy_order) {
        if (trade.quantity > buy_order->remaining_quantity) {
            result.valid = false;
            std::stringstream ss;
            ss << "Fill quantity " << trade.quantity
               << " exceeds buy order remaining " << buy_order->remaining_quantity
               << " (order " << trade.buy_order_id << ")";
            result.reason = ss.str();
            return result;
        }
    }

    // Check sell order exists and has sufficient remaining quantity.
    const auto* sell_order = book.get_order(trade.sell_order_id);
    if (sell_order) {
        if (trade.quantity > sell_order->remaining_quantity) {
            result.valid = false;
            std::stringstream ss;
            ss << "Fill quantity " << trade.quantity
               << " exceeds sell order remaining " << sell_order->remaining_quantity
               << " (order " << trade.sell_order_id << ")";
            result.reason = ss.str();
            return result;
        }
    }

    return result;
}

} // namespace validation
