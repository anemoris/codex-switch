package runner

import "os/exec"

type Runner interface {
	Run(cmd *exec.Cmd) error
}

type RealRunner struct{}

func (RealRunner) Run(cmd *exec.Cmd) error {
	return cmd.Run()
}
