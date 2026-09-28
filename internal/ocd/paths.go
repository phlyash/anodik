package ocd

import (
	"anodik/internal/platform"
	"path/filepath"
	"strings"
)

func RuntimeDir() string {
	return filepath.Join(platform.RuntimeBaseDir(), "ocd")
}

func SessionName(iface string) string {
	return "openocd_" + sanitizeInterface(iface)
}

func SessionPath(iface string) string {
	return filepath.Join(RuntimeDir(), SessionName(iface)+".session")
}

func LogPath(iface string) string {
	return filepath.Join(RuntimeDir(), SessionName(iface)+".log")
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
