# AI AGENT IMPLEMENTATION PLAN

## Project Goal

Build a distributed benchmarking and hosting platform for evaluating contestant-submitted trading infrastructure.

The platform should:

1. Accept contestant submissions
2. Securely sandbox and deploy them
3. Launch distributed trading bots
4. Stress test the deployed system
5. Collect telemetry
6. Validate correctness
7. Stream live leaderboard metrics

---

# CRITICAL ENGINEERING REQUIREMENTS

The codebase MUST prioritize:

- Clean Architecture
- SOLID Principles
- Modularity
- Maintainability
- Scalability
- Readability
- Extensibility
- Strong typing
- Proper abstraction layers

DO NOT generate monolithic or tightly coupled code.

---

# ARCHITECTURE STYLE

Follow:

- Microservices Architecture
- Event-Driven Architecture
- Domain-Driven Design (DDD)
- Clean Architecture

---

# REPOSITORY STRUCTURE

```text
/services
    /api-gateway-go
    /submission-service-go
    /benchmark-orchestrator-go

/high-performance
    /bot-engine-cpp
    /validation-engine-cpp
    /telemetry-core-cpp

/frontend
    /dashboard-nextjs

/infrastructure
    /k8s
    /terraform
    /docker

/proto
/docs
```

---

# LANGUAGE REQUIREMENTS

## Go Services

Use Go for:

- API Gateway
- Submission Service
- Benchmark Orchestrator
- Infrastructure Controllers

### Requirements

- Use Fiber or Gin
- Use gRPC for internal communication
- Use dependency injection
- Use interfaces heavily
- Avoid global state
- Use context propagation
- Proper structured logging
- Graceful shutdown handling

---

## C++ Services

Use modern C++20 for:

- Distributed Bot Engine
- Validation Engine
- Telemetry Core

### Requirements

- Use RAII
- Use smart pointers
- Avoid raw pointer ownership
- Use thread-safe abstractions
- Prefer composition over inheritance
- Use lock-free structures where beneficial
- Avoid giant classes
- Use modular architecture
- Follow SOLID principles

### Suggested Libraries

| Purpose | Library |
|---|---|
| Networking | Boost.Asio / Beast |
| Logging | spdlog |
| Serialization | protobuf |
| Testing | GoogleTest |

---

# FRONTEND REQUIREMENTS

Use:

- Next.js
- TypeScript
- TailwindCSS
- shadcn/ui

### Frontend Features

- Live leaderboard
- Real-time graphs
- Benchmark status
- Submission dashboard
- Metrics visualization

---

# INFRASTRUCTURE REQUIREMENTS

Must include:

- Docker
- Kubernetes manifests
- Terraform (optional but preferred)
- Prometheus
- Grafana

---

# DATABASE REQUIREMENTS

| Purpose | Database |
|---|---|
| Metadata | PostgreSQL |
| Cache | Redis |
| Time-series Metrics | TimescaleDB |

---

# MESSAGE BUS

Use:

- Redpanda (preferred)
OR
- Kafka

Use event-driven communication wherever appropriate.

---

# COMMUNICATION PROTOCOLS

## Internal

Use:
- gRPC
- protobuf

## External

Support:
- REST
- WebSocket

Optional:
- FIX protocol

---

# CODE QUALITY REQUIREMENTS

The generated code MUST:

- Be production-style
- Include proper folder structure
- Include interfaces
- Include abstractions
- Include comments only where necessary
- Avoid duplicated logic
- Avoid God classes
- Avoid tight coupling

---

# SERVICE IMPLEMENTATION ORDER

Follow this exact order.

---

# PHASE 1 — FOUNDATION

## Goals

Set up:

- monorepo
- folder structure
- shared protobuf definitions
- shared configs
- logging utilities
- docker setup

### Deliverables

- root workspace
- shared tooling
- CI-ready structure

---

# PHASE 2 — SUBMISSION SYSTEM

## Build

### submission-service-go

Features:

- file upload
- submission metadata
- validation
- deployment request generation

### Requirements

- REST APIs
- PostgreSQL integration
- object storage abstraction
- clean handler/service/repository separation

---

# PHASE 3 — SANDBOX DEPLOYMENT

## Build

### benchmark-orchestrator-go

Features:

- deployment orchestration
- Kubernetes pod management
- container lifecycle management
- resource limit enforcement

### Requirements

- isolated execution
- configurable limits
- deployment status tracking

---

# PHASE 4 — DISTRIBUTED BOT ENGINE

## Build

### bot-engine-cpp

Features:

- high-concurrency trading bots
- WebSocket connections
- order generation
- traffic simulation

### Requirements

- scalable worker pools
- asynchronous networking
- configurable strategies
- efficient memory usage

---

# PHASE 5 — TELEMETRY PIPELINE

## Build

### telemetry-core-cpp

Features:

- metrics ingestion
- latency measurement
- TPS tracking
- event streaming

### Requirements

- low-latency processing
- histogram support
- percentile computation

---

# PHASE 6 — VALIDATION ENGINE

## Build

### validation-engine-cpp

Features:

- orderbook reconstruction
- fill validation
- price-time priority checks
- correctness scoring

### Requirements

- deterministic processing
- efficient matching logic
- replayable validation

---

# PHASE 7 — LEADERBOARD

## Build

### dashboard-nextjs

Features:

- live metrics
- dynamic rankings
- latency graphs
- benchmark monitoring

### Requirements

- websocket streaming
- responsive UI
- real-time updates

---

# PHASE 8 — OBSERVABILITY

## Add

- Prometheus
- Grafana
- distributed logging
- tracing support

---

# PHASE 9 — SCALING & HARDENING

## Add

- autoscaling
- fault tolerance
- retry mechanisms
- backpressure handling
- load balancing

---

# TESTING REQUIREMENTS

Every service MUST include:

- unit tests
- integration tests
- configuration validation
- health endpoints

---

# SECURITY REQUIREMENTS

Must enforce:

- sandbox isolation
- restricted networking
- CPU limits
- memory limits
- execution timeouts

---

# PERFORMANCE GOALS

Target:

- thousands of concurrent bots
- low telemetry overhead
- stable websocket streaming
- horizontally scalable workers

---

# IMPORTANT IMPLEMENTATION RULES

## DO

- Build incrementally
- Keep services independent
- Keep interfaces small
- Use event-driven patterns
- Use protobuf contracts
- Keep code modular

---

## DO NOT

- Generate monolithic services
- Hardcode configuration
- Mix business logic with handlers
- Create giant classes
- Create tightly coupled modules

---

# FINAL EXPECTATION

The final system should resemble a production-grade distributed exchange benchmarking platform with:

- scalable infrastructure
- secure sandboxing
- distributed concurrency
- real-time analytics
- maintainable architecture
- strong systems engineering principles