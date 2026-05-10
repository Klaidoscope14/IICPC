#pragma once

#include <string>
#include <chrono>
#include <random>
#include <vector>
#include "bot_engine/config.hpp"

namespace bot_engine {

/// Side of an order.
enum class Side { Buy, Sell };

/// Represents a trading order to send to the contestant's exchange.
struct Order {
    std::string id;
    std::string symbol;
    Side side;
    double price;
    int32_t quantity;
    std::string type = "limit"; // "limit" or "market"
};

/// Generates random trading orders based on configured parameters.
///
/// Thread-safe: each instance uses its own random engine.
class OrderGenerator {
public:
    explicit OrderGenerator(const BotConfig& config);

    /// Generate a single random order.
    Order generate();

    /// Generate a batch of random orders.
    std::vector<Order> generate_batch(int count);

private:
    BotConfig config_;
    std::mt19937 rng_;
    std::uniform_int_distribution<int> symbol_dist_;
    std::uniform_int_distribution<int> side_dist_;
    std::uniform_real_distribution<double> price_dist_;
    std::uniform_int_distribution<int> qty_dist_;
    uint64_t counter_{0};

    std::string next_id();
};

} // namespace bot_engine
