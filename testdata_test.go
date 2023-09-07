package etxt

import "testing"

// Tests using pre-generated test data for comparison
// of Ebitengine and gtxt rendering results. Take a look
// at testdata_generate.go for more details.

var testdata = make(map[string][]byte)

func TestTestdataBlendRand(t *testing.T) {
	// get test data
	valuesE, foundE := testdata["blend_rand_ebiten"]
	valuesG, foundG := testdata["blend_rand_gtxt"]
	valuesB, foundB := testdata["blend_rand_ebiten_gtxt"]
	if foundE != foundG || foundE != foundB {
		panic("incorrect test data generation or setup")
	}
	if !foundE { t.SkipNow() }

	// compare values
	if !similarByteSlices(valuesE, valuesG) {
		t.Fatalf("Mismatched testdata blend_rand results:\nEbitengine results: %v\ngtxt results: %v", valuesE, valuesG)
	}
	if !similarByteSlices(valuesE, valuesB) {
		t.Fatalf("Mismatched testdata blend_rand results:\nEbitengine results: %v\nEbitengine + gtxt results: %v", valuesE, valuesB)
	}
}

func similarByteSlices(a, b []byte) bool {
	if len(a) != len(b) { return false }
	for i := 0; i < len(a); i++ {
		if a[i] == b[i] { continue }
		if a[i] < b[i] && a[i] + 1 == b[i] { continue }
		if b[i] < a[i] && b[i] + 1 == a[i] { continue }
		return false
	}
	return true
}
