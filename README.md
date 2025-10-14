# js-wasm-runner-go

Download `javy`:

```bash
wget https://github.com/bytecodealliance/javy/releases/download/v7.0.0/javy-arm-macos-v7.0.0.gz
```

Check hashsum:
```bash
wget https://github.com/bytecodealliance/javy/releases/download/v7.0.0/javy-arm-macos-v7.0.0.gz.sha256
echo "$(cat javy-arm-macos-v7.0.0.gz.sha256)" | shasum -a 256 javy-arm-macos-v7.0.0.gz | shasum -c
```

Install `javy`:

```bash
gzip -dk javy-arm-macos-v7.0.0.gz
chmod +x javy-arm-macos-v7.0.0
```

Emit QuickJS engine compiled to wasm:

```bash
./javy-arm-macos-v7.0.0 emit-plugin -o quickjs.plugin.wasm
```

Compile:
```bash
# Static, with QuickJS embedded in the final wasm.
./javy-arm-macos-v7.0.0 build ./js/hello.js -o static-hello.wasm 
# Dynamic, without QuickJS embedded in the final wasm.
./javy-arm-macos-v7.0.0 build -C dynamic -C plugin=quickjs.plugin.wasm -o dyn-hello.wasm ./js/hello.js 
```

