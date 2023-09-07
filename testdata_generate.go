package etxt

// Ebitengine is hard to test with "go test" because you need to create
// standalone programs with a main function and so on. There are multiple
// ways to work around that, and the truth is that for testing etxt there's
// already the gtxt CPU version that allows achieving high test coverage
// with a fairly decent degree of reliability. Still, in order to make sure
// that the gtxt version (CPU rendering) and the default Ebitengine version
// (GPU rendering [notice that rasterization still happens on CPU]) results
// are matching, we can use go:generate to run standalone Ebitengine programs,
// get some raw image results, plug that into basic tests and print it all
// as static test files.

// Fixed seed fuzzy test of compositing with gtxt vs Ebitengine.
//go:generate go run -tags "GENERATE_ETXT_TESTDATA" test/generate/blend_rand/ebiten.go
//go:generate go run -tags "GENERATE_ETXT_TESTDATA gtxt" test/generate/blend_rand/ebiten_gtxt.go
//go:generate go run -tags "GENERATE_ETXT_TESTDATA gtxt" test/generate/blend_rand/gtxt.go
