GO_BIN ?= go
all: build install
.PHONY: all

build:
	$(GO_BIN) build ./cmd/ent

install:
	$(GO_BIN) install ./cmd/ent
