package runner

import (
	"os/exec"
	"testing"
)

func TestRealRunnerExecutesCommand(t *testing.T) {
	r := RealRunner{}
	cmd := exec.Command("echo", "hello")
	if err := r.Run(cmd); err != nil {
		t.Fatalf("expected echo to succeed: %v", err)
	}
}

func TestRealRunnerReturnsError(t *testing.T) {
	r := RealRunner{}
	cmd := exec.Command("false")
	if err := r.Run(cmd); err == nil {
		t.Fatal("expected 'false' command to return error")
	}
}
