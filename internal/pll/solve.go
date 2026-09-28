package pll

import (
	"fmt"
	"math"
	"sort"
)

type SolveOptions struct {
	SoC     string
	FRef    float64
	FOut1   float64
	FOut2   *float64
	Epsilon float64
}

func Solve(opts SolveOptions) ([]Config, error) {
	epsilon := opts.Epsilon
	if epsilon == 0 {
		epsilon = 1e-3
	}

	chip, err := resolveChip(opts.SoC)
	if err != nil {
		return nil, err
	}

	hasDual := dualOutputChips[chip]

	if opts.FOut2 != nil && !hasDual {
		return nil, fmt.Errorf("chip %q does not support a second output; FOut2 must be nil", opts.SoC)
	}

	var raw []Config
	if opts.FOut2 != nil && hasDual {
		raw, err = solveBothConfigs(chip, opts.FRef, opts.FOut1, *opts.FOut2, epsilon)
	} else {
		raw, err = solveConfig(chip, opts.FRef, opts.FOut1, epsilon)
	}
	if err != nil {
		return nil, err
	}

	sort.SliceStable(raw, func(i, j int) bool {
		return lessBySortKey(opts.FRef, &raw[i], &raw[j])
	})

	return raw, nil
}

func resolveChip(soc string) (chipKind, error) {
	name := toLower(soc)
	chip, ok := chipRegistry[name]
	if !ok {
		return 0, fmt.Errorf("unknown chip %q; available: %v", soc, sortedChipNames())
	}
	return chip, nil
}

func sortedChipNames() []string {
	names := make([]string, 0, len(chipRegistry))
	for name := range chipRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		out[i] = b
	}
	return string(out)
}

func solveConfig(chip chipKind, fRef, fOut, epsilon float64) ([]Config, error) {
	lim := chipLimits(chip)

	if !lim.isFRefInRange(fRef) {
		return nil, fmt.Errorf("f_ref=%g kHz is out of the allowed range", fRef)
	}
	if !lim.isFOutInRange(fOut) {
		return nil, fmt.Errorf("f_out=%g kHz is out of the allowed range", fOut)
	}

	if math.Abs(fRef-fOut) < epsilon {
		c := Config{chip: chip}
		c.DivA, c.DivB = 0, 0
		c.PreDiv = 1
		c.RDiv = 1
		c.NDiv = 16
		c.Frac = 0
		c.Mod = 1
		c.DoFrac = false
		c.DoBypass = true
		c.IsCalculated = true
		return []Config{c.snapshot()}, nil
	}

	type scored struct {
		err float64
		cfg Config
	}
	var solutions []scored
	bestErr := math.Inf(1)
	var best *Config

	c := Config{chip: chip}

	for _, a := range lim.aRange() {
		c.DivA = a
		for _, b := range lim.bRange() {
			c.DivB = b
			for _, r := range lim.rRange() {
				c.RDiv = r
				for _, p := range lim.pRange() {
					c.PreDiv = p

					requiredN := c.calcRequiredN(fRef, fOut)

					roundedN := math.Round(requiredN)
					if math.Abs(requiredN-roundedN) <= 1e-9 {
						requiredN = roundedN
						c.DoFrac = false
						c.NDiv = int(roundedN)
						c.Frac, c.Mod = 0, 1
					} else {
						fracPart := requiredN - math.Floor(requiredN)
						c.DoFrac = true
						c.NDiv = int(math.Floor(requiredN))
						c.Frac, c.Mod = findFractionVGxT(fracPart, 1e-9)
					}

					if !lim.isNInRange(c.DoFrac, requiredN) {
						continue
					}
					if !lim.isFVCOInRange(c.FVCO(fRef)) {
						continue
					}
					if !lim.isFPFDInRange(&c, fRef) {
						continue
					}

					errVal := math.Abs(c.FOut(fRef) - fOut)
					snap := c.snapshot()
					snap.IsCalculated = true

					if errVal < bestErr {
						bestErr = errVal
						b := snap
						best = &b
					}
					if errVal <= epsilon {
						solutions = append(solutions, scored{errVal, snap})
					}
				}
			}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no solution found for any parameter combination")
	}
	if len(solutions) == 0 {
		return []Config{*best}, nil
	}

	sort.SliceStable(solutions, func(i, j int) bool { return solutions[i].err < solutions[j].err })
	out := make([]Config, len(solutions))
	for i, s := range solutions {
		out[i] = s.cfg
	}
	return out, nil
}

type dualScored struct {
	rankErr float64
	cfg     Config
}

func solveBothConfigs(chip chipKind, fRef, fOut1, fOut2, epsilon float64) ([]Config, error) {
	lim := chipLimits(chip)

	if !lim.isFRefInRange(fRef) {
		return nil, fmt.Errorf("f_ref=%g kHz is out of the allowed range", fRef)
	}
	if !lim.isFOutInRange(fOut1) {
		return nil, fmt.Errorf("f_out1=%g kHz is out of the allowed range", fOut1)
	}
	if !lim.isFOutInRange(fOut2) {
		return nil, fmt.Errorf("f_out2=%g kHz is out of the allowed range", fOut2)
	}

	var solutions []dualScored
	bestErr := math.Inf(1)
	var best *Config

	c := Config{chip: chip}
	c.PreDiv = 1

	tryCandidate := func() {
		if !lim.isFVCOInRange(c.FVCO(fRef)) {
			return
		}
		if !lim.isFPFDInRange(&c, fRef) {
			return
		}

		fVCO := c.FVCO(fRef)
		divs1 := findDividers(lim, fVCO, fOut1, epsilon)
		divs2 := findDividers(lim, fVCO, fOut2, epsilon)

		if len(divs1) == 0 || len(divs2) == 0 {
			return
		}

		for _, d1 := range divs1 {
			for _, d2 := range divs2 {
				rankErr := d1.err + d2.err
				acceptErr := math.Max(d1.err, d2.err)

				c.DivA, c.DivB = d1.a, d1.b
				c.Div2A, c.Div2B = d2.a, d2.b
				c.IsCalculated = true

				snap := c.snapshot()

				if rankErr < bestErr {
					bestErr = rankErr
					b := snap
					best = &b
				}
				if acceptErr <= epsilon {
					solutions = append(solutions, dualScored{rankErr, snap})
				}
			}
		}
	}

	for _, r := range lim.rRange() {
		c.RDiv = r

		// do_frac=false pass: ndiv in [1, 160]
		c.DoFrac = false
		for ndiv := 1; ndiv <= 160; ndiv++ {
			c.NDiv = ndiv
			c.Frac, c.Mod = 0, 1
			tryCandidate()
		}

		// do_frac=true pass: ndiv in [20, 160]
		c.DoFrac = true
		for ndiv := 20; ndiv <= 160; ndiv++ {
			c.NDiv = ndiv

			for _, fracPart := range fracCandidatesVG015(&c, lim, fRef, fOut1, fOut2) {
				c.Frac, c.Mod = findFractionVG015(fracPart)
				tryCandidate()
			}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no solution found")
	}
	if len(solutions) == 0 {
		return []Config{*best}, nil
	}

	sort.SliceStable(solutions, func(i, j int) bool { return solutions[i].rankErr < solutions[j].rankErr })
	out := make([]Config, len(solutions))
	for i, s := range solutions {
		out[i] = s.cfg
	}
	return out, nil
}

type divCandidate struct {
	a, b int
	err  float64
}

func findDividers(lim limits, fVCO, fOutTarget, epsilon float64) []divCandidate {
	var out []divCandidate
	for _, a := range lim.aRange() {
		for _, b := range lim.bRange() {
			errVal := math.Abs(fVCO/float64((a+1)*(b+1)) - fOutTarget)
			if errVal <= epsilon {
				out = append(out, divCandidate{a, b, errVal})
			}
		}
	}
	return out
}

func fracCandidatesVG015(c *Config, lim limits, fRef, fOut1, fOut2 float64) []float64 {
	fPFD := c.FPFD(fRef)
	fracLo := math.Max(0.0, 200e3/fPFD-float64(c.NDiv))
	fracHi := math.Min(1-1e-12, 1600e3/fPFD-float64(c.NDiv))

	if fracLo >= fracHi {
		return nil
	}

	aMax := maxInt(lim.aRange()) + 1
	bMax := maxInt(lim.bRange()) + 1

	seen := make(map[float64]bool)
	var out []float64

	for ab := 1; ab <= aMax*bMax; ab++ {
		for _, target := range [2]float64{fOut1, fOut2} {
			frac := target*float64(ab)/fPFD - float64(c.NDiv)
			if frac >= fracLo && frac <= fracHi && !seen[frac] {
				seen[frac] = true
				out = append(out, frac)
			}
		}
	}

	return out
}

func maxInt(vals []int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func lessBySortKey(fRef float64, a, b *Config) bool {
	ka := sortKeyOf(fRef, a)
	kb := sortKeyOf(fRef, b)
	for i := range ka {
		if ka[i] != kb[i] {
			return ka[i] < kb[i]
		}
	}
	return false
}

func sortKeyOf(fRef float64, cfg *Config) [7]float64 {
	fVCO := cfg.FVCO(fRef)

	a1 := float64(cfg.DivA + 1)
	b1 := float64(cfg.DivB + 1)

	penaltyFrac := 0.0
	if cfg.DoFrac {
		penaltyFrac = 1
	}
	penaltyBalance := math.Abs(a1 - b1)
	penaltyTotalDiv := -(a1 * b1)
	penaltyVCO := 0.0
	if !(fVCO >= 600e3 && fVCO <= 1000e3) {
		penaltyVCO = 1
	}
	penaltyNDiv := -float64(cfg.NDiv)
	penaltyRDiv := float64(cfg.RDiv)
	penaltyPreDiv := float64(cfg.PreDiv)

	if cfg.HasSecondOutput() {
		a2 := float64(cfg.Div2A + 1)
		b2 := float64(cfg.Div2B + 1)
		penaltyBalance += math.Abs(a2 - b2)
		penaltyTotalDiv += -(a2 * b2)
	}

	return [7]float64{
		penaltyFrac,
		penaltyBalance,
		penaltyTotalDiv,
		penaltyVCO,
		penaltyNDiv,
		penaltyRDiv,
		penaltyPreDiv,
	}
}
