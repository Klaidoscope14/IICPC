FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy the entire mono-repo for building
COPY . .

# Set working directory to the service
WORKDIR /app/services/websocket-service-go

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -o websocket-service cmd/main.go

# Production stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/services/websocket-service-go/websocket-service .

# Copy .env file if it exists (optional)
COPY --from=builder /app/services/websocket-service-go/.env.example ./.env

EXPOSE 8086

CMD ["./websocket-service"]
