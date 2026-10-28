package hotrod_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	sharedContainer testcontainers.Container
	sharedAddr      string
	serverVersion   string // e.g. "Infinispan 'N/A' 16.3.0-SNAPSHOT", fetched from GET /rest/v3/server; empty if unavailable
)

// fetchServerVersion queries the server info endpoint (served on the Hot Rod port since the single-port design).
func fetchServerVersion(addr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/rest/v3/server", nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: could not fetch server version: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var info struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil || info.Version == "" {
		fmt.Fprintf(os.Stderr, "warn: could not decode server version: %v\n", err)
		return
	}
	serverVersion = info.Version
}

func isShort() bool {
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") {
			return true
		}
	}
	return false
}

func TestMain(m *testing.M) {
	if isShort() {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        serverImage(),
			ExposedPorts: []string{"11222/tcp"},
			Env: map[string]string{
				"USER": "admin",
				"PASS": "password",
			},
			WaitingFor: wait.ForLog("ISPN080001").WithStartupTimeout(120 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		cancel()
		fmt.Fprintf(os.Stderr, "start infinispan: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		cancel()
		_ = container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "get host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "11222")
	if err != nil {
		cancel()
		_ = container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "get port: %v\n", err)
		os.Exit(1)
	}
	cancel()

	sharedContainer = container
	sharedAddr = fmt.Sprintf("%s:%s", host, port.Port())

	code := m.Run()
	_ = container.Terminate(context.Background())
	os.Exit(code)
}
