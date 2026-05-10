# API Gateway (Go)

> **Status:** Not yet implemented — planned for a future phase.

This service will serve as the single entry point for all external traffic. Responsibilities:

- Request routing to internal microservices
- Rate limiting
- Authentication / authorization
- Request/response transformation
- TLS termination

## Planned Tech Stack

- Go + Gin/Fiber
- gRPC for internal service communication
- JWT-based auth
