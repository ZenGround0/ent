GO_BIN ?= go
all: deps install
.PHONY: all

ffi:
	git submodule update --init --recursive && cd extern/filecoin-ffi && $(MAKE)

build:
	$(GO_BIN) build ./cmd/ent

install:
	$(GO_BIN) install ./cmd/ent
