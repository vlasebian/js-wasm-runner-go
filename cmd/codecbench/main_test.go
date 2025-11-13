package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	BenchCodecBaseDir = "../../lorawan-devices/vendor"
)

func BenchmarkCodecRunner(b *testing.B) {
	vendor, device := "tektelic", "t000589x"

	if v := os.Getenv("CODEC_VENDOR"); v != "" {
		vendor = v
	}
	if d := os.Getenv("CODEC_DEVICE"); d != "" {
		device = d
	}

	jsPath, jsCode, input, err := loadDeviceCodec(BenchCodecBaseDir, vendor, device)
	if err != nil {
		log.Fatalf("LoadDeviceCodec: %v", err)
	}
	jsCodeStr := string(jsCode)
	b.Logf("codec: %s, size: %.02fkb", jsPath, float64(len(jsCodeStr))/1024)

	mode := os.Getenv("CODEC_RUNNER")
	switch {
	case mode == "goja":
		benchmarkGoja(b, jsCodeStr, input)
		return
	case mode == "wasm":
		benchmarkWASM(b, jsCodeStr, input)
		return
	default:
		b.Fatal("Please set CODEC_RUNNER environment variable to either 'Goja' or 'WASM'")
	}
}

func benchmarkWASM(b *testing.B, jsCodeStr string, input decodeUplinkInput) {
	rtc := wazero.NewRuntimeConfig().
		WithDebugInfoEnabled(true)

	ctx := context.Background()
	rt := wazero.NewRuntimeWithConfig(ctx, rtc)
	defer rt.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	wasm, err := os.ReadFile("../../wasm/qjs.wasm")
	if err != nil {
		b.Fatal("read qjs.wasm:", err)
	}

	qjsEvalMod, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		b.Fatal("error compiling quickjs wasm:", err)
	}
	defer qjsEvalMod.Close(ctx)

	wrappedScript := wrapUplinkDecoderScriptForWASM(jsCodeStr,
		jsObjectFromDecodeUplinkInput(input))

	qjsEvalModConfig := wazero.NewModuleConfig().
		WithName("qjs").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin).
		WithArgs([]string{"qjs", "--std", "-C", "-e", string(wrappedScript)}...)

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

func benchmarkGoja(b *testing.B, jsCodeStr string, input decodeUplinkInput) {
	wrappedScript := wrapUplinkDecoderScriptForGoja(jsCodeStr)

	for b.Loop() {
		result, err := runScript(wrappedScript, input)
		if err != nil {
			log.Fatalf("Error running script: %v", err)
		}
		_ = result
		// fmt.Printf("Result: %+v\n", result)
	}
}
