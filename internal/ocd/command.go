package ocd

import (
	"anodik/internal/env"
	"anodik/internal/toolchain"
	"strings"
)

// Command builds the full openocd argv for cfg/ports. tail is appended
// verbatim after the interface/target config files and any extra args from
// settings — used to inject a one-off -c command such as flashing or
// service erase.
func Command(s *env.Settings, sdkRoot string, cfg Config, ports Ports, tail []string) []string {
	args := []string{toolchain.OpenOCD(s), "-s", HelpersDir(sdkRoot)}

	args = append(args, "-c", "gdb_port "+ports.GDB)
	args = append(args, "-c", "telnet_port "+ports.Telnet)
	args = append(args, "-c", "tcl_port "+ports.TCL)

	args = append(args, "-f", cfg.InterfaceCfg, "-f", cfg.TargetCfg)

	if cfg.ExtraArgs != "" {
		args = append(args, splitShellArgs(cfg.ExtraArgs)...)
	}

	return append(args, tail...)
}

// splitShellArgs is a minimal shlex.split equivalent: splits on whitespace,
// honoring single and double quotes. Extra OpenOCD args are simple flag
// lists in practice, so this covers the real-world cases without pulling in
// a shell-parsing dependency.
func splitShellArgs(s string) []string {
	var args []string
	var current strings.Builder
	var quote rune

	flush := func() {
		if current.Len() > 0 {
			args = append(args, current.String())
			current.Reset()
		}
	}

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()

	return args
}
