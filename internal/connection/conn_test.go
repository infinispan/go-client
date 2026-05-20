package connection

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"infinispan.org/go-client/internal/auth"
	"infinispan.org/go-client/internal/codec"
	"infinispan.org/go-client/internal/operation"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

func serverImage() string {
	if img := os.Getenv("INFINISPAN_SERVER_IMAGE"); img != "" {
		return img
	}
	return "quay.io/infinispan/server:latest"
}

func startInfinispan(t *testing.T) (string, testcontainers.Container) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        serverImage(),
		ExposedPorts: []string{"11222/tcp"},
		Env: map[string]string{
			"USER": "admin",
			"PASS": "password",
		},
		WaitingFor: wait.ForLog("ISPN080001").WithStartupTimeout(120 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start infinispan: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	port, err := container.MappedPort(ctx, "11222")
	if err != nil {
		t.Fatalf("get port: %v", err)
	}
	return fmt.Sprintf("%s:%s", host, port.Port()), container
}

func createCache(t *testing.T, container testcontainers.Container, name string) {
	t.Helper()
	ctx := context.Background()
	cmd := fmt.Sprintf("create cache --template=org.infinispan.DIST_SYNC %s", name)
	_, output, err := container.Exec(ctx, []string{
		"bash", "-c",
		fmt.Sprintf("echo '%s' | /opt/infinispan/bin/cli.sh -c http://admin:password@localhost:11222", cmd),
	})
	if err != nil {
		t.Fatalf("exec create cache: %v", err)
	}
	body, _ := io.ReadAll(output)
	t.Logf("create cache output: %s", body)
}

func dialAndAuth(t *testing.T, addr string) *Conn {
	t.Helper()
	conn, err := Dial(context.Background(), addr, func(update *codec.TopologyUpdate, cacheName string) {}, codec.IntelligenceHashDistAware, 0, testLogger)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.Authenticate(ctx, auth.NewScramSHA256("admin", "password")); err != nil {
		conn.Close()
		t.Fatalf("auth: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestPingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	addr, _ := startInfinispan(t)
	conn, err := Dial(context.Background(), addr, nil, codec.IntelligenceHashDistAware, 0, testLogger)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := conn.Execute(ctx, &operation.PingOp{})
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	if result != true {
		t.Errorf("ping result = %v, want true", result)
	}
}

func TestAuthMechListIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	addr, _ := startInfinispan(t)
	conn, err := Dial(context.Background(), addr, nil, codec.IntelligenceHashDistAware, 0, testLogger)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mechResult, err := conn.Execute(ctx, &operation.AuthMechListOp{})
	if err != nil {
		t.Fatalf("auth mech list: %v", err)
	}
	mechs := mechResult.([]string)
	t.Logf("available mechanisms: %v", mechs)
	if len(mechs) == 0 {
		t.Fatal("expected at least one mechanism")
	}
}

func TestScramSHA256Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	addr, _ := startInfinispan(t)
	conn := dialAndAuth(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := conn.Execute(ctx, &operation.PingOp{})
	if err != nil {
		t.Fatalf("ping after auth: %v", err)
	}
	if result != true {
		t.Errorf("ping result = %v, want true", result)
	}
	t.Log("SCRAM-SHA-256 auth and post-auth ping succeeded")
}

func TestPutGetIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	addr, container := startInfinispan(t)
	createCache(t, container, "test")
	conn := dialAndAuth(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := conn.Execute(ctx, &operation.PutOp{
		Cache: "test",
		Key:   []byte("hello"),
		Value: []byte("world"),
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	result, err := conn.Execute(ctx, &operation.GetOp{
		Cache: "test",
		Key:   []byte("hello"),
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp := result.(*operation.GetResponse)
	if !resp.Found {
		t.Fatal("expected key to be found")
	}
	if string(resp.Value) != "world" {
		t.Errorf("value = %q, want %q", string(resp.Value), "world")
	}

	result, err = conn.Execute(ctx, &operation.GetOp{
		Cache: "test",
		Key:   []byte("nonexistent"),
	})
	if err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
	resp = result.(*operation.GetResponse)
	if resp.Found {
		t.Error("expected key not found")
	}
}
