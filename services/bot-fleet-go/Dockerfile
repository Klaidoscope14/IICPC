FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/bot-fleet-go/go.mod services/bot-fleet-go/go.sum ./services/bot-fleet-go/
COPY services/api-gateway-go/go.mod services/api-gateway-go/go.sum ./services/api-gateway-go/
COPY services/benchmark-orchestrator-go/go.mod services/benchmark-orchestrator-go/go.sum ./services/benchmark-orchestrator-go/
COPY services/submission-service-go/go.mod services/submission-service-go/go.sum ./services/submission-service-go/
COPY services/validation-service-go/go.mod services/validation-service-go/go.sum ./services/validation-service-go/
COPY services/correctness-engine-go/go.mod ./services/correctness-engine-go/
RUN go work sync

WORKDIR /src
COPY pkg ./pkg
COPY services/bot-fleet-go ./services/bot-fleet-go
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bot-fleet ./services/bot-fleet-go/cmd/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /out/bot-fleet /app/bot-fleet
RUN mkdir -p /app/traces && chown -R appuser:appuser /app

EXPOSE 8085

USER appuser

CMD ["/app/bot-fleet"]
