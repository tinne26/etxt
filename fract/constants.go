package fract

// Minimum and maximum constants.
const (
	MaxUnit Unit = +0x7FFFFFFF
	MinUnit Unit = -0x7FFFFFFF - 1
	One Unit = 64 // fract.One.ToInt() == 1
	MaxInt int = +33554431
	MinInt int = -33554432
	MaxFloat64 float64 = +33554431.984375
	MinFloat64 float64 = -33554432
	Delta float64 = 0.015625 // 1.0/64.0
	HalfDelta float64 = 0.0078125 // 1.0/128.0
)
