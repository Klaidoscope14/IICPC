# Sample Trading Engine

A simple C++ orderbook matching engine for the IICPC Distributed Benchmarking Platform.

## Features

- **Orderbook Management**: Price-time priority matching
- **Multiple Symbols**: Support for multiple trading symbols
- **Real-time Matching**: Efficient order matching algorithm
- **Trade Execution**: Automatic trade generation and tracking
- **Market Depth**: Best bid/ask and depth information

## Architecture

- `OrderBook`: Core matching engine with buy/sell order books
- `Exchange`: Multi-symbol order management
- `Order`: Order representation with metadata
- `Trade`: Trade execution records

## Building

```bash
mkdir build && cd build
cmake ..
make -j$(nproc)
```

## Running

```bash
./trading_engine
```

## Docker Support

```bash
docker build -t trading-engine .
docker run -p 8080:8080 trading-engine
```

## Performance

- Optimized for high-frequency trading
- Price-time priority enforcement
- Efficient memory management
- Thread-safe design (ready for extension)

## API (Future)

The engine is designed to be extended with REST/WebSocket APIs for:
- Order submission
- Market data streaming
- Trade notifications
- Real-time depth updates
