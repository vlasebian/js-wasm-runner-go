package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func formatBytes(bytes []byte) string {
	s := strings.Builder{}
	s.WriteString("[ ")
	for _, b := range bytes {
		s.WriteString(fmt.Sprintf("0x%02X, ", b))
	}
	s.WriteString("]")
	return s.String()
}

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

	script, err := os.ReadFile("./js/the-things-uno-quickstart.js")
	if err != nil {
		log.Fatalf("read js code: %v", err)
	}

	bytes := []byte{0x00}
	fport := 1

	jsCode := fmt.Sprintf(`
%s

function main() {
	const bytes = %s;
	const fPort = %d;
	return decodeDownlink({ bytes, fPort });
}

let r = JSON.stringify(main())
console.log(r);
	`, script, formatBytes(bytes), fport)

	qjsEvalModConfig := wazero.NewModuleConfig().
		WithName("qjs").
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
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
