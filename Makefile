BIN_DIR  :=$(shell pwd)/bin
PKG_DIR  :=$(shell pwd)/pkg
WASM_DIR :=$(shell pwd)/wasm

all: 

.PHONY: deps
deps:
	./scripts/install.sh all

.PHONY: clean-deps
clean-deps:
	./scripts/install.sh clean
