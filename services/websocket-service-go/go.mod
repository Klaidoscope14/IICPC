module github.com/iicpc/websocket-service-go

go 1.26.3

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/gorilla/websocket v1.5.1
	github.com/iicpc/pkg v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/twmb/franz-go v1.16.1
)

replace github.com/iicpc/pkg => ../../pkg
