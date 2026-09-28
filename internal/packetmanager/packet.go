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
	archiveZip   = "zip"
	archiveTarGz = "tar.gz"
)

const (
	anodik   PacketType = "anodik"
	cmake    PacketType = "cmake"
	compiler PacketType = "compiler"
	ninja    PacketType = "ninja"
	openocd  PacketType = "openocd"
)

func IsValidName(name PacketType) bool {
	switch name {
	case
		anodik,
		cmake,
		compiler,
		ninja,
		openocd:
		return true
	}
	return false
}

func ParsePacketList(data []byte) ([]Packet, error) {
	var packets []Packet
	if err := json.Unmarshal(data, &packets); err != nil {
		return nil, err
	}
	return packets, nil
}

func preferredArchive() string {
	if runtime.GOOS == "windows" {
		return archiveZip
	}
	return archiveTarGz
}

func supportsHost(p Packet) bool {
	return p.OS == runtime.GOOS && packetArchToGoArch[p.Arch] == runtime.GOARCH
}

func FilterPackets(packets []Packet) []Packet {
	preferred := preferredArchive()

	order := make([]string, 0, len(packets))
	chosen := make(map[string]Packet, len(packets))

	for _, p := range packets {
		if !supportsHost(p) {
			continue
		}

		key := p.Type + "@" + p.Version

		current, seen := chosen[key]
		if !seen {
			order = append(order, key)
			chosen[key] = p
			continue
		}

		if current.Archiv != preferred && p.Archiv == preferred {
			chosen[key] = p
		}
	}

	filtered := make([]Packet, 0, len(order))
	for _, key := range order {
		filtered = append(filtered, chosen[key])
	}

	return filtered
}

func (p Packet) String() string {
	return fmt.Sprintf("%s - %s", p.Type, p.Version)
}

func (p Packet) Directory(installDir string) string {
	return filepath.Join(installDir, p.Type)
}
