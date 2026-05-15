#include "Exchange.h"
#include <iostream>
#include <stdexcept>

namespace Mercury {

void Exchange::createOrderBook(const std::string& symbol) {
    std::unique_lock lock(mutex_);
    if (engines_.find(symbol) == engines_.end()) {
        engines_[symbol] = std::make_unique<MatchingEngine>();
    }
}

MatchingEngine* Exchange::getEngine(const std::string& symbol) {
    auto it = engines_.find(symbol);
    if (it != engines_.end()) {
        return it->second.get();
    }
    return nullptr;
}

std::vector<std::string> Exchange::getSymbols() const {
    std::shared_lock lock(mutex_);
    std::vector<std::string> symbols;
    symbols.reserve(engines_.size());
    for (const auto& pair : engines_) {
        symbols.push_back(pair.first);
    }
    return symbols;
}

ExecutionResult Exchange::submitOrder(const std::string& symbol, Order order) {
    std::shared_lock lock(mutex_);
    MatchingEngine* engine = getEngine(symbol);
    if (!engine) {
        ExecutionResult result;
        result.status = ExecutionStatus::Rejected;
        result.orderId = order.orderId;
        return result;
    }
    return engine->submitOrder(std::move(order));
}

ExecutionResult Exchange::cancelOrder(const std::string& symbol, uint64_t orderId) {
    std::shared_lock lock(mutex_);
    MatchingEngine* engine = getEngine(symbol);
    if (!engine) {
        ExecutionResult result;
        result.status = ExecutionStatus::Rejected;
        result.orderId = orderId;
        return result;
    }
    return engine->cancelOrder(orderId);
}

ExecutionResult Exchange::modifyOrder(const std::string& symbol, uint64_t orderId, int64_t newPrice, uint64_t newQuantity) {
    std::shared_lock lock(mutex_);
    MatchingEngine* engine = getEngine(symbol);
    if (!engine) {
        ExecutionResult result;
        result.status = ExecutionStatus::Rejected;
        result.orderId = orderId;
        return result;
    }
    return engine->modifyOrder(orderId, newPrice, newQuantity);
}

MarketSnapshot Exchange::getSnapshot(const std::string& symbol, size_t maxLevels) {
    std::shared_lock lock(mutex_);
    MatchingEngine* engine = getEngine(symbol);
    if (!engine) {
        return MarketSnapshot{};
    }
    return engine->getOrderBook().getSnapshot(maxLevels);
}

} // namespace Mercury
