package jobs

import (
	"anodik/internal/env"
	"anodik/internal/process"
)

type Job interface {
	Prepare(*env.Settings, *process.Manager) error
	Run(*env.Settings, *process.Manager) error
	Finish(*env.Settings, *process.Manager) error
	RequiresSDKRoot() bool
}

type Stoppable interface {
	Job
	Stop(*env.Settings, *process.Manager) error
}

type SDKRootRequirement struct{}

func (SDKRootRequirement) RequiresSDKRoot() bool {
	return true
}

type NoSDKRootRequirement struct{}

func (NoSDKRootRequirement) RequiresSDKRoot() bool {
	return false
}
