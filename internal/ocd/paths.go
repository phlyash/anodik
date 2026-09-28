package ocd

import (
	"path/filepath"
	"strings"
)

func RuntimeDir(workingDirectory string) string {
	/// maybe move openocd runtime dir with logs and session file to somewhere like
	/// .local/share/niiet/openocd/run or smth like this.
	return filepath.Join(workingDirectory, "build")
}

func SessionName(iface string) string {
	return "openocd_" + sanitizeInterface(iface)
}

func LogPath(workingDirectory, iface string) string {
	return filepath.Join(RuntimeDir(workingDirectory), SessionName(iface)+".log")
}

func sanitizeInterface(iface string) string {
	iface = strings.TrimSpace(iface)
	if iface == "" {
		return "unknown"
	}

	iface = strings.ToLower(filepath.Base(iface))
	iface = strings.TrimSuffix(iface, ".cfg")

	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, iface)

	if strings.Trim(cleaned, "_") == "" {
		return "unknown"
	}
	return cleaned
}
