package ocd

import "fmt"

func PrintInfo(cfg Config, ports *Ports) {
	fmt.Printf("  SoC       : %s\n", cfg.SoC)
	fmt.Printf("  Interface : %s\n", cfg.Interface)
	fmt.Printf("  Target cfg: %s\n", cfg.TargetCfg)
	fmt.Printf("  Iface cfg : %s\n", cfg.InterfaceCfg)
	fmt.Printf("  GDB port  : %s\n", cfg.GDBPort)

	if ports != nil {
		fmt.Printf("  Telnet/tcl: %s / %s\n", ports.Telnet, ports.TCL)
	}
	if cfg.HasELFPath {
		fmt.Printf("  ELF       : %s\n", cfg.ELFPath)
	}
}
