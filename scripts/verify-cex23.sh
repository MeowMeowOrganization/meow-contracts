#!/usr/bin/env sh
set -eu

# Run from the repository root after installing Buf and a suitable Go toolchain.
buf lint
buf build
buf generate

go test ./...
go vet ./...
