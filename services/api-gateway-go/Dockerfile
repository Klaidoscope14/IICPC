FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/api-gateway-go/go.mod services/api-gateway-go/go.sum ./services/api-gateway-go/
WORKDIR /src/services/api-gateway-go
RUN go mod download

WORKDIR /src
COPY pkg ./pkg
COPY services/api-gateway-go ./services/api-gateway-go
WORKDIR /src/services/api-gateway-go
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./cmd/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /out/api-gateway /app/api-gateway

EXPOSE 8082

USER appuser

CMD ["/app/api-gateway"]
