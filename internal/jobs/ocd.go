package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"fmt"
)

const ocdProcessName = "openocd"

type OCDJob struct {
	SDKRootRequirement
	ExampleOptions
	SoCOptions
	DebugOptions
	ProcessOptions
}

func (j *OCDJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *OCDJob) Run(s *env.Settings, manager *process.Manager) error {
	if j.StopProcess {
		return ocdStop(manager)
	}
	if j.Status {
		return ocdStatus(manager)
	}

	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, j.GDBPort)
	if err != nil {
		return err
	}
	if err := checkOCDConfigs(cfg); err != nil {
		return err
	}

	ports := ocd.AllocatePorts(cfg.GDBPort)
	args := ocd.Command(s, s.SDKRoot, cfg, ports, nil)

	if j.RunInBackground {
		return startOCDBackground(s, manager, cfg, ports, args)
	}
	return startOCDForeground(manager, cfg, ports, args)
}

func (j *OCDJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *OCDJob) Stop(_ *env.Settings, manager *process.Manager) error {
	return ocdStop(manager)
}

func (j *OCDJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "start (or stop/query) a standalone OpenOCD server",
		Usage:   "[example] [flags]",
	}.Merge(j.ExampleOptions.Help(), j.SoCOptions.Help(), j.DebugOptions.Help(), j.ProcessOptions.Help())
}

// startOCDForeground runs openocd attached to the terminal. There is no
// persistent session file here — ocd stop/status only track background
// servers started with --background, matching the python tool's behavior
// where a foreground openocd is owned by whoever's terminal it's in.
func startOCDForeground(manager *process.Manager, cfg ocd.Config, ports ocd.Ports, args []string) error {
	fmt.Println("Starting OpenOCD...")
	ocd.PrintInfo(cfg, &ports)

	return manager.RunForeground(args[0], args[1:])
}

func startOCDBackground(s *env.Settings, manager *process.Manager, cfg ocd.Config, ports ocd.Ports, args []string) error {
	fmt.Println("Starting OpenOCD in background...")
	ocd.PrintInfo(cfg, &ports)

	pid, alive, err := manager.RunDetachedLogged(args[0], args[1:], ocd.LogPath(s.SDKRoot), ports.GDB)
	if err != nil {
		return err
	}

	if !alive {
		fmt.Println("ERROR: OpenOCD failed to start. See log:", ocd.LogPath(s.SDKRoot))
		return fmt.Errorf("openocd did not stay running")
	}

	fmt.Printf("OpenOCD started (PID: %d, GDB port: %s)\n", pid, ports.GDB)
	return nil
}

func ocdStop(manager *process.Manager) error {
	if err := manager.Kill(ocdProcessName); err != nil {
		fmt.Println("OpenOCD not running (stale or absent session)")
		return nil
	}
	fmt.Println("OpenOCD stopped")
	return nil
}

func ocdStatus(manager *process.Manager) error {
	pid, metadata, ok := manager.Session(ocdProcessName)
	if !ok {
		fmt.Println("OpenOCD is NOT running")
		return nil
	}

	state := "running"
	if !manager.IsAlive(ocdProcessName) {
		state = "NOT running (stale session)"
	}

	port := metadata
	if port == "" {
		port = "?"
	}

	fmt.Printf("OpenOCD is %s (PID: %d, GDB port: %s)\n", state, pid, port)
	return nil
}
