.DEFAULT_GOAL := help

export LOCAL_UID := $(shell id -u)
export LOCAL_GID := $(shell id -g)
COMPOSE := docker compose
RUN := $(COMPOSE) run --rm dev
HOST_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH ?= $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
ARGS ?= --help

.PHONY: help build run binary fmt vet test up down stop restart logs ps shell

help:
	@echo "build   Build the Docker development image"
	@echo "run     Run the CLI in Docker (ARGS defaults to --help)"
	@echo "binary  Build bin/sift for this host (HOST_OS/HOST_ARCH override)"
	@echo "fmt     Format Go source"
	@echo "vet     Check Go source"
	@echo "test    Run tests (no behavioral tests exist yet)"
	@echo "up      Start optional workspace container in foreground"
	@echo "shell   Open shell in running workspace container"
	@echo "down / stop / restart / logs / ps   Manage workspace container"

build:
	$(COMPOSE) build

run:
	$(RUN) go run ./cmd/sift $(ARGS)

binary:
	$(COMPOSE) run --rm -e CGO_ENABLED=0 -e GOOS=$(HOST_OS) -e GOARCH=$(HOST_ARCH) dev go build -o bin/sift ./cmd/sift

fmt:
	$(RUN) go fmt ./...

vet:
	$(RUN) go vet ./...

test:
	$(RUN) go test ./...

up:
	$(COMPOSE) up

down:
	$(COMPOSE) down

stop:
	$(COMPOSE) stop

restart:
	$(COMPOSE) restart

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

shell:
	$(COMPOSE) exec dev sh
