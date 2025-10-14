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

func main() {
	wasmPath := "./wasm/static-hello.wasm"
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
