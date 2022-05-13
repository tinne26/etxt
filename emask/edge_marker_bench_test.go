//go:build gtxt

package emask

import "os"
import "log"
import "math/rand"
import "testing"

import "golang.org/x/image/math/fixed"

const rastBenchSeed = int64(0) // use 0 for PID-based seed
func makeRng() *rand.Rand {
	seed := rastBenchSeed
	if seed == 0 { seed = int64(os.Getpid()) }
	return rand.New(rand.NewSource(seed))
}

func BenchmarkStdRast(b *testing.B) {
	rng := makeRng()
	rast := &DefaultRasterizer{}
	for n := 0; n < b.N; n++ {
		for size := 16; size <= 512; size *= 2 {
			shape := randomShape(rng, 16, size, size)
			segments := shape.Segments()
			_, err := Rasterize(segments, rast, fixed.Point26_6{})
			if err != nil { log.Fatalf("rasterization error: %s", err.Error()) }
		}
	}
}

func BenchmarkEdgeRast(b *testing.B) {
	rng := makeRng()
	rast := NewStdEdgeMarkerRasterizer()
	for n := 0; n < b.N; n++ {
		for size := 16; size <= 512; size *= 2 {
			shape := randomShape(rng, 16, size, size)
			segments := shape.Segments()
			_, err := Rasterize(segments, rast, fixed.Point26_6{})
			if err != nil { log.Fatalf("rasterization error: %s", err.Error()) }
		}
	}
}
