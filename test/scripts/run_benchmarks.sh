#!/bin/bash

# Notice: requires one .ttf font file in etxt/test/fonts/
#         (any normal font will do)
echo "[benchmarking with gtxt...]"
go test -bench "." -tags "gtxt bench" ./... | grep "^[^?]"

# ebitengine pass is the same at the moment so it's disabled
# echo ""
# echo "[Ebitengine pass...]"
# go test -bench "." -tags "bench" ./... | grep "^[^?]"

# You may also use -benchmem
# go test -bench "." -benchmem -tags "gtxt bench" ./... | grep "^[^?]"
