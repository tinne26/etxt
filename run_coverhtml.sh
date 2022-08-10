#!/bin/bash

# Notice: requires a test_font.ttf file in etxt/
#         (any normal font will work)
go test -tags gtxt ./... -coverprofile cover_prof.out > /dev/null
go tool cover -html=cover_prof.out
rm cover_prof.out
