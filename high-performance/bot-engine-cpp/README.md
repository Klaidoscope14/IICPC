# Distributed Bot Engine (C++20)

> **Status:** Not yet implemented — planned for Phase 4.

High-concurrency trading bot fleet for stress-testing contestant submissions.

## Planned Features

- Scalable worker pools with async I/O (Boost.Asio)
- WebSocket + REST order generation
- Configurable trading strategies
- Burst traffic simulation

## Planned Libraries

| Purpose | Library |
|---|---|
| Networking | Boost.Asio / Beast |
| Logging | spdlog |
| Serialization | protobuf |
| Testing | GoogleTest |
