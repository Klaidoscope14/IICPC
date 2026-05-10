cmake_minimum_required(VERSION 3.20)
project(telemetry-core VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

find_package(Threads REQUIRED)

# Telemetry core library.
add_library(telemetry_core_lib
    src/config/telemetry_config.cpp
    src/ingestion/metrics_ingester.cpp
    src/aggregation/histogram.cpp
    src/aggregation/sliding_window.cpp
    src/aggregation/metrics_aggregator.cpp
    src/storage/metrics_store.cpp
    src/streaming/stream_server.cpp
)

target_include_directories(telemetry_core_lib PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

target_link_libraries(telemetry_core_lib PUBLIC
    Threads::Threads
)

# Main executable.
add_executable(telemetry_core src/main.cpp)
target_link_libraries(telemetry_core PRIVATE telemetry_core_lib)

# Tests.
option(BUILD_TESTS "Build unit tests" OFF)
if(BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()
