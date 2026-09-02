package ocd

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

const tclTerminator = '\x1a'

func QueryTCL(addr, command string, timeout time.Duration) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return "", fmt.Errorf("connect to openocd tcl port: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return "", fmt.Errorf("set tcl connection deadline: %w", err)
	}

	if _, err := conn.Write([]byte(command + string(tclTerminator))); err != nil {
		return "", fmt.Errorf("send tcl command: %w", err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString(tclTerminator)
	if err != nil {
		return "", fmt.Errorf("read tcl response: %w", err)
	}

	response = strings.TrimSuffix(response, string(tclTerminator))
	return strings.TrimSpace(response), nil
}

func TargetExamined(response string) bool {
	lower := strings.ToLower(response)
	if strings.Contains(lower, "unknown") {
		return false
	}
	if strings.Contains(lower, "error") {
		return false
	}
	if strings.Contains(lower, "not examined") {
		return false
	}
	return true
}

func TargetNames(addr string, timeout time.Duration) ([]string, error) {
	resp, err := QueryTCL(addr, "target names", timeout)
	if err != nil {
		return nil, err
	}
	return strings.Fields(resp), nil
}

func ResolveTargetName(addr, soc string, timeout time.Duration) (string, error) {
	names, err := TargetNames(addr, timeout)
	if err != nil {
		return "", err
	}

	prefix := strings.ToUpper(soc) + ".cpu"
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			return name, nil
		}
	}

	return "", fmt.Errorf("no target found for soc %q among %v", soc, names)
}

func CheckTargetExamined(addr, targetName string, timeout time.Duration) (examined bool, detail string, err error) {
	stateResp, err := QueryTCL(addr, fmt.Sprintf(`capture "%s curstate"`, targetName), timeout)
	fmt.Printf("[TCL CHECK STATE] %s\n", stateResp)
	if err != nil {
		return false, "", err
	}

	return TargetExamined(stateResp), stateResp, nil
}
