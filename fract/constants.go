package fract

// Miscellaneous constants related to [Unit].
const (
	MaxUnit Unit = +0x7FFFFFFF
	MinUnit Unit = -0x7FFFFFFF - 1
	One Unit = 64 // fract.One.ToInt() == 1
	MaxInt int = +33554431 // max representable int
	MinInt int = -33554432 // min representable int
	MaxFloat64 float64 = +33554431.984375 // max representable float
	MinFloat64 float64 = -33554432        // min representable float
	Delta float64 = 0.015625 // float equivalent of Unit(1) => 1.0/64.0
	HalfDelta float64 = 0.0078125 // 1.0/128.0 (used for rounding)
)
