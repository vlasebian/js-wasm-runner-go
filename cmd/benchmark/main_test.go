package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/dop251/goja"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Run: go test ./cmd/wasm-benchmark -bench=.
// For more options: go help testflag (e.g., -benchtime=5s, -cpu=4 for parallel).
// Add more benchmarks (e.g., BenchmarkTargetFunctionLarge) for different inputs/sizes.
// For allocations/memory: go test ./cmd/wasm-benchmark -bench=. -benchmem

func BenchmarkWASM(b *testing.B) {
	wasmPath := "../../wasm/qjs.wasm"
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		log.Fatalf("read qjs.wasm: %v", err)
	}

	rtc := wazero.NewRuntimeConfig().
		WithDebugInfoEnabled(true)

	ctx := context.Background()
	rt := wazero.NewRuntimeWithConfig(ctx, rtc)
	defer rt.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	jsCode, err := os.ReadFile("../../js/bench.qjs")
	if err != nil {
		log.Fatalf("read js code: %v", err)
	}
	_ = jsCode

	qjsEvalModConfig := wazero.NewModuleConfig().
		WithName("qjs").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin).
		WithArgs([]string{"qjs", "-e", string(jsCode)}...)

	qjsEvalMod, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error compiling quickjs wasm: %v\n", err)
		return
	}
	defer qjsEvalMod.Close(ctx)

	for b.Loop() {
		qjsEval, err := rt.InstantiateModule(ctx, qjsEvalMod, qjsEvalModConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error instantiating quickjs wasm: %v\n", err)
			return
		}
		if err := qjsEval.Close(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error closing quickjs wasm: %v\n", err)
			return
		}
	}
}

func BenchmarkGoja(b *testing.B) {
	vm := goja.New()
	script, err := os.ReadFile("../../js/bench.qjs")
	if err != nil {
		log.Fatalf("error reading script: %v", err)
	}
	scriptStr := string(script)
	for b.Loop() {
		_, err := vm.RunString(scriptStr)
		if err != nil {
			log.Fatalf("error running goja: %v", err)
		}
	}
}
