@echo off

:: Notice: tests need fonts in etxt/font/test/, see test/README.md
go test -tags gtxt ./... -coverprofile cover_prof.out >NUL
go tool cover -html=cover_prof.out
del cover_prof.out
