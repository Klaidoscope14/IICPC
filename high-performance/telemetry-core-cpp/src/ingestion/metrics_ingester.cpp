#include "telemetry/metrics_ingester.hpp"
#include <sys/socket.h>
#include <netinet/in.h>
#include <unistd.h>
#include <cstring>
#include <iostream>
#include <sstream>

namespace telemetry {

MetricsIngester::MetricsIngester(int port, MetricCallback callback)
    : port_(port)
    , callback_(std::move(callback))
{}

MetricsIngester::~MetricsIngester() {
    stop();
}

void MetricsIngester::start() {
    if (running_) return;
    running_ = true;
    server_thread_ = std::thread([this]() { run(); });
}

void MetricsIngester::stop() {
    running_ = false;
    if (server_fd_ >= 0) {
        shutdown(server_fd_, SHUT_RDWR);
        close(server_fd_);
        server_fd_ = -1;
    }
    if (server_thread_.joinable()) {
        server_thread_.join();
    }
}

void MetricsIngester::run() {
    server_fd_ = socket(AF_INET, SOCK_STREAM, 0);
    if (server_fd_ < 0) {
        std::cerr << "[Ingester] Failed to create socket" << std::endl;
        running_ = false;
        return;
    }

    int opt = 1;
    setsockopt(server_fd_, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    struct sockaddr_in addr{};
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port = htons(port_);

    if (bind(server_fd_, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        std::cerr << "[Ingester] Failed to bind to port " << port_ << std::endl;
        running_ = false;
        close(server_fd_);
        return;
    }

    listen(server_fd_, 128);
    std::cout << "[Ingester] Listening on port " << port_ << std::endl;

    while (running_) {
        struct sockaddr_in client_addr{};
        socklen_t client_len = sizeof(client_addr);

        int client_fd = accept(server_fd_, (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) {
            if (running_) std::cerr << "[Ingester] Accept failed" << std::endl;
            continue;
        }

        // Handle client in-line (simple single-threaded for now).
        handle_client(client_fd);
    }
}

void MetricsIngester::handle_client(int client_fd) {
    char buffer[4096];
    std::string accumulated;

    while (running_) {
        int bytes_read = recv(client_fd, buffer, sizeof(buffer) - 1, 0);
        if (bytes_read <= 0) break;

        buffer[bytes_read] = '\0';
        accumulated += buffer;

        // Process complete lines (newline-delimited JSON).
        size_t pos;
        while ((pos = accumulated.find('\n')) != std::string::npos) {
            std::string line = accumulated.substr(0, pos);
            accumulated = accumulated.substr(pos + 1);

            if (!line.empty()) {
                auto metric = parse_metric(line);
                callback_(metric);
                metrics_count_++;
            }
        }
    }

    close(client_fd);
}

RawMetric MetricsIngester::parse_metric(const std::string& json) {
    RawMetric metric;
    metric.timestamp = std::chrono::system_clock::now();

    // Simple JSON parsing (production would use a proper JSON library).
    auto extract = [&](const std::string& key) -> std::string {
        auto key_pos = json.find("\"" + key + "\"");
        if (key_pos == std::string::npos) return "";
        auto colon = json.find(':', key_pos);
        if (colon == std::string::npos) return "";
        auto start = json.find_first_not_of(" \t\"", colon + 1);
        if (start == std::string::npos) return "";
        if (json[start] == '"') {
            auto end = json.find('"', start + 1);
            return json.substr(start + 1, end - start - 1);
        }
        auto end = json.find_first_of(",} \t\n", start);
        return json.substr(start, end - start);
    };

    metric.benchmark_id = extract("benchmark_id");
    metric.order_id = extract("order_id");
    metric.symbol = extract("symbol");
    metric.side = extract("side");
    metric.success = extract("success") == "true";
    metric.error_message = extract("error");

    auto latency_str = extract("latency_us");
    if (!latency_str.empty()) metric.latency_us = std::stoll(latency_str);

    auto price_str = extract("price");
    if (!price_str.empty()) metric.price = std::stod(price_str);

    auto qty_str = extract("quantity");
    if (!qty_str.empty()) metric.quantity = std::stoi(qty_str);

    return metric;
}

} // namespace telemetry
