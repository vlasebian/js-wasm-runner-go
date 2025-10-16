#!/bin/bash

set -euo pipefail

RUN_COUNT=10
RUN_TIME=30s

# Fail if benchstat is not installed.
if ! command -v benchstat &> /dev/null; then
  echo "benchstat could not be found"
  echo "install with: go install golang.org/x/perf/cmd/benchstat@latest"
  exit 1
fi

for i in {1..$RUN_COUNT}; do
  go test ./cmd/benchmark -bench BenchmarkGoja -benchmem -benchtime=$RUN_TIME >> goja.txt
done

for i in {1..$RUN_COUNT}; do
  go test ./cmd/benchmark -bench BenchmarkWASM -benchmem -benchtime=$RUN_TIME >> wasm.txt
done

benchstat goja.txt wasm.txt > bench.txt