#!/bin/bash

# Notice: testing requires two .ttf font files in etxt/test/fonts/
#         (any normal fonts will do)
echo "[testing with gtxt...]"
go test -tags "gtxt test" -count "1" -cover ./... | grep "^[^?]"

# ebitengine pass is barely relevant at the moment,
# but it helps catch build tag mixups
echo ""
echo "[testing with Ebitengine...]"
go test -tags "test" -count "1" -cover ./... | grep "^[^?]"
