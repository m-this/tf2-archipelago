SHELL := /bin/sh

# Every tool version comes from one file. Never inline a version here: compose,
# the plugin build and this Makefile all read the same values, so they cannot
# drift apart.
include deploy/env/versions.env
export

# go.mod owns the Go version: the `go` directive must be a literal.
GO_VERSION := $(shell sed -n 's/^go //p' go.mod)
export GO_VERSION

# go.mod owns the defender mod's version too. It is a module dependency rather
# than a checkout, so the requirement is the pin and there is no line in
# versions.env to keep in step with it. Read with sed rather than `go list -m`
# so a Makefile parse costs nothing and works without a toolchain; the build
# itself resolves it properly.
DEFENDERBOTS_VERSION := $(shell sed -n 's|^[[:space:]]*github.com/m-this/tf2-mvm-bots-go \(v[^ ]*\).*|\1|p' go.mod)
export DEFENDERBOTS_VERSION

# The apworld owns the release version, because that is the one a release tag is
# checked against (see version-check). Everything that has to state a version of
# this project reads it from here.
RELEASE_VERSION := $(shell sed -n 's/.*"world_version": "\([^"]*\)".*/\1/p' apworld/tf2_mvm/archipelago.json)

# Which build this is, for the window title and the debug bundle. Between
# releases a dozen builds carry the same version, and the commit is the only
# thing that tells them apart. A tree with uncommitted work says so, because a
# report from one cannot be traced to anything else.
BUILD_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)$(shell git diff --quiet HEAD 2>/dev/null || echo +dirty)

# Which channel this build came from, for the window title and the bundle.
#
# A build made from main after 1.9.0 called itself "1.9.0-837b556", which reads
# as the release it is not: a player on it reports a 1.9.0 bug and nobody knows
# the difference until somebody reads the commit. The nightly workflow passes
# BUILD_CHANNEL=nightly so the title says what it is.
#
# Only the displayed string. RELEASE_VERSION stays the release the apworld owns,
# because the Windows version resource below takes it as $(RELEASE_VERSION).0
# and that field has to be four numbers.
BUILD_CHANNEL ?=
LAUNCHER_VERSION := $(if $(BUILD_CHANNEL),$(BUILD_CHANNEL),$(RELEASE_VERSION))-$(BUILD_COMMIT)

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

# The browser interface. `ng build` writes straight into the package that
# serves it, so there is one copy of the bundle and it cannot go stale.
WEB := launcher/web
SPA_DIST := launcher/internal/spa/dist
GEN_GO := launcher/internal/gen/tf2ap/launcher/v1
PROTO_SRC := $(shell find proto -name '*.proto')

# Tools of record: pinned and run through `go run` or `uv run`, so no host
# install is needed and a local run is byte-identical to CI.
GOFUMPT := go run mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
RUFF := uv run --quiet --with ruff==$(RUFF_VERSION) ruff
BUF := go run github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
# npm is the one tool of record that is not a Go program, so it runs in a
# container the way honkit does: nothing is installed on a laptop, and the
# -direct targets below are for CI, which already stands in a node image.
# The whole repository is mounted, not just $(WEB): `ng build` writes its output
# into launcher/internal/spa, which is outside the frontend project.
NPM := docker run --rm -u $$(id -u):$$(id -g) \
	-v $(CURDIR):/work -w /work/$(WEB) \
	-e HOME=/tmp -e npm_config_cache=/tmp/.npm \
	node:$(NODE_VERSION) npm
# Ours only, and ours means written here. deploy/bots/build/ holds seven
# repositories this project fetches and compiles, and one of them now carries Go
# of its own; launcher/web holds npm's tree, which carries Go too; and
# launcher/internal/gen is buf's output, which answers to the .proto files.
# Formatting somebody else's tree is not this project's business, and a fresh
# checkout of one must not be able to fail our own format check.
GO_SRC := $$(find . -type f -name '*.go' -not -path './deploy/bots/build/*' -not -path './launcher/internal/gen/*' -not -path './launcher/web/*')

.PHONY: help seed up down restart logs ps rcon community-check \
        check fmt fmt-check vet lint lint-fix fix-check vuln compile test \
        test-fast export apworld-lint \
		apworld-fmt apworld-test apworld-build apworld-package plugin bots bots-from-source \
        integration build docs \
        docs-build docs-down dist compose-release version-check clean \
        go-version-check \
        launcher launcher-assets launcher-assets-common \
        proto proto-lint proto-fmt proto-deps \
        web-ready web-install web-build tracker-data tracker-build web-lint web-test web-e2e web-e2e-real \
        web-captures web-check \
        web-ready-direct web-install-direct web-build-direct web-lint-direct \
        web-test-direct \
        web-e2e-direct web-check-direct \
        launcher-linux launcher-assets-linux captures embed-placeholders toolchain

help:
	@echo "tf2-archipelago"
	@echo "  make seed          Generate a seed in ./seed to upload to archipelago.gg"
	@echo "  make up            Start the stack (docker compose)"
	@echo "  make down          Stop the stack"
	@echo "  make logs          Follow logs"
	@echo "  make rcon          Send a server command: make rcon CMD='sm_ap_status'"
	@echo "  make check         The gate: everything CI runs"
	@echo "  make proto         Regenerate the launcher contract from proto/"
	@echo "  make web-build     Build the browser interface into the launcher"
	@echo "  make tracker-build Build the public campaign tracker into ./dist/tracker"
	@echo "  make web-e2e       Drive the interface in a browser against the fake launcher"
	@echo "  make export        Regenerate apworld/tf2_mvm/data from gamedata/"
	@echo "  make community-check Validate community.json against community-content/tf"
	@echo "  make plugin        Compile the SourceMod plugin"
	@echo "  make bots          Stage the MvM defender bots the image installs"
	@echo "  make apworld-test  Run the apworld's tests inside Archipelago"
	@echo "  make integration   Bring up Archipelago and the bridge, drive them"
	@echo "  make dist          Build everything a release attaches into ./dist"
	@echo "  make launcher      Cross-compile tf2ap.exe (Windows) into ./dist"
	@echo "  make launcher-linux Build tf2ap-linux-amd64 into ./dist"
	@echo "  make captures      Redraw the terminal captures in docs/images"
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
	go run ./launcher/cmd/rcon

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

# The launcher embeds build artefacts, and Go refuses to compile a package
# whose //go:embed pattern matches nothing. Every Go target here depends on
# this so a fresh clone can vet, lint, build and test without first building
# the plugin, staging the bots and running the apworld through Docker.
# The placeholders are a zero-byte plugin and the smallest valid zip; the
# launcher-assets targets replace them with the real files and never see these.
#
# Both platforms are listed. A build tag picks which pair a binary embeds, but
# lint and vet read every file in the package whatever they are building for.
EMBED_PLACEHOLDERS = $(EMBED)/tf2_archipelago.smx $(EMBED)/tf2_mvm.apworld \
	$(EMBED)/sm-ripext-windows.zip $(EMBED)/defender-bots-windows.zip \
	$(EMBED)/sm-ripext-linux.zip $(EMBED)/defender-bots-linux.zip

embed-placeholders:
	@mkdir -p $(EMBED)
	@for f in $(EMBED_PLACEHOLDERS); do \
		[ -e "$$f" ] && continue; \
		case "$$f" in \
			*.smx) : > "$$f" ;; \
		*) printf 'PK\005\006\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000' > "$$f" ;; \
		esac; \
	done
	@mkdir -p $(SPA_DIST)
	@[ -e $(SPA_DIST)/index.html ] || \
		printf '<!doctype html><meta charset="utf-8"><title>tf2ap</title>\n<p>The browser interface was not built. Run <code>make web-build</code>.\n' \
			> $(SPA_DIST)/index.html

# --- The browser interface ---
#
# proto/ is the contract between the launcher and the Angular app. Both sides
# are generated and neither is committed: one source of truth, and a checked-in
# copy could only ever be a second answer to what the .proto files say. So every
# Go target depends on `proto`, which is a no-op once the tree is newer than the
# protos it was generated from.
proto: $(GEN_GO)

$(GEN_GO): $(PROTO_SRC) proto/buf.gen.yaml proto/buf.yaml proto/buf.lock
	cd proto && $(BUF) generate
	@touch $(GEN_GO)

proto-lint:
	cd proto && $(BUF) lint
	cd proto && $(BUF) format --diff --exit-code

proto-fmt:
	cd proto && $(BUF) format -w

# buf.lock names the commit of every dependency, so moving one is a decision
# rather than something a build does behind you. Not in `check`.
proto-deps:
	cd proto && $(BUF) dep update

web-install:
	$(NPM) ci --prefer-offline --no-audit --no-fund

# Everything a frontend target needs that a fresh clone does not have.
#
# Three things, and forgetting any one of them fails only in CI, because a
# machine that has built once already has all three:
#
#   proto               the generated contract. eslint is type-aware, so
#                       without the TypeScript every @gen import is unresolved
#                       and the no-unsafe-* family fires on every line.
#   embed-placeholders  the files internal/assets embeds. The fake launcher is
#                       a Go binary and will not compile without them.
#   node_modules        the obvious one.
#
# Named once so a target added later cannot quietly want a fourth thing and
# get away with it on a laptop that already has one.
web-ready: proto embed-placeholders web-install

web-build: web-ready
	$(NPM) run build

# The public tracker is a second entry into the same Angular source tree. It
# shares the launcher's components and theme, but its output needs no launcher
# or game server and can be served by any static host.
# Angular will not reach outside its workspace for assets. Stage the two
# generated catalogues rather than commit a second copy of either one.
tracker-data:
	mkdir -p $(WEB)/src/tracker/data
	cp apworld/tf2_mvm/data/missions.json apworld/tf2_mvm/data/weapon_classes.json $(WEB)/src/tracker/data/

tracker-build: web-ready tracker-data
	$(NPM) run build:tracker

web-lint: web-ready
	$(NPM) run lint
	$(NPM) run format:check

web-test: web-ready
	$(NPM) test

# The browser tests drive the real app against launcher/cmd/fakelauncher: the
# real handlers, the real WebSocket and the real form model, with no game server
# behind them. Playwright brings the fake up itself, so this needs a browser on
# the machine and not much else. Not in `check`: the browser is a 150 MB
# download that no other target needs, and CI installs it in the web job.
web-e2e: web-build
	cd $(WEB) && npx playwright test

# The browser tests against a launcher somebody started, rather than the fake.
# It asks the half a fake cannot: that the binary a player downloads carries the
# interface, serves it, and answers with a form.Model built from their own
# settings. Start one first, on either platform:
#
#   ./dist/tf2ap-linux-amd64 -no-browser -addr 127.0.0.1:8477
#   make web-e2e-real REAL=http://127.0.0.1:8477
REAL ?= http://127.0.0.1:8477

web-e2e-real:
	cd $(WEB) && TF2AP_REAL=$(REAL) npx playwright test e2e/real-launcher.spec.ts

# The pictures in the README and the book, redrawn against the fake launcher so
# the same run draws the same image every time: no machine's fonts, no player's
# home directory, no state left over from an evening of playing. Not in `check`:
# a screenshot that differs by a pixel is not a failure.
web-captures: web-build
	cd $(WEB) && TF2AP_CAPTURE=1 npx playwright test e2e/screenshots.spec.ts

web-check: web-lint web-test web-build tracker-build

# Direct targets: host npm, for CI, which already runs inside a node image, and
# for a developer who would rather not pay the container round trip.
web-install-direct:
	cd $(WEB) && npm ci --prefer-offline --no-audit --no-fund

web-ready-direct: proto embed-placeholders web-install-direct

web-build-direct: web-ready-direct
	cd $(WEB) && npm run build

web-lint-direct: web-ready-direct
	cd $(WEB) && npm run lint
	cd $(WEB) && npm run format:check

web-test-direct: web-ready-direct
	cd $(WEB) && npm test

web-e2e-direct: web-build-direct
	cd $(WEB) && npx playwright install --with-deps chromium
	cd $(WEB) && npx playwright test

web-check-direct: web-lint-direct web-test-direct web-build-direct

# Not in `check`: every analyzer it registers is in golangci-lint's govet.
vet: embed-placeholders proto
	go vet ./...

lint: embed-placeholders proto
	$(GOLANGCI_LINT) run ./...

lint-fix: embed-placeholders proto
	$(GOLANGCI_LINT) run --fix ./...

# `go fix` must be a no-op: whatever it would rewrite belongs in the commit.
# `-diff` reports without touching the tree, so this is safe on a dirty working
# copy.
fix-check: embed-placeholders proto
	@out="$$(go fix -diff ./...)"; \
	if [ -n "$$out" ]; then \
		printf '%s\n' "$$out"; \
		echo "go fix would rewrite code: apply it with 'go fix ./...' and commit"; \
		exit 1; \
	fi

# Reports only the vulnerabilities whose vulnerable symbol this code can
# actually reach, so a hit is a bug to fix rather than a number to argue with.
vuln: embed-placeholders proto
	$(GOVULNCHECK) ./...

compile: embed-placeholders proto
	go build ./...

# The race detector is the only tool that sees a data race, and the bridge is
# three goroutines around one state store. -shuffle=on breaks the accidental
# ordering that makes a suite pass in one order and fail in another.
#
# This also guards the committed export: TestCommittedExportIsCurrent
# regenerates it and fails if the tree is stale, which is why there is no
# separate freshness target.
# Not toolchain: CI runs this target on its own, and building SourcePawn from
# source there would add several minutes to every run for tests that CI has no
# way to require. `check` builds the toolchain before it gets here and sets
# TF2AP_REQUIRE_SPSHELL, so the gate runs the drivers and refuses to skip them.
# Run `make check` or `make toolchain` once to have them locally.
test: embed-placeholders proto
	CGO_ENABLED=1 $(SPENV) $(REQUIRE_SPSHELL) go test -race -shuffle=on ./...

test-fast: embed-placeholders proto
	$(SPENV) go test ./...

# --- SourcePawn under its own VM ---
#
# The plugin is the one component nothing here could run, so it was checked by
# reading its source for substrings. That catches a rename and misses a changed
# sum. spcomp and SourcePawn's standalone VM compile and run one function of it
# on inputs a test chooses, which catches the sum.
#
# The toolchain is the defender mod's: it pins the sourcepawn commit, patches
# the clang-only tree GCC rejects and caches the build. Running its script from
# the module cache rather than copying it means the pin cannot drift between
# the two repositories, and SPWORK puts the output here because the module
# cache is read-only.
SPWORK ?= $(CURDIR)/toolchain
SPROOT := $(SPWORK)/sourcepawn
SPENV := SPCOMP=$(SPROOT)/objdir/spcomp/linux-x86_64/spcomp \
	SPSHELL=$(SPROOT)/objdir/spshell/linux-x86_64/spshell \
	SPINCLUDE=$(SPROOT)/include/core

# Idempotent: a second run finds the two binaries and exits.
toolchain:
	SPWORK=$(SPWORK) sh $(BOTS_MOD)/tools/spshell.sh

export:
	go generate ./gamedata

COMMUNITY_CONTENT ?= ./community-content/tf
community-check:
	go run ./gamedata/cmd/communitycheck $(COMMUNITY_CONTENT)

# The weapon catalogue and the schema import read a TF2 install: any one will
# do, and ~/tf2-native is the one the bot test-bed keeps. The pools come from
# the bot mod this module pins, so the catalogue follows the version in go.mod.
TF2_DIR ?= $(HOME)/tf2-native/tf-dedicated/tf
BOTS_MOD = $$(go list -m -f '{{.Dir}}' github.com/m-this/tf2-mvm-bots-go)
weapons:
	go run ./gamedata/cmd/weapons \
		-pools $(BOTS_MOD)/plugin/source/redbots3/generated/loadouts.sp \
		-schema $(TF2_DIR)/scripts/items/items_game.txt \
		-english $(TF2_DIR)/resource/tf_english.txt > gamedata/weapons_generated.go

import-weapons:
	go run ./gamedata/cmd/importweapons $(TF2_DIR)/scripts/items/items_game.txt $(TF2_DIR)/resource/tf_english.txt

# --- The apworld ---

PYTHON_SRC := apworld/

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

# The standalone launcher only needs the packaged world, not an Archipelago
# server image. Archipelago 0.6.7's APWorldContainer format is version 7; the
# standard-library packager mirrors its archive layout and manifest stamping so
# WSL users do not need Docker (or the optional zip command) to build the exe.
apworld-package:
	go run ./launcher/cmd/packagezip apworld apworld/tf2_mvm $(DIST)/tf2_mvm.apworld \
		--container-version 7

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
# The .smx is built as part of every launcher asset build. spcomp is a Linux
# binary, so release launchers are built on Linux or WSL just like CI. A direct
# `go build` may still use the placeholder for compile-only development, but a
# launcher produced by this target must never silently package an old plugin.
EMBED := launcher/internal/assets/embedded
LAUNCHER_LDFLAGS := -X github.com/m-this/tf2-archipelago/launcher/internal/assets.SourcemodBranch=$(SOURCEMOD_BRANCH) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.SourcemodVersion=$(SOURCEMOD_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.MetamodBranch=$(MMSOURCE_BRANCH) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.MetamodVersion=$(MMSOURCE_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.RipextVersion=$(RIPEXT_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.ArchipelagoVersion=$(ARCHIPELAGO_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.DefenderbotsVersion=$(DEFENDERBOTS_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.LauncherVersion=$(LAUNCHER_VERSION)

# The bots go in as a Windows-only zip: the staged tree carries both platforms'
# extensions, and the 20 MB of Linux .so has no business inside a .exe.
# The apworld and the plugin, which are the same bytes on either platform.
launcher-assets-common: plugin bots apworld-package
	mkdir -p $(EMBED)
	cp $(DIST)/tf2_mvm.apworld $(EMBED)/tf2_mvm.apworld
	cp plugin/gamedata/tf2_archipelago.txt $(EMBED)/tf2_archipelago.txt
	cp plugin/build/tf2_archipelago.smx $(EMBED)/tf2_archipelago.smx

# One platform's binaries per build: SourceMod loads the .so or the .dll by
# platform and ignores the other, so each launcher carries only its own.
launcher-assets: launcher-assets-common
	go run ./launcher/cmd/packagezip tree deploy/bots/build/package \
		$(EMBED)/defender-bots-windows.zip --exclude-suffix .so
	curl -fsSL -o $(EMBED)/sm-ripext-windows.zip \
		"https://github.com/ErikMinekus/sm-ripext/releases/download/$(RIPEXT_VERSION)/sm-ripext-$(RIPEXT_VERSION)-windows.zip"

launcher-assets-linux: launcher-assets-common
	go run ./launcher/cmd/packagezip tree deploy/bots/build/package \
		$(EMBED)/defender-bots-linux.zip --exclude-suffix .dll
	curl -fsSL -o $(EMBED)/sm-ripext-linux.zip \
		"https://github.com/ErikMinekus/sm-ripext/releases/download/$(RIPEXT_VERSION)/sm-ripext-$(RIPEXT_VERSION)-linux.zip"

# -H windowsgui links for the windows subsystem: a double-click opens the
# window and no console behind it. The flags that print keep working, because
# the launcher attaches to the terminal's console when it was given arguments.
#
# The .syso carries the icon, the manifest and a VERSIONINFO resource. The last
# one is not cosmetic: an exe with no CompanyName, ProductName or FileVersion is
# an anonymous blob to SmartScreen and to Defender's heuristics, and this one
# already looks like a dropper to them, because it unpacks archives and starts a
# game server. Signing is item 2 in TODO.md; this is what costs nothing.
#
# pechecksum runs last because it rewrites a header field over the linked file,
# and the Go linker leaves that field at zero. It is in-tree rather than a pinned
# tool: the sum is fifteen lines, and the release job already has Go.
#
# The version is read from the apworld, which is what `version-check` compares a
# tag against, so the resource and the release cannot disagree. The manifest's
# assemblyIdentity gets the same number, which is why it is generated and not
# committed with a version baked into it.
launcher: launcher-assets web-build
	mkdir -p $(DIST)
	sed 's/version="0\.0\.0\.0"/version="$(RELEASE_VERSION).0"/' \
		launcher/cmd/tf2ap/tf2ap.manifest > $(DIST)/tf2ap.manifest
	go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@$(GOVERSIONINFO_VERSION) \
		-64 -platform-specific=false \
		-icon launcher/cmd/tf2ap/tf2ap.ico \
		-manifest $(DIST)/tf2ap.manifest \
		-file-version "$(RELEASE_VERSION).0" \
		-product-version "$(RELEASE_VERSION).0" \
		-propagate-ver-strings \
		-o launcher/cmd/tf2ap/rsrc_windows_amd64.syso \
		launcher/cmd/tf2ap/versioninfo.json
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -H windowsgui $(LAUNCHER_LDFLAGS)" \
		-o $(DIST)/tf2ap.exe ./launcher/cmd/tf2ap
	go run ./launcher/cmd/pechecksum $(DIST)/tf2ap.exe

# No window: walk is a Win32 binding, so the Linux build is the console flow
# the compose stack already uses. Everything else is the same program.
launcher-linux: launcher-assets-linux web-build
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w $(LAUNCHER_LDFLAGS)" \
		-o $(DIST)/tf2ap-linux-amd64 ./launcher/cmd/tf2ap

# The captures in the README and the book. They are SVG of the Linux
# launcher's own output, so a diff says what changed in one and no machine's
# font choices reach the picture.
# CAPTURE_ENV keeps whoever runs this out of the picture: the paths a capture
# prints come from the environment, and a committed image should not carry the
# home directory of the person who last redrew it.
#
# XDG_CONFIG_HOME points at an empty directory for the same reason. The
# launcher reads the config file before the environment, so without this a
# capture shows the port and the reach of the machine that drew it rather than
# what a player who has just installed it sees.
CAPTURE_ENV = TF2AP_INSTALL_ROOT=/home/player/tf2-archipelago \
	TF2AP_ARCHIPELAGO_DIR=/home/player/Archipelago \
	XDG_CONFIG_HOME=$(CURDIR)/dist/capture-config \
	AP_HOST=archipelago.gg AP_PORT=12345 SRCDS_RCONPW=hidden

# The sed is the last of it: the launcher lists every folder it looked in for
# the Archipelago app, and one of them is always the real home of whoever ran
# this.
captures: launcher-linux
	rm -rf dist/capture-config
	mkdir -p dist/capture-config
	$(CAPTURE_ENV) ./dist/tf2ap-linux-amd64 -version \
		| sed "s|$$HOME|/home/player|g" \
		| ./docs/capture.sh 'tf2ap-linux-amd64 -version' docs/images/linux-version.svg
	$(CAPTURE_ENV) ./dist/tf2ap-linux-amd64 -status \
		| sed "s|$$HOME|/home/player|g" \
		| ./docs/capture.sh 'tf2ap-linux-amd64 -status' docs/images/linux-status.svg

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
dist: apworld-build plugin bots launcher launcher-linux compose-release
	cp plugin/build/tf2_archipelago.smx $(DIST)/
	cp apworld/tf2_mvm/data/*.json $(DIST)/
	# env.example, not .env.example: gh release create renames an asset whose
	# name starts with a dot, so the documented URL 404'd on every release
	# while default.env.example quietly served the file (apw-6xe).
	cp deploy/.env.example $(DIST)/env.example
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
		$(COMPOSE_RELEASE) --profile selfhost --profile seed --profile tailscale-fastdl config --no-interpolate \
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
# go-version-check first: a builder on the wrong Go makes lint fail in a way
# that reads as a linter bug rather than a stale pin.
# Set on the gate, not on `test`, and the difference matters: CI runs `make
# test` on its own, so setting it there turned a skip into a failure on a
# machine that has no toolchain and no way to build one in reasonable time. The
# target-specific variable reaches test through the prerequisite, which is the
# whole point: the gate refuses to skip a differential test, and a developer
# who has not run `make toolchain` gets a skip that names what is missing.
check: REQUIRE_SPSHELL := TF2AP_REQUIRE_SPSHELL=1
check: go-version-check fmt-check proto-lint lint fix-check compile web-check toolchain test vuln apworld-lint plugin apworld-test docs-build compose-release integration

# The go directive owns the version. Two pins cannot read it, so this says when
# they have drifted rather than leaving it to whoever hits the failure.
go-version-check:
	./deploy/check-go-version.sh

clean: .env
	$(COMPOSE) down -v
	$(COMPOSE_SEED) down -v
	$(COMPOSE_TEST) down -v
	$(COMPOSE_DOCS) down -v
	rm -rf plugin/build/ deploy/bots/build/ $(DIST)/
