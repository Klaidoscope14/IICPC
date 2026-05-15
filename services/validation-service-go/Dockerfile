FROM golang:1.25.0-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache build-base

# Copy shared pkg and service code
COPY pkg/ ./pkg/
COPY services/validation-service-go/ ./services/validation-service-go/

# Build
WORKDIR /app/services/validation-service-go
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/validation-service cmd/main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/validation-service /app/validation-service

# Use unprivileged user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8081

CMD ["/app/validation-service"]
