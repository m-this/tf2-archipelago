SHELL := /bin/sh

# Every tool version comes from one file. Never inline a version here: compose,
# the plugin build and this Makefile all read the same values, so they cannot
# drift apart.
include deploy/env/versions.env
export

# go.mod owns the Go version: the `go` directive must be a literal.
GO_VERSION := $(shell sed -n 's/^go //p' go.mod)
export GO_VERSION

# --project-directory pins relative paths in the compose files to the repository
# root. --env-file replaces the default .env rather than adding to it, so both
# files have to be named: the pins first, then the operator's settings, which
# win.
COMPOSE_BASE := docker compose --project-directory . --env-file deploy/env/versions.env --env-file .env
COMPOSE := $(COMPOSE_BASE) -f deploy/compose.yml
COMPOSE_SEED := $(COMPOSE_BASE) -f deploy/compose.seed.yml
COMPOSE_TEST := $(COMPOSE_BASE) -f deploy/compose.test.yml
COMPOSE_DOCS := $(COMPOSE_BASE) -f deploy/compose.docs.yml

# The release file must render without an operator's .env, so this reads the
# pins and nothing else.
COMPOSE_RELEASE := docker compose --project-directory . \
	--env-file deploy/env/versions.env \
	-f deploy/compose.yml -f deploy/compose.seed.yml -f deploy/compose.release.yml

DIST := dist

# Tools of record: pinned and run through `go run` or `uv run`, so no host
# install is needed and a local run is byte-identical to CI.
GOFUMPT := go run mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
RUFF := uv run --quiet --with ruff==$(RUFF_VERSION) ruff
GO_SRC := $$(find . -type f -name '*.go')

.PHONY: help seed up down restart logs ps rcon \
        check fmt fmt-check vet lint lint-fix fix-check vuln compile test \
        test-fast export apworld-lint \
        apworld-fmt apworld-test apworld-build plugin bots bots-from-source \
        integration build docs \
        docs-build docs-down dist compose-release version-check clean \
        launcher launcher-assets

help:
	@echo "tf2-archipelago"
	@echo "  make seed          Generate a seed in ./seed to upload to archipelago.gg"
	@echo "  make up            Start the stack (docker compose)"
	@echo "  make down          Stop the stack"
	@echo "  make logs          Follow logs"
	@echo "  make rcon          Send a server command: make rcon CMD='sm_ap_status'"
	@echo "  make check         The gate: everything CI runs"
	@echo "  make export        Regenerate apworld/tf2_mvm/data from gamedata/"
	@echo "  make plugin        Compile the SourceMod plugin"
	@echo "  make bots          Stage the MvM defender bots the image installs"
	@echo "  make apworld-test  Run the apworld's tests inside Archipelago"
	@echo "  make integration   Bring up Archipelago and the bridge, drive them"
	@echo "  make dist          Build everything a release attaches into ./dist"
	@echo "  make launcher      Cross-compile tf2ap.exe (Windows) into ./dist"
	@echo "  make docs          Build the book and serve it on 127.0.0.1"
	@echo "  make clean         Stop, remove volumes, remove build output"

# --- The stack ---

# Every compose target needs a .env, so it is a real prerequisite rather than a
# check repeated in each recipe.
.env:
	cp deploy/.env.example .env
	@echo "wrote .env from the example. Set SRCDS_RCONPW before starting."

# The seed goes to archipelago.gg, so it has to land on the host rather than in
# a volume. The directory is created here because Docker would create it as
# root, and the image generates as an unprivileged user.
seed: .env
	mkdir -p seed
	$(COMPOSE_SEED) run --rm --build seed

up: .env
	$(COMPOSE) up -d

down: .env
	$(COMPOSE) down

restart: down up

logs: .env
	$(COMPOSE) logs -f --tail=200

ps: .env
	$(COMPOSE) ps

# Read by hand, not sourced: .env holds unquoted values with spaces, which a shell cannot source.
# The server reads SRCDS_RCONPW at boot, so a value changed since then needs 'make restart'.
RCON := SRCDS_RCONPW="$$(sed -n 's/^SRCDS_RCONPW=//p' .env)" \
	SRCDS_PORT="$$(sed -n 's/^SRCDS_PORT=//p' .env)" \
	python3 deploy/rcon.py

# Silenced so the password does not reach the terminal in the echoed recipe.
rcon: .env
	@$(RCON) $(CMD)

build: .env
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

PYTHON_SRC := apworld/ deploy/rcon.py

apworld-fmt:
	$(RUFF) format $(PYTHON_SRC)

apworld-lint:
	$(RUFF) format --check $(PYTHON_SRC)
	$(RUFF) check $(PYTHON_SRC)

# The apworld's tests need Archipelago to run inside, so they run in the image
# that has it. The stage puts the world back in worlds/ as a folder and drops
# the packaged copy, because loading both would be the same game twice.
apworld-test:
	docker build --target apworld-test \
		--build-arg ARCHIPELAGO_VERSION=$(ARCHIPELAGO_VERSION) \
		-f deploy/Dockerfile.archipelago -t tf2-archipelago-apworld-test .
	docker run --rm tf2-archipelago-apworld-test

# The image's build stage already zips the world for custom_worlds/, which takes
# zips only. The release copies that zip rather than build a second one.
apworld-build:
	mkdir -p $(DIST)
	docker build --target build \
		--build-arg ARCHIPELAGO_VERSION=$(ARCHIPELAGO_VERSION) \
		-f deploy/Dockerfile.archipelago -t tf2-archipelago-apworld-build .
	id=$$(docker create tf2-archipelago-apworld-build); \
	docker cp "$$id:/ap/custom_worlds/tf2_mvm.apworld" $(DIST)/tf2_mvm.apworld; \
	docker rm "$$id"

# --- The plugin ---

plugin:
	./plugin/build.sh

# --- The defender bots ---

# Stages the MvM bot stack into deploy/bots/build/package: four plugins
# compiled from patched source, two extensions from the pinned upstream
# releases for both Linux and Windows. The image runs this in its own stage;
# this target is for looking at what it produces.
bots:
	./deploy/bots/build.sh

# The same, but compiling the two extensions here instead of downloading them.
# Needs clang and a 32-bit toolchain. Linux only, and only worth it when a TF2
# update breaks CBaseNPC and the fix has to be ours.
bots-from-source:
	BOTS_BUILD_EXTENSIONS=1 ./deploy/bots/build.sh

# --- The launcher ---

# The all-in-one Windows exe. The .smx and the ripext Windows zip are embedded
# in the binary, so launcher-assets fetches them into the embed dir first.
#
# Version strings are injected from deploy/env/versions.env via -ldflags, so the
# versions.env stays the single source of truth and a hand `go build` (which
# leaves them empty) is caught by assets.RequireVersions at runtime.
#
# The .smx is built by `make plugin`, which only runs on Linux (spcomp is a
# Linux binary). CI runs `make plugin` before `make launcher` on its Linux
# runner, so the real .smx is in place. On a non-Linux host the launcher still
# builds with whatever .smx the embed dir holds (a placeholder for dev), because
# the plugin compile is a separate concern.
EMBED := launcher/internal/assets/embedded
LAUNCHER_LDFLAGS := -X github.com/m-this/tf2-archipelago/launcher/internal/assets.SourcemodBranch=$(SOURCEMOD_BRANCH) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.SourcemodVersion=$(SOURCEMOD_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.MetamodBranch=$(MMSOURCE_BRANCH) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.MetamodVersion=$(MMSOURCE_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.RipextVersion=$(RIPEXT_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.ArchipelagoVersion=$(ARCHIPELAGO_VERSION)

# The bots go in as a Windows-only zip: the staged tree carries both platforms'
# extensions, and the 20 MB of Linux .so has no business inside a .exe.
launcher-assets: bots apworld-build
	mkdir -p $(EMBED)
	cp $(DIST)/tf2_mvm.apworld $(EMBED)/tf2_mvm.apworld
	rm -f $(EMBED)/defender-bots-windows.zip
	cd deploy/bots/build/package && zip -qr $(CURDIR)/$(EMBED)/defender-bots-windows.zip \
		addons -x '*.so'
	curl -fsSL -o $(EMBED)/sm-ripext-windows.zip \
		"https://github.com/ErikMinekus/sm-ripext/releases/download/$(RIPEXT_VERSION)/sm-ripext-$(RIPEXT_VERSION)-windows.zip"
	@if [ -f plugin/build/tf2_archipelago.smx ]; then \
		cp plugin/build/tf2_archipelago.smx $(EMBED)/tf2_archipelago.smx; \
		echo "copied plugin/build/tf2_archipelago.smx into the embed dir"; \
	else \
		echo "no plugin/build/tf2_archipelago.smx (run 'make plugin' on Linux, or CI will) — building with the placeholder"; \
	fi

# -H windowsgui links for the windows subsystem: a double-click opens the
# window and no console behind it. The flags that print keep working, because
# the launcher attaches to the terminal's console when it was given arguments.
launcher: launcher-assets
	mkdir -p $(DIST)
	go run github.com/akavel/rsrc@$(RSRC_VERSION) \
		-manifest launcher/cmd/tf2ap/tf2ap.manifest \
		-arch amd64 -o launcher/cmd/tf2ap/rsrc_windows_amd64.syso
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -H windowsgui $(LAUNCHER_LDFLAGS)" \
		-o $(DIST)/tf2ap.exe ./launcher/cmd/tf2ap

# --- Integration ---

# Archipelago and the bridge, for real, driven the way the plugin drives them.
# The game server is not in it: it downloads 14 GB and proves nothing without a
# human at a Team Fortress 2 client.
integration:
	./deploy/integration-test.sh

# --- The book ---

# In `check` because the book is a published site now: one that no longer builds
# should fail here rather than at the domain.
docs-build:
	docker build --build-arg HONKIT_VERSION=$(HONKIT_VERSION) \
		-f deploy/Dockerfile.docs -t tf2-archipelago-docs .

# Rebuilt on every run: honkit reads the whole of docs/ and the build takes
# seconds, so there is nothing to be gained from asking what changed.
docs: .env
	$(COMPOSE_DOCS) up -d --build
	@echo "the book is on http://127.0.0.1:$${DOCS_PORT:-8081}"

docs-down: .env
	$(COMPOSE_DOCS) down

# --- The release ---

# .github/workflows/release.yml calls this and builds nothing of its own.
dist: apworld-build plugin bots launcher compose-release
	cp plugin/build/tf2_archipelago.smx $(DIST)/
	cp apworld/tf2_mvm/data/*.json $(DIST)/
	cp deploy/.env.example $(DIST)/.env.example
	# The bot stack as one archive rooted at addons/, so a server that is not
	# this image installs it by unzipping into the game directory. Both the
	# Linux .so and the Windows .dll are in it: SourceMod takes the one its
	# platform needs. This is what a Windows server gets.
	rm -f $(DIST)/tf2-defender-bots.zip
	cd deploy/bots/build/package && zip -qr $(CURDIR)/$(DIST)/tf2-defender-bots.zip addons

# --no-interpolate leaves every ${VAR} alone, so the operator's .env still fills
# them in. The awk drops the build: blocks, which point at a repository a
# release has no copy of. The sed undoes the absolute path compose gave the seed
# bind mount.
compose-release:
	mkdir -p $(DIST)
	@{ \
		echo '# tf2-archipelago. Generated: rendered from deploy/compose.yml,'; \
		echo '# deploy/compose.seed.yml and deploy/compose.release.yml.'; \
		echo '#'; \
		echo '# Put it next to a .env, then:'; \
		echo '#'; \
		echo '#   docker compose --profile seed run --rm seed  # writes ./seed, upload it'; \
		echo '#   docker compose up -d'; \
		echo '#'; \
		echo '# TF2AP_VERSION picks the release the images come from.'; \
		echo '# https://github.com/m-this/tf2-archipelago'; \
		$(COMPOSE_RELEASE) --profile selfhost --profile seed config --no-interpolate \
			| awk '$$0 == "    build:" { skip = 1; next } skip { if (match($$0, /^      /)) next; skip = 0 } { print }' \
			| sed 's|$(CURDIR)/|./|g'; \
	} > $(DIST)/compose.yaml

# A Go test keeps the plugin and the apworld manifest on one version. The tag is
# the third place that names it, and no file in the tree can read a tag.
version-check:
	@want=$${VERSION:?pass VERSION=1.0.0}; \
	got=$$(sed -n 's/.*"world_version": "\([^"]*\)".*/\1/p' apworld/tf2_mvm/archipelago.json); \
	if [ "$$want" != "$$got" ]; then \
		echo "the tag says $$want, apworld/tf2_mvm/archipelago.json says $$got" >&2; \
		exit 1; \
	fi

# --- The gate ---

# Everything CI runs, cheapest failure first. Green here means green there.
check: fmt-check lint fix-check compile test vuln apworld-lint plugin apworld-test docs-build compose-release integration

clean: .env
	$(COMPOSE) down -v
	$(COMPOSE_SEED) down -v
	$(COMPOSE_TEST) down -v
	$(COMPOSE_DOCS) down -v
	rm -rf plugin/build/ deploy/bots/build/ $(DIST)/
