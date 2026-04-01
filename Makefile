.EXPORT_ALL_VARIABLES:

APP_HTTP_PORT ?= 8081
PROMETHEUS_PORT ?= 9090
GRAFANA_PORT ?= 3000
KAFKA_EXTERNAL_PORT ?= 9094
KAFKA_EXPORTER_PORT ?= 9308

# Нагрузка на API (cmd/load): см. -pattern steady|wave|steps|burst|mixed
# Быстрый прогон: make load LOAD_DURATION=30s LOAD_PATTERN=steady
LOAD_URL ?= http://localhost:$(APP_HTTP_PORT)
LOAD_DURATION ?= 30m
LOAD_PATTERN ?= mixed
LOAD_QPS ?= 6
LOAD_WORKERS ?= 4

.PHONY: run run-consumer dlq-replay load docker-up docker-stop docker-down-v help

help:
	@echo "Targets:"
	@echo "  make run           - run Go app locally on :8080 (cwd: cmd/server)"
	@echo "  make run-consumer  - run Kafka consumer locally (metrics :2112, cwd: cmd/consumer)"
	@echo "  make dlq-replay    - replay DLQ -> main topic: make dlq-replay ARGS='-limit 5'"
	@echo "  make load          - нагрузка POST /items (LOAD_DURATION, LOAD_PATTERN, LOAD_QPS, LOAD_URL)"
	@echo "  make docker-up     - db + app + consumer + prometheus + grafana + kafka (ports from .env)"
	@echo "                     - app: http://localhost:$(APP_HTTP_PORT)  prometheus: http://localhost:$(PROMETHEUS_PORT)"
	@echo "                     - consumer metrics: http://localhost:$${CONSUMER_METRICS_PORT:-2112}/metrics"
	@echo "                     - kafka-exporter (lag): http://localhost:$(KAFKA_EXPORTER_PORT)/metrics"
	@echo "                     - grafana: http://localhost:$(GRAFANA_PORT) (admin / admin)"
	@echo "                     - kafka: localhost:$(KAFKA_EXTERNAL_PORT)"
	@echo "  make docker-stop   - docker compose stop"
	@echo "  make docker-down-v - docker compose down -v"

run:
	cd cmd/server && go run .

run-consumer:
	cd cmd/consumer && go run .

dlq-replay:
	cd cmd/dlqreplay && go run . $(ARGS)

load:
	cd cmd/load && go run . -url "$(LOAD_URL)" -duration "$(LOAD_DURATION)" -pattern "$(LOAD_PATTERN)" -qps "$(LOAD_QPS)" -workers "$(LOAD_WORKERS)" $(LOAD_ARGS)

docker-up:
	docker compose up -d --build

docker-stop:
	docker compose stop

docker-down-v:
	docker compose down -v
