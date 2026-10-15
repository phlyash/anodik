package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"fmt"
)

type OCDJob struct {
	SDKRootRequirement
	ExampleOptions
	SoCOptions
	DebugOptions
	ProcessOptions
	JSONOutput
}

func (j *OCDJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *OCDJob) Run(s *env.Settings, manager *process.Manager) error {
	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, j.GDBPort)
	if err != nil {
		return err
	}

	if j.JSON && !j.Status {
		return fmt.Errorf("--json is only supported with --status")
	}
	if j.StopProcess {
		return ocdStop(manager, cfg.Interface)
	}
	if j.Status {
		if j.JSON {
			status, _ := ocdStatusOf(manager, cfg.Interface)
			return printJSON(status)
		}
		return ocdStatus(manager, cfg.Interface)
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
	args := ocd.Command(s, s.SDKRoot, cfg, ports, nil)

	if j.RunInBackground {
		return startOCDBackground(s, manager, cfg, ports, args)
	}
	return startOCDForeground(manager, cfg, ports, args)
}

func (j *OCDJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *OCDJob) Stop(s *env.Settings, manager *process.Manager) error {
	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, j.GDBPort)
	if err != nil {
		return err
	}
	return ocdStop(manager, cfg.Interface)
}

func (j *OCDJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "start (or stop/query) a standalone OpenOCD server",
		Usage:   "[example] [flags]",
	}.Merge(j.ExampleOptions.Help(), j.SoCOptions.Help(), j.DebugOptions.Help(), j.ProcessOptions.Help(), HelpInfo{
		Flags: []FlagHelp{{Name: "json", Desc: "with --status: print machine-readable JSON"}},
	})
}

func startOCDForeground(manager *process.Manager, cfg ocd.Config, ports ocd.Ports, args []string) error {
	fmt.Println("Starting OpenOCD...")
	ocd.PrintInfo(cfg, &ports)

	return manager.RunForeground(args[0], args[1:])
}

func startOCDBackground(s *env.Settings, manager *process.Manager, cfg ocd.Config, ports ocd.Ports, args []string) error {
	fmt.Println("Starting OpenOCD in background...")
	ocd.PrintInfo(cfg, &ports)

	session := ocd.SessionPath(cfg.Interface)
	logPath := ocd.LogPath(cfg.Interface)

	pid, alive, err := manager.RunDetachedLogged(session, args[0], args[1:], logPath, ports.GDB)
	if err != nil {
		return err
	}

	if !alive {
		fmt.Println("ERROR: OpenOCD failed to start. See log:", logPath)
		return fmt.Errorf("openocd did not stay running")
	}

	fmt.Printf("OpenOCD started (PID: %d, interface: %s, GDB port: %s)\n", pid, cfg.Interface, ports.GDB)
	fmt.Printf("log: %s\n", logPath)
	return nil
}

func ocdStop(manager *process.Manager, iface string) error {
	if err := manager.Kill(ocd.SessionPath(iface)); err != nil {
		fmt.Printf("OpenOCD not running on interface %s (stale or absent session)\n", iface)
		return nil
	}
	fmt.Printf("OpenOCD on interface %s stopped\n", iface)
	return nil
}

type OCDStatus struct {
	Interface string `json:"interface"`
	Running   bool   `json:"running"`
	Stale   bool   `json:"stale"`
	PID     int    `json:"pid,omitempty"`
	GDBPort string `json:"gdbPort,omitempty"`
}

func ocdStatusOf(manager *process.Manager, iface string) (OCDStatus, bool) {
	session := ocd.SessionPath(iface)
	status := OCDStatus{Interface: iface}

	pid, metadata, ok := manager.Session(session)
	if !ok {
		return status, false
	}
	status.PID = pid
	status.GDBPort = metadata
	status.Running = manager.IsAlive(session)
	status.Stale = !status.Running
	return status, true
}

func ocdStatus(manager *process.Manager, iface string) error {
	status, found := ocdStatusOf(manager, iface)
	if !found {
		fmt.Printf("OpenOCD is NOT running on interface %s\n", iface)
		return nil
	}

	state := "running"
	if status.Stale {
		state = "NOT running (stale session)"
	}

	port := status.GDBPort
	if port == "" {
		port = "?"
	}

	fmt.Printf("OpenOCD on interface %s is %s (PID: %d, GDB port: %s)\n", iface, state, status.PID, port)
	return nil
}
