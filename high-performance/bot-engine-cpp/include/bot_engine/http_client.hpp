#pragma once

#include "bot_engine/order_generator.hpp"
#include <chrono>
#include <string>

namespace bot_engine {

/// Simple HTTP client for submitting orders to a contestant's REST endpoint.
///
/// This is a minimal implementation using POSIX sockets.
/// For production, replace with Boost.Beast or libcurl.
class HttpClient {
public:
  explicit HttpClient(const std::string &base_url);

  /// POST an order as JSON to the target endpoint.
  /// Returns {success, latency, error_message}.
  struct Response {
    bool success;
    int status_code;
    std::string body;
    std::chrono::microseconds latency;
  };

  Response post_order(const Order &order);

private:
  std::string base_url_;
  std::string host_;
  int port_;

  void parse_url(const std::string &url);
};

} // namespace bot_engine
