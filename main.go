package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	wasmPath := "./static-hello.wasm"
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

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	_, err = rt.InstantiateModule(ctx, compiledMod, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error instantiating wasm binary: %v\n", err)
		return
	}

	// quickJSWasm, err := os.ReadFile("./quickjs.plugin.wasm")
	// if err != nil {
	// 	log.Fatalf("read wasm: %v", err)
	// }

	// _, err = r.Instantiate(ctx, quickJSWasm)
	// if err != nil {
	// 	log.Fatalf("instantiate wasm: %v", err)
	// }

	// modConfig := wazero.NewModuleConfig().
	// 	WithStdout(os.Stdout).
	// 	WithStderr(os.Stderr)

	// mod, err := r.InstantiateWithConfig(ctx, wasm, modConfig)
	// if err != nil {
	// 	log.Fatalf("instantiate wasm: %v", err)
	// }
	// if mod == nil {
	// 	log.Fatalf("module is nil")
	// }
	// if mod.ExportedFunction("main") == nil {
	// 	log.Fatalf("function main is not exported")
	// }

	// _, err = mod.ExportedFunction("main").Call(ctx)
	// if err != nil {
	// 	log.Fatalf("call wasm: %v", err)
	// }
}
