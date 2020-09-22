GO_BIN ?= go
all: ffi install build
.PHONY: all

ffi:
	git submodule update --init --recursive && cd extern/filecoin-ffi && $(MAKE)

build:
	$(GO_BIN) build ./cmd/ent

install:
	$(GO_BIN) install ./cmd/ent
