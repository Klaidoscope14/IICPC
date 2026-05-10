#include "bot_engine/config.hpp"
#include <cstdlib>

namespace bot_engine {

BotConfig BotConfig::from_env() {
    BotConfig config;

    if (auto* val = std::getenv("TARGET_URL"))        config.target_url = val;
    if (auto* val = std::getenv("BOT_COUNT"))          config.bot_count = std::atoi(val);
    if (auto* val = std::getenv("DURATION_SECONDS"))   config.duration_seconds = std::atoi(val);
    if (auto* val = std::getenv("ORDERS_PER_SECOND"))  config.orders_per_second = std::atoi(val);
    if (auto* val = std::getenv("BENCHMARK_ID"))       config.benchmark_id = val;
    if (auto* val = std::getenv("METRICS_ENDPOINT"))   config.metrics_endpoint = val;

    return config;
}

} // namespace bot_engine
