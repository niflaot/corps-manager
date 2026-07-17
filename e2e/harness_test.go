package e2e

import (
	"os"
	"os/exec"
	"testing"
)

type processResult struct {
	output string
	err    error
}

func runHarness(t *testing.T, args []string, environment ...string) processResult {
	t.Helper()
	command := exec.Command(testBinary, args...)
	command.Env = append(os.Environ(), environment...)
	output, err := command.CombinedOutput()
	return processResult{output: string(output), err: err}
}
