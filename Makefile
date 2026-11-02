.DEFAULT_GOAL := help

export LOCAL_UID := $(shell id -u)
export LOCAL_GID := $(shell id -g)
COMPOSE := docker compose
RUN := $(COMPOSE) run --rm dev
HOST_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH ?= $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
ARGS ?= --help
VERSION ?= dev
GO_LDFLAGS := -X main.version=$(VERSION)
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: help build run binary install fmt vet test up down stop restart logs ps shell

help:
	@echo "build   Build the Docker development image"
	@echo "run     Run the CLI in Docker (ARGS defaults to --help)"
	@echo "binary  Build bin/sift for this host (VERSION=dev, HOST_OS/HOST_ARCH override)"
	@echo "install Build and install sift (INSTALL_DIR defaults to ~/.local/bin)"
	@echo "fmt     Format Go source"
	@echo "vet     Check Go source"
	@echo "test    Run tests"
	@echo "up      Start optional workspace container in foreground"
	@echo "shell   Open shell in running workspace container"
	@echo "down / stop / restart / logs / ps   Manage workspace container"

build:
	$(COMPOSE) build

run:
	$(RUN) go run -ldflags '$(GO_LDFLAGS)' ./cmd/sift $(ARGS)

binary:
	$(COMPOSE) run --rm -e CGO_ENABLED=0 -e GOOS=$(HOST_OS) -e GOARCH=$(HOST_ARCH) dev go build -ldflags '$(GO_LDFLAGS)' -o bin/sift ./cmd/sift

install: binary
	mkdir -p "$(INSTALL_DIR)"
	install -m 0755 bin/sift "$(INSTALL_DIR)/sift"
	@printf 'Installed %s\nEnsure %s is on your PATH.\n' "$(INSTALL_DIR)/sift" "$(INSTALL_DIR)"

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
