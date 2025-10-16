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

run-qjs-plain-js:
	go run cmd/qjs-plain-js/main.go

run-javy-dyn:
	./bin/javy emit-plugin -o wasm/quickjs.plugin.wasm
	./bin/javy build -o wasm/dyn-hello.wasm -C plugin=wasm/quickjs.plugin.wasm js/console-hello.js
	go run cmd/javy-dyn/main.go

run-javy-static:
	./bin/javy build -o wasm/static-hello.wasm js/console-hello.js
	go run cmd/javy-static/main.go
	
run-codec-example:
	go run cmd/codec-example/main.go

stress:
	/bin/wazero run -mount=/tmp -mount=. ./wasm/qjs.wasm --memory-limit 4096 --stack-size 2 ./js/stress-test.qjs
	
bench:
	./scripts/bench.sh