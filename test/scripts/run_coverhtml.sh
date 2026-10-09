#!/bin/bash

# Notice: tests need fonts in etxt/font/test/, see test/README.md
go test -tags gtxt ./... -coverprofile cover_prof.out > /dev/null
go tool cover -html=cover_prof.out
rm cover_prof.out
