package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"anodik/internal/toolchain"
	"fmt"
	"time"
)

type RunJob struct {
	SDKRootRequirement
	ExampleOptions
	BuildOptions
	SoCOptions
	DebugOptions
	ProcessOptions
}

func (j *RunJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *RunJob) Run(s *env.Settings, manager *process.Manager) error {
	buildType := j.BuildType.Resolved(s.BuildType)
	if err := buildOne(s, manager, j.ProjectDir, j.Example, buildType, j.Jobs); err != nil {
		return err
	}

	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, j.GDBPort)
	if err != nil {
		return err
	}
	if err := checkOCDConfigs(cfg); err != nil {
		return err
	}
	if err := checkELF(cfg, j.Example); err != nil {
		return err
	}

	if manager.IsAlive(ocdProcessName) {
		fmt.Println("Different session of openocd running in background. Stopping...")
		err = manager.Kill(ocdProcessName)
		if err != nil {
			return fmt.Errorf("Cannot stop: %w", err)
		}
		fmt.Println("Stopped successfully.")
	}

	ports := ocd.AllocatePorts(cfg.GDBPort)

	fmt.Println("Starting OpenOCD in background...")
	ocd.PrintInfo(cfg, &ports)

	ocdArgs := ocd.Command(s, s.SDKRoot, cfg, ports, nil)
	pid, alive, err := manager.RunDetachedLogged(ocdArgs[0], ocdArgs[1:], ocd.LogPath(s.SDKRoot), ports.GDB)
	if err != nil {
		return err
	}
	if !alive {
		fmt.Println("error: OpenOCD failed to start. See log:", ocd.LogPath(s.SDKRoot))
		return fmt.Errorf("openocd did not stay running")
	}
	fmt.Printf("OpenOCD started (PID: %d)\n", pid)

	targetName, err := ocd.ResolveTargetName("127.0.0.1:"+ports.TCL, cfg.SoC, 2*time.Second)
	if err != nil {
		fmt.Println("error: could not find OpenOCD target for this SoC. See log:", ocd.LogPath(s.SDKRoot))
		return err
	}

	fmt.Printf("[TARGET] %s\n", targetName)

	if examined, detail, err := ocd.CheckTargetExamined("127.0.0.1:"+ports.TCL, targetName, 2*time.Second); err != nil {
		fmt.Println("error: OpenOCD is not responding on its TCL port. See log:", ocd.LogPath(s.SDKRoot))
		return fmt.Errorf("openocd tcl check failed: %w", err)
	} else if !examined {
		fmt.Println("error: OpenOCD could not find the target chip (check programmer/wiring). See log:", ocd.LogPath(s.SDKRoot))
		return fmt.Errorf("openocd target not examined: %s", detail)
	}

	fmt.Printf("Starting GDB, connecting to localhost:%s...\n", ports.GDB)
	gdbArgs := []string{
		"-q",
		cfg.ELFPath,
		"-ex", "target extended-remote localhost:" + ports.GDB,
		"-ex", "monitor reset halt",
		"-ex", "load",
	}

	return manager.RunForeground(toolchain.GDB(s), gdbArgs)
}

func (j *RunJob) Finish(settings *env.Settings, manager *process.Manager) error {
	return manager.Kill(ocdProcessName)

}

func (j *RunJob) Stop(*env.Settings, *process.Manager) error {
	return nil
}

func (j *RunJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "build, flash, and attach GDB in one step",
		Usage:   "<example> [flags]",
	}.Merge(j.ExampleOptions.Help(), j.BuildOptions.Help(), j.SoCOptions.Help(), j.DebugOptions.Help())
}
