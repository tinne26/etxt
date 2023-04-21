# How to `go test` etxt

Testing etxt requires placing two different `.ttf` fonts in `/font/test/`. Some tests can be run even without these fonts, but you will get a failure due to missing assets and some warnings related to it.

The main testing command is the following:
```
go test -tags gtxt ./...
```
While you can test without the `gtxt` tag, most tests won't run because there's no easy way to test Ebitengine graphical output. Ebitengine tests exist almost only to detect build problems.

Many scripts are provided in `/test/scripts`. For example, you can run the tests with `run_tests.sh` (or `run_tests.bat` if you are on Windows). Here are the results of an example run:
```
$ ./test/scripts/run_tests.sh
[testing with gtxt...]
ok      github.com/tinne26/etxt 0.384s  coverage: 81.5% of statements
ok      github.com/tinne26/etxt/ecache  1.362s  coverage: 90.0% of statements
ok      github.com/tinne26/etxt/efixed  0.300s  coverage: 81.8% of statements
ok      github.com/tinne26/etxt/emask   0.330s  coverage: 74.0% of statements

[testing with Ebitengine...]
ok      github.com/tinne26/etxt 0.389s  coverage: 33.9% of statements
ok      github.com/tinne26/etxt/efixed  0.299s  coverage: 81.8% of statements
ok      github.com/tinne26/etxt/emask   0.315s  coverage: 74.0% of statements
```

Scripts also include generation of static documentation, coverage and some benchmarking of custom rasterizers.
