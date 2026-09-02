package pll

import (
	"fmt"
	"strings"
	"testing"
)

// TestSolveKnownVG1TCoefficients checks the three reference solutions from
// the SDK's example PLL header:
//
//	FROM_16_TO_125:              16 MHz -> 125 MHz, ref_div=1 fb_mult=125 out_div_a=1 out_div_b=3 prediv=1
//	APP_PLL_FROM_16MHZ_TO_140MHZ: 16 MHz -> 140 MHz, ref_div=1 fb_mult=105 out_div_a=1 out_div_b=2 prediv=1
//	USB_PLL_FROM_16MHZ_TO_60MHZ:  16 MHz -> 60 MHz,  ref_div=1 fb_mult=120 out_div_a=3 out_div_b=3 prediv=1
//
// All three exceed K1921VG3T/VG5T/VG7T's f_out ceiling (120 MHz / 100 MHz /
// 100 MHz respectively), so they can only have come from K1921VG1T, whose
// ConfigBase f_out range (200 Hz .. 204 MHz) is unrestricted.
func TestSolveKnownVG1TCoefficients(t *testing.T) {
	cases := []struct {
		name                                               string
		fRef, fOut                                         float64
		wantRDiv, wantNDiv, wantDivA, wantDivB, wantPreDiv int
	}{
		{"16MHz_to_125MHz", 16000, 125000, 1, 125, 1, 3, 1},
		{"16MHz_to_140MHz", 16000, 140000, 1, 105, 1, 2, 1},
		{"16MHz_to_60MHz", 16000, 60000, 1, 120, 3, 3, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sols, err := Solve(SolveOptions{
				SoC:   "k1921vg1t",
				FRef:  c.fRef,
				FOut1: c.fOut,
			})
			if err != nil {
				t.Fatalf("Solve(%g -> %g): unexpected error: %v", c.fRef, c.fOut, err)
			}
			if len(sols) == 0 {
				t.Fatalf("Solve(%g -> %g): no solutions returned", c.fRef, c.fOut)
			}

			best := sols[0]

			if best.RDiv != c.wantRDiv {
				t.Errorf("RDiv = %d, want %d", best.RDiv, c.wantRDiv)
			}
			if best.NDiv != c.wantNDiv {
				t.Errorf("NDiv = %d, want %d", best.NDiv, c.wantNDiv)
			}
			if best.DivA != c.wantDivA {
				t.Errorf("DivA = %d, want %d", best.DivA, c.wantDivA)
			}
			if best.DivB != c.wantDivB {
				t.Errorf("DivB = %d, want %d", best.DivB, c.wantDivB)
			}
			if best.PreDiv != c.wantPreDiv {
				t.Errorf("PreDiv = %d, want %d", best.PreDiv, c.wantPreDiv)
			}
			if best.DoFrac {
				t.Errorf("DoFrac = true, want an integer-mode solution (all three reference cases hit their target exactly)")
			}

			gotFOut := best.FOut(c.fRef)
			if diff := absDiff(gotFOut, c.fOut); diff > 1e-6 {
				t.Errorf("FOut(%g) = %g, want %g (diff %g)", c.fRef, gotFOut, c.fOut, diff)
			}
		})
	}
}

// TestGenerateHeaderMatchesReference renders each known solution to C and
// checks that the emitted ll_pll_config_t fields and LL_PLL_F_OUT_M call
// match the reference header byte for byte on the lines that matter (the
// struct fields and the static assert), using the same prefixes as the
// original example (none, "APP", "USB").
func TestGenerateHeaderMatchesReference(t *testing.T) {
	cases := []struct {
		name       string
		fRef, fOut float64
		prefix     string
		wantMacro  string
		wantFOut   string
		wantStatic string
	}{
		{
			name: "no_prefix_125MHz", fRef: 16000, fOut: 125000, prefix: "",
			wantMacro:  "#define PLL_FROM_16MHZ_TO_125MHZ \\",
			wantFOut:   "#define F_OUT \\",
			wantStatic: `_Static_assert(F_OUT == 125000, "Wrong PLL math");`,
		},
		{
			name: "APP_prefix_140MHz", fRef: 16000, fOut: 140000, prefix: "APP",
			wantMacro:  "#define APP_PLL_FROM_16MHZ_TO_140MHZ \\",
			wantFOut:   "#define APP_F_OUT \\",
			wantStatic: `_Static_assert(APP_F_OUT == 140000, "Wrong PLL math");`,
		},
		{
			name: "USB_prefix_60MHz", fRef: 16000, fOut: 60000, prefix: "USB",
			wantMacro:  "#define USB_PLL_FROM_16MHZ_TO_60MHZ \\",
			wantFOut:   "#define USB_F_OUT \\",
			wantStatic: `_Static_assert(USB_F_OUT == 60000, "Wrong PLL math");`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sols, err := Solve(SolveOptions{SoC: "k1921vg1t", FRef: c.fRef, FOut1: c.fOut})
			if err != nil {
				t.Fatalf("Solve: unexpected error: %v", err)
			}

			out, err := GenerateHeader(sols[:1], CodegenOptions{
				CPU:    "k1921vg1t",
				FRef:   c.fRef,
				FOut1:  c.fOut,
				Prefix: c.prefix,
			})
			if err != nil {
				t.Fatalf("GenerateHeader: unexpected error: %v", err)
			}

			for _, want := range []string{c.wantMacro, c.wantFOut, c.wantStatic} {
				if !strings.Contains(out, want) {
					t.Errorf("generated header missing expected line %q\n--- full output ---\n%s", want, out)
				}
			}

			// The register field block should match the reference example
			// exactly for the 125 MHz case (the one given verbatim in the
			// original request).
			if c.name == "no_prefix_125MHz" {
				wantFields := []string{
					".ref_div   = 1, \\",
					".fb_mult   = 125, \\",
					".frac_num  = 0, \\",
					".frac_mod  = 1, \\",
					".out_div_a = 1, \\",
					".out_div_b = 3, \\",
					".prediv    = 1, \\",
				}
				for _, f := range wantFields {
					if !strings.Contains(out, f) {
						t.Errorf("generated header missing expected field line %q", f)
					}
				}
			}
		})
	}
}

// TestSolveValidation checks that out-of-range inputs and unsupported
// dual-output requests are rejected, matching calc.py's ValueError cases.
func TestSolveValidation(t *testing.T) {
	t.Run("unknown_soc", func(t *testing.T) {
		if _, err := Solve(SolveOptions{SoC: "not-a-real-chip", FRef: 16000, FOut1: 100000}); err == nil {
			t.Error("expected an error for an unknown SoC, got nil")
		}
	})

	t.Run("fref_out_of_range", func(t *testing.T) {
		// K1921VG1T requires 10e3 <= f_ref <= 100e3 kHz.
		if _, err := Solve(SolveOptions{SoC: "k1921vg1t", FRef: 1000, FOut1: 100000}); err == nil {
			t.Error("expected an error for f_ref out of range, got nil")
		}
	})

	t.Run("fout_out_of_range", func(t *testing.T) {
		// K1921VG3T caps f_out at 120e3 kHz.
		if _, err := Solve(SolveOptions{SoC: "k1921vg3t", FRef: 16000, FOut1: 125000}); err == nil {
			t.Error("expected an error for f_out exceeding K1921VG3T's range, got nil")
		}
	})

	t.Run("fout2_on_single_output_chip", func(t *testing.T) {
		f2 := 60000.0
		if _, err := Solve(SolveOptions{SoC: "k1921vg1t", FRef: 16000, FOut1: 100000, FOut2: &f2}); err == nil {
			t.Error("expected an error when FOut2 is given for a single-output chip, got nil")
		}
	})
}

// TestSolveDualOutputVG015 exercises the K1921VG015 dual-output path.
// There's no reference C example for this chip in the request, so this
// only checks internal consistency: both outputs land within epsilon of
// their targets, and GenerateHeader correctly refuses to emit
// ll_pll_config_t for a dual-output solution.
func TestSolveDualOutputVG015(t *testing.T) {
	fOut2 := 20000.0
	sols, err := Solve(SolveOptions{
		SoC:   "k1921vg015",
		FRef:  20000,
		FOut1: 40000,
		FOut2: &fOut2,
	})
	if err != nil {
		t.Fatalf("Solve: unexpected error: %v", err)
	}
	if len(sols) == 0 {
		t.Fatal("Solve: no solutions returned")
	}

	best := sols[0]
	if !best.HasSecondOutput() {
		t.Fatal("HasSecondOutput() = false for a K1921VG015 solution")
	}

	gotFOut1 := best.FOut(20000)
	gotFOut2 := best.FOut2(20000)
	if diff := absDiff(gotFOut1, 40000); diff > 1 {
		t.Errorf("FOut = %g, want ~40000 (diff %g)", gotFOut1, diff)
	}
	if diff := absDiff(gotFOut2, fOut2); diff > 1 {
		t.Errorf("FOut2 = %g, want ~%g (diff %g)", gotFOut2, fOut2, diff)
	}

	if _, err := GenerateHeader(sols[:1], CodegenOptions{CPU: "k1921vg015", FRef: 20000, FOut1: 40000, FOut2: &fOut2}); err != nil {
		t.Errorf("GenerateHeader: unexpected error for a dual-output solution: %v", err)
	}
	if out, err := GenerateHeader(sols[:1], CodegenOptions{CPU: "k1921vg015", FRef: 20000, FOut1: 40000}); err != nil {
		t.Errorf("GenerateHeader: unexpected error when FOut2 is omitted for a dual-output solution: %v", err)
	} else if strings.Contains(out, "_OUT1") || strings.Contains(out, "_OUT2") {
		t.Errorf("GenerateHeader without FOut2 should render a single unsuffixed block, got:\n%s", out)
	}
}

// TestSolveBypassCase checks the f_ref == f_out short-circuit path
// (solve_config's bypass branch in calc.py), which returns a fixed
// bypass configuration rather than searching.
func TestSolveBypassCase(t *testing.T) {
	sols, err := Solve(SolveOptions{SoC: "k1921vg1t", FRef: 16000, FOut1: 16000})
	if err != nil {
		t.Fatalf("Solve: unexpected error: %v", err)
	}
	if len(sols) != 1 {
		t.Fatalf("len(sols) = %d, want 1 for the bypass case", len(sols))
	}
	if !sols[0].DoBypass {
		t.Error("DoBypass = false, want true when f_ref == f_out")
	}
}

// TestGenerateHeaderDualOutputSharesVCOFields checks that a dual-output
// (K1921VG015) solution renders as two ll_pll_config_t blocks — one per
// output — sharing every field except out_div_a/out_div_b, tagged _OUT1
// and _OUT2 so the macro names don't collide.
func TestGenerateHeaderDualOutputSharesVCOFields(t *testing.T) {
	fOut2 := 20000.0
	sols, err := Solve(SolveOptions{SoC: "k1921vg015", FRef: 20000, FOut1: 40000, FOut2: &fOut2})
	if err != nil {
		t.Fatalf("Solve: unexpected error: %v", err)
	}
	best := sols[0]

	out, err := GenerateHeader(sols[:1], CodegenOptions{CPU: "k1921vg015", FRef: 20000, FOut1: 40000, FOut2: &fOut2})
	if err != nil {
		t.Fatalf("GenerateHeader: unexpected error: %v", err)
	}

	for _, want := range []string{
		"#define PLL_FROM_20MHZ_TO_40MHZ_OUT1 \\",
		"#define PLL_FROM_20MHZ_TO_20MHZ_OUT2 \\",
		"#define F_OUT_OUT1 \\",
		"#define F_OUT_OUT2 \\",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("generated header missing expected line %q\n--- full output ---\n%s", want, out)
		}
	}

	sharedFields := []string{
		fmt.Sprintf(".ref_div   = %d, \\", best.RDiv),
		fmt.Sprintf(".fb_mult   = %d, \\", best.NDiv),
		fmt.Sprintf(".prediv    = %d, \\", best.PreDiv),
	}
	for _, f := range sharedFields {
		if n := strings.Count(out, f); n != 2 {
			t.Errorf("expected shared field %q to appear exactly twice (once per output block), got %d", f, n)
		}
	}

	wantA1 := fmt.Sprintf(".out_div_a = %d, \\", best.DivA)
	wantA2 := fmt.Sprintf(".out_div_a = %d, \\", best.Div2A)
	if !strings.Contains(out, wantA1) {
		t.Errorf("missing primary out_div_a line %q", wantA1)
	}
	if !strings.Contains(out, wantA2) {
		t.Errorf("missing secondary out_div_a line %q", wantA2)
	}
}

// TestGenerateHeaderVG015SingleOutputRequest reproduces the reported bug:
// requesting a single output frequency on a dual-output chip (no --output2)
// must succeed and render exactly one block using the solution's primary
// output — the second output divider pair the solver always computes for
// this chip is simply not printed.
func TestGenerateHeaderVG015SingleOutputRequest(t *testing.T) {
	sols, err := Solve(SolveOptions{SoC: "k1921vg015", FRef: 20000, FOut1: 40000})
	if err != nil {
		t.Fatalf("Solve: unexpected error: %v", err)
	}
	if len(sols) == 0 {
		t.Fatal("Solve: no solutions returned")
	}

	out, err := GenerateHeader(sols[:1], CodegenOptions{CPU: "k1921vg015", FRef: 20000, FOut1: 40000})
	if err != nil {
		t.Fatalf("GenerateHeader: unexpected error: %v", err)
	}

	if strings.Count(out, "ll_pll_config_t") != 1 {
		t.Errorf("expected exactly one ll_pll_config_t block, got:\n%s", out)
	}
	if !strings.Contains(out, "#define F_OUT \\") {
		t.Errorf("expected an unsuffixed F_OUT macro, got:\n%s", out)
	}
	if strings.Contains(out, "_OUT1") || strings.Contains(out, "_OUT2") {
		t.Errorf("did not expect OUT1/OUT2 suffixes for a single-output request, got:\n%s", out)
	}
}

func absDiff(a, b float64) float64 {
	if a < b {
		return b - a
	}
	return a - b
}
