package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"go.uber.org/goleak"
)

var testBinary string

// TestMain builds the real process once and verifies harness cleanup.
func TestMain(main *testing.M) {
	temporary, err := os.MkdirTemp("", "discord-bot-e2e-")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testBinary = filepath.Join(temporary, "discord-bot")
	if runtime.GOOS == "windows" {
		testBinary += ".exe"
	}
	command := exec.Command("go", "build", "-o", testBinary, "../cmd")
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "build e2e binary: %v\n%s", buildErr, output)
		_ = os.RemoveAll(temporary)
		os.Exit(1)
	}
	code := main.Run()
	if leakErr := goleak.Find(); leakErr != nil {
		_, _ = fmt.Fprintln(os.Stderr, leakErr)
		code = 1
	}
	_ = os.RemoveAll(temporary)
	os.Exit(code)
}
