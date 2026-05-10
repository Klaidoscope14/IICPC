#include "bot_engine/order_generator.hpp"
#include <sstream>

namespace bot_engine {

OrderGenerator::OrderGenerator(const BotConfig& config)
    : config_(config)
    , rng_(std::random_device{}())
    , symbol_dist_(0, static_cast<int>(config.symbols.size()) - 1)
    , side_dist_(0, 1)
    , price_dist_(config.min_price, config.max_price)
    , qty_dist_(1, config.max_quantity)
{}

Order OrderGenerator::generate() {
    Order order;
    order.id = next_id();
    order.symbol = config_.symbols[symbol_dist_(rng_)];
    order.side = side_dist_(rng_) == 0 ? Side::Buy : Side::Sell;
    order.price = std::round(price_dist_(rng_) * 100.0) / 100.0;
    order.quantity = qty_dist_(rng_);
    order.type = "limit";
    return order;
}

std::vector<Order> OrderGenerator::generate_batch(int count) {
    std::vector<Order> orders;
    orders.reserve(count);
    for (int i = 0; i < count; ++i) {
        orders.push_back(generate());
    }
    return orders;
}

std::string OrderGenerator::next_id() {
    std::stringstream ss;
    ss << "BOT-" << ++counter_;
    return ss.str();
}

} // namespace bot_engine
