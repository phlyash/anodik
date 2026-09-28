package ocd

import (
	"net"
	"strconv"
	"testing"
)

func occupy(t *testing.T, port int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Skipf("cannot occupy port %d: %v", port, err)
	}
	t.Cleanup(func() { ln.Close() })
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func TestExplicitBusyPortFails(t *testing.T) {
	port := freePort(t)
	occupy(t, port)

	if _, err := ResolvePorts(strconv.Itoa(port), true); err == nil {
		t.Fatalf("explicit busy port %d must fail, got no error", port)
	}
}

func TestExplicitFreePortIsHonoredExactly(t *testing.T) {
	port := freePort(t)

	ports, err := ResolvePorts(strconv.Itoa(port), true)
	if err != nil {
		t.Fatal(err)
	}
	if ports.GDB != strconv.Itoa(port) {
		t.Errorf("GDB = %s, want %s", ports.GDB, strconv.Itoa(port))
	}
}

func TestExplicitPortNotShiftedByTelnetCollision(t *testing.T) {
	occupy(t, DefaultTelnetPort)
	port := freePort(t)

	ports, err := ResolvePorts(strconv.Itoa(port), true)
	if err != nil {
		t.Fatal(err)
	}
	if ports.GDB != strconv.Itoa(port) {
		t.Errorf("GDB = %s, want %s: a telnet collision must not move the requested gdb port", ports.GDB, strconv.Itoa(port))
	}
	if ports.Telnet == strconv.Itoa(DefaultTelnetPort) {
		t.Error("telnet port should have moved off the occupied default")
	}
}

func TestDefaultPortAllocatesAround(t *testing.T) {
	port := freePort(t)
	occupy(t, port)

	ports, err := ResolvePorts(strconv.Itoa(port), false)
	if err != nil {
		t.Fatal(err)
	}
	if ports.GDB == strconv.Itoa(port) {
		t.Errorf("GDB = %s, want a shifted port: a non-explicit request must allocate around a busy port", ports.GDB)
	}
}

func TestInvalidPortFails(t *testing.T) {
	for _, in := range []string{"", "abc", "0", "-1", "70000"} {
		if _, err := ResolvePorts(in, false); err == nil {
			t.Errorf("ResolvePorts(%q) must fail", in)
		}
	}
}
