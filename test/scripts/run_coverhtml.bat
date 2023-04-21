@echo off

:: Notice: requires one .ttf font file in etxt/font/test/
::         (any normal font will do)
go test -tags gtxt ./... -coverprofile cover_prof.out >NUL
go tool cover -html=cover_prof.out
del cover_prof.out
