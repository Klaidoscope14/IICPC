#pragma once

#include <string>
#include <vector>
#include <cstdint>

namespace bot_engine {

/// Configuration for the bot engine runtime.
struct BotConfig {
    /// Target endpoint URL of the contestant's exchange.
    std::string target_url = "http://localhost:8080";

    /// Number of concurrent bot workers.
    int32_t bot_count = 100;

    /// Duration of the benchmark in seconds.
    int32_t duration_seconds = 60;

    /// Target orders per second across all bots.
    int32_t orders_per_second = 1000;

    /// Protocols to use for order submission.
    std::vector<std::string> protocols = {"rest"};

    /// Trading symbols to generate orders for.
    std::vector<std::string> symbols = {"AAPL", "GOOGL", "MSFT"};

    /// Minimum order price.
    double min_price = 100.0;

    /// Maximum order price.
    double max_price = 200.0;

    /// Maximum order quantity.
    int32_t max_quantity = 100;

    /// ID of the benchmark this run belongs to (for reporting).
    std::string benchmark_id;

    /// Endpoint for reporting metrics back to the orchestrator.
    std::string metrics_endpoint = "http://localhost:8081";

    /// Load from environment variables with fallback to defaults.
    static BotConfig from_env();
};

} // namespace bot_engine
