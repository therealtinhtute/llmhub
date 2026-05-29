REPO ?= therealtinhtute/llmhub
BINARY ?= llmhub
PREFIX ?= /usr/local/bin
RELEASE_BRANCH ?= master
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X 'main.Version=$(VERSION)' -X 'main.Commit=$(COMMIT)' -X 'main.BuildDate=$(BUILD_DATE)'
GORELEASER_VERSION ?= v2.16.0
GORELEASER ?= $(shell command -v goreleaser 2>/dev/null || echo "go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)")

.PHONY: help build-web embed build dev release-check release-snapshot release-preflight release download-latest install-latest clean

help:
	@echo "Available commands:"
	@echo "  make build             Build web assets and compile $(BINARY)"
	@echo "  make dev               Build web assets and run the server"
	@echo "  make build-web         Build the React management panel"
	@echo "  make embed             Build and embed the management panel"
	@echo "  make release-check     Validate .goreleaser.yml with $(GORELEASER)"
	@echo "  make release-snapshot  Build local GoReleaser snapshot archives"
	@echo "  make release-preflight Verify the worktree and local release build prerequisites"
	@echo "  make release           Create and push TAG=... to trigger the GitHub release workflow"
	@echo "  make download-latest   Download the latest GitHub Release binary"
	@echo "  make install-latest    Install the latest GitHub Release binary to PREFIX=$(PREFIX)"
	@echo "  make clean             Remove build, web, and release artifacts"

build-web:
	cd web && bun install && bun run build

embed: build-web
	mkdir -p internal/managementasset/static
	cp web/dist/index.html internal/managementasset/static/management.html

build: embed
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/server/

dev: embed
	go run ./cmd/server/

release-check:
	$(GORELEASER) check

release-snapshot:
	$(GORELEASER) release --snapshot --clean

release-preflight:
	@set -eu; \
	worktree_status=$$(git status --porcelain 2>/dev/null); \
	if [ -n "$$worktree_status" ]; then \
		echo "release-preflight requires a clean git worktree" >&2; \
		exit 1; \
	fi
	git fetch origin $(RELEASE_BRANCH) --tags
	$(MAKE) release-check
	$(MAKE) release-snapshot
	@set -eu; \
	head_sha=$$(git rev-parse HEAD); \
	remote_sha=$$(git rev-parse refs/remotes/origin/$(RELEASE_BRANCH)); \
	if [ "$$head_sha" != "$$remote_sha" ]; then \
		echo "HEAD ($$head_sha) is not up to date with origin/$(RELEASE_BRANCH) ($$remote_sha)" >&2; \
		exit 1; \
	fi; \
	echo "release preflight passed for origin/$(RELEASE_BRANCH)"

release: release-preflight
	@set -eu; \
	tag="$(TAG)"; \
	if [ -z "$$tag" ]; then \
		echo "TAG is required, for example: make release TAG=v1.2.3" >&2; \
		exit 1; \
	fi; \
	if ! printf '%s\n' "$$tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?(\+([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?$$'; then \
		echo "TAG must be a SemVer release tag like v1.2.3 or v1.2.3-rc1" >&2; \
		exit 1; \
	fi; \
	if git show-ref --verify --quiet "refs/tags/$$tag"; then \
		echo "tag $$tag already exists locally" >&2; \
		exit 1; \
	fi; \
	if git ls-remote --exit-code --tags origin "refs/tags/$$tag" >/dev/null 2>&1; then \
		echo "tag $$tag already exists on origin" >&2; \
		exit 1; \
	fi; \
	git tag -a "$$tag" -m "$$tag"; \
	git push origin "refs/tags/$$tag"; \
	echo "Pushed $$tag. GitHub Actions is now responsible for publishing the release."; \
	echo "Watch the workflow:"; \
	echo "  gh run list --workflow release.yaml --branch $$tag --limit 1"; \
	echo "  gh run watch \"$$(gh run list --workflow release.yaml --branch $$tag --json databaseId --jq '.[0].databaseId')\" --exit-status"; \
	echo "Inspect the published release:"; \
	echo "  gh release view $$tag"

download-latest:
	@set -eu; \
	os=$$(go env GOOS); \
	arch=$$(go env GOARCH); \
	case "$$arch" in arm64) asset_arch=aarch64 ;; *) asset_arch=$$arch ;; esac; \
	case "$$os" in windows) ext=zip ;; *) ext=tar.gz ;; esac; \
	latest_url=$$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$(REPO)/releases/latest"); \
	tag=$${latest_url##*/}; \
	version=$${tag#v}; \
	asset="$(BINARY)_$${version}_$${os}_$${asset_arch}.$${ext}"; \
	url="https://github.com/$(REPO)/releases/download/$${tag}/$${asset}"; \
	mkdir -p dist/downloads; \
	echo "Downloading $${url}"; \
	curl -fL "$${url}" -o "dist/downloads/$${asset}"; \
	echo "Saved dist/downloads/$${asset}"

install-latest:
	@set -eu; \
	os=$$(go env GOOS); \
	arch=$$(go env GOARCH); \
	case "$$arch" in arm64) asset_arch=aarch64 ;; *) asset_arch=$$arch ;; esac; \
	case "$$os" in windows) ext=zip ;; *) ext=tar.gz ;; esac; \
	case "$$os" in windows) exe=.exe ;; *) exe= ;; esac; \
	latest_url=$$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$(REPO)/releases/latest"); \
	tag=$${latest_url##*/}; \
	version=$${tag#v}; \
	asset="$(BINARY)_$${version}_$${os}_$${asset_arch}.$${ext}"; \
	url="https://github.com/$(REPO)/releases/download/$${tag}/$${asset}"; \
	archive="dist/downloads/$${asset}"; \
	if [ ! -f "$${archive}" ]; then \
		mkdir -p dist/downloads; \
		echo "Downloading $${url}"; \
		curl -fL "$${url}" -o "$${archive}"; \
	fi; \
	tmpdir=$$(mktemp -d); \
	trap 'rm -rf "$${tmpdir}"' EXIT; \
	case "$${ext}" in \
		zip) unzip -q "$${archive}" -d "$${tmpdir}" ;; \
		tar.gz) tar -xzf "$${archive}" -C "$${tmpdir}" ;; \
	esac; \
	install -d "$(PREFIX)"; \
	install -m 0755 "$${tmpdir}/$(BINARY)$${exe}" "$(PREFIX)/$(BINARY)$${exe}"; \
	echo "Installed $(BINARY)$${exe} to $(PREFIX)/$(BINARY)$${exe}"

clean:
	rm -rf web/dist web/node_modules internal/managementasset/static $(BINARY) dist
