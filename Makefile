APP_NAME := allinme.core-api
MODULE   := github.com/magicvr/allinme.core-api
BIN_DIR  := bin
VERSION  ?= 0.1.0
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -X $(MODULE)/pkg/version.Version=$(VERSION) \
            -X $(MODULE)/pkg/version.Commit=$(COMMIT) \
            -X $(MODULE)/pkg/version.BuiltAt=$(BUILT_AT)

.PHONY: run build test tidy fmt vet clean

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
