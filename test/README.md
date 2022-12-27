# How to `go test` etxt

Testing etxt requires placing two different `.ttf` fonts in `/test/fonts/`. Some tests can be run even without these fonts, but you will get a failure due to missing assets and some warnings related to it.

The most basic form requires both `gtxt` and `test` tags:
```
go test -tags "gtxt test" ./...
```

Many scripts are provided in `test/scripts`. For example, you can run the tests with `run_tests.sh`. Here are the results of an example run[^1]:
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

Scripts also include generation of static documentation, coverage and some benchmarking of custom rasterizers. For Window users, `.bat` versions are provided in some cases.

[^1]: The Ebitengine test results are almost irrelevant and only exist to detect build problems. It's not easy to test with Ebitengine images (you need to spawn separate processes that use `RunGame()` directly from the tests), so the only tests run are those that don't require any rasterization targets, which are already included in the `gtxt` tests.
