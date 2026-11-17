@echo off

:: Notice: tests need fonts in etxt/font/test/, see test/README.md
echo [testing with gtxt...]
go test -tags gtxt -count "1" -cover -v ./... | findstr /V /R /C:"=== " /C:"--- PASS" /C:"^PASS$" /C:"^coverage: " /C:"test fonts: " /C:"^?"

:: ebitengine pass is barely relevant at the moment,
:: but it helps catch build tag mixups
echo.
echo [testing with Ebitengine...]
go test -count "1" -cover -v ./... | findstr /V /R /C:"=== " /C:"--- PASS" /C:"^PASS$" /C:"^coverage: " /C:"test fonts: " /C:"^?"
