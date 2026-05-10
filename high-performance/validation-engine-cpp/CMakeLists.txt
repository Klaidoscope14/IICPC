cmake_minimum_required(VERSION 3.20)
project(validation-engine VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

find_package(Threads REQUIRED)

add_library(validation_engine_lib
    src/orderbook/order.cpp
    src/orderbook/orderbook_reconstructor.cpp
    src/validation/price_time_validator.cpp
    src/validation/fill_validator.cpp
    src/validation/validation_engine.cpp
    src/scoring/correctness_scorer.cpp
)

target_include_directories(validation_engine_lib PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

target_link_libraries(validation_engine_lib PUBLIC Threads::Threads)

add_executable(validation_engine src/main.cpp)
target_link_libraries(validation_engine PRIVATE validation_engine_lib)
