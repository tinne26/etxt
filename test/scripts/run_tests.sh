#!/bin/bash

# Notice: tests need fonts in etxt/font/test/, see test/README.md
filter="=== |--- PASS|^PASS$|^coverage: |test fonts: |^\?"

echo "[testing with gtxt...]"
go test -tags gtxt -count "1" -cover -v ./... | grep -v -E "$filter"

# ebitengine pass is barely relevant at the moment,
# but it helps catch build tag mixups
echo ""
echo "[testing with Ebitengine...]"
go test -count "1" -cover -v ./... | grep -v -E "$filter"
