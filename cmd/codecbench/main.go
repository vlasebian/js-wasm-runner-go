package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"gopkg.in/yaml.v3"
)

type decodeUplinkInput struct {
	Bytes    []uint8 `json:"bytes"`
	FPort    uint8   `json:"fPort"`
	RecvTime int64   `json:"recvTime"` // UnixNano
}

type JavaScriptExecutor interface {
	Execute(jsCode string, input decodeUplinkInput) error
}

const (
	CodecBaseDir string = "./lorawan-devices/vendor"
	Vendor       string = "tektelic"
	Model        string = "t000589x"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <wasm|goja>", os.Args[0])
	}

	mode := os.Args[1]
	if mode != "wasm" && mode != "goja" {
		log.Fatalf("Invalid mode: %s. Must be 'wasm' or 'goja'", mode)
	}

	_, jsCode, input, err := loadDeviceCodec(CodecBaseDir, Vendor, Model)
	if err != nil {
		log.Fatalf("loadDeviceCodec: %v", err)
	}

	jsCodeStr := string(jsCode)
	executor := NewJavaScriptExecutor(mode)
	if err := executor.Execute(jsCodeStr, input); err != nil {
		log.Fatalf("Execution error: %v", err)
	}
}

func NewJavaScriptExecutor(mode string) JavaScriptExecutor {
	if mode == "wasm" {
		return &wasmExecutor{}
	}
	return &gojaExecutor{}
}

type wasmExecutor struct{}

func (e *wasmExecutor) Execute(jsCode string, input decodeUplinkInput) error {
	rtc := wazero.NewRuntimeConfig().
		WithDebugInfoEnabled(true)

	ctx := context.Background()
	rt := wazero.NewRuntimeWithConfig(ctx, rtc)
	defer rt.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	wasmPath := "wasm/qjs.wasm"
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("read qjs.wasm: %w", err)
	}

	qjsEvalMod, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		return fmt.Errorf("error compiling quickjs wasm: %w", err)
	}
	defer qjsEvalMod.Close(ctx)

	wrappedScript := wrapUplinkDecoderScriptForWASM(jsCode, jsObjectFromDecodeUplinkInput(input))

	qjsEvalModConfig := wazero.NewModuleConfig().
		WithName("qjs").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin).
		WithArgs([]string{"qjs", "--std", "-C", "-e", string(wrappedScript)}...)

	qjsEval, err := rt.InstantiateModule(ctx, qjsEvalMod, qjsEvalModConfig)
	if err != nil {
		return fmt.Errorf("error instantiating quickjs wasm: %w", err)
	}

	if err := qjsEval.Close(ctx); err != nil {
		return fmt.Errorf("error closing quickjs wasm: %w", err)
	}

	return nil
}

func wrapUplinkDecoderScriptForWASM(script string, input string) string {
	// This wrapper executes decodeUplink() if it is defined. Then, it executes normalizeUplink() if it is defined too,
	// and if the output of decodeUplink() didn't return errors.
	// Fallback to Decoder() for backwards compatibility with The Things Network Stack V2 payload functions.
	return fmt.Sprintf(`
%s

function main() {
	%s

	const { bytes, fPort, recvTime } = input;

	// Convert UnixNano to JavaScript Date.
	const jsDate = new Date(Number(BigInt(recvTime) / 1000000n));

	if (typeof decodeUplink === 'function') {
		const decoded = decodeUplink({ bytes, fPort, recvTime: jsDate });
		let normalized;
		const { data, errors } = decoded;
		if ((!errors || !errors.length) && data && typeof normalizeUplink === 'function') {
			normalized = normalizeUplink({ data });
		}
		// console.log('Decoded:', JSON.stringify(decoded));
		return { decoded, normalized };
	}
	return {
		decoded: {
			data: Decoder(bytes, fPort)
		}
	}
}

// In WASM you must call main explicitly, you cannot execute a specific function directly.
main();

	`, script, input)
}

func jsObjectFromDecodeUplinkInput(input decodeUplinkInput) string {
	var bytesStr string
	if len(input.Bytes) == 0 {
		bytesStr = "[]"
	} else {
		byteVals := make([]string, len(input.Bytes))
		for i, b := range input.Bytes {
			byteVals[i] = fmt.Sprintf("%d", b)
		}
		bytesStr = "[" + strings.Join(byteVals, ", ") + "]"
	}
	js := fmt.Sprintf(
		"let input = {  bytes: %s,  fPort: %d,  recvTime: %d};\n",
		bytesStr, input.FPort, input.RecvTime,
	)
	return js
}

type gojaExecutor struct{}

func (e *gojaExecutor) Execute(jsCode string, input decodeUplinkInput) error {
	wrappedScript := wrapUplinkDecoderScriptForGoja(jsCode)

	// Run the script
	result, err := runScript(wrappedScript, input)
	if err != nil {
		return fmt.Errorf("Error running script: %v", err)
	}

	// Print the result
	fmt.Printf("Result: %+v\n", result)
	return nil
}

func wrapUplinkDecoderScriptForGoja(script string) string {
	// This wrapper executes decodeUplink() if it is defined. Then, it executes normalizeUplink() if it is defined too,
	// and if the output of decodeUplink() didn't return errors.
	// Fallback to Decoder() for backwards compatibility with The Things Network Stack V2 payload functions.
	return fmt.Sprintf(`
		%s

		function main(input) {
			const bytes = input.bytes.slice();
			const { fPort, recvTime } = input;

			// Convert UnixNano to JavaScript Date.
			const jsDate = new Date(Number(BigInt(recvTime) / 1000000n));

			if (typeof decodeUplink === 'function') {
				const decoded = decodeUplink({ bytes, fPort, recvTime: jsDate });
				let normalized;
				const { data, errors } = decoded;
				if ((!errors || !errors.length) && data && typeof normalizeUplink === 'function') {
					normalized = normalizeUplink({ data });
				}
				return { decoded, normalized };
			}
			return {
				decoded: {
					data: Decoder(bytes, fPort)
				}
			}
		}
	`, script)
}

func runScript(script string, input decodeUplinkInput) (map[string]interface{}, error) {
	// Create a new Goja VM
	vm := goja.New()

	// Set field name mapper to use JSON tags
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// Run the script to load the function
	_, err := vm.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("failed to load script: %w", err)
	}

	// Get the main function
	mainFunc, ok := goja.AssertFunction(vm.Get("main"))
	if !ok {
		return nil, fmt.Errorf("main function not found")
	}

	// Convert input to Goja value
	inputValue := vm.ToValue(input)

	// Call the main function
	result, err := mainFunc(goja.Undefined(), inputValue)
	if err != nil {
		return nil, fmt.Errorf("failed to execute main: %w", err)
	}

	// Export result to Go map
	var output map[string]interface{}
	err = vm.ExportTo(result, &output)
	if err != nil {
		return nil, fmt.Errorf("failed to export result: %w", err)
	}

	return output, nil
}

// LoadDeviceCodec loads the JS codec and example input for the given device. The device should be passed as
// "brand/model". the function loads the file at lorawan-devices/vendor/brand/model-codec.yaml.
func loadDeviceCodec(baseDir, vendor, model string) (string, []byte, decodeUplinkInput, error) {
	yamlPath := filepath.Join(baseDir, vendor, fmt.Sprintf("%s-codec.yaml", model))

	// Read YAML codec spec
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		return "", nil, decodeUplinkInput{}, fmt.Errorf("read yaml codec (%s): %w", yamlPath, err)
	}

	codecSpec, err := parseCodecYAML(yamlBytes)
	if err != nil {
		return "", nil, decodeUplinkInput{}, fmt.Errorf("parse yaml codec: %w", err)
	}

	if len(codecSpec.UplinkDecoder.Examples) == 0 {
		return "", nil, decodeUplinkInput{}, errors.New("no examples found (uplinkDecoder.examples is empty)")
	}

	// Read JS codec file
	jsPath := filepath.Join(baseDir, vendor, codecSpec.UplinkDecoder.Filename)
	jsCode, err := os.ReadFile(jsPath)
	if err != nil {
		return "", nil, decodeUplinkInput{}, fmt.Errorf("read js codec (%s): %w", jsPath, err)
	}

	first := codecSpec.UplinkDecoder.Examples[0].Input

	// Convert bytes (int slice) -> []uint8
	converted := make([]uint8, len(first.Bytes))
	for i, v := range first.Bytes {
		if v < 0 || v > 255 {
			return "", nil, decodeUplinkInput{}, fmt.Errorf("byte value out of range at index %d: %d", i, v)
		}
		converted[i] = uint8(v)
	}

	recvTime := time.Now().UnixNano()

	input := decodeUplinkInput{
		Bytes:    converted,
		FPort:    uint8(first.FPort),
		RecvTime: recvTime,
	}

	return jsPath, jsCode, input, nil
}

type codecYAML struct {
	UplinkDecoder struct {
		Filename string `yaml:"fileName"`
		Examples []struct {
			Input struct {
				FPort int   `yaml:"fPort"`
				Bytes []int `yaml:"bytes"`
			} `yaml:"input"`
		} `yaml:"examples"`
	} `yaml:"uplinkDecoder"`
}

func parseCodecYAML(b []byte) (codecYAML, error) {
	var spec codecYAML
	if err := yaml.Unmarshal(b, &spec); err != nil {
		return codecYAML{}, err
	}
	return spec, nil
}
