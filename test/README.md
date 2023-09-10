# How to `go test` etxt

Testing etxt requires placing two different `.ttf` fonts in `/font/test/`. Some tests can be executed even without these fonts, but you will get one failure due to missing assets.

The main testing command is the following:
```
go test -tags gtxt ./...
```
While you can test without the `gtxt` tag, most tests won't run because there's no easy way to test Ebitengine graphical output. I made an effort to compare gtxt and Ebitengine results using `go generate`. That's explained in the next section. Outside that, Ebitengine-specific tests exist mainly to help detect build problems.

Many scripts are provided in `/test/scripts`. For example, you can run the tests with `run_tests.sh` (or `run_tests.bat` if you are on Windows). Here are the results of an example run:
```
$ ./test/scripts/run_tests.sh
[testing with gtxt...]
ok      github.com/tinne26/etxt 0.066s  coverage: 29.4% of statements
ok      github.com/tinne26/etxt/cache   0.052s  coverage: 86.0% of statements
ok      github.com/tinne26/etxt/font    0.059s  coverage: 88.1% of statements
ok      github.com/tinne26/etxt/fract   0.057s  coverage: 92.7% of statements
ok      github.com/tinne26/etxt/mask    0.076s  coverage: 84.9% of statements

[testing with Ebitengine...]
ok      github.com/tinne26/etxt 0.105s  coverage: 17.9% of statements
ok      github.com/tinne26/etxt/cache   0.141s  coverage: 86.0% of statements
ok      github.com/tinne26/etxt/font    0.072s  coverage: 88.1% of statements
ok      github.com/tinne26/etxt/fract   0.140s  coverage: 92.1% of statements
ok      github.com/tinne26/etxt/mask    0.158s  coverage: 84.9% of statements
```

Scripts also include generation of static documentation, coverage and some benchmarking of custom rasterizers.


## Testing Ebitengine vs gtxt

As explained in the previous section, Ebitengine's graphical output is hard to test. The logic between the default etxt version and the `-tags gtxt` version is almost entirely shared, so testing only with `gtxt` is still a fairly decent guarantee that things will also work on Ebitengine. The main difference are blend modes and glyph compositing over a target surface. To help out cover this gap, we can use `go generate` from the base `etxt` directory:
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
- Maturity of v0.0.9's API is still quite heterogeneous.

v0.0.9 is still on its infancy and it's likely to be much less stable than v0.0.8. At the same time, it also fixes a few big bugs and makes many, many small quality improvements over v0.0.8.
