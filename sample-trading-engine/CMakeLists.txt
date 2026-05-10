cmake_minimum_required(VERSION 3.10)
project(TradingEngine VERSION 1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Add executable
add_executable(trading_engine
    main.cpp
    exchange.cpp
    orderbook.cpp
)

# Set compiler flags
target_compile_options(trading_engine PRIVATE -O3 -Wall -Wextra)

# Install target
install(TARGETS trading_engine DESTINATION bin)
