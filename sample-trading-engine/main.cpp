#include "exchange.hpp"
#include <iostream>
#include <thread>
#include <chrono>
#include <random>
#include <sstream>

/// TradingEngine orchestrates the exchange and simulates trading activity.
///
/// Owns the Exchange instance and a background worker thread that generates
/// random orders. Call start() to begin and stop() for a clean shutdown.
class TradingEngine {
private:
    Exchange exchange_;
    std::atomic<bool> running_{false};
    std::thread worker_;
    uint64_t order_counter_{1};
    
public:
    ~TradingEngine() {
        stop();
    }

    /// Start the trading engine and spawn the background simulation thread.
    void start() {
        running_ = true;
        std::cout << "Trading Engine Started" << std::endl;
        
        // Create some initial orderbooks.
        exchange_.createOrderBook("AAPL");
        exchange_.createOrderBook("GOOGL");
        exchange_.createOrderBook("MSFT");
        
        // Launch simulation on a joinable thread.
        worker_ = std::thread([this]() { simulateTrading(); });
    }
    
    /// Gracefully stop the engine and wait for the worker thread to finish.
    void stop() {
        if (!running_.exchange(false)) {
            return; // Already stopped.
        }
        
        if (worker_.joinable()) {
            worker_.join();
        }
        
        std::cout << "Trading Engine Stopped" << std::endl;
    }
    
    /// Print market depth for a given symbol.
    void printMarketDepth(const std::string& symbol) {
        auto* orderbook = exchange_.getOrderBook(symbol);
        if (!orderbook) {
            std::cout << "No orderbook for " << symbol << std::endl;
            return;
        }
        
        auto [best_bid, bid_qty] = orderbook->getBestBid();
        auto [best_ask, ask_qty] = orderbook->getBestAsk();
        
        std::cout << "\n=== " << symbol << " Market Depth ===" << std::endl;
        std::cout << "Best Bid: " << best_bid << " x" << bid_qty << std::endl;
        std::cout << "Best Ask: " << best_ask << " x" << ask_qty << std::endl;
        std::cout << "Spread: " << (best_ask - best_bid) << std::endl;
    }
    
    /// Print aggregate trading statistics.
    void printStats() {
        auto all_trades = exchange_.getAllTrades();
        std::cout << "\n=== Trading Statistics ===" << std::endl;
        std::cout << "Total Trades: " << all_trades.size() << std::endl;
        
        if (!all_trades.empty()) {
            double total_volume = 0;
            for (const auto& trade : all_trades) {
                total_volume += trade.price * trade.quantity;
            }
            std::cout << "Total Volume: $" << total_volume << std::endl;
        }
        
        auto symbols = exchange_.getSymbols();
        std::cout << "Active Symbols: ";
        for (size_t i = 0; i < symbols.size(); ++i) {
            std::cout << symbols[i];
            if (i < symbols.size() - 1) std::cout << ", ";
        }
        std::cout << std::endl;
    }

private:
    /// Background worker: generates random orders until running_ is set to false.
    void simulateTrading() {
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> symbol_dist(0, 2);
        std::uniform_int_distribution<> side_dist(0, 1);
        std::uniform_real_distribution<> price_dist(100.0, 200.0);
        std::uniform_int_distribution<> qty_dist(1, 100);
        
        const std::vector<std::string> symbols = {"AAPL", "GOOGL", "MSFT"};
        
        while (running_) {
            std::string symbol = symbols[symbol_dist(gen)];
            Side side = side_dist(gen) == 0 ? Side::Buy : Side::Sell;
            double price = std::round(price_dist(gen) * 100) / 100.0;
            int quantity = qty_dist(gen);
            
            std::string order_id = generateOrderId();
            auto order = std::make_shared<Order>(order_id, symbol, side, price, quantity);
            
            auto trades = exchange_.addOrder(order);
            
            if (!trades.empty()) {
                for (const auto& trade : trades) {
                    std::cout << "TRADE: " << trade.symbol << " " << trade.price 
                              << " x" << trade.quantity << std::endl;
                }
            }
            
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
        }
    }
    
    /// Generate a sequential order ID (e.g., "ORD1", "ORD2", ...).
    std::string generateOrderId() {
        std::stringstream ss;
        ss << "ORD" << order_counter_++;
        return ss.str();
    }
};

int main() {
    TradingEngine engine;
    
    engine.start();
    
    // Run for demonstration.
    std::this_thread::sleep_for(std::chrono::seconds(5));
    
    // Print some statistics.
    engine.printMarketDepth("AAPL");
    engine.printStats();
    
    engine.stop();
    
    std::cout << "Trading Engine Demo Complete" << std::endl;
    return 0;
}
