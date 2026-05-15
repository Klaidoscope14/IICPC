FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache build-base

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/validation-service-go/go.mod services/validation-service-go/go.sum ./services/validation-service-go/
WORKDIR /src/services/validation-service-go
RUN go mod download

WORKDIR /src
COPY pkg ./pkg
COPY services/validation-service-go ./services/validation-service-go
WORKDIR /src/services/validation-service-go
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/validation-service ./cmd/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /out/validation-service /app/validation-service

EXPOSE 8084

USER appuser

CMD ["/app/validation-service"]
