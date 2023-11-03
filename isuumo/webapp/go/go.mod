module github.com/isucon/isucon10-qualify/isuumo

go 1.14

require (
	github.com/XSAM/otelsql v0.26.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/labstack/echo/v4 v4.11.2
	github.com/labstack/gommon v0.4.0
	github.com/mackee/pgx-replaced v0.0.0-20230218024503-3dae8b2f6855
	github.com/paulmach/orb v0.10.0
	github.com/redis/go-redis/v9 v9.3.0
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.45.0
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.19.0
	go.opentelemetry.io/otel/sdk v1.19.0
)
