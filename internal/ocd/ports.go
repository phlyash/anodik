package ocd

import (
	"fmt"
	"net"
	"strconv"
)

const (
	DefaultTelnetPort = 4444
	DefaultTCLPort    = 6666
	portScanSpan      = 64
)

type Ports struct {
	GDB    string
	Telnet string
	TCL    string
}

func ResolvePorts(gdbPort string, explicit bool) (Ports, error) {
	base, err := strconv.Atoi(gdbPort)
	if err != nil || base < 1 || base > 65535 {
		return Ports{}, fmt.Errorf("invalid gdb port %q", gdbPort)
	}

	if explicit {
		if !portFree(base) {
			return Ports{}, fmt.Errorf("gdb port %d is busy: stop whatever is using it, or ask for a different port", base)
		}

		offset, ok := findFreeOffset([]int{DefaultTelnetPort, DefaultTCLPort}, portScanSpan)
		if !ok {
			return Ports{}, fmt.Errorf("no free telnet/tcl port pair near %d/%d", DefaultTelnetPort, DefaultTCLPort)
		}

		return Ports{
			GDB:    strconv.Itoa(base),
			Telnet: strconv.Itoa(DefaultTelnetPort + offset),
			TCL:    strconv.Itoa(DefaultTCLPort + offset),
		}, nil
	}

	offset, ok := findFreeOffset([]int{base, DefaultTelnetPort, DefaultTCLPort}, portScanSpan)
	if !ok {
		return Ports{}, fmt.Errorf("no free gdb/telnet/tcl port triple near %d/%d/%d", base, DefaultTelnetPort, DefaultTCLPort)
	}

	return Ports{
		GDB:    strconv.Itoa(base + offset),
		Telnet: strconv.Itoa(DefaultTelnetPort + offset),
		TCL:    strconv.Itoa(DefaultTCLPort + offset),
	}, nil
}

func findFreeOffset(bases []int, span int) (int, bool) {
	for offset := 0; offset < span; offset++ {
		allFree := true
		for _, base := range bases {
			if !portFree(base + offset) {
				allFree = false
				break
			}
		}
		if allFree {
			return offset, true
		}
	}
	return 0, false
}

func portFree(port int) bool {
	if port < 1 || port > 65535 {
		return false
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
