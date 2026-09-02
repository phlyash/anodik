package pll

import "math"

const vgxtMaxMod = (1 << 24) - 1

// findFractionVGxT is a direct port of ConfigBase.find_fraction: a
// Stern-Brocot / continued-fraction search for the best num/mod
// approximation of fraction with mod <= vgxtMaxMod.
//
// fraction must be in (0, 1) and epsilon in (0, 1), matching the asserts
// in calc.py; callers here never violate that (see solveConfig / dual
// solve), so this doesn't re-validate.
func findFractionVGxT(fraction, epsilon float64) (num, mod int) {
	p0, q0 := 0, 1
	p1, q1 := 1, 1
	bestFrac, bestMod := 0, 1
	bestDiff := math.Abs(fraction)

	updateBest := func(p, q int) {
		diff := math.Abs(fraction - float64(p)/float64(q))
		if diff < bestDiff {
			bestDiff = diff
			bestFrac, bestMod = p, q
		}
	}

	for {
		pm, qm := p0+p1, q0+q1

		if qm > vgxtMaxMod {
			if fraction > float64(p0)/float64(q0) {
				k := (vgxtMaxMod - q0) / q1
				if k >= 1 {
					updateBest(p0+k*p1, q0+k*q1)
				}
			} else {
				k := (vgxtMaxMod - q1) / q0
				if k >= 1 {
					updateBest(k*p0+p1, k*q0+q1)
				}
			}
			break
		}

		medVal := float64(pm) / float64(qm)
		updateBest(pm, qm)

		if math.Abs(fraction-medVal) <= epsilon {
			return pm, qm
		}

		if fraction < medVal {
			p1, q1 = pm, qm
		} else {
			p0, q0 = pm, qm
		}
	}

	return bestFrac, bestMod
}

// vg015Mod is the fixed frac_mod K1921VG015 uses (1 << 24, calc.py's
// K1921VG015Config.find_fraction).
const vg015Mod = 1 << 24

// findFractionVG015 is a direct port of K1921VG015Config.find_fraction:
// unlike the VGxT search, VG015's fractional field is a plain fixed-point
// numerator over a constant power-of-two modulus, so this just rounds and
// clamps into [1, mod-1].
func findFractionVG015(fraction float64) (num, mod int) {
	frac := int(math.Round(vg015Mod * fraction))
	if frac < 1 {
		frac = 1
	}
	if frac > vg015Mod-1 {
		frac = vg015Mod - 1
	}
	return frac, vg015Mod
}
