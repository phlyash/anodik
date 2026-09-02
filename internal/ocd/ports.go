package ocd

import (
	"net"
	"strconv"
)

const (
	DefaultTelnetPort = 4444
	DefaultTCLPort    = 6666
	portScanSpan      = 64
)

// Ports is the resolved gdb/telnet/tcl port triple OpenOCD will listen on.
type Ports struct {
	GDB    string
	Telnet string
	TCL    string
}

// AllocatePorts slides the gdb/telnet/tcl triple by a shared offset onto
// free ports, keeping the familiar port layout (telnet = gdb_base_offset +
// 4444, tcl = gdb_base_offset + 6666) even when the preferred gdb port is
// busy.
func AllocatePorts(gdbPort string) Ports {
	gdbBase, err := strconv.Atoi(gdbPort)
	if err != nil {
		gdbBase = 0
	}

	bases := []int{gdbBase, DefaultTelnetPort, DefaultTCLPort}
	offset := findFreeOffset(bases, portScanSpan)

	return Ports{
		GDB:    strconv.Itoa(gdbBase + offset),
		Telnet: strconv.Itoa(DefaultTelnetPort + offset),
		TCL:    strconv.Itoa(DefaultTCLPort + offset),
	}
}

func findFreeOffset(bases []int, span int) int {
	for offset := 0; offset < span; offset++ {
		allFree := true
		for _, base := range bases {
			if !portFree(base + offset) {
				allFree = false
				break
			}
		}
		if allFree {
			return offset
		}
	}
	return 0
}

func portFree(port int) bool {
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
