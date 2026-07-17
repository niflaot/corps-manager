package e2e

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

type harness struct {
	command *exec.Cmd
	output  bytes.Buffer
	baseURL string
	client  *http.Client
}

func startHarness(t *testing.T) *harness {
	t.Helper()
	port := freePort(t)
	runtime := &harness{
		baseURL: "http://127.0.0.1:" + strconv.Itoa(port),
		client:  &http.Client{Timeout: time.Second},
	}
	runtime.command = exec.Command(testBinary, "serve")
	runtime.command.Env = append(os.Environ(),
		"DISCORD_BOT_ENVIRONMENT=test",
		"DISCORD_BOT_HOST=127.0.0.1",
		"DISCORD_BOT_PORT="+strconv.Itoa(port),
		"DISCORD_BOT_TOKEN=",
		"DISCORD_BOT_REDIS_ADDR=127.0.0.1:1",
		"DISCORD_BOT_REDIS_DIAL_TIMEOUT=20ms",
		"DISCORD_BOT_REDIS_HEALTH_TIMEOUT=20ms",
		"DISCORD_BOT_POSTGRES_HOST=127.0.0.1",
		"DISCORD_BOT_POSTGRES_PORT=1",
		"DISCORD_BOT_POSTGRES_CONNECT_TIMEOUT=20ms",
		"DISCORD_BOT_POSTGRES_HEALTH_TIMEOUT=20ms",
	)
	runtime.command.Stdout = &runtime.output
	runtime.command.Stderr = &runtime.output
	if err := runtime.command.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := runtime.client.Get(runtime.baseURL + "/openapi.json")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				t.Cleanup(runtime.close)
				return runtime
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	runtime.close()
	t.Fatalf("process did not become ready: %s", runtime.output.String())
	return nil
}

func (runtime *harness) close() {
	if transport, ok := runtime.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	if runtime.command == nil || runtime.command.Process == nil {
		return
	}
	_ = runtime.command.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_ = runtime.command.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = runtime.command.Process.Kill()
		<-done
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close port allocator: %v", err)
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	if port == 0 {
		t.Fatal(fmt.Errorf("allocated invalid port"))
	}
	return port
}
