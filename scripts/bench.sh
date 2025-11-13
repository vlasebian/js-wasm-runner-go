#!/bin/bash
#
# Usage: bench.sh <vendor> <model>
#
# Example runs: 
#   ./scripts/bench.sh abeeway asset-tracker-2
#   ./scripts/bench.sh tektelic t000589x
# (make sure there is a file available under <vendor>/<model>-codec.yaml)

set -euo pipefail

CODEC_VENDOR=$1
CODEC_MODEL=$2

RUN_COUNT=10
BENCHMARK_PKG="./cmd/codecbench"

OUTPUT_FILE="benchstat-summary-$CODEC_VENDOR-$CODEC_MODEL-$(date +%Y-%m-%d-%H-%M-%S).txt"

# Fail if benchstat is not installed.
if ! command -v benchstat &> /dev/null; then
  echo "benchstat could not be found"
  echo "install with: go install golang.org/x/perf/cmd/benchstat@latest"
  exit 1
fi

GOJA_RESULTS_FILE="goja-$CODEC_VENDOR-$CODEC_MODEL-$(date +%Y-%m-%d-%H-%M-%S).txt"
CODEC_RUNNER=goja CODEC_VENDOR=${CODEC_VENDOR} CODEC_DEVICE=${CODEC_MODEL} \
  go test -count=${RUN_COUNT} -benchmem -bench=. ${BENCHMARK_PKG} | tee ${GOJA_RESULTS_FILE}

WASM_RESULTS_FILE="wasm-$CODEC_VENDOR-$CODEC_MODEL-$(date +%Y-%m-%d-%H-%M-%S).txt"
CODEC_RUNNER=wasm CODEC_VENDOR=${CODEC_VENDOR} CODEC_DEVICE=${CODEC_MODEL} \
  go test -count=${RUN_COUNT} -benchmem -bench=. ${BENCHMARK_PKG} | tee ${WASM_RESULTS_FILE}

BENCHSTAT_RESULTS_FILE="benchstat-$CODEC_VENDOR-$CODEC_MODEL-$(date +%Y-%m-%d-%H-%M-%S).txt"
benchstat -format=csv ${GOJA_RESULTS_FILE} ${WASM_RESULTS_FILE} > ${BENCHSTAT_RESULTS_FILE}

awk -F',' '
BEGIN {
    print "| Metric | goja | wasm | Change | P-value |"
    print "|--------|------|------|---------|---------|"
}

# Identify which section we are in (sec/op, B/op, allocs/op)
/^,sec\/op/     { section="sec/op"; next }
/^,B\/op/       { section="B/op";   next }
/^,allocs\/op/  { section="allocs/op"; next }

# Extract the useful rows
/^CodecRunner/ {
    name=$1
    go_val=$2
    go_ci=$3
    wasm_val=$4
    wasm_ci=$5
    diff=$6
    p=$7

    go_fmt = go_val " ±" go_ci
    wasm_fmt = wasm_val " ±" wasm_ci

    print "| " section " | " go_fmt " | " wasm_fmt " | " diff " | " p " |"
}
' ${BENCHSTAT_RESULTS_FILE} > ${OUTPUT_FILE}

echo "Benchmark summary written to ${OUTPUT_FILE}"
# rm goja.txt wasm.txt bench.csv
mdcat ${OUTPUT_FILE}
