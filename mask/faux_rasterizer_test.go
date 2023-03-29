package mask

import "time"
import "math/rand"
import "testing"

func TestFloat32UnitRangeStability(t *testing.T) {
	if unitFP32FromUint16(0) != -1 { t.Fatal("expected minus one") }
	if unitFP32FromUint16(65535) != 1 { t.Fatal("expected one") }

	i := uint16(0)
	for {
		fp32 := unitFP32FromUint16(i)
		if fp32 ==  0 { t.Fatalf("unexpected zero on i = %d", i) }
		u16  := uint16FromUnitFP32(fp32)
		if u16 != i {
			t.Fatalf("i = %d, fp32 = %f, u16 => %d", i, fp32, u16)
		}

		if i == 65535 { break }
		i += 1 // exhaustive testing
	}
}

func TestFloat32UnitRangeStabilityRng(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 4096; i++ {
		fp32 := float32(rng.Float64()*2 - 1.0)
		if fp32 == 0 { continue }
		if fp32 < -1 || fp32 > 1 { panic("incorrect test code") }

		u16  := uint16FromUnitFP32(fp32)
		re32 := unitFP32FromUint16(u16)
		if re32 == 0 { t.Fatalf("got zero from %f", fp32) }
		re16 := uint16FromUnitFP32(re32)
		if re16 != u16 {
			t.Fatalf("unstability with fp32 = %f, u16 => %d, re32 = %f, re16 = %d", fp32, u16, re32, re16)
		}
	}
}
