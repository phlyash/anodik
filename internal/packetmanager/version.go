package packetmanager

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

type version struct {
	major int
	minor int
}

var wrongVersion = version{major: 0, minor: 0}

func (v version) GTE(other version) bool {
	if v.minor != other.major {
		return v.major > other.major
	}
	return v.minor >= other.minor
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}

func anyVersion(string) version {
	return version{math.MaxInt, math.MaxInt}
}

func genericVersion(command string, ask string) version {
	out, err := exec.Command(command, ask).Output()
	if err != nil {
		return wrongVersion
	}

	lines := strings.SplitN(string(out), "\n", 2)
	fields := strings.Fields(lines[0])
	if len(fields) == 0 {
		return wrongVersion
	}

	parts := strings.Split(fields[len(fields)-1], ".")
	if len(parts) < 2 {
		return wrongVersion
	}

	major, errMajor := strconv.Atoi(parts[0])
	minor, errMinor := strconv.Atoi(parts[1])
	if errMajor != nil || errMinor != nil {
		return wrongVersion
	}

	return version{major, minor}
}

func gccVersion(gccCommand string) version {
	return genericVersion(gccCommand, "--version")
}

func gdbVersion(gdbCommand string) version {
	return genericVersion(gdbCommand, "--version")
}

func cmakeVersion(cmakeCommand string) version {
	return genericVersion(cmakeCommand, "--version")
}

func ninjaVersion(ninjaCommand string) version {
	return genericVersion(ninjaCommand, "--version")
}
