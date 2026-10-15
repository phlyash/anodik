package packetmanager

import (
	"anodik/internal/env"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
)

func Install(packet *Packet, target_path string) error {
	url := packet.Url
	tmpArchive, err := Download(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer os.Remove(tmpArchive)

	fmt.Println("checking sha256 sum")

	if err := CheckSHA(packet, tmpArchive); err != nil {
		return err
	}

	fmt.Println("extracting into", target_path)

	if err := extractInto(tmpArchive, target_path, packet.Archiv); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	fmt.Println("done")
	return nil
}

func Download(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	client, err := httpClient("")
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "riscv-sdk-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	fmt.Println("downloading", url)

	pr := &progressReader{r: resp.Body, total: resp.ContentLength}
	if _, err := io.Copy(tmp, pr); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	fmt.Println()

	return tmp.Name(), nil
}

func Uninstall(packet Packet, installDir string, assumeYes bool) (removed bool, err error) {
	packetDir := packet.Directory(installDir)

	if _, err := os.Lstat(packetDir); err != nil {
		return false, fmt.Errorf("Packet %s is not installed in %s", packet.Type, installDir)
	}

	if !assumeYes {
		answer, err := Ask(
			fmt.Sprintf("Do you want to delete directory %s and all its contents", packetDir),
			Yes|No,
		)
		if err != nil {
			return false, err
		}
		if answer == No {
			return false, nil
		}
	}

	if err := os.RemoveAll(packetDir); err != nil {
		return false, fmt.Errorf("remove %s: %w", packetDir, err)
	}

	settings, err := env.ReadToolchainEnv()
	if err != nil {
		return true, fmt.Errorf("cannot clear .env settings: %w", err)
	}

	if clean, ok := CleanEnvTable[packet.Type]; ok {
		settings = clean(settings)
	}

	return true, env.Save(settings)
}

func CheckSHA(packet *Packet, archivePath string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	expected := packet.Sha256

	h := sha256.New()
	if _, err := io.Copy(h, archive); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))

	if actual != expected {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}
