# IICPC Summer Hackathon 2026 — Distributed Benchmarking & Hosting Platform

## Overview

This project is a high-performance distributed benchmarking platform built for the IICPC Summer Hackathon 2026.

The platform is designed to securely host contestant-submitted trading infrastructure such as:

- Simulated orderbooks
- Matching engines
- Exchange cores
- Trading APIs

The system dynamically deploys these submissions inside isolated sandboxed environments, launches a distributed fleet of trading bots to stress-test them under extreme load conditions, collects real-time telemetry, validates correctness of execution, and streams live analytics to a real-time leaderboard.

The architecture is inspired by real-world low-latency trading infrastructure and modern cloud-native distributed systems.

---

# Problem Statement

Contestants upload their exchange or trading-engine implementation.

The platform must:

1. Securely containerize and deploy the submission
2. Expose predefined APIs/WebSocket endpoints
3. Launch distributed trading bots
4. Simulate peak market conditions
5. Measure:
   - latency
   - throughput
   - stability
   - correctness
6. Dynamically rank contestants on a live leaderboard

---

# Core Objectives

The system focuses heavily on:

- Distributed Systems
- Concurrency
- Scalability
- Low-Latency Engineering
- Container Isolation
- Telemetry & Observability
- Real-Time Analytics
- Infrastructure Automation

---

# High-Level Architecture

```text
Contestant Upload
        ↓
Submission Service
        ↓
Sandbox Deployment Engine
        ↓
Containerized Trading Engine
        ↓
Distributed Bot Fleet
        ↓
Telemetry Collector
        ↓
Metrics Pipeline
        ↓
Validation Engine
        ↓
Leaderboard & Analytics
```
---

<img width="1536" height="1024" alt="architecture_IICPC" src="https://github.com/user-attachments/assets/ae3b1a6e-49a9-4408-ac8c-606182f4a8e2" />


---

# Major Components

## 1. Submission & Sandboxing Engine

Responsible for:

- Uploading contestant binaries/source code
- Building containers
- Isolating execution environments
- Applying resource constraints
- Exposing controlled endpoints

### Features

- Docker-based isolation
- Kubernetes deployment
- CPU & memory limits
- Network restrictions
- Secure execution

---

## 2. Distributed Load Generator (Bot Fleet)

A horizontally scalable traffic generation engine.

Responsible for:

- Simulating thousands of trading bots
- Generating realistic market activity
- Sending:
  - Limit Orders
  - Market Orders
  - Cancels
- Supporting:
  - REST
  - WebSocket
  - FIX (optional)

### Features

- Massive concurrency
- High-throughput request generation
- Distributed workers
- Configurable trading strategies
- Burst traffic simulation

---

## 3. Telemetry & Validation Engine

Responsible for:

- Collecting real-time metrics
- Measuring:
  - p50 latency
  - p90 latency
  - p99 latency
  - TPS
- Detecting failures
- Validating correctness of exchange behavior

### Validation Goals

- Price-time priority
- Fill correctness
- Matching consistency
- Sequence integrity

---

## 4. Real-Time Leaderboard

Responsible for:

- Streaming live benchmark data
- Ranking submissions dynamically
- Displaying:
  - TPS
  - latency graphs
  - success rate
  - correctness score

---

# Technology Stack

## Backend Services

| Component | Tech |
|---|---|
| API Gateway | Go |
| Submission Service | Go |
| Orchestrator | Go |
| Distributed Bot Fleet | C++20 |
| Validation Engine | C++20 |
| Telemetry Core | C++20 |

---

## Frontend

| Component | Tech |
|---|---|
| Framework | Next.js |
| Language | TypeScript |
| Styling | TailwindCSS |
| Components | shadcn/ui |

---

## Infrastructure

| Component | Tech |
|---|---|
| Containerization | Docker |
| Orchestration | Kubernetes |
| Messaging | Redpanda |
| Metrics | Prometheus |
| Visualization | Grafana |

---

## Databases

| Purpose | Database |
|---|---|
| Metadata | PostgreSQL |
| Cache | Redis |
| Time-Series Metrics | TimescaleDB |

---

# System Design Principles

This project follows:

- Clean Architecture
- SOLID Principles
- Domain-Driven Design (DDD)
- Event-Driven Architecture
- Microservices Architecture

Goals:

- High maintainability
- Easy onboarding
- Clear separation of concerns
- Scalability
- Testability

---

# Repository Structure

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

# Core Engineering Challenges

## Secure Sandboxing

Running untrusted contestant code safely.

---

## Massive Concurrency

Simulating thousands of concurrent trading bots.

---

## Accurate Telemetry

Capturing precise latency metrics without introducing bottlenecks.

---

## Correctness Validation

Ensuring exchange behavior follows expected matching logic.

---

## Real-Time Streaming

Streaming live metrics to dashboards efficiently.

---

# Development Phases

## Phase 1 — MVP

- Upload service
- Container deployment
- Basic benchmark execution

---

## Phase 2 — Distributed Bot Fleet

- Trading bot workers
- WebSocket load generation
- Distributed orchestration

---

## Phase 3 — Telemetry Pipeline

- Metrics ingestion
- Latency tracking
- TPS computation

---

## Phase 4 — Validation Engine

- Orderbook reconstruction
- Matching verification
- Correctness scoring

---

## Phase 5 — Leaderboard & Analytics

- Live dashboard
- Dynamic rankings
- WebSocket streaming

---

## Phase 6 — Infrastructure & Scaling

- Kubernetes deployment
- Horizontal autoscaling
- Distributed infrastructure

---

# Engineering Goals

This project prioritizes:

- Performance
- Scalability
- Reliability
- Security
- Maintainability
- Extensibility

over purely visual demonstrations.

---

# Team Vision

The goal is to build infrastructure that resembles real-world exchange benchmarking systems and demonstrates strong systems engineering principles, distributed computing knowledge, and low-latency architecture design.
