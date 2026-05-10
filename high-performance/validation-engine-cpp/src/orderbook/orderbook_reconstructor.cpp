#include "validation/orderbook_reconstructor.hpp"
#include <algorithm>

namespace validation {

OrderbookReconstructor::OrderbookReconstructor(const std::string& symbol)
    : symbol_(symbol)
{}

void OrderbookReconstructor::apply_order(const LogOrder& order) {
    if (order.is_cancel) {
        apply_cancel(order.id);
        return;
    }

    if (order.side == Side::Buy) {
        bids_[order.price].push_back(order);
    } else {
        asks_[order.price].push_back(order);
    }

    order_index_[order.id] = {order.side, order.price};
}

void OrderbookReconstructor::apply_cancel(const std::string& order_id) {
    auto it = order_index_.find(order_id);
    if (it == order_index_.end()) return;

    auto [side, price] = it->second;

    auto erase_from = [&](auto& book) {
        auto level_it = book.find(price);
        if (level_it != book.end()) {
            auto& orders = level_it->second;
            orders.erase(
                std::remove_if(orders.begin(), orders.end(),
                    [&](const LogOrder& o) { return o.id == order_id; }),
                orders.end()
            );
            if (orders.empty()) {
                book.erase(level_it);
            }
        }
    };

    if (side == Side::Buy) {
        erase_from(bids_);
    } else {
        erase_from(asks_);
    }

    order_index_.erase(it);
}

void OrderbookReconstructor::apply_fill(const std::string& order_id, int32_t fill_qty) {
    auto it = order_index_.find(order_id);
    if (it == order_index_.end()) return;

    auto [side, price] = it->second;

    auto fill_in = [&](auto& book) {
        auto level_it = book.find(price);
        if (level_it == book.end()) return;
        for (auto& order : level_it->second) {
            if (order.id == order_id) {
                order.remaining_quantity -= fill_qty;
                if (order.remaining_quantity <= 0) {
                    apply_cancel(order_id);
                }
                return;
            }
        }
    };

    if (side == Side::Buy) {
        fill_in(bids_);
    } else {
        fill_in(asks_);
    }
}

double OrderbookReconstructor::best_bid() const {
    if (bids_.empty()) return 0.0;
    return bids_.begin()->first;
}

double OrderbookReconstructor::best_ask() const {
    if (asks_.empty()) return 0.0;
    return asks_.begin()->first;
}

const LogOrder* OrderbookReconstructor::front_at_price(Side side, double price) const {
    if (side == Side::Buy) {
        auto it = bids_.find(price);
        if (it != bids_.end() && !it->second.empty()) {
            return &it->second.front();
        }
    } else {
        auto it = asks_.find(price);
        if (it != asks_.end() && !it->second.empty()) {
            return &it->second.front();
        }
    }
    return nullptr;
}

bool OrderbookReconstructor::has_order(const std::string& order_id) const {
    return order_index_.count(order_id) > 0;
}

const LogOrder* OrderbookReconstructor::get_order(const std::string& order_id) const {
    auto it = order_index_.find(order_id);
    if (it == order_index_.end()) return nullptr;

    auto [side, price] = it->second;

    auto find_in = [&](const auto& book) -> const LogOrder* {
        auto level_it = book.find(price);
        if (level_it == book.end()) return nullptr;
        for (const auto& order : level_it->second) {
            if (order.id == order_id) return &order;
        }
        return nullptr;
    };

    if (side == Side::Buy) {
        return find_in(bids_);
    } else {
        return find_in(asks_);
    }
}

} // namespace validation
