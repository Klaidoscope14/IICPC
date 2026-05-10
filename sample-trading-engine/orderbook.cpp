#include "orderbook.hpp"
#include <algorithm>

std::vector<Trade> OrderBook::addOrder(std::shared_ptr<Order> order) {
    std::unique_lock lock(mutex_);

    auto trades = matchOrders(order);

    // If the order wasn't fully filled, rest it on the book.
    if (order->quantity > 0) {
        if (order->side == Side::Buy) {
            buy_orders_[order->price].push_back(order);
        } else {
            sell_orders_[order->price].push_back(order);
        }
    }

    return trades;
}

std::vector<Trade> OrderBook::matchOrders(std::shared_ptr<Order> incoming_order) {
    std::vector<Trade> new_trades;
    
    if (incoming_order->side == Side::Buy) {
        // Buy order matches against sell orders (lowest ask first).
        auto it = sell_orders_.begin();
        while (incoming_order->quantity > 0 && it != sell_orders_.end() && 
               incoming_order->price >= it->first) {
            
            auto& orders_at_price = it->second;
            auto order_it = orders_at_price.begin();
            
            while (incoming_order->quantity > 0 && order_it != orders_at_price.end()) {
                auto resting_order = *order_it;
                int trade_qty = std::min(incoming_order->quantity, resting_order->quantity);
                
                Trade trade(incoming_order->id, resting_order->id, symbol_, 
                           resting_order->price, trade_qty);
                trades_.push_back(trade);
                new_trades.push_back(trade);
                
                incoming_order->quantity -= trade_qty;
                resting_order->quantity -= trade_qty;
                
                if (resting_order->quantity == 0) {
                    order_it = orders_at_price.erase(order_it);
                } else {
                    ++order_it;
                }
            }
            
            if (orders_at_price.empty()) {
                it = sell_orders_.erase(it);
            } else {
                ++it;
            }
        }
    } else {
        // Sell order matches against buy orders (highest bid first).
        auto it = buy_orders_.begin();
        while (incoming_order->quantity > 0 && it != buy_orders_.end() && 
               incoming_order->price <= it->first) {
            
            auto& orders_at_price = it->second;
            auto order_it = orders_at_price.begin();
            
            while (incoming_order->quantity > 0 && order_it != orders_at_price.end()) {
                auto resting_order = *order_it;
                int trade_qty = std::min(incoming_order->quantity, resting_order->quantity);
                
                Trade trade(resting_order->id, incoming_order->id, symbol_, 
                           resting_order->price, trade_qty);
                trades_.push_back(trade);
                new_trades.push_back(trade);
                
                incoming_order->quantity -= trade_qty;
                resting_order->quantity -= trade_qty;
                
                if (resting_order->quantity == 0) {
                    order_it = orders_at_price.erase(order_it);
                } else {
                    ++order_it;
                }
            }
            
            if (orders_at_price.empty()) {
                it = buy_orders_.erase(it);
            } else {
                ++it;
            }
        }
    }
    
    return new_trades;
}

bool OrderBook::cancelOrder(const std::string& order_id) {
    std::unique_lock lock(mutex_);

    // Search in buy orders.
    for (auto& [price, orders] : buy_orders_) {
        auto it = std::find_if(orders.begin(), orders.end(),
            [&order_id](const std::shared_ptr<Order>& order) {
                return order->id == order_id;
            });
        
        if (it != orders.end()) {
            orders.erase(it);
            if (orders.empty()) {
                buy_orders_.erase(price);
            }
            return true;
        }
    }
    
    // Search in sell orders.
    for (auto& [price, orders] : sell_orders_) {
        auto it = std::find_if(orders.begin(), orders.end(),
            [&order_id](const std::shared_ptr<Order>& order) {
                return order->id == order_id;
            });
        
        if (it != orders.end()) {
            orders.erase(it);
            if (orders.empty()) {
                sell_orders_.erase(price);
            }
            return true;
        }
    }
    
    return false;
}

std::map<double, int> OrderBook::getBuyDepth() const {
    std::shared_lock lock(mutex_);

    std::map<double, int> depth;
    for (const auto& [price, orders] : buy_orders_) {
        int total_qty = 0;
        for (const auto& order : orders) {
            total_qty += order->quantity;
        }
        depth[price] = total_qty;
    }
    return depth;
}

std::map<double, int> OrderBook::getSellDepth() const {
    std::shared_lock lock(mutex_);

    std::map<double, int> depth;
    for (const auto& [price, orders] : sell_orders_) {
        int total_qty = 0;
        for (const auto& order : orders) {
            total_qty += order->quantity;
        }
        depth[price] = total_qty;
    }
    return depth;
}

std::pair<double, int> OrderBook::getBestBid() const {
    std::shared_lock lock(mutex_);

    if (buy_orders_.empty()) {
        return {0.0, 0};
    }
    
    auto it = buy_orders_.begin();
    int total_qty = 0;
    for (const auto& order : it->second) {
        total_qty += order->quantity;
    }
    return {it->first, total_qty};
}

std::pair<double, int> OrderBook::getBestAsk() const {
    std::shared_lock lock(mutex_);

    if (sell_orders_.empty()) {
        return {0.0, 0};
    }
    
    auto it = sell_orders_.begin();
    int total_qty = 0;
    for (const auto& order : it->second) {
        total_qty += order->quantity;
    }
    return {it->first, total_qty};
}

std::vector<Trade> OrderBook::getTrades() const {
    std::shared_lock lock(mutex_);
    return trades_;
}
