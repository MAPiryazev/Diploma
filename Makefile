.EXPORT_ALL_VARIABLES:

APP_HTTP_PORT ?= 8081
PROMETHEUS_PORT ?= 9090
GRAFANA_PORT ?= 3000
KAFKA_EXTERNAL_PORT ?= 9094
KAFKA_EXPORTER_PORT ?= 9308

# Сценарная нагрузка на API (cmd/load): smoke|balanced|stress|negative
# Демо без ожидаемых ошибок: make load-demo
# Высокая нагрузка: make load-stress
# Демонстрация ошибок/idempotency: make load-errors
LOAD_URL ?= http://localhost:$(APP_HTTP_PORT)
LOAD_DURATION ?= 30m
LOAD_PROFILE ?= balanced
LOAD_QPS ?= 8
LOAD_WORKERS ?= 6
LOAD_AUTH_TOKEN ?= dev-token

LOAD_DEMO_DURATION ?= 2m
LOAD_DEMO_QPS ?= 6
LOAD_DEMO_WORKERS ?= 6

LOAD_STRESS_DURATION ?= 5m
LOAD_STRESS_QPS ?= 60
LOAD_STRESS_WORKERS ?= 24

LOAD_ERRORS_DURATION ?= 2m
LOAD_ERRORS_QPS ?= 8
LOAD_ERRORS_WORKERS ?= 6

.PHONY: run run-consumer dlq-replay load load-demo load-stress load-errors docker-up docker-stop docker-down-v help

help:
	@echo "Targets:"
	@echo "  make run           - run Go app locally on :8080 (cwd: cmd/server)"
	@echo "  make run-consumer  - run Kafka consumer locally (metrics :2112, cwd: cmd/consumer)"
	@echo "  make dlq-replay    - replay DLQ -> main topic: make dlq-replay ARGS='-limit 5'"
	@echo "  make load          - сценарная нагрузка с ручными параметрами"
	@echo "                     - параметры: LOAD_DURATION, LOAD_PROFILE, LOAD_QPS, LOAD_WORKERS, LOAD_URL"
	@echo "  make load-demo     - обычный чистый прогон без ожидаемых 4xx/5xx (balanced, low QPS)"
	@echo "  make load-stress   - высокая нагрузка без ожидаемых ошибок (stress, high QPS)"
	@echo "  make load-errors   - демонстрация idempotency conflict / forbidden / unauthorized"
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
	cd cmd/load && go run . -url "$(LOAD_URL)" -duration "$(LOAD_DURATION)" -profile "$(LOAD_PROFILE)" -qps "$(LOAD_QPS)" -workers "$(LOAD_WORKERS)" -auth-token "$(LOAD_AUTH_TOKEN)" $(LOAD_ARGS)

load-demo:
	$(MAKE) load LOAD_PROFILE=balanced LOAD_DURATION="$(LOAD_DEMO_DURATION)" LOAD_QPS="$(LOAD_DEMO_QPS)" LOAD_WORKERS="$(LOAD_DEMO_WORKERS)" LOAD_ARGS="$(LOAD_ARGS)"

load-stress:
	$(MAKE) load LOAD_PROFILE=stress LOAD_DURATION="$(LOAD_STRESS_DURATION)" LOAD_QPS="$(LOAD_STRESS_QPS)" LOAD_WORKERS="$(LOAD_STRESS_WORKERS)" LOAD_ARGS="$(LOAD_ARGS)"

load-errors:
	$(MAKE) load LOAD_PROFILE=negative LOAD_DURATION="$(LOAD_ERRORS_DURATION)" LOAD_QPS="$(LOAD_ERRORS_QPS)" LOAD_WORKERS="$(LOAD_ERRORS_WORKERS)" LOAD_ARGS="$(LOAD_ARGS)"

docker-up:
	docker compose up -d --build

docker-stop:
	docker compose stop

docker-down-v:
	docker compose down -v
