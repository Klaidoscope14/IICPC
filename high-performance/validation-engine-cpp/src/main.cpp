#include "validation/validation_engine.hpp"
#include <iostream>
#include <fstream>
#include <sstream>

using namespace validation;

/// Parse orders from a CSV file: id,symbol,side,price,quantity,timestamp_epoch_us,type
std::vector<LogOrder> load_orders_csv(const std::string& filename) {
    std::vector<LogOrder> orders;
    std::ifstream file(filename);
    if (!file.is_open()) {
        std::cerr << "Cannot open orders file: " << filename << std::endl;
        return orders;
    }

    std::string line;
    std::getline(file, line); // Skip header.

    while (std::getline(file, line)) {
        std::stringstream ss(line);
        LogOrder order;
        std::string side_str, ts_str;

        std::getline(ss, order.id, ',');
        std::getline(ss, order.symbol, ',');
        std::getline(ss, side_str, ',');

        std::string price_str, qty_str;
        std::getline(ss, price_str, ',');
        std::getline(ss, qty_str, ',');
        std::getline(ss, ts_str, ',');
        std::getline(ss, order.type, ',');

        order.side = (side_str == "buy") ? Side::Buy : Side::Sell;
        order.price = std::stod(price_str);
        order.quantity = std::stoi(qty_str);
        order.remaining_quantity = order.quantity;

        int64_t epoch_us = std::stoll(ts_str);
        order.timestamp = std::chrono::system_clock::time_point(
            std::chrono::microseconds(epoch_us));

        orders.push_back(order);
    }

    return orders;
}

/// Parse trades from a CSV file: trade_id,buy_order_id,sell_order_id,symbol,price,quantity,timestamp_epoch_us
std::vector<LogTrade> load_trades_csv(const std::string& filename) {
    std::vector<LogTrade> trades;
    std::ifstream file(filename);
    if (!file.is_open()) {
        std::cerr << "Cannot open trades file: " << filename << std::endl;
        return trades;
    }

    std::string line;
    std::getline(file, line); // Skip header.

    while (std::getline(file, line)) {
        std::stringstream ss(line);
        LogTrade trade;
        std::string price_str, qty_str, ts_str;

        std::getline(ss, trade.trade_id, ',');
        std::getline(ss, trade.buy_order_id, ',');
        std::getline(ss, trade.sell_order_id, ',');
        std::getline(ss, trade.symbol, ',');
        std::getline(ss, price_str, ',');
        std::getline(ss, qty_str, ',');
        std::getline(ss, ts_str, ',');

        trade.price = std::stod(price_str);
        trade.quantity = std::stoi(qty_str);

        int64_t epoch_us = std::stoll(ts_str);
        trade.timestamp = std::chrono::system_clock::time_point(
            std::chrono::microseconds(epoch_us));

        trades.push_back(trade);
    }

    return trades;
}

int main(int argc, char* argv[]) {
    std::cout << "=== IICPC Validation Engine ===" << std::endl;

    if (argc < 3) {
        std::cerr << "Usage: validation_engine <orders.csv> <trades.csv>" << std::endl;
        std::cerr << std::endl;
        std::cerr << "Orders CSV: id,symbol,side,price,quantity,timestamp_epoch_us,type" << std::endl;
        std::cerr << "Trades CSV: trade_id,buy_order_id,sell_order_id,symbol,price,quantity,timestamp_epoch_us" << std::endl;
        return 1;
    }

    std::string orders_file = argv[1];
    std::string trades_file = argv[2];

    auto orders = load_orders_csv(orders_file);
    auto trades = load_trades_csv(trades_file);

    std::cout << "Loaded " << orders.size() << " orders and " << trades.size() << " trades" << std::endl;

    ValidationEngine engine;
    engine.load_orders(orders);
    engine.load_trades(trades);

    auto result = engine.validate();

    std::cout << "\n=== Validation Results ===" << std::endl;
    std::cout << "Total Orders:      " << result.total_orders << std::endl;
    std::cout << "Total Trades:      " << result.total_trades << std::endl;
    std::cout << "Valid Trades:      " << result.valid_trades << std::endl;
    std::cout << "Invalid Trades:    " << result.invalid_trades << std::endl;
    std::cout << "Correctness Score: " << result.correctness_score << "/100" << std::endl;

    if (!result.violations.empty()) {
        std::cout << "\n--- Violations ---" << std::endl;
        int shown = 0;
        for (const auto& v : result.violations) {
            std::cout << "  • " << v << std::endl;
            if (++shown >= 20) {
                std::cout << "  ... and " << (result.violations.size() - shown) << " more" << std::endl;
                break;
            }
        }
    }

    std::cout << "\n=== Validation Complete ===" << std::endl;
    return (result.correctness_score >= 95.0) ? 0 : 1;
}
