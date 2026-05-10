#include "exchange.hpp"

std::vector<Trade> Exchange::addOrder(std::shared_ptr<Order> order) {
    // Ensure the orderbook exists (takes exclusive lock briefly).
    createOrderBook(order->symbol);

    // The orderbook has its own internal mutex, so we only need a shared lock here
    // to protect the map lookup.
    std::shared_lock lock(mutex_);
    auto it = orderbooks_.find(order->symbol);
    auto trades = it->second->addOrder(order);

    // Upgrade to exclusive lock to store trades in the global log.
    lock.unlock();
    std::unique_lock write_lock(mutex_);
    all_trades_.insert(all_trades_.end(), trades.begin(), trades.end());

    return trades;
}

bool Exchange::cancelOrder(const std::string& order_id, const std::string& symbol) {
    std::shared_lock lock(mutex_);

    auto it = orderbooks_.find(symbol);
    if (it == orderbooks_.end()) {
        return false;
    }
    
    return it->second->cancelOrder(order_id);
}

OrderBook* Exchange::getOrderBook(const std::string& symbol) {
    std::shared_lock lock(mutex_);

    auto it = orderbooks_.find(symbol);
    if (it == orderbooks_.end()) {
        return nullptr;
    }
    
    return it->second.get();
}

void Exchange::createOrderBook(const std::string& symbol) {
    std::unique_lock lock(mutex_);

    if (orderbooks_.find(symbol) == orderbooks_.end()) {
        orderbooks_[symbol] = std::make_unique<OrderBook>(symbol);
    }
}

std::vector<Trade> Exchange::getAllTrades() const {
    std::shared_lock lock(mutex_);
    return all_trades_;
}

std::vector<std::string> Exchange::getSymbols() const {
    std::shared_lock lock(mutex_);

    std::vector<std::string> symbols;
    symbols.reserve(orderbooks_.size());
    for (const auto& [symbol, _] : orderbooks_) {
        symbols.push_back(symbol);
    }
    return symbols;
}
