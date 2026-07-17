APP_NAME := allinme.core-api
MODULE   := github.com/magicvr/allinme.core-api
BIN_DIR  := bin
IMAGE    ?= allinme.core-api
VERSION  ?= 0.1.0
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -X $(MODULE)/pkg/version.Version=$(VERSION) \
            -X $(MODULE)/pkg/version.Commit=$(COMMIT) \
            -X $(MODULE)/pkg/version.BuiltAt=$(BUILT_AT)

.PHONY: run build test tidy fmt vet clean \
	docker-build docker-run docker-up docker-down docker-logs docker-rebuild

run:
	go run ./cmd/server

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

# ---- Docker ----

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILT_AT=$(BUILT_AT) \
		-t $(IMAGE):$(VERSION) \
		-t $(IMAGE):latest \
		.

docker-run: docker-build
	docker run --rm -p 8080:8080 \
		-e APP_ENV=development \
		-e LOG_LEVEL=info \
		--name allinme-core-api \
		$(IMAGE):$(VERSION)

docker-up:
	APP_VERSION=$(VERSION) COMMIT=$(COMMIT) BUILT_AT=$(BUILT_AT) \
		docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api

docker-rebuild:
	docker compose build --no-cache
	APP_VERSION=$(VERSION) COMMIT=$(COMMIT) BUILT_AT=$(BUILT_AT) \
		docker compose up -d
