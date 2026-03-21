.PHONY: run docker-up docker-stop docker-down-v help

help:
	@echo "Targets:"
	@echo "  make run           - run Go app locally on :8080 (cwd: cmd/server)"
	@echo "  make docker-up     - db + app + prometheus (app UI: http://localhost:8081)"
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
