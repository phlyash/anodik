package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"fmt"
	"strings"
)

type SrvEraseJob struct {
	SDKRootRequirement
	ExampleOptions
	SoCOptions
	ProcessOptions
}

func (j *SrvEraseJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SrvEraseJob) Run(s *env.Settings, manager *process.Manager) error {
	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, "")
	if err != nil {
		return err
	}
	if err := checkOCDConfigs(cfg); err != nil {
		return err
	}

	if err := ocd.StopInterface(manager, cfg.Interface); err != nil {
		return err
	}

	ports, err := ocd.ResolvePorts(cfg.GDBPort, cfg.GDBPortExplicit)
	if err != nil {
		return err
	}

	fmt.Println("Erasing flash...")
	ocd.PrintInfo(cfg, &ports)

	driverTail := []string{"-c", "init", "-c", "halt", "-c", cfg.SoC + " srv_erase", "-c", "exit"}
	driverArgs := ocd.Command(s, s.SDKRoot, cfg, ports, driverTail)

	output, err := manager.RunForegroundCaptured(driverArgs[0], driverArgs[1:])
	if err == nil && !ocdCommandFailed(output) {
		return nil
	}

	fmt.Printf("\n'%s srv_erase' did not run, falling back to sector erase...\n", cfg.SoC)

	fallbackTail := []string{"-c", "init", "-c", "reset halt", "-c", "flash erase_sector 0 0 last", "-c", "reset", "-c", "exit"}
	fallbackArgs := ocd.Command(s, s.SDKRoot, cfg, ports, fallbackTail)

	return manager.RunForeground(fallbackArgs[0], fallbackArgs[1:])
}

func ocdCommandFailed(output string) bool {
	return strings.Contains(output, "invalid command name") || strings.Contains(output, "ERROR:")
}

func (j *SrvEraseJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SrvEraseJob) Stop(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SrvEraseJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "mass-erase flash via OpenOCD",
		Usage:   "[example] [flags]",
	}.Merge(j.ExampleOptions.Help(), j.SoCOptions.Help())
}
