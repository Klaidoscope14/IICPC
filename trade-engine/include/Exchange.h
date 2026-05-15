#pragma once

#include "MatchingEngine.h"
#include <string>
#include <unordered_map>
#include <memory>
#include <shared_mutex>
#include <vector>

namespace Mercury {

class Exchange {
public:
    Exchange() = default;
    ~Exchange() = default;

    // Delete copy and move
    Exchange(const Exchange&) = delete;
    Exchange& operator=(const Exchange&) = delete;
    Exchange(Exchange&&) = delete;
    Exchange& operator=(Exchange&&) = delete;

    /**
     * @brief Create a new order book / matching engine for a symbol
     */
    void createOrderBook(const std::string& symbol);

    /**
     * @brief Submit an order to the given symbol
     */
    ExecutionResult submitOrder(const std::string& symbol, Order order);

    /**
     * @brief Cancel an order by ID for the given symbol
     */
    ExecutionResult cancelOrder(const std::string& symbol, uint64_t orderId);

    /**
     * @brief Modify an order by ID for the given symbol
     */
    ExecutionResult modifyOrder(const std::string& symbol, uint64_t orderId, int64_t newPrice, uint64_t newQuantity);

    /**
     * @brief Get a list of all supported symbols
     */
    std::vector<std::string> getSymbols() const;

    /**
     * @brief Get market depth (snapshot) for a symbol up to N levels
     */
    MarketSnapshot getSnapshot(const std::string& symbol, size_t maxLevels = 5);

private:
    mutable std::shared_mutex mutex_;
    std::unordered_map<std::string, std::unique_ptr<MatchingEngine>> engines_;

    MatchingEngine* getEngine(const std::string& symbol);
};

} // namespace Mercury
