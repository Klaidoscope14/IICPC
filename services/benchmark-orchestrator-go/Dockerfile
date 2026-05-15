FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/benchmark-orchestrator-go/go.mod services/benchmark-orchestrator-go/go.sum ./services/benchmark-orchestrator-go/
WORKDIR /src/services/benchmark-orchestrator-go
RUN go mod download

WORKDIR /src
COPY pkg ./pkg
COPY services/benchmark-orchestrator-go ./services/benchmark-orchestrator-go
WORKDIR /src/services/benchmark-orchestrator-go
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/orchestrator-service ./cmd/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata docker-cli && adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /out/orchestrator-service /app/orchestrator-service

EXPOSE 8081

USER appuser

CMD ["/app/orchestrator-service"]
