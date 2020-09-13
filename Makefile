GO_BIN ?= go
all: build
.PHONY: all

build:
	$(GO_BIN) build ./...
