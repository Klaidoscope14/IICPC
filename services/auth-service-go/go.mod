module github.com/iicpc/auth-service-go

go 1.25.0

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.12.3
	golang.org/x/crypto v0.31.0
	github.com/iicpc/pkg v0.0.0
)

replace github.com/iicpc/pkg => ../../pkg
