package ocd

import "fmt"

func PrintInfo(cfg Config, ports *Ports) {
	fmt.Printf("  SoC       : %s\n", cfg.SoC)
	fmt.Printf("  Interface : %s\n", cfg.Interface)
	fmt.Printf("  Target cfg: %s\n", cfg.TargetCfg)
	fmt.Printf("  Iface cfg : %s\n", cfg.InterfaceCfg)
	gdbPort := cfg.GDBPort
	if ports != nil {
		gdbPort = ports.GDB
	}
	fmt.Printf("  GDB port  : %s\n", gdbPort)

	if ports != nil {
		fmt.Printf("  Telnet/tcl: %s / %s\n", ports.Telnet, ports.TCL)
	}
	if cfg.HasELFPath {
		fmt.Printf("  ELF       : %s\n", cfg.ELFPath)
	}
}

type Info struct {
	Example      string `json:"example,omitempty"`
	SoC          string `json:"soc"`
	Interface    string `json:"interface"`
	TargetCfg    string `json:"targetCfg"`
	InterfaceCfg string `json:"interfaceCfg"`
	GDBPort      string `json:"gdbPort"`
	TelnetPort   string `json:"telnetPort,omitempty"`
	TCLPort      string `json:"tclPort,omitempty"`
	ELF          string `json:"elf,omitempty"`
}

func NewInfo(example string, cfg Config, ports *Ports) Info {
	info := Info{
		Example:      example,
		SoC:          cfg.SoC,
		Interface:    cfg.Interface,
		TargetCfg:    cfg.TargetCfg,
		InterfaceCfg: cfg.InterfaceCfg,
		GDBPort:      cfg.GDBPort,
	}
	if ports != nil {
		info.GDBPort = ports.GDB
		info.TelnetPort = ports.Telnet
		info.TCLPort = ports.TCL
	}
	if cfg.HasELFPath {
		info.ELF = cfg.ELFPath
	}
	return info
}
