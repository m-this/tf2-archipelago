SHELL := /bin/sh

# Every tool version comes from one file. Never inline a version here: compose,
# the plugin build and this Makefile all read the same values, so they cannot
# drift apart.
include deploy/env/versions.env
export

# go.mod owns the Go version: the `go` directive must be a literal.
GO_VERSION := $(shell sed -n 's/^go //p' go.mod)
export GO_VERSION

COMPOSE := docker compose --env-file deploy/env/versions.env -f deploy/compose.yml
COMPOSE_TEST := docker compose --env-file deploy/env/versions.env -f deploy/compose.test.yml

# Tools of record: pinned and run through `go run` or `uv run`, so no host
# install is needed and a local run is byte-identical to CI.
GOFUMPT := go run mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
RUFF := uv run --quiet --with ruff==$(RUFF_VERSION) ruff
GO_SRC := $$(find . -type f -name '*.go')

.PHONY: help up down restart logs ps check fmt fmt-check vet lint lint-fix \
        fix-check vuln compile test test-fast export apworld-lint apworld-fmt \
        apworld-test plugin integration build clean

help:
	@echo "tf2-archipelago"
	@echo "  make up            Start the stack (docker compose)"
	@echo "  make down          Stop the stack"
	@echo "  make logs          Follow logs"
	@echo "  make check         The gate: everything CI runs"
	@echo "  make export        Regenerate apworld/tf2_mvm/data from gamedata/"
	@echo "  make plugin        Compile the SourceMod plugin"
	@echo "  make apworld-test  Run the apworld's tests inside Archipelago"
	@echo "  make integration   Bring up Archipelago and the bridge, drive them"
	@echo "  make clean         Stop, remove volumes, remove build output"

# --- The stack ---

up:
	@if [ ! -f .env ]; then cp deploy/.env.example .env; echo "wrote .env from the example, set SRCDS_RCONPW"; fi
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart: down up

logs:
	$(COMPOSE) logs -f --tail=200

ps:
	$(COMPOSE) ps

build:
	$(COMPOSE) build

# --- Go ---

fmt:
	$(GOFUMPT) -w $(GO_SRC)

fmt-check:
	@files="$$($(GOFUMPT) -l $(GO_SRC))"; \
	if [ -n "$$files" ]; then \
		printf '%s\n' "$$files"; \
		exit 1; \
	fi

# Not in `check`: every analyzer it registers is in golangci-lint's govet.
vet:
	go vet ./...

lint:
	$(GOLANGCI_LINT) run ./...

lint-fix:
	$(GOLANGCI_LINT) run --fix ./...

# `go fix` must be a no-op: whatever it would rewrite belongs in the commit.
# `-diff` reports without touching the tree, so this is safe on a dirty working
# copy.
fix-check:
	@out="$$(go fix -diff ./...)"; \
	if [ -n "$$out" ]; then \
		printf '%s\n' "$$out"; \
		echo "go fix would rewrite code: apply it with 'go fix ./...' and commit"; \
		exit 1; \
	fi

# Reports only the vulnerabilities whose vulnerable symbol this code can
# actually reach, so a hit is a bug to fix rather than a number to argue with.
vuln:
	$(GOVULNCHECK) ./...

compile:
	go build ./...

# The race detector is the only tool that sees a data race, and the bridge is
# three goroutines around one state store. -shuffle=on breaks the accidental
# ordering that makes a suite pass in one order and fail in another.
#
# This also guards the committed export: TestCommittedExportIsCurrent
# regenerates it and fails if the tree is stale, which is why there is no
# separate freshness target.
test:
	CGO_ENABLED=1 go test -race -shuffle=on ./...

test-fast:
	go test ./...

export:
	go generate ./gamedata

# --- The apworld ---

apworld-fmt:
	$(RUFF) format apworld/

apworld-lint:
	$(RUFF) format --check apworld/
	$(RUFF) check apworld/

# The apworld's tests need Archipelago to run inside, so they run in the image
# that has it. The stage puts the world back in worlds/ as a folder and drops
# the packaged copy, because loading both would be the same game twice.
apworld-test:
	docker build --target apworld-test \
		--build-arg ARCHIPELAGO_VERSION=$(ARCHIPELAGO_VERSION) \
		-f deploy/Dockerfile.archipelago -t tf2-archipelago-apworld-test .
	docker run --rm tf2-archipelago-apworld-test

# --- The plugin ---

plugin:
	./plugin/build.sh

# --- Integration ---

# Archipelago and the bridge, for real, driven the way the plugin drives them.
# The game server is not in it: it downloads 14 GB and proves nothing without a
# human at a Team Fortress 2 client.
integration:
	./deploy/integration-test.sh

# --- The gate ---

# Everything CI runs, cheapest failure first. Green here means green there.
check: fmt-check lint fix-check compile test vuln apworld-lint plugin apworld-test integration

clean:
	$(COMPOSE) down -v
	$(COMPOSE_TEST) down -v
	rm -rf plugin/build/
