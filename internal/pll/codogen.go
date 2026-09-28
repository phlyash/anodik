package pll

import (
	"fmt"
	"strconv"
	"strings"
)

type CodegenOptions struct {
	CPU          string
	FRef         float64
	FOut1        float64
	FOut2        *float64
	Prefix       string
	MaxSolutions int
}

func GenerateHeader(solutions []Config, opts CodegenOptions) (string, error) {
	maxSolutions := opts.MaxSolutions
	if maxSolutions <= 0 || maxSolutions > len(solutions) {
		maxSolutions = len(solutions)
	}
	used := solutions[:maxSolutions]

	if opts.FOut2 != nil {
		for _, cfg := range used {
			if !cfg.HasSecondOutput() {
				return "", fmt.Errorf("CodegenOptions.FOut2 is set but this solution has no second output")
			}
		}
	}

	var body strings.Builder
	body.WriteString("#pragma once\n\n#include <ll_pll.h>\n")

	total := len(used)
	for i, cfg := range used {
		if opts.FOut2 != nil {
			block1, err := makeStructBlock(&cfg, cfg.DivA, cfg.DivB, opts.CPU, opts.FRef, opts.FOut1, opts.Prefix, "OUT1", i+1, total)
			if err != nil {
				return "", err
			}
			block2, err := makeStructBlock(&cfg, cfg.Div2A, cfg.Div2B, opts.CPU, opts.FRef, *opts.FOut2, opts.Prefix, "OUT2", i+1, total)
			if err != nil {
				return "", err
			}
			body.WriteString(block1)
			body.WriteString(block2)
			continue
		}

		block, err := makeStructBlock(&cfg, cfg.DivA, cfg.DivB, opts.CPU, opts.FRef, opts.FOut1, opts.Prefix, "", i+1, total)
		if err != nil {
			return "", err
		}
		body.WriteString(block)
	}
	body.WriteString("\n")

	return body.String(), nil
}

func makeStructBlock(cfg *Config, divA, divB int, cpu string, fRef, requestedFOut float64, prefix, outputTag string, index, total int) (string, error) {
	fReal := cfg.FVCO(fRef) / float64((divA+1)*(divB+1))
	fReq := requestedFOut
	if fReq == 0 {
		fReq = fReal
	}
	errVal := absF(fReal - fReq)
	fVCO := cfg.FVCO(fRef)

	fracNum, fracMod := 0, 1
	if cfg.DoFrac {
		fracNum, fracMod = cfg.Frac, cfg.Mod
	}

	fRefInt := int64(roundF(fRef))
	fOutInt := int64(roundF(fReal))

	p := sanitizeIdent(prefix)
	suffix := ""
	if total > 1 {
		suffix = fmt.Sprintf("_%d", index)
	}
	if outputTag != "" {
		suffix += "_" + outputTag
	}

	macroName := makeConfigName(prefix, fRef, fReq, index, total, outputTag)

	fOutMacro := fmt.Sprintf("F_OUT%s", suffix)
	if p != "" {
		fOutMacro = fmt.Sprintf("%s_F_OUT%s", p, suffix)
	}

	label := fmt.Sprintf("solution %d", index)
	if outputTag != "" {
		label = fmt.Sprintf("solution %d, %s", index, outputTag)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n/* %s for %s\n", label, strings.ToUpper(cpu))
	fmt.Fprintf(&b, " * request : %s kHz\n", formatF(fReq, 3))
	fmt.Fprintf(&b, " * actual  : %s kHz\n", formatF(fReal, 3))
	fmt.Fprintf(&b, " * error   : %s kHz\n", formatF(errVal, 6))
	fmt.Fprintf(&b, " * f_vco   : %s kHz\n", formatF(fVCO, 3))
	b.WriteString(" */\n")
	fmt.Fprintf(&b, "#define %s \\\n", macroName)
	b.WriteString("    ((ll_pll_config_t){ \\\n")
	fmt.Fprintf(&b, "        .ref_div   = %d, \\\n", cfg.RDiv)
	fmt.Fprintf(&b, "        .fb_mult   = %d, \\\n", cfg.NDiv)
	fmt.Fprintf(&b, "        .frac_num  = %d, \\\n", fracNum)
	fmt.Fprintf(&b, "        .frac_mod  = %d, \\\n", fracMod)
	fmt.Fprintf(&b, "        .out_div_a = %d, \\\n", divA)
	fmt.Fprintf(&b, "        .out_div_b = %d, \\\n", divB)
	fmt.Fprintf(&b, "        .prediv    = %d, \\\n", cfg.PreDiv)
	b.WriteString("    })\n\n")
	fmt.Fprintf(&b, "#define %s \\\n", fOutMacro)
	fmt.Fprintf(&b, "    LL_PLL_F_OUT_M(%d, \\\n", fRefInt)
	fmt.Fprintf(&b, "        %d, %d, %d, %d, %d, %d, %d)\n\n", cfg.RDiv, cfg.NDiv, fracNum, fracMod, divA, divB, cfg.PreDiv)
	fmt.Fprintf(&b, "_Static_assert(%s == %d, \"Wrong PLL math\");\n", fOutMacro, fOutInt)

	return b.String(), nil
}

func makeConfigName(prefix string, fRef, fOut float64, index, total int, outputTag string) string {
	var parts []string
	if p := sanitizeIdent(prefix); p != "" {
		parts = append(parts, p)
	}
	parts = append(parts, "PLL", "FROM", freqToIdent(fRef), "TO", freqToIdent(fOut))

	name := strings.Join(parts, "_")
	if total > 1 {
		name += fmt.Sprintf("_%d", index)
	}
	if outputTag != "" {
		name += "_" + outputTag
	}
	return name
}

func freqToIdent(freqKHz float64) string {
	khz := int64(roundF(freqKHz))
	if khz%1000 == 0 {
		return fmt.Sprintf("%dMHZ", khz/1000)
	}
	return fmt.Sprintf("%dKHZ", khz)
}

func sanitizeIdent(text string) string {
	var out strings.Builder
	prevUnderscore := false

	for _, r := range strings.ToUpper(text) {
		if isAlnum(r) {
			out.WriteRune(r)
			prevUnderscore = false
		} else if !prevUnderscore {
			out.WriteByte('_')
			prevUnderscore = true
		}
	}

	ident := strings.Trim(out.String(), "_")
	if ident != "" && isDigit(rune(ident[0])) {
		ident = "_" + ident
	}
	return ident
}

func isAlnum(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func absF(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func roundF(f float64) float64 {
	if f < 0 {
		return -roundF(-f)
	}
	i := float64(int64(f))
	if f-i >= 0.5 {
		i++
	}
	return i
}

func formatF(f float64, prec int) string {
	return strconv.FormatFloat(f, 'f', prec, 64)
}
