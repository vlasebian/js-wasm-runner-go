package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed quickjs.plugin.wasm
var quickJSWasm []byte

//go:embed dyn-hello.wasm
var dynHelloWasm []byte

func main() {
	quickJSwasmPath := "./quickjs.plugin.wasm"
	quickJSwasmArgs := []string{}
	quickJSwasmBin := filepath.Base(quickJSwasmPath)
	quickJSwasm, err := os.ReadFile(quickJSwasmPath)
	if err != nil {
		log.Fatalf("read quickjs wasm: %v", err)
	}

	wasmPath := "./dyn-hello.wasm"
	wasmArgs := []string{}
	// Binary name without path.
	wasmBin := filepath.Base(wasmPath)
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		log.Fatalf("read wasm: %v", err)
	}

	rtc := wazero.NewRuntimeConfig()
	rtc = rtc.WithDebugInfoEnabled(true)

	ctx := context.Background()
	rt := wazero.NewRuntimeWithConfig(ctx, rtc)
	defer rt.Close(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	quickJSModConfig := wazero.NewModuleConfig().
		WithName("javy-default-plugin-v1").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin).
		WithArgs(append([]string{quickJSwasmBin}, quickJSwasmArgs...)...)

	quickJSMod, err := rt.CompileModule(ctx, quickJSwasm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error compiling quickjs wasm: %v\n", err)
		return
	}
	_, err = rt.InstantiateModule(ctx, quickJSMod, quickJSModConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error instantiating quickjs wasm: %v\n", err)
		return
	}

	cfg := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin).
		WithArgs(append([]string{wasmBin}, wasmArgs...)...)

	compiledMod, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error compiling wasm binary: %v\n", err)
		return
	}

	_, err = rt.InstantiateModule(ctx, compiledMod, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error instantiating wasm binary: %v\n", err)
		return
	}
}
