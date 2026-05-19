#!/bin/bash

# Notice: requires one .ttf font file in etxt/test/fonts/
#         (any normal font will do)
go test -tags "gtxt test" ./... -coverprofile cover_prof.out > /dev/null
go tool cover -html=cover_prof.out
rm cover_prof.out
