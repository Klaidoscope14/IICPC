FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/api-gateway-go/go.mod services/api-gateway-go/go.sum ./services/api-gateway-go/
COPY services/benchmark-orchestrator-go/go.mod services/benchmark-orchestrator-go/go.sum ./services/benchmark-orchestrator-go/
COPY services/submission-service-go/go.mod services/submission-service-go/go.sum ./services/submission-service-go/
COPY services/validation-service-go/go.mod services/validation-service-go/go.sum ./services/validation-service-go/
COPY services/bot-fleet-go/go.mod services/bot-fleet-go/go.sum ./services/bot-fleet-go/
COPY services/correctness-engine-go/go.mod services/correctness-engine-go/go.sum ./services/correctness-engine-go/
COPY services/websocket-service-go/go.mod ./services/websocket-service-go/
COPY services/admin-service-go/go.mod ./services/admin-service-go/
COPY services/auth-service-go/go.mod ./services/auth-service-go/
RUN go work sync

WORKDIR /src
COPY pkg ./pkg
COPY services/api-gateway-go ./services/api-gateway-go
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./services/api-gateway-go/cmd/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /out/api-gateway /app/api-gateway

EXPOSE 8082

USER appuser

CMD ["/app/api-gateway"]
