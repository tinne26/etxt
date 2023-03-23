package fract

// Fast conversion from int to [Unit]. If the int value is not
// representable with a [Unit], the result is undefined. If you
// want to account for overflows, check [MinInt] <= value <= [MaxInt].
func FromInt(value int) Unit { return Unit(value << 6) }

// Converts a float64 to the closest Unit, rounding up in case
// of ties. Doesn't account for NaNs, infinites nor overflows.
func FromFloat64Up(value float64) Unit {
	unitApprox := Unit(value*64)
	fp64Approx := unitApprox.ToFloat64()
	if fp64Approx == value { return unitApprox }
	if fp64Approx > value {
		unitApprox -= 1
		fp64Approx = unitApprox.ToFloat64()
	}

	if value - fp64Approx >= HalfDelta { unitApprox += 1 }
	return unitApprox
}

// Converts a float64 to the closest Unit, rounding down in case
// of ties. Doesn't account for NaNs, infinites nor overflows.
func FromFloat64Down(value float64) Unit {
	unitApprox := Unit(value*64)
	fp64Approx := unitApprox.ToFloat64()
	if fp64Approx == value { return unitApprox }
	if fp64Approx > value {
		unitApprox -= 1
		fp64Approx = unitApprox.ToFloat64()
	}

	if value - fp64Approx > HalfDelta { unitApprox += 1 }
	return unitApprox
}

