WEB_DIR := web/admin
WEB_DIST := web/dist
EMBED_STATIC := internal/admin/http/static

.PHONY: test test-go test-web build build-web sync-web fetch-geo-data run

test: test-go test-web

test-go:
	go test ./... -v

test-web:
	cd $(WEB_DIR) && npm test

build: build-web sync-web
	go build ./cmd/pipescope

build-web:
	cd $(WEB_DIR) && npm run build

sync-web:
	rm -rf $(WEB_DIST)
	mkdir -p $(WEB_DIST)
	cp -R $(WEB_DIR)/dist/* $(WEB_DIST)/
	rm -rf $(EMBED_STATIC)
	mkdir -p $(EMBED_STATIC)
	cp -R $(WEB_DIR)/dist/* $(EMBED_STATIC)/

fetch-geo-data:
	./scripts/fetch-geo-data.sh

run:
	go run ./cmd/pipescope -config assets/config.example.yaml
