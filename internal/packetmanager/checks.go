package packetmanager

import (
	"fmt"
	"os/exec"
	"strings"
)

func gccCheck(gccCommand string) error {
	out, err := exec.Command(gccCommand, "--target-help").CombinedOutput()

	if err != nil && len(out) == 0 {
		return fmt.Errorf("[WARN] tool changed API: %w", err)
	}

	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "-mfix-cloudbear-0001" {
			return nil
		}
	}

	return fmt.Errorf("[WARN] riscv gcc doesnt have -mfix-cloudbear-0001 fix for k1921vg015")
}

func anyCheck(string) error {
	return nil
}
