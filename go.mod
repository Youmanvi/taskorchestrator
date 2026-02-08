module github.com/vihan/taskorchestrator

go 1.22

require (
	github.com/microsoft/durabletask-go v0.1.0
	github.com/microsoft/durabletask-go/backend/sqlite v0.1.0
	github.com/spf13/viper v1.18.2
	github.com/rs/zerolog v1.31.0
	github.com/prometheus/client_golang v1.18.0
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/sdk v1.22.0
	go.opentelemetry.io/otel/exporters/zipkin v1.22.0
	github.com/stretchr/testify v1.8.4
	github.com/shopspring/decimal v1.3.1
	github.com/sony/gobreaker v0.5.0
	github.com/mattn/go-sqlite3 v1.14.19
)
