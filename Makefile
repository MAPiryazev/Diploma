.EXPORT_ALL_VARIABLES:

APP_HTTP_PORT ?= 8081
PROMETHEUS_PORT ?= 9090
GRAFANA_PORT ?= 3000

.PHONY: run docker-up docker-stop docker-down-v help

help:
	@echo "Targets:"
	@echo "  make run           - run Go app locally on :8080 (cwd: cmd/server)"
	@echo "  make docker-up     - db + app + prometheus + grafana (ports from .env)"
	@echo "                     - app: http://localhost:$(APP_HTTP_PORT)  prometheus: http://localhost:$(PROMETHEUS_PORT)"
	@echo "                     - grafana: http://localhost:$(GRAFANA_PORT) (admin / admin)"
	@echo "  make docker-stop   - docker compose stop"
	@echo "  make docker-down-v - docker compose down -v"

run:
	cd cmd/server && go run .

docker-up:
	docker compose up -d --build

docker-stop:
	docker compose stop

docker-down-v:
	docker compose down -v
