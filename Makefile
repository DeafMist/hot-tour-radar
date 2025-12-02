SHELL := /bin/bash
.DEFAULT_GOAL := help

BACKEND_DIR := backend
GO_TEST_ENV := GOPATH= GOMODCACHE=
SERVICE ?= worker

.PHONY: help backend-test backend-fmt docker-build compose-up compose-down clean

help:
	@echo "Available targets:"
	@echo "  backend-test   - Run Go tests for all backend packages"
	@echo "  backend-fmt    - Format backend Go code with gofmt"
	@echo "  docker-build   - Build backend service image (override SERVICE=name)"
	@echo "  compose-up     - Start full stack via docker-compose"
	@echo "  compose-down   - Stop the docker-compose stack"
	@echo "  clean          - Remove temporary build artifacts"

backend-test:
	@cd $(BACKEND_DIR) && $(GO_TEST_ENV) go test ./...

backend-fmt:
	@cd $(BACKEND_DIR) && gofmt -w ./api ./internal ./retention ./worker

# Build a specific backend service by setting SERVICE (worker, api, retention)
docker-build:
	docker build --build-arg SERVICE=$(SERVICE) -f backend/Dockerfile -t hot-tour-$(SERVICE) .

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

clean:
	@find $(BACKEND_DIR) -name '*.out' -delete
