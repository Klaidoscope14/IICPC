cmake_minimum_required(VERSION 3.20)
project(bot-engine VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# Find required packages.
find_package(Threads REQUIRED)

# Bot engine library.
add_library(bot_engine_lib
    src/config/bot_config.cpp
    src/bot/order_generator.cpp
    src/bot/bot_worker.cpp
    src/bot/bot_pool.cpp
    src/metrics/metrics_reporter.cpp
    src/net/http_client.cpp
)

target_include_directories(bot_engine_lib PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

target_link_libraries(bot_engine_lib PUBLIC
    Threads::Threads
)

# Main executable.
add_executable(bot_engine src/main.cpp)
target_link_libraries(bot_engine PRIVATE bot_engine_lib)

# Tests (optional).
option(BUILD_TESTS "Build unit tests" OFF)
if(BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()
