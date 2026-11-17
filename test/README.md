# How to `go test` etxt

Most tests need fonts placed in `/font/test/` (`.ttf` or `.otf`). To cover all current tests, you will need at least a Latin and a Han + Latin font (e.g. Noto Sans SC). The package's test run will fail listing the skipped tests when suitable fonts can't be found.

The main testing command is the following:
```
go test -tags gtxt ./...
```
While you can test without the `gtxt` tag, most tests won't run because there's no easy way to test Ebitengine's graphical output. I made an effort to compare gtxt and Ebitengine results using `go generate`. That's explained in the next section. Outside that, Ebitengine-specific tests exist mainly to help detect build problems.

Many scripts are provided in `/test/scripts`. For example, you can run the tests with `run_tests.sh` (or `run_tests.bat` if you are on Windows), which also lists skipped tests with the reason. Here are the results of an example run:
```
$ ./test/scripts/run_tests.sh
[testing with gtxt...]
ok      github.com/tinne26/etxt         0.212s  coverage: 75.5% of statements
ok      github.com/tinne26/etxt/cache   0.004s  coverage: 83.8% of statements
ok      github.com/tinne26/etxt/font    0.016s  coverage: 81.8% of statements
ok      github.com/tinne26/etxt/fract   0.007s  coverage: 89.4% of statements
ok      github.com/tinne26/etxt/mask    0.017s  coverage: 78.9% of statements

[testing with Ebitengine...]
ok      github.com/tinne26/etxt         0.248s  coverage: 54.5% of statements
ok      github.com/tinne26/etxt/cache   0.041s  coverage: 83.8% of statements
ok      github.com/tinne26/etxt/font    0.017s  coverage: 81.8% of statements
ok      github.com/tinne26/etxt/fract   0.055s  coverage: 88.9% of statements
ok      github.com/tinne26/etxt/mask    0.065s  coverage: 78.9% of statements
```

Tests pick fonts in the same order on every run, sorted by name. You can set the `ETXT_TEST_SEED` environment variable to any integer in order to shuffle fonts in a reproducible way.

## Testing Ebitengine vs gtxt

As explained in the previous section, Ebitengine's graphical output is hard to test. The logic between the default etxt version and the `-tags gtxt` version is almost entirely shared, so testing only with `gtxt` is still a fairly decent guarantee that things will also work on Ebitengine. The main difference are blend modes and glyph compositing over a target surface. To help cover this gap, we can use `go generate` from the base `etxt` directory:
```
$ go generate
Generating 'testdata_blend_rand_ebiten_test.go'... OK
Generating 'testdata_blend_rand_ebiten_gtxt_test.go'... OK
Generating 'testdata_blend_rand_gtxt_test.go'... OK
```

This will generate a few additional test files that contain only raw render data. Running `go test .` or `go test -tags gtxt .` afterwards will include this data on existing conditional tests. These tests will compare the compositing results of etxt's different modes and report if the results vary in any meaningful way.

To be honest, this set of tests is fairly limited and simplistic at the moment, but it's still much better than having no cross comparison tests at all.

## Honest reliability assessment

High test coverage percentages don't really mean much. Some examples:
- The whole `go generate` stuff for Ebitengine doesn't even increase coverage.
- You often have to write many more tests than what's strictly required for coverage to be really confident that something works as intended. I have written many such tests, but many more are still missing.
- Examples go a long way in improving my confidence that something is working, even if this isn't reflected on tests coverage.
