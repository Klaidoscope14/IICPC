#include "bot_engine/http_client.hpp"
#include <sstream>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <unistd.h>
#include <cstring>
#include <mutex>

namespace bot_engine {

static std::mutex dns_mutex;

HttpClient::HttpClient(const std::string& base_url)
    : base_url_(base_url)
{
    parse_url(base_url);
}

HttpClient::~HttpClient() {
    disconnect_socket();
}

void HttpClient::parse_url(const std::string& url) {
    // Simple URL parsing: http://host:port
    std::string stripped = url;
    if (stripped.starts_with("http://")) {
        stripped = stripped.substr(7);
    }

    auto colon_pos = stripped.find(':');
    if (colon_pos != std::string::npos) {
        host_ = stripped.substr(0, colon_pos);
        port_ = std::stoi(stripped.substr(colon_pos + 1));
    } else {
        host_ = stripped;
        port_ = 80;
    }
}

void HttpClient::disconnect_socket() {
    if (sock_ >= 0) {
        close(sock_);
        sock_ = -1;
    }
}

bool HttpClient::connect_socket() {
    if (sock_ >= 0) {
        return true; // Already connected
    }

    if (!addr_resolved_) {
        std::lock_guard<std::mutex> lock(dns_mutex);
        struct hostent* server = gethostbyname(host_.c_str());
        if (!server) {
            return false;
        }
        addr_resolved_ = true;
        server_addr_.sin_family = AF_INET;
        server_addr_.sin_port = htons(port_);
        std::memcpy(&server_addr_.sin_addr.s_addr, server->h_addr, server->h_length);
    }

    sock_ = socket(AF_INET, SOCK_STREAM, 0);
    if (sock_ < 0) {
        return false;
    }

    if (connect(sock_, (struct sockaddr*)&server_addr_, sizeof(server_addr_)) < 0) {
        disconnect_socket();
        return false;
    }

    return true;
}

HttpClient::Response HttpClient::post_order(const Order& order) {
    auto start = std::chrono::steady_clock::now();

    // Build JSON body.
    std::stringstream body;
    body << R"({"id":")" << order.id
         << R"(","symbol":")" << order.symbol
         << R"(","side":")" << (order.side == Side::Buy ? "buy" : "sell")
         << R"(","price":)" << order.price
         << R"(,"quantity":)" << order.quantity
         << R"(,"type":")" << order.type << R"("})";

    std::string body_str = body.str();

    // Build HTTP request.
    std::stringstream request;
    request << "POST /api/v1/orders HTTP/1.1\r\n"
            << "Host: " << host_ << ":" << port_ << "\r\n"
            << "Content-Type: application/json\r\n"
            << "Content-Length: " << body_str.size() << "\r\n"
            << "Connection: keep-alive\r\n"
            << "\r\n"
            << body_str;

    std::string req_str = request.str();

    // Reconnect logic
    for (int retry = 0; retry < 2; ++retry) {
        if (!connect_socket()) {
            auto latency = std::chrono::duration_cast<std::chrono::microseconds>(
                std::chrono::steady_clock::now() - start);
            return {false, 0, "connection failed", latency};
        }

        // Send request.
        ssize_t sent = send(sock_, req_str.c_str(), req_str.size(), 0);
        if (sent < 0) {
            disconnect_socket();
            continue; // try again
        }

        // Read response (simple: just read first chunk).
        char buffer[4096];
        int bytes_read = recv(sock_, buffer, sizeof(buffer) - 1, 0);
        
        if (bytes_read <= 0) {
            disconnect_socket();
            continue; // try again
        }

        auto latency = std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - start);

        buffer[bytes_read] = '\0';
        std::string response_str(buffer);

        // Parse status code from "HTTP/1.1 XXX ...".
        int status_code = 0;
        auto space_pos = response_str.find(' ');
        if (space_pos != std::string::npos) {
            status_code = std::stoi(response_str.substr(space_pos + 1, 3));
        }

        bool success = (status_code >= 200 && status_code < 300);
        return {success, status_code, response_str, latency};
    }

    auto latency = std::chrono::duration_cast<std::chrono::microseconds>(
        std::chrono::steady_clock::now() - start);
    return {false, 0, "no response after retry", latency};
}

} // namespace bot_engine
