.EXPORT_ALL_VARIABLES:

APP_HTTP_PORT ?= 8081
ANALYTICS_PORT ?= 8082
RELAY_PORT ?= 8083
PROJECTION_BUILDER_METRICS_PORT ?= 2112
RISK_EVALUATION_METRICS_PORT ?= 2113
PROMETHEUS_PORT ?= 9090
GRAFANA_PORT ?= 3000
KAFKA_EXTERNAL_PORT ?= 9094
KAFKA_EXPORTER_PORT ?= 9308
PROJECTION_BUILDER_GROUP_ID ?= diploma-projection-builder
RISK_EVALUATION_GROUP_ID ?= diploma-risk-evaluation

# Сценарная нагрузка на API (cmd/load): smoke|balanced|stress|events|negative
# Демо без ожидаемых ошибок: make load-demo
# Высокая нагрузка: make load-stress
# Демонстрация ошибок/idempotency: make load-errors
LOAD_URL ?= http://localhost:$(APP_HTTP_PORT)
LOAD_DURATION ?= 30m
LOAD_PROFILE ?= balanced
LOAD_QPS ?= 8
LOAD_WORKERS ?= 6
JWT_USER_ID ?= 11111111-1111-1111-1111-111111111111
JWT_ROLE ?= operator
JWT_TTL ?= 1h
LOAD_AUTH_TOKEN ?= $(shell cd cmd/jwttoken && go run . -user-id "$(JWT_USER_ID)" -role "$(JWT_ROLE)" -ttl "$(JWT_TTL)")

LOAD_DEMO_DURATION ?= 2m
LOAD_DEMO_QPS ?= 6
LOAD_DEMO_WORKERS ?= 6

LOAD_STRESS_DURATION ?= 5m
LOAD_STRESS_QPS ?= 120
LOAD_STRESS_WORKERS ?= 24

LOAD_ERRORS_DURATION ?= 2m
LOAD_ERRORS_QPS ?= 8
LOAD_ERRORS_WORKERS ?= 6

LOAD_EVENTS_DURATION ?= 90s
LOAD_EVENTS_QPS ?= 4
LOAD_EVENTS_WORKERS ?= 3

.PHONY: help run run-analytics run-relay run-projectionbuilder run-riskevaluation dlq-replay jwt-token load load-demo load-stress load-events load-errors compose-config compose-up compose-up-core compose-stop compose-down-v compose-logs verify verify-test verify-build verify-compose

help:
	@echo "Targets:"
	@echo "  make run                  - run transaction-api locally on :8080"
	@echo "  make run-analytics        - run analytics-query locally on :$(ANALYTICS_PORT)"
	@echo "  make run-relay            - run outbox-relay locally on :$(RELAY_PORT)"
	@echo "  make run-projectionbuilder - run projection-builder locally on :$(PROJECTION_BUILDER_METRICS_PORT)"
	@echo "  make run-riskevaluation   - run risk-evaluation locally on :$(RISK_EVALUATION_METRICS_PORT)"
	@echo "  make dlq-replay           - replay DLQ -> main topic: make dlq-replay ARGS='-limit 5'"
	@echo "  make jwt-token            - print demo JWT for LOAD_AUTH_TOKEN/localStorage authToken"
	@echo "  make load                 - сценарная нагрузка с ручными параметрами"
	@echo "                     - параметры: LOAD_DURATION, LOAD_PROFILE, LOAD_QPS, LOAD_WORKERS, LOAD_URL, LOAD_AUTH_TOKEN"
	@echo "  make load-demo            - обычный чистый прогон без ожидаемых 4xx/5xx"
	@echo "  make load-stress          - высокая нагрузка без ожидаемых ошибок"
	@echo "  make load-events          - create/update/delete + /analytics/stream"
	@echo "  make load-errors          - демонстрация idempotency conflict / forbidden / unauthorized"
	@echo "  make compose-config       - render docker compose topology"
	@echo "  make compose-up-core      - start core banking/event stack without observability"
	@echo "  make compose-up           - start full demo stack with observability profile"
	@echo "  make compose-stop         - docker compose stop"
	@echo "  make compose-down-v       - docker compose down -v"
	@echo "  make compose-logs SERVICE=transaction-api - tail logs for one service"
	@echo "  make verify              - go test + build key binaries + docker compose config"

run:
	cd cmd/server && go run .

run-analytics:
	cd cmd/analytics && go run .

run-relay:
	cd cmd/relay && go run .

run-projectionbuilder:
	cd cmd/projectionbuilder && CONSUMER_METRICS_PORT=$(PROJECTION_BUILDER_METRICS_PORT) KAFKA_CONSUMER_GROUP_ID=$(PROJECTION_BUILDER_GROUP_ID) go run .

run-riskevaluation:
	cd cmd/riskevaluation && CONSUMER_METRICS_PORT=$(RISK_EVALUATION_METRICS_PORT) KAFKA_CONSUMER_GROUP_ID=$(RISK_EVALUATION_GROUP_ID) go run .

dlq-replay:
	cd cmd/dlqreplay && go run . $(ARGS)

jwt-token:
	@cd cmd/jwttoken && go run . -user-id "$(JWT_USER_ID)" -role "$(JWT_ROLE)" -ttl "$(JWT_TTL)"

load:
	@cd cmd/load && go run . -url "$(LOAD_URL)" -duration "$(LOAD_DURATION)" -profile "$(LOAD_PROFILE)" -qps "$(LOAD_QPS)" -workers "$(LOAD_WORKERS)" -auth-token "$(LOAD_AUTH_TOKEN)" $(LOAD_ARGS)

load-demo:
	$(MAKE) load LOAD_PROFILE=balanced LOAD_DURATION="$(LOAD_DEMO_DURATION)" LOAD_QPS="$(LOAD_DEMO_QPS)" LOAD_WORKERS="$(LOAD_DEMO_WORKERS)" LOAD_ARGS="$(LOAD_ARGS)"

load-stress:
	$(MAKE) load LOAD_PROFILE=stress LOAD_DURATION="$(LOAD_STRESS_DURATION)" LOAD_QPS="$(LOAD_STRESS_QPS)" LOAD_WORKERS="$(LOAD_STRESS_WORKERS)" LOAD_ARGS="$(LOAD_ARGS)"

load-events:
	$(MAKE) load LOAD_PROFILE=events LOAD_DURATION="$(LOAD_EVENTS_DURATION)" LOAD_QPS="$(LOAD_EVENTS_QPS)" LOAD_WORKERS="$(LOAD_EVENTS_WORKERS)" LOAD_ARGS="$(LOAD_ARGS)"

load-errors:
	$(MAKE) load LOAD_PROFILE=negative LOAD_DURATION="$(LOAD_ERRORS_DURATION)" LOAD_QPS="$(LOAD_ERRORS_QPS)" LOAD_WORKERS="$(LOAD_ERRORS_WORKERS)" LOAD_ARGS="$(LOAD_ARGS)"

compose-config:
	docker compose --profile observability config

compose-up-core:
	docker compose up -d --build

compose-up:
	docker compose --profile observability up -d --build

compose-stop:
	docker compose stop

compose-down-v:
	docker compose down -v

compose-logs:
	docker compose logs -f $(SERVICE)

verify: verify-test verify-build verify-compose

verify-test:
	go test ./...

verify-build:
	go build ./cmd/server ./cmd/analytics ./cmd/relay ./cmd/projectionbuilder ./cmd/riskevaluation ./cmd/dlqreplay

verify-compose:
	docker compose --profile observability config > NUL
