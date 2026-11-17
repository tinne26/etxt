#!/bin/bash

# Notice: benchmarks need fonts in etxt/font/test/, see test/README.md
echo "[benchmarking with gtxt...]"
go test -run "^$" -bench "." -tags "gtxt" ./... | grep "^[^?]"

# ebitengine pass is the same at the moment so it's disabled
# echo ""
# echo "[Ebitengine pass...]"
# go test -run "^$" -bench "." ./... | grep "^[^?]"

# You may also use -benchmem
# go test -run "^$" -bench "." -benchmem -tags "gtxt" ./... | grep "^[^?]"
