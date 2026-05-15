FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/api-gateway-go/go.mod services/api-gateway-go/go.sum ./services/api-gateway-go/
COPY services/benchmark-orchestrator-go/go.mod services/benchmark-orchestrator-go/go.sum ./services/benchmark-orchestrator-go/
COPY services/submission-service-go/go.mod services/submission-service-go/go.sum ./services/submission-service-go/
COPY services/validation-service-go/go.mod services/validation-service-go/go.sum ./services/validation-service-go/
RUN go work sync

WORKDIR /src
COPY pkg ./pkg
COPY services/submission-service-go ./services/submission-service-go
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/submission-service ./services/submission-service-go/cmd/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /out/submission-service /app/submission-service
RUN mkdir -p /data/iicpc-storage && chown -R appuser:appuser /data

EXPOSE 8080

USER appuser

CMD ["/app/submission-service"]
