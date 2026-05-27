.PHONY: build-web embed build dev clean

build-web:
	cd web && bun install && bun run build

embed: build-web
	mkdir -p internal/managementasset/static
	cp web/dist/index.html internal/managementasset/static/management.html

build: embed
	go build -o llmhub ./cmd/server/

dev: embed
	go run ./cmd/server/

clean:
	rm -rf web/dist web/node_modules internal/managementasset/static llmhub
