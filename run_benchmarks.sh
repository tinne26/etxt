#!/bin/bash

# Notice: requires a test_font.ttf file in etxt/
#         (any normal font will work)
go test -bench=. ./... -tags gtxt | grep "^[^?]"

# You may also use -benchmem
# go test -bench=. ./... -tags gtxt -benchmem | grep "^[^?]"
