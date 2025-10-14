#!/bin/bash

set -euo pipefail

BIN_DIR=$(pwd)/bin
PKG_DIR=$(pwd)/pkg
WASM_DIR=$(pwd)/wasm

# Versions and URLs
WAZERO_BIN=wazero
WAZERO_URL=https://wazero.io/install.sh

JAVY_VERSION=7.0.1
JAVY_ARCH=arm
JAVY_OS=macos
JAVY_BIN=javy
JAVY_URL="https://github.com/bytecodealliance/javy/releases/download/v${JAVY_VERSION}/javy-${JAVY_ARCH}-${JAVY_OS}-v${JAVY_VERSION}.gz"

WASI_SDK_VERSION=27
WASI_SDK_ARCH=arm64
WASI_SDK_OS=macos
WASI_SDK_URL="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-${WASI_SDK_VERSION}/wasi-sdk-${WASI_SDK_VERSION}.0-${WASI_SDK_ARCH}-${WASI_SDK_OS}.tar.gz"

ensure_dirs() {
	mkdir -p "$BIN_DIR" "$PKG_DIR" "$WASM_DIR"
}

install_javy() {
	ensure_dirs
	if [ ! -f "$BIN_DIR/$JAVY_BIN" ]; then
		curl -sSL "$JAVY_URL" -o "$BIN_DIR/$JAVY_BIN.gz"
		gunzip -f "$BIN_DIR/$JAVY_BIN.gz"
		chmod +x "$BIN_DIR/$JAVY_BIN"
		"$BIN_DIR/$JAVY_BIN" --version
	else
		echo "javy already installed in $BIN_DIR"
	fi
}

install_wazero() {
	ensure_dirs
	if [ ! -f "$BIN_DIR/$WAZERO_BIN" ]; then
		# Install wazero CLI into BIN_DIR if installer supports BINDIR, else fallback
		# The official installer respects BINDIR env var
		BINDIR="$BIN_DIR" curl -fsSL "$WAZERO_URL" | sh
		"$BIN_DIR/$WAZERO_BIN" version
	else
		echo "wazero already installed in $BIN_DIR"
	fi
}

install_wasi_sdk() {
	ensure_dirs
	if [ ! -d "$PKG_DIR/wasi-sdk" ]; then
		mkdir -p "$PKG_DIR/wasi-sdk"
		curl -sSL "$WASI_SDK_URL" -o "$PKG_DIR/wasi-sdk.tar.gz"
		tar -xzf "$PKG_DIR/wasi-sdk.tar.gz" -C "$PKG_DIR/wasi-sdk" --strip-components=1
		rm -f "$PKG_DIR/wasi-sdk.tar.gz"
	else
		echo "wasi-sdk already installed in $PKG_DIR/wasi-sdk"
	fi
}

build_quickjs_ng() {
	ensure_dirs
	if ! git -C "$PKG_DIR/quickjs-ng" rev-parse 2>/dev/null; then
		git clone https://github.com/quickjs-ng/quickjs.git "$PKG_DIR/quickjs-ng"
	else
		git -C "$PKG_DIR/quickjs-ng" pull --ff-only || true
	fi

	cmake -S "$PKG_DIR/quickjs-ng" -B "$PKG_DIR/quickjs-ng/build" -DCMAKE_TOOLCHAIN_FILE="$PKG_DIR/wasi-sdk/share/cmake/wasi-sdk-p1.cmake"
	make -C "$PKG_DIR/quickjs-ng/build" qjsc
	make -C "$PKG_DIR/quickjs-ng/build" qjs_exe
	cp "$PKG_DIR/quickjs-ng/build/qjsc" "$WASM_DIR/qjsc.wasm"
	cp "$PKG_DIR/quickjs-ng/build/qjs" "$WASM_DIR/qjs.wasm"
}

clean_deps() {
	rm -rf "$BIN_DIR"/* "$PKG_DIR"/* "$WASM_DIR"/*
}

install_all() {
	install_javy
	install_wazero
	install_wasi_sdk
	build_quickjs_ng
}

print_usage() {
	cat <<EOF
Usage: $(basename "$0") [command]

Commands:
  all             Install all components
  javy            Install javy
  wazero          Install wazero CLI
  wasi-sdk        Install wasi-sdk toolchain
  quickjs-ng      Build quickjs-ng and copy wasm artifacts
  clean           Remove installed binaries, packages, and wasm outputs
  help            Show this help message

If no command is provided, 'all' is executed.
EOF
}

main() {
	cmd="${1:-all}"
	case "$cmd" in
		all)
			install_all
			;;
		javy)
			install_javy
			;;
		wazero)
			install_wazero
			;;
		wasi-sdk)
			install_wasi_sdk
			;;
		quickjs-ng)
			build_quickjs_ng
			;;
		clean)
			clean_deps
			;;
		help|-h|--help)
			print_usage
			;;
		*)
			echo "Unknown command: $cmd" >&2
			print_usage >&2
			exit 1
			;;
	 esac
}

main "$@"
