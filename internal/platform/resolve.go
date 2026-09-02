package platform

import (
	"fmt"
	"os"
)

type Context struct {
	WorkingDir string
	SDKRoot    string
}

func Resolve() (Context, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return Context{}, fmt.Errorf("resolve working directory: %w", err)
	}

	sdkRoot, err := ResolveSdkRoot()
	if err != nil {
		return Context{WorkingDir: workingDir}, err
	}

	return Context{WorkingDir: workingDir, SDKRoot: sdkRoot}, nil
}
