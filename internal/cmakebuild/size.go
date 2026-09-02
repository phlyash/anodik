package cmakebuild

import (
	"anodik/internal/env"
	"anodik/internal/toolchain"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Size prints a berkeley-format size report for the ELF built at
// buildDir/<elfName>. Returns false if no ELF was found there.
func Size(s *env.Settings, buildDir, elfName string) (found bool, err error) {
	elfPath := filepath.Join(buildDir, elfName)
	if _, statErr := os.Stat(elfPath); statErr != nil {
		return false, nil
	}

	fmt.Printf("=== %s ===\n", elfPath)

	cmd := exec.Command(toolchain.Size(s), "--format=berkeley", elfPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return true, fmt.Errorf("run size: %w", err)
	}

	fmt.Println()
	return true, nil
}
