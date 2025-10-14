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
	jsCode, err := os.ReadFile("./js/console-hello.js")
	if err != nil {
		log.Fatalf("read js code: %v", err)
	}

	quickJSwasmPath := "./quickjs.plugin.wasm"
	quickJSwasm, err := os.ReadFile(quickJSwasmPath)
	if err != nil {
		log.Fatalf("read quickjs wasm: %v", err)
	}

	rtc := wazero.NewRuntimeConfig().WithDebugInfoEnabled(true)

	ctx := context.Background()
	rt := wazero.NewRuntimeWithConfig(ctx, rtc)
	defer rt.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	quickJSModConfig := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin)

	quickJSMod, err := rt.CompileModule(ctx, quickJSwasm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error compiling quickjs wasm: %v\n", err)
		return
	}
	defer quickJSMod.Close(ctx)

	qjs, err := rt.InstantiateModule(ctx, quickJSMod, quickJSModConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error instantiating quickjs wasm: %v\n", err)
		return
	}
	defer qjs.Close(ctx)

	mem := qjs.Memory()
	realloc := qjs.ExportedFunction("cabi_realloc")
	compile := qjs.ExportedFunction("compile-src")
	invoke := qjs.ExportedFunction("invoke")

	if realloc == nil || compile == nil || invoke == nil {
		log.Fatalf("missing one or more required exports")
	}

	// --- helper funcs ---
	alloc := func(n uint32) (uint32, error) {
		res, err := realloc.Call(ctx, 0, 0, 1, uint64(n))
		if err != nil {
			return 0, err
		}
		return uint32(res[0]), nil
	}
	readU32 := func(p uint32) uint32 {
		b, ok := mem.Read(p, 4)
		if !ok {
			log.Fatalf("invalid read at %d", p)
		}
		return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	}
	readList := func(ptr uint32) []byte {
		dataPtr := readU32(ptr)
		dataLen := readU32(ptr + 4)
		if dataLen == 0 {
			return nil
		}
		data, ok := mem.Read(dataPtr, dataLen)
		if !ok {
			log.Fatalf("invalid read of list at %d", dataPtr)
		}
		return data
	}

	// --- compile-src(js) -> bytecode ---
	srcPtr, err := alloc(uint32(len(jsCode)))
	if err != nil {
		log.Fatalf("alloc src: %v", err)
	}
	if !mem.Write(srcPtr, jsCode) {
		log.Fatal("mem.Write(js) failed")
	}

	// compile-src returns a pointer to a 8-byte list descriptor (data_ptr,len)
	res, err := compile.Call(ctx, uint64(srcPtr), uint64(len(jsCode)))
	if err != nil {
		log.Fatalf("compile-src: %v", err)
	}
	retCompile := uint32(res[0])
	bytecode := readList(retCompile)

	fmt.Printf("compiled bytecode: %d bytes\n", len(bytecode))

	// --- invoke(bytecode) ---
	bcPtr, err := alloc(uint32(len(bytecode)))
	if err != nil {
		log.Fatalf("alloc bytecode: %v", err)
	}
	if !mem.Write(bcPtr, bytecode) {
		log.Fatal("mem.Write(bytecode) failed")
	}

	retInvoke, err := alloc(8)
	if err != nil {
		log.Fatalf("alloc retInvoke: %v", err)
	}

	// invoke(ret_ptr, bc_ptr, bc_len, arg_ptr, arg_len)
	if _, err := invoke.Call(ctx,
		uint64(retInvoke),
		uint64(bcPtr),
		uint64(len(bytecode)),
		0, 0,
	); err != nil {
		log.Fatalf("invoke: %v", err)
	}

	if out := readList(retInvoke); len(out) > 0 {
		fmt.Printf("invoke output: %s\n", string(out))
	}
}
