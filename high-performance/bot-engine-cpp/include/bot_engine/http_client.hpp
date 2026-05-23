#pragma once

#include "bot_engine/order_generator.hpp"
#include <chrono>
#include <string>
#include <netinet/in.h>

namespace bot_engine {

/// Simple HTTP client for submitting orders to a contestant's REST endpoint.
///
/// This is a minimal implementation using POSIX sockets.
/// For production, replace with Boost.Beast or libcurl.
class HttpClient {
public:
  explicit HttpClient(const std::string &base_url);
  ~HttpClient();

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

  int sock_ = -1;
  struct sockaddr_in server_addr_{};
  bool addr_resolved_ = false;

  void parse_url(const std::string &url);
  bool connect_socket();
  void disconnect_socket();
};

} // namespace bot_engine
