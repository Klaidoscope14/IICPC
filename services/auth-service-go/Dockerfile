FROM golang:1.25.0-alpine AS builder

WORKDIR /app
COPY pkg/ /pkg/
WORKDIR /app/services/auth-service-go

COPY services/auth-service-go/go.mod services/auth-service-go/go.sum* ./
RUN go mod download

COPY services/auth-service-go/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service cmd/main.go

FROM alpine:latest

WORKDIR /
COPY --from=builder /auth-service /auth-service

EXPOSE 8082
ENTRYPOINT ["/auth-service"]
