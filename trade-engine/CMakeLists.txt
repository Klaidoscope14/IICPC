cmake_minimum_required(VERSION 3.10)

if(POLICY CMP0169)
    cmake_policy(SET CMP0169 OLD)
endif()

project(IICPC_trade_engine_simple VERSION 1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED True)

# ============================================================================
# THREADING SUPPORT
# ============================================================================
set(THREADS_PREFER_PTHREAD_FLAG ON)
find_package(Threads REQUIRED)

# ============================================================================
# COMPILER FLAGS
# ============================================================================
if(MSVC)
    set(CMAKE_CXX_FLAGS_RELWITHDEBINFO "/O2 /Zi /DNDEBUG")
    add_compile_options(/W4)
else()
    set(CMAKE_CXX_FLAGS_RELWITHDEBINFO "-O2 -g -DNDEBUG")
endif()

# ============================================================================
# DEPENDENCIES
# ============================================================================
include(FetchContent)

# Abseil (Required for O(1) hash maps and fast ladders)
set(ABSL_PROPAGATE_CXX_STD ON CACHE BOOL "" FORCE)
FetchContent_Declare(
    abseil_cpp
    GIT_REPOSITORY https://github.com/abseil/abseil-cpp.git
    GIT_TAG        20240116.2
)
FetchContent_MakeAvailable(abseil_cpp)

# Google Test
FetchContent_Declare(
    googletest
    GIT_REPOSITORY https://github.com/google/googletest.git
    GIT_TAG        v1.15.2
)
set(gtest_force_shared_crt ON CACHE BOOL "" FORCE)
FetchContent_MakeAvailable(googletest)

# ============================================================================
# CORE ENGINE LIBRARY
# ============================================================================
set(MERCURY_CORE_SOURCES
    src/OrderBook.cpp
    src/MatchingEngine.cpp
    src/Exchange.cpp
)

set(MERCURY_HEADERS
    include/Order.h
    include/OrderBook.h
    include/OrderNode.h
    include/PriceLevel.h
    include/HashMap.h
    include/IntrusiveList.h
    include/ObjectPool.h
    include/MatchingEngine.h
    include/Exchange.h
    include/MarketData.h
)

add_library(iicpc_engine_lib
    ${MERCURY_CORE_SOURCES}
    ${MERCURY_HEADERS}
)
target_include_directories(iicpc_engine_lib PUBLIC include)
target_link_libraries(iicpc_engine_lib PUBLIC Threads::Threads absl::flat_hash_map absl::btree)

# ============================================================================
# EXECUTABLE
# ============================================================================
add_executable(iicpc_engine
    src/main.cpp
)
target_link_libraries(iicpc_engine PRIVATE iicpc_engine_lib)

# ============================================================================
# TESTS
# ============================================================================
enable_testing()

add_executable(iicpc_engine_tests
    tests/order_book_test.cpp
    tests/matching_engine_test.cpp
)
target_link_libraries(iicpc_engine_tests
    iicpc_engine_lib
    GTest::gtest_main
)

include(GoogleTest)
gtest_discover_tests(iicpc_engine_tests)
