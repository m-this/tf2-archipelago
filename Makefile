SHELL := /bin/sh

# Every tool version comes from one file. Never inline a version here: compose,
# the plugin build and this Makefile all read the same values, so they cannot
# drift apart.
include deploy/env/versions.env
export

# go.mod owns the Go version: the `go` directive must be a literal.
GO_VERSION := $(shell sed -n 's/^go //p' go.mod)
export GO_VERSION

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

# Tools of record: pinned and run through `go run` or `uv run`, so no host
# install is needed and a local run is byte-identical to CI.
GOFUMPT := go run mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
RUFF := uv run --quiet --with ruff==$(RUFF_VERSION) ruff
SHADOW := uv run --quiet --with pillow==$(PILLOW_VERSION) python docs/shadow.py
# Ours only. deploy/bots/build/ holds seven repositories this project fetches
# and compiles, and one of them now carries Go of its own: formatting somebody
# else's tree is not this project's business, and a fresh checkout of it must
# not be able to fail our own format check.
GO_SRC := $$(find . -type f -name '*.go' -not -path './deploy/bots/build/*')

.PHONY: help seed up down restart logs ps rcon community-check \
        check fmt fmt-check vet lint lint-fix fix-check vuln compile test shadows \
        window-captures \
        test-fast export apworld-lint \
		apworld-fmt apworld-test apworld-build apworld-package plugin bots bots-from-source \
        integration build docs \
        docs-build docs-down dist compose-release version-check clean \
        go-version-check \
        launcher launcher-assets launcher-assets-common \
        launcher-linux launcher-assets-linux captures embed-placeholders

help:
	@echo "tf2-archipelago"
	@echo "  make seed          Generate a seed in ./seed to upload to archipelago.gg"
	@echo "  make up            Start the stack (docker compose)"
	@echo "  make down          Stop the stack"
	@echo "  make logs          Follow logs"
	@echo "  make rcon          Send a server command: make rcon CMD='sm_ap_status'"
	@echo "  make check         The gate: everything CI runs"
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
	@echo "  make shadows       Drop-shadow the window screenshots in docs/images/raw"
	@echo "  make window-captures Rephotograph the launcher's window through Wine"
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

# Not in `check`: every analyzer it registers is in golangci-lint's govet.
vet: embed-placeholders
	go vet ./...

lint: embed-placeholders
	$(GOLANGCI_LINT) run ./...

lint-fix: embed-placeholders
	$(GOLANGCI_LINT) run --fix ./...

# `go fix` must be a no-op: whatever it would rewrite belongs in the commit.
# `-diff` reports without touching the tree, so this is safe on a dirty working
# copy.
fix-check: embed-placeholders
	@out="$$(go fix -diff ./...)"; \
	if [ -n "$$out" ]; then \
		printf '%s\n' "$$out"; \
		echo "go fix would rewrite code: apply it with 'go fix ./...' and commit"; \
		exit 1; \
	fi

# Reports only the vulnerabilities whose vulnerable symbol this code can
# actually reach, so a hit is a bug to fix rather than a number to argue with.
vuln: embed-placeholders
	$(GOVULNCHECK) ./...

compile: embed-placeholders
	go build ./...

# The race detector is the only tool that sees a data race, and the bridge is
# three goroutines around one state store. -shuffle=on breaks the accidental
# ordering that makes a suite pass in one order and fail in another.
#
# This also guards the committed export: TestCommittedExportIsCurrent
# regenerates it and fails if the tree is stale, which is why there is no
# separate freshness target.
test: embed-placeholders
	CGO_ENABLED=1 go test -race -shuffle=on ./...

test-fast: embed-placeholders
	go test ./...

export:
	go generate ./gamedata

COMMUNITY_CONTENT ?= ./community-content/tf
community-check:
	go run ./gamedata/cmd/communitycheck $(COMMUNITY_CONTENT)

# --- The apworld ---

PYTHON_SRC := apworld/ deploy/rcon.py deploy/player-yaml.py

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
	python3 deploy/package-zip.py apworld apworld/tf2_mvm $(DIST)/tf2_mvm.apworld \
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
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.ArchipelagoVersion=$(ARCHIPELAGO_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.DefenderbotsVersion=$(DEFENDERBOTS_VERSION) \
	-X github.com/m-this/tf2-archipelago/launcher/internal/assets.LauncherVersion=$(LAUNCHER_VERSION)

# The bots go in as a Windows-only zip: the staged tree carries both platforms'
# extensions, and the 20 MB of Linux .so has no business inside a .exe.
# The apworld and the plugin, which are the same bytes on either platform.
launcher-assets-common: bots apworld-package
	mkdir -p $(EMBED)
	cp $(DIST)/tf2_mvm.apworld $(EMBED)/tf2_mvm.apworld
	cp plugin/gamedata/tf2_archipelago.txt $(EMBED)/tf2_archipelago.txt
	@if [ -f plugin/build/tf2_archipelago.smx ]; then \
		cp plugin/build/tf2_archipelago.smx $(EMBED)/tf2_archipelago.smx; \
		echo "copied plugin/build/tf2_archipelago.smx into the embed dir"; \
	else \
		echo "no plugin/build/tf2_archipelago.smx (run 'make plugin' on Linux, or CI will) — building with the placeholder"; \
	fi

# One platform's binaries per build: SourceMod loads the .so or the .dll by
# platform and ignores the other, so each launcher carries only its own.
launcher-assets: launcher-assets-common
	python3 deploy/package-zip.py tree deploy/bots/build/package \
		$(EMBED)/defender-bots-windows.zip --exclude-suffix .so
	curl -fsSL -o $(EMBED)/sm-ripext-windows.zip \
		"https://github.com/ErikMinekus/sm-ripext/releases/download/$(RIPEXT_VERSION)/sm-ripext-$(RIPEXT_VERSION)-windows.zip"

launcher-assets-linux: launcher-assets-common
	python3 deploy/package-zip.py tree deploy/bots/build/package \
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
launcher: launcher-assets
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
launcher-linux: launcher-assets-linux
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

# The launcher's window, photographed. walk is a Win32 binding, so the window
# runs under Wine on a virtual display and ImageMagick takes the picture; see
# docs/window-shot.sh for what that needs installed. A shot taken by hand on a
# real Windows machine and dropped in docs/images/raw/ goes through the same
# second half.
window-captures: launcher
	mkdir -p docs/images/raw
	./docs/window-shot.sh $(DIST)/tf2ap.exe docs/images/raw/launcher-main.png 30 main
	./docs/window-shot.sh $(DIST)/tf2ap.exe docs/images/raw/launcher-settings.png 30 dialog
	$(MAKE) shadows

# The drop shadow and the transparent margins, over whatever is in
# docs/images/raw/. Separate from taking the picture, because a picture taken
# on Windows needs this half and not the other one.
shadows:
	@for raw in docs/images/raw/*.png; do \
		[ -e "$$raw" ] || { echo "nothing in docs/images/raw"; exit 0; }; \
		$(SHADOW) "$$raw" "docs/images/$$(basename $$raw)"; \
	done

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
# go-version-check first: a builder on the wrong Go makes lint fail in a way
# that reads as a linter bug rather than a stale pin.
check: go-version-check fmt-check lint fix-check compile test vuln apworld-lint plugin apworld-test docs-build compose-release integration

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
