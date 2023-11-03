OTEL_EXPORTER_OTLP_ENDPOINT="http://monitoring:4318"
OTEL_SERVICE_NAME="isuumo"
DB_HOSTNAME="192.168.0.12"
DB_DATABASE="isuumo"
PG_PASSWORD="isucon"
if [ -f appversion.sh ]; then
  source appversion.sh
fi
