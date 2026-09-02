package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/pll"
	"anodik/internal/process"
	"fmt"
)

type PllJob struct {
	NoSDKRootRequirement
	ExampleOptions
	SoCOptions
	PllOptions
}

func (j *PllJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *PllJob) Run(s *env.Settings, _ *process.Manager) error {
	soc, err := j.resolveSoC(s)
	if err != nil {
		return err
	}

	opts := pll.SolveOptions{
		SoC:   soc,
		FRef:  j.Input,
		FOut1: j.Output,
	}
	if j.HasOutput2 {
		opts.FOut2 = &j.Output2
	}

	solutions, err := pll.Solve(opts)
	if err != nil {
		return fmt.Errorf("solve PLL config: %w", err)
	}

	maxSolutions := j.NumSolutions
	if maxSolutions <= 0 {
		maxSolutions = 1
	}
	if maxSolutions > len(solutions) {
		maxSolutions = len(solutions)
	}

	header, err := pll.GenerateHeader(solutions[:maxSolutions], pll.CodegenOptions{
		CPU:          soc,
		FRef:         j.Input,
		FOut1:        j.Output,
		FOut2:        opts.FOut2,
		MaxSolutions: maxSolutions,
	})
	if err != nil {
		return fmt.Errorf("generate PLL header: %w", err)
	}

	fmt.Print(header)
	return nil
}

func (j *PllJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *PllJob) resolveSoC(s *env.Settings) (string, error) {
	if err := j.ExampleOptions.ResolveOptional(s.SDKRoot, s.WorkingDir); err != nil {
		return "", err
	}

	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, "")
	if err != nil {
		return "", err
	}
	return cfg.SoC, nil
}

func (j *PllJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "compute PLL coefficients for a reference/output frequency and print generated C code",
		Usage:   "[example] --input <kHz> --output <kHz> [flags]",
	}.Merge(j.ExampleOptions.Help(), j.SoCOptions.Help(), j.PllOptions.Help())
}
