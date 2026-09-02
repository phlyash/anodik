package jobs

import (
	"anodik/internal/env"
	"anodik/internal/platform"
	"anodik/internal/process"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type RegisterJob struct {
	NoSDKRootRequirement
	target string
}

var ErrNotSDKRoot = errors.New("this directory is not a SDK root. ./tools/cmake/RiscvSdk not found")

func (j *RegisterJob) Prepare(settings *env.Settings, _ *process.Manager) error {
	sdkRegisterAnchor := filepath.Join(settings.WorkingDir, "tools", "cmake", "RiscvSdk")

	info, err := os.Stat(sdkRegisterAnchor)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotSDKRoot
		}
		return err
	}

	if !info.IsDir() {
		return ErrNotSDKRoot
	}

	j.target = sdkRegisterAnchor
	fmt.Printf("[SDK ROOT] %s\n", settings.WorkingDir)
	fmt.Printf("[SDK TARGET] %s\n", sdkRegisterAnchor)

	return nil
}

func (j *RegisterJob) Run(*env.Settings, *process.Manager) error {
	if err := platform.RegisterSdk(j.target); err != nil {
		return fmt.Errorf("cannot register sdk: %w", err)
	}
	return nil
}

func (j *RegisterJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *RegisterJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "register the current directory as the SDK root",
	}
}
