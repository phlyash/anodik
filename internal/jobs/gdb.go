package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"anodik/internal/toolchain"
	"fmt"
)

type GDBJob struct {
	SDKRootRequirement
	ExampleOptions
	SoCOptions
	DebugOptions
	ProcessOptions
}

func (j *GDBJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *GDBJob) Run(s *env.Settings, manager *process.Manager) error {
	cliGDBPort := j.GDBPort
	if cliGDBPort == "" {
		// Prefer whatever port a running background openocd session
		// picked, so `anodik ocd --background` followed by `anodik gdb`
		// just works without repeating --gdb-port.
		cliGDBPort = ocd.SessionGDBPort(s.SDKRoot)
	}

	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, cliGDBPort)
	if err != nil {
		return err
	}
	if err := checkELF(cfg, j.Example); err != nil {
		return err
	}

	fmt.Printf("Starting GDB, connecting to localhost:%s...\n", cfg.GDBPort)

	args := []string{
		"-q",
		cfg.ELFPath,
		"-ex", "target extended-remote localhost:" + cfg.GDBPort,
	}

	return manager.RunForeground(toolchain.GDB(s), args)
}

func (j *GDBJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *GDBJob) Stop(*env.Settings, *process.Manager) error {
	return nil
}

func (j *GDBJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "attach GDB to a running (or self-started) OpenOCD session",
		Usage:   "<example> [flags]",
	}.Merge(j.ExampleOptions.Help(), j.SoCOptions.Help(), j.DebugOptions.Help())
}
