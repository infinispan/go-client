package hotrod_test

import (
	"context"
	"fmt"
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
)

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
	defer cancel()

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
		fmt.Fprintf(os.Stderr, "start infinispan: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "get host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "11222")
	if err != nil {
		container.Terminate(context.Background())
		fmt.Fprintf(os.Stderr, "get port: %v\n", err)
		os.Exit(1)
	}

	sharedContainer = container
	sharedAddr = fmt.Sprintf("%s:%s", host, port.Port())

	code := m.Run()
	container.Terminate(context.Background())
	os.Exit(code)
}
