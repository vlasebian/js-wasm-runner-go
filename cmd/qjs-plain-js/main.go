package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func main() {
	wasmPath := "./wasm/qjs.wasm"
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

	// qjsArgsModConfig := wazero.NewModuleConfig().
	// 	WithName("qjs").
	// 	WithFS(os.DirFS(".")).
	// 	WithStdout(os.Stdout).
	// 	WithStderr(os.Stderr).
	// 	WithStdin(os.Stdin).
	// 	WithArgs([]string{"qjs", "./js/console-hello.js"}...)

	// qjsArgsMod, err := rt.CompileModule(ctx, wasm)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "error compiling quickjs wasm: %v\n", err)
	// 	return
	// }
	// defer qjsArgsMod.Close(ctx)

	// qjsArgs, err := rt.InstantiateModule(ctx, qjsArgsMod, qjsArgsModConfig)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "error instantiating quickjs wasm: %v\n", err)
	// 	return
	// }
	// defer qjsArgs.Close(ctx)

	jsCode, err := os.ReadFile("./js/console-hello.js")
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

	qjsEval, err := rt.InstantiateModule(ctx, qjsEvalMod, qjsEvalModConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error instantiating quickjs wasm: %v\n", err)
		return
	}
	defer qjsEval.Close(ctx)
}
