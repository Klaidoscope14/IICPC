#include "telemetry/stream_server.hpp"
#include <sys/socket.h>
#include <netinet/in.h>
#include <unistd.h>
#include <iostream>
#include <sstream>
#include <iomanip>
#include <ctime>

namespace telemetry {

StreamServer::StreamServer(int port, const MetricsStore& store)
    : port_(port)
    , store_(store)
{}

StreamServer::~StreamServer() {
    stop();
}

void StreamServer::start() {
    if (running_) return;
    running_ = true;
    accept_thread_ = std::thread([this]() { run(); });
}

void StreamServer::stop() {
    running_ = false;
    if (server_fd_ >= 0) {
        shutdown(server_fd_, SHUT_RDWR);
        close(server_fd_);
        server_fd_ = -1;
    }
    if (accept_thread_.joinable()) {
        accept_thread_.join();
    }
    for (auto& t : client_threads_) {
        if (t.joinable()) t.join();
    }
    client_threads_.clear();
}

void StreamServer::run() {
    server_fd_ = socket(AF_INET, SOCK_STREAM, 0);
    if (server_fd_ < 0) {
        std::cerr << "[StreamServer] Failed to create socket" << std::endl;
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
        std::cerr << "[StreamServer] Failed to bind to port " << port_ << std::endl;
        running_ = false;
        close(server_fd_);
        return;
    }

    listen(server_fd_, 32);
    std::cout << "[StreamServer] Listening on port " << port_ << std::endl;

    while (running_) {
        struct sockaddr_in client_addr{};
        socklen_t client_len = sizeof(client_addr);

        int client_fd = accept(server_fd_, (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) continue;

        client_count_++;
        client_threads_.emplace_back([this, client_fd]() {
            handle_client(client_fd);
            client_count_--;
        });
    }
}

void StreamServer::handle_client(int client_fd) {
    std::cout << "[StreamServer] Client connected" << std::endl;

    while (running_) {
        const auto* snap = store_.latest();
        if (snap) {
            std::string json = snapshot_to_json(*snap) + "\n";
            int sent = send(client_fd, json.c_str(), json.size(), 0);
            if (sent <= 0) break;
        }

        std::this_thread::sleep_for(std::chrono::seconds(1));
    }

    close(client_fd);
    std::cout << "[StreamServer] Client disconnected" << std::endl;
}

std::string StreamServer::snapshot_to_json(const AggregatedSnapshot& snap) const {
    auto time_t_val = std::chrono::system_clock::to_time_t(snap.timestamp);

    std::stringstream ss;
    ss << std::fixed << std::setprecision(2);
    ss << "{";
    ss << "\"benchmark_id\":\"" << snap.benchmark_id << "\",";
    ss << "\"timestamp\":" << time_t_val << ",";
    ss << "\"current_tps\":" << snap.current_tps << ",";
    ss << "\"total_orders_sent\":" << snap.total_orders_sent << ",";
    ss << "\"total_orders_acknowledged\":" << snap.total_orders_acknowledged << ",";
    ss << "\"total_errors\":" << snap.total_errors << ",";
    ss << "\"avg_latency_us\":" << snap.avg_latency_us << ",";
    ss << "\"p50_latency_us\":" << snap.p50_latency_us << ",";
    ss << "\"p90_latency_us\":" << snap.p90_latency_us << ",";
    ss << "\"p99_latency_us\":" << snap.p99_latency_us << ",";
    ss << "\"min_latency_us\":" << snap.min_latency_us << ",";
    ss << "\"max_latency_us\":" << snap.max_latency_us << ",";
    ss << "\"active_connections\":" << snap.active_connections;
    ss << "}";

    return ss.str();
}

} // namespace telemetry
