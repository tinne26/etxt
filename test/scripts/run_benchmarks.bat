@echo off

:: Notice: benchmarks need fonts in etxt/font/test/, see test/README.md
echo [benchmarking with gtxt...]
go test -run "^$" -bench "." -tags "gtxt" ./... | findstr /R "^[^?]"

:: ebitengine pass is the same at the moment so it's disabled
:: echo.
:: echo [benchmarking with Ebitengine...]
:: go test -run "^$" -bench "." ./... | findstr /R "^[^?]"

:: You may also use -benchmem
:: go test -run "^$" -bench "." -benchmem -tags "gtxt" ./... | findstr /R "^[^?]"
