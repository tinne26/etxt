@echo off

:: Notice: requires one .ttf font file in etxt/test/fonts/
::         (any normal font will do)
echo [benchmarking with gtxt...]
go test -bench "." -tags "gtxt bench" ./... | findstr /R "^[^?]"

:: ebitengine pass is the same at the moment so it's disabled
:: echo.
:: echo [benchmarking with Ebitengine...]
:: go test -bench "." -tags "bench" ./... | findstr /R "^[^?]"

:: You may also use -benchmem
:: go test -bench "." -benchmem -tags "gtxt bench" ./... | findstr /R "^[^?]"
