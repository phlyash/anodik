package pll

type Config struct {
	DivA   int // out_div_a register field; effective divider = DivA+1
	DivB   int // out_div_b register field; effective divider = DivB+1
	PreDiv int // prediv register field: 1, 2, or 4 (used as-is, not +1)
	RDiv   int // ref_div register field; effective divider = RDiv+1
	NDiv   int // integer part of the feedback multiplier
	Mod    int // fractional modulus (frac_mod)
	Frac   int // fractional numerator (frac_num)

	// K1921VG015 only: second output divider pair, same encoding as DivA/DivB.
	// Zero value on every other chip.
	Div2A int
	Div2B int

	DoFrac       bool
	DoBypass     bool
	IsCalculated bool

	chip chipKind
}

// N is the effective feedback multiplier: NDiv, or NDiv + Frac/Mod when a fractional solution was chosen.
func (c *Config) N() float64 {
	if c.DoFrac {
		return float64(c.NDiv) + float64(c.Frac)/float64(c.Mod)
	}
	return float64(c.NDiv)
}

// FVCO is the PLL's internal VCO frequency for the given reference frequency (kHz).
// VG015 uses a different formula (no PreDiv term, plain RDiv rather than RDiv+1) — see (*Config).isVG015.
func (c *Config) FVCO(fRef float64) float64 {
	if c.chip == chipVG015 {
		return (fRef * c.N()) / float64(c.RDiv)
	}
	return (fRef * c.N() * float64(c.PreDiv)) / float64(c.RDiv+1)
}

// FOut is the primary output frequency for the given reference frequency (kHz).
func (c *Config) FOut(fRef float64) float64 {
	return c.FVCO(fRef) / float64((c.DivA+1)*(c.DivB+1))
}

// FOut2 is the secondary output frequency (K1921VG015 only).
// Calling this on a single-output chip is a caller error — Div2A/Div2B are zero there and the result is meaningless.
func (c *Config) FOut2(fRef float64) float64 {
	return c.FVCO(fRef) / float64((c.Div2A+1)*(c.Div2B+1))
}

// FPFD is the phase-frequency-detector frequency for the given reference frequency (kHz).
func (c *Config) FPFD(fRef float64) float64 {
	if c.chip == chipVG015 {
		return fRef / float64(c.RDiv)
	}
	return fRef / float64(c.RDiv+1)
}

// calcRequiredN is the feedback multiplier needed to hit fOut exactly, given the divider fields currently set on c.
func (c *Config) calcRequiredN(fRef, fOut float64) float64 {
	ab := float64((c.DivA + 1) * (c.DivB + 1))
	if c.chip == chipVG015 {
		return (fOut * float64(c.RDiv) * ab) / fRef
	}
	r1 := float64(c.RDiv + 1)
	return (fOut * r1 * ab) / fRef
}

func (c *Config) snapshot() Config {
	return *c
}

func (c *Config) HasSecondOutput() bool {
	return c.chip == chipVG015
}

type chipKind int

const (
	chipVG1T chipKind = iota
	chipVG3T
	chipVG5T
	chipVG7T
	chipVG015
)

type limits struct {
	aRange func() []int
	bRange func() []int
	rRange func() []int
	pRange func() []int

	isNInRange    func(doFrac bool, n float64) bool
	isFRefInRange func(fRef float64) bool
	isFVCOInRange func(fVCO float64) bool
	isFOutInRange func(fOut float64) bool
	isFPFDInRange func(c *Config, fRef float64) bool
}

func intRange(lo, hi int) []int {
	out := make([]int, 0, hi-lo)
	for i := lo; i < hi; i++ {
		out = append(out, i)
	}
	return out
}

func defaultLimits() limits {
	return limits{
		aRange: func() []int { return intRange(0, 8) },
		bRange: func() []int { return intRange(0, 64) },
		// 1..63 inclusive: kept within LL_ASSERT(ref_div in [1,64]) in
		// ll_pll_configure(), same comment as calc.py's r_range().
		rRange: func() []int { return intRange(1, 64) },
		pRange: func() []int { return []int{1, 2, 4} },

		isNInRange: func(doFrac bool, n float64) bool {
			if doFrac {
				return n >= 20 && n <= 160
			}
			return n >= 1 && n <= 160
		},
		isFRefInRange: func(fRef float64) bool { return fRef >= 10e3 && fRef <= 100e3 },
		isFVCOInRange: func(fVCO float64) bool { return fVCO >= 600e3 && fVCO <= 1200e3 },
		isFOutInRange: func(fOut float64) bool { return fOut >= 200 && fOut <= 204e3 },
		isFPFDInRange: func(c *Config, fRef float64) bool {
			fpfd := c.FPFD(fRef)
			if c.DoFrac {
				return fpfd >= 8e3 && fpfd <= c.FVCO(fRef)/20
			}
			return fpfd >= 8e3 && fpfd <= c.FVCO(fRef)/16
		},
	}
}

func chipLimits(chip chipKind) limits {
	l := defaultLimits()

	switch chip {
	case chipVG1T:
		// no overrides

	case chipVG3T:
		l.isFOutInRange = func(fOut float64) bool { return fOut >= 200 && fOut <= 120e3 }

	case chipVG5T, chipVG7T:
		l.isFOutInRange = func(fOut float64) bool { return fOut >= 200 && fOut <= 100e3 }

	case chipVG015:
		l.isFOutInRange = func(fOut float64) bool { return fOut >= 390 && fOut <= 60e3 }
		l.isFRefInRange = func(fRef float64) bool { return fRef >= 10e3 && fRef <= 30e3 }
		l.isFVCOInRange = func(fVCO float64) bool { return fVCO >= 200e3 && fVCO <= 1600e3 }
		l.rRange = func() []int { return intRange(1, 65) }
		l.pRange = func() []int { return []int{1} }
	}

	return l
}

var chipRegistry = map[string]chipKind{
	"k1921vg1t":  chipVG1T,
	"k1921vg3t":  chipVG3T,
	"k1921vg5t":  chipVG5T,
	"k1921vg7t":  chipVG7T,
	"k1921vg015": chipVG015,
}

var dualOutputChips = map[chipKind]bool{
	chipVG015: true,
}
