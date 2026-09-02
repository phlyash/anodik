package packetmanager

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
)

/// json scheme:
/// [
///   {
///     "os": "windows|darwin|linux",
///     "arch": "x86_64|aarch64"
///     "type": "anodik|compiler|openocd|tools|cmake|ninja",
///     "version": "string",
///     "archiv": "zip|tar.gz",
///     "url": "string",
///     "sha256": "string"
///   }
/// ]

type Packet struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Archiv  string `json:"archiv"`
	Url     string `json:"url"`
	Sha256  string `json:"sha256"`
}

var packetArchToGoArch = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

type PacketType = string

const (
	anodik   PacketType = "anodik"
	cmake    PacketType = "cmake"
	compiler PacketType = "compiler"
	ninja    PacketType = "ninja"
	openocd  PacketType = "openocd"
	tools    PacketType = "tools"
)

func IsValidName(name PacketType) bool {
	switch name {
	case
		anodik,
		cmake,
		compiler,
		ninja,
		openocd,
		tools:
		return true
	}
	return false
}

func ParsePacketList(data []byte) ([]Packet, error) {
	var packets []Packet
	if err := json.Unmarshal(data, &packets); err != nil {
		return nil, err
	}

	const arch = runtime.GOARCH
	const os = runtime.GOOS

	filtered := make([]Packet, len(packets))
	for _, p := range packets {
		if packetArchToGoArch[p.Arch] == arch && p.OS == os {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

func (p Packet) String() string {
	return fmt.Sprintf("%s - %s", p.Type, p.Version)
}

func (p Packet) Directory(installDir string) string {
	return filepath.Join(installDir, p.Type)
}
