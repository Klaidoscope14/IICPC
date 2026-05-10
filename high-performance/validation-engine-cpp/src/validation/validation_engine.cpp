#include "validation/validation_engine.hpp"

namespace validation {

void ValidationEngine::load_orders(const std::vector<LogOrder>& orders) {
    orders_ = orders;
}

void ValidationEngine::load_trades(const std::vector<LogTrade>& trades) {
    trades_ = trades;
}

void ValidationEngine::ensure_book(const std::string& symbol) {
    if (books_.find(symbol) == books_.end()) {
        books_[symbol] = std::make_unique<OrderbookReconstructor>(symbol);
    }
}

ValidationResult ValidationEngine::validate() {
    ValidationResult result;
    result.total_orders = static_cast<int>(orders_.size());
    result.total_trades = static_cast<int>(trades_.size());
    result.valid_trades = 0;
    result.invalid_trades = 0;

    // Phase 1: Replay orders into the reconstructed orderbook.
    size_t order_idx = 0;

    for (const auto& trade : trades_) {
        // Apply all orders that arrived before this trade.
        while (order_idx < orders_.size() &&
               orders_[order_idx].timestamp <= trade.timestamp) {
            const auto& order = orders_[order_idx];
            ensure_book(order.symbol);
            books_[order.symbol]->apply_order(order);
            order_idx++;
        }

        ensure_book(trade.symbol);
        auto& book = *books_[trade.symbol];

        // Phase 2: Validate price-time priority.
        auto pt_result = price_time_validator_.validate(trade, book);
        if (!pt_result.valid) {
            result.invalid_trades++;
            result.violations.push_back(pt_result.reason);
            result.trade_validations.push_back(pt_result);
        }

        // Phase 3: Validate fill correctness.
        auto fill_result = fill_validator_.validate(trade, book);
        if (!fill_result.valid) {
            result.invalid_trades++;
            result.violations.push_back(fill_result.reason);
            result.trade_validations.push_back(fill_result);
        }

        if (pt_result.valid && fill_result.valid) {
            result.valid_trades++;
            TradeValidation ok;
            ok.trade_id = trade.trade_id;
            ok.valid = true;
            result.trade_validations.push_back(ok);
        }

        // Phase 4: Update the book with the fill.
        book.apply_fill(trade.buy_order_id, trade.quantity);
        book.apply_fill(trade.sell_order_id, trade.quantity);
    }

    // Apply remaining orders.
    while (order_idx < orders_.size()) {
        const auto& order = orders_[order_idx];
        ensure_book(order.symbol);
        books_[order.symbol]->apply_order(order);
        order_idx++;
    }

    // Compute score.
    result.correctness_score = CorrectnessScorer::score(result);

    return result;
}

const OrderbookReconstructor* ValidationEngine::get_book(const std::string& symbol) const {
    auto it = books_.find(symbol);
    if (it != books_.end()) return it->second.get();
    return nullptr;
}

} // namespace validation
