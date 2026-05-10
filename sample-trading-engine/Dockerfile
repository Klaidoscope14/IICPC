FROM ubuntu:22.04

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    && rm -rf /var/lib/apt/lists/*

# Copy source files
COPY . .

# Build the trading engine
RUN mkdir build && cd build && \
    cmake .. && \
    make -j$(nproc)

# Expose port for potential HTTP API
EXPOSE 8080

# Run the trading engine
CMD ["./build/trading_engine"]
