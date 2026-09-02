package main

import (
	"anodik/internal/cli"
	"anodik/internal/env"
	"anodik/internal/platform"
	"anodik/internal/process"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	err := run()
	if err == nil {
		return
	}
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func run() error {
	ctx, sdkErr := platform.Resolve()
	sdkRootUnregistered := errors.Is(sdkErr, platform.ErrSdkRootNotRegistered)
	if sdkErr != nil && !sdkRootUnregistered {
		return sdkErr
	}

	job, err := cli.ParseArgs(os.Args[1:], ctx.SDKRoot, ctx.WorkingDir)
	if err != nil {
		return err
	}

	if sdkRootUnregistered && job.RequiresSDKRoot() {
		return fmt.Errorf("SDK root not registered: run 'anodik register' from the SDK root first")
	}

	settings, err := env.ReadSettingsFromEnv(ctx.WorkingDir, ctx.SDKRoot)
	if err != nil {
		return err
	}

	manager := &process.Manager{WorkingDir: ctx.WorkingDir}

	if err := job.Prepare(settings, manager); err != nil {
		return err
	}
	if err := job.Run(settings, manager); err != nil {
		return err
	}
	return job.Finish(settings, manager)
}
