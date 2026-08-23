package hotrod_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"infinispan.org/go-client/hotrod"
)

func writePEM(t *testing.T, path, blockType string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		t.Fatal(err)
	}
}

type tlsCerts struct {
	dir            string
	caFile         string
	serverCertFile string
	serverKeyFile  string
	clientCertFile string
	clientKeyFile  string
	serverP12      string
	clientP12      string
}

func generateTLSCerts(t *testing.T, clientCN ...string) *tlsCerts {
	t.Helper()

	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available, skipping TLS integration test")
	}

	dir := t.TempDir()
	c := &tlsCerts{dir: dir}

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	c.caFile = filepath.Join(dir, "ca.pem")
	writePEM(t, c.caFile, "CERTIFICATE", caDER)

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.ParseIP("::1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	c.serverCertFile = filepath.Join(dir, "server.pem")
	writePEM(t, c.serverCertFile, "CERTIFICATE", serverDER)

	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		t.Fatal(err)
	}
	c.serverKeyFile = filepath.Join(dir, "server-key.pem")
	writePEM(t, c.serverKeyFile, "EC PRIVATE KEY", serverKeyDER)

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cn := "Test Client"
	if len(clientCN) > 0 && clientCN[0] != "" {
		cn = clientCN[0]
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	c.clientCertFile = filepath.Join(dir, "client.pem")
	writePEM(t, c.clientCertFile, "CERTIFICATE", clientDER)

	clientKeyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatal(err)
	}
	c.clientKeyFile = filepath.Join(dir, "client-key.pem")
	writePEM(t, c.clientKeyFile, "EC PRIVATE KEY", clientKeyDER)

	// Server keystore (PKCS12 with server key + cert + CA chain)
	c.serverP12 = filepath.Join(dir, "server.p12")
	runOpenSSL(t, "pkcs12", "-export",
		"-in", c.serverCertFile,
		"-inkey", c.serverKeyFile,
		"-certfile", c.caFile,
		"-out", c.serverP12,
		"-name", "server",
		"-passout", "pass:changeit",
	)

	// Client keystore (PKCS12 for mTLS — used by CLI inside container)
	c.clientP12 = filepath.Join(dir, "client.p12")
	runOpenSSL(t, "pkcs12", "-export",
		"-in", c.clientCertFile,
		"-inkey", c.clientKeyFile,
		"-certfile", c.caFile,
		"-out", c.clientP12,
		"-name", "client",
		"-passout", "pass:changeit",
	)

	return c
}

func runOpenSSL(t *testing.T, args ...string) {
	t.Helper()
	out, err := exec.Command("openssl", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("openssl %v: %v\n%s", args[:2], err, out)
	}
}

const infinispanTLSXML = `<infinispan
      xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
      xsi:schemaLocation="urn:infinispan:config:16.0 https://infinispan.org/schemas/infinispan-config-16.0.xsd
                            urn:infinispan:server:16.0 https://infinispan.org/schemas/infinispan-server-16.0.xsd"
      xmlns="urn:infinispan:config:16.0"
      xmlns:server="urn:infinispan:server:16.0">

   <cache-container name="default" statistics="true">
      <transport cluster="${infinispan.cluster.name:tls-cluster}" stack="${infinispan.cluster.stack:tcp}" node-name="${infinispan.node.name:}"/>
      <security>
         <authorization/>
      </security>
   </cache-container>

   <server xmlns="urn:infinispan:server:16.0">
      <interfaces>
         <interface name="public">
            <inet-address value="${infinispan.bind.address:0.0.0.0}"/>
         </interface>
      </interfaces>

      <socket-bindings default-interface="public" port-offset="${infinispan.socket.binding.port-offset:0}">
         <socket-binding name="default" port="${infinispan.bind.port:11222}"/>
      </socket-bindings>

      <security>
         <security-realms>
            <security-realm name="default">
               <server-identities>
                  <ssl>
                     <keystore path="server.p12"
                               password="changeit" alias="server"/>
                  </ssl>
               </server-identities>
               <properties-realm/>
            </security-realm>
         </security-realms>
      </security>

      <endpoints socket-binding="default" security-realm="default"/>
   </server>
</infinispan>
`

// mTLS XML uses a truststore and requires client cert auth.
// The trust.p12 is created inside the container by the entrypoint script
// using keytool (openssl is not available in the Infinispan image).
// EXTERNAL auth XML uses truststore-realm for certificate-based authentication
// with common-name-role-mapper to map the client cert CN to a role.
const infinispanExternalXML = `<infinispan
      xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
      xsi:schemaLocation="urn:infinispan:config:16.0 https://infinispan.org/schemas/infinispan-config-16.0.xsd
                            urn:infinispan:server:16.0 https://infinispan.org/schemas/infinispan-server-16.0.xsd"
      xmlns="urn:infinispan:config:16.0"
      xmlns:server="urn:infinispan:server:16.0">

   <cache-container name="default" statistics="true">
      <transport cluster="${infinispan.cluster.name:ext-cluster}" stack="${infinispan.cluster.stack:tcp}" node-name="${infinispan.node.name:}"/>
      <distributed-cache name="ext-test"/>
   </cache-container>

   <server xmlns="urn:infinispan:server:16.0">
      <interfaces>
         <interface name="public">
            <inet-address value="${infinispan.bind.address:0.0.0.0}"/>
         </interface>
      </interfaces>

      <socket-bindings default-interface="public" port-offset="${infinispan.socket.binding.port-offset:0}">
         <socket-binding name="default" port="${infinispan.bind.port:11222}"/>
      </socket-bindings>

      <security>
         <security-realms>
            <security-realm name="default">
               <server-identities>
                  <ssl>
                     <keystore path="server.p12"
                               password="changeit" alias="server"/>
                     <truststore path="trust.p12"
                                 password="changeit"/>
                  </ssl>
               </server-identities>
               <truststore-realm/>
            </security-realm>
         </security-realms>
      </security>

      <endpoints>
         <endpoint socket-binding="default" security-realm="default" require-ssl-client-auth="true"/>
      </endpoints>
   </server>
</infinispan>
`

// externalEntrypoint imports both the CA cert and the client cert into the
// trust store. Infinispan's <truststore-realm/> uses Elytron's
// KeyStoreBackedSecurityRealm which requires the exact client certificate
// in the trust store (direct trust), not just the signing CA.
var externalEntrypoint = "#!/bin/bash\nset -e\n" +
	"CONF=/opt/infinispan/server/conf\n" +
	"KEYTOOL=/usr/lib/jvm/default-java/bin/keytool\n\n" +
	"$KEYTOOL -importcert -noprompt -alias ca \\\n" +
	"  -file $CONF/ca.pem \\\n" +
	"  -keystore $CONF/trust.p12 \\\n" +
	"  -storetype PKCS12 -storepass changeit\n\n" +
	"$KEYTOOL -exportcert -alias client \\\n" +
	"  -keystore $CONF/client.p12 \\\n" +
	"  -storetype PKCS12 -storepass changeit \\\n" +
	"  -file $CONF/client-cert.der\n\n" +
	"$KEYTOOL -importcert -noprompt -alias client \\\n" +
	"  -file $CONF/client-cert.der \\\n" +
	"  -keystore $CONF/trust.p12 \\\n" +
	"  -storetype PKCS12 -storepass changeit\n\n" +
	"exec /opt/infinispan/bin/server.sh\n"

const infinispanMTLSXML = `<infinispan
      xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
      xsi:schemaLocation="urn:infinispan:config:16.0 https://infinispan.org/schemas/infinispan-config-16.0.xsd
                            urn:infinispan:server:16.0 https://infinispan.org/schemas/infinispan-server-16.0.xsd"
      xmlns="urn:infinispan:config:16.0"
      xmlns:server="urn:infinispan:server:16.0">

   <cache-container name="default" statistics="true">
      <transport cluster="${infinispan.cluster.name:mtls-cluster}" stack="${infinispan.cluster.stack:tcp}" node-name="${infinispan.node.name:}"/>
      <security>
         <authorization/>
      </security>
   </cache-container>

   <server xmlns="urn:infinispan:server:16.0">
      <interfaces>
         <interface name="public">
            <inet-address value="${infinispan.bind.address:0.0.0.0}"/>
         </interface>
      </interfaces>

      <socket-bindings default-interface="public" port-offset="${infinispan.socket.binding.port-offset:0}">
         <socket-binding name="default" port="${infinispan.bind.port:11222}"/>
      </socket-bindings>

      <security>
         <security-realms>
            <security-realm name="default">
               <server-identities>
                  <ssl>
                     <keystore path="server.p12"
                               password="changeit" alias="server"/>
                     <truststore path="trust.p12"
                                 password="changeit"/>
                  </ssl>
               </server-identities>
               <properties-realm/>
            </security-realm>
         </security-realms>
      </security>

      <endpoints>
         <endpoint socket-binding="default" security-realm="default" require-ssl-client-auth="true"/>
      </endpoints>
   </server>
</infinispan>
`

// mtlsEntrypoint creates the trust store using keytool (the only cert tool
// available in the Infinispan image), creates the admin user, and starts the
// server. This replaces the default entrypoint because the trust store must
// exist before the server reads the config.
const mtlsEntrypoint = `#!/bin/bash
set -e
CONF=/opt/infinispan/server/conf
KEYTOOL=/usr/lib/jvm/default-java/bin/keytool

$KEYTOOL -importcert -noprompt -alias ca \
  -file $CONF/ca.pem \
  -keystore $CONF/trust.p12 \
  -storetype PKCS12 -storepass changeit

/opt/infinispan/bin/cli.sh user create "$USER" -p "$PASS" -g admin

exec /opt/infinispan/bin/server.sh
`

func startInfinispanTLS(t *testing.T, certs *tlsCerts, mtls bool) (string, testcontainers.Container) {
	t.Helper()
	ctx := context.Background()

	xmlContent := infinispanTLSXML
	if mtls {
		xmlContent = infinispanMTLSXML
	}
	xmlFile := filepath.Join(certs.dir, "infinispan.xml")
	if err := os.WriteFile(xmlFile, []byte(xmlContent), 0644); err != nil {
		t.Fatal(err)
	}

	confDir := "/opt/infinispan/server/conf"
	files := []testcontainers.ContainerFile{
		{HostFilePath: xmlFile, ContainerFilePath: confDir + "/infinispan.xml", FileMode: 0644},
		{HostFilePath: certs.serverP12, ContainerFilePath: confDir + "/server.p12", FileMode: 0644},
	}

	req := testcontainers.ContainerRequest{
		Image:        serverImage(),
		ExposedPorts: []string{"11222/tcp"},
		Env: map[string]string{
			"USER": "admin",
			"PASS": "password",
		},
		Files:      files,
		WaitingFor: wait.ForLog("ISPN080001").WithStartupTimeout(120 * time.Second),
	}

	if mtls {
		// mTLS needs a custom entrypoint to create the trust store with keytool
		// before the server starts, and mounts the CA PEM + client keystore.
		scriptFile := filepath.Join(certs.dir, "entrypoint.sh")
		if err := os.WriteFile(scriptFile, []byte(mtlsEntrypoint), 0755); err != nil {
			t.Fatal(err)
		}
		req.Files = append(req.Files,
			testcontainers.ContainerFile{HostFilePath: certs.caFile, ContainerFilePath: confDir + "/ca.pem", FileMode: 0644},
			testcontainers.ContainerFile{HostFilePath: certs.clientP12, ContainerFilePath: confDir + "/client.p12", FileMode: 0644},
			testcontainers.ContainerFile{HostFilePath: scriptFile, ContainerFilePath: confDir + "/entrypoint.sh", FileMode: 0755},
		)
		req.Entrypoint = []string{"bash", confDir + "/entrypoint.sh"}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start infinispan TLS: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

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

func createTestCacheTLS(t *testing.T, container testcontainers.Container, name string) {
	t.Helper()
	ctx := context.Background()
	cmd := fmt.Sprintf("create cache --template=org.infinispan.DIST_SYNC %s", name)
	_, output, err := container.Exec(ctx, []string{
		"bash", "-c",
		fmt.Sprintf("echo '%s' | /opt/infinispan/bin/cli.sh --trustall -c https://admin:password@localhost:11222", cmd),
	})
	if err != nil {
		t.Fatalf("exec create cache: %v", err)
	}
	body, _ := io.ReadAll(output)
	t.Logf("create cache output: %s", body)
}

func createTestCacheMTLS(t *testing.T, container testcontainers.Container, name string) {
	t.Helper()
	ctx := context.Background()
	confDir := "/opt/infinispan/server/conf"
	cmd := fmt.Sprintf("create cache --template=org.infinispan.DIST_SYNC %s", name)
	_, output, err := container.Exec(ctx, []string{
		"bash", "-c",
		fmt.Sprintf("echo '%s' | /opt/infinispan/bin/cli.sh --trustall --keystore=%s/client.p12 --keystore-password=changeit -c https://admin:password@localhost:11222", cmd, confDir),
	})
	if err != nil {
		t.Fatalf("exec create cache: %v", err)
	}
	body, _ := io.ReadAll(output)
	t.Logf("create cache output: %s", body)
}

func TestTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	certs := generateTLSCerts(t)
	addr, container := startInfinispanTLS(t, certs, false)
	createTestCacheTLS(t, container, "tls-test")

	uri := fmt.Sprintf("hotrods://admin:password@%s?trust_store_file_name=%s&sni_host_name=localhost",
		addr, certs.caFile)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient TLS: %v", err)
	}
	defer client.Close()

	cache := client.Cache("tls-test")

	if err := cache.Put(ctx, []byte("tls-key"), []byte("tls-value")); err != nil {
		t.Fatalf("Put over TLS: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("tls-key"))
	if err != nil {
		t.Fatalf("Get over TLS: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if string(val) != "tls-value" {
		t.Errorf("value = %q, want %q", string(val), "tls-value")
	}
}

func TestTLS_WithOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	certs := generateTLSCerts(t)
	addr, container := startInfinispanTLS(t, certs, false)
	createTestCacheTLS(t, container, "tls-opts")

	uri := fmt.Sprintf("hotrods://admin:password@%s", addr)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri,
		hotrod.WithTrustStore(certs.caFile),
		hotrod.WithSNIHostName("localhost"),
	)
	if err != nil {
		t.Fatalf("NewClient TLS with options: %v", err)
	}
	defer client.Close()

	cache := client.Cache("tls-opts")

	if err := cache.Put(ctx, []byte("k"), []byte("v")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("k"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(val) != "v" {
		t.Errorf("expected found=true val=%q, got found=%v val=%q", "v", found, string(val))
	}
}

func TestMTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	certs := generateTLSCerts(t)
	addr, container := startInfinispanTLS(t, certs, true)
	createTestCacheMTLS(t, container, "mtls-test")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uri := fmt.Sprintf("hotrods://admin:password@%s?trust_ca=%s&client_cert=%s&client_key=%s&sni_host=localhost",
		addr, certs.caFile, certs.clientCertFile, certs.clientKeyFile)

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient mTLS: %v", err)
	}
	defer client.Close()

	cache := client.Cache("mtls-test")

	if err := cache.Put(ctx, []byte("mtls-key"), []byte("mtls-value")); err != nil {
		t.Fatalf("Put over mTLS: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("mtls-key"))
	if err != nil {
		t.Fatalf("Get over mTLS: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if string(val) != "mtls-value" {
		t.Errorf("value = %q, want %q", string(val), "mtls-value")
	}
}

func TestMTLS_WithoutClientCert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	certs := generateTLSCerts(t)
	addr, _ := startInfinispanTLS(t, certs, true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := fmt.Sprintf("hotrods://admin:password@%s?trust_ca=%s&sni_host=localhost",
		addr, certs.caFile)

	_, err := hotrod.NewClient(ctx, uri)
	if err == nil {
		t.Fatal("expected connection to fail without client certificate when server requires mTLS")
	}
	t.Logf("expected error: %v", err)
}

func startInfinispanExternal(t *testing.T, certs *tlsCerts) (string, testcontainers.Container) {
	t.Helper()
	ctx := context.Background()

	xmlFile := filepath.Join(certs.dir, "infinispan.xml")
	if err := os.WriteFile(xmlFile, []byte(infinispanExternalXML), 0644); err != nil {
		t.Fatal(err)
	}
	scriptFile := filepath.Join(certs.dir, "entrypoint.sh")
	if err := os.WriteFile(scriptFile, []byte(externalEntrypoint), 0755); err != nil {
		t.Fatal(err)
	}

	confDir := "/opt/infinispan/server/conf"
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        serverImage(),
			ExposedPorts: []string{"11222/tcp"},
			Files: []testcontainers.ContainerFile{
				{HostFilePath: xmlFile, ContainerFilePath: confDir + "/infinispan.xml", FileMode: 0644},
				{HostFilePath: certs.serverP12, ContainerFilePath: confDir + "/server.p12", FileMode: 0644},
				{HostFilePath: certs.caFile, ContainerFilePath: confDir + "/ca.pem", FileMode: 0644},
				{HostFilePath: certs.clientP12, ContainerFilePath: confDir + "/client.p12", FileMode: 0644},
				{HostFilePath: scriptFile, ContainerFilePath: confDir + "/entrypoint.sh", FileMode: 0755},
			},
			Entrypoint: []string{"bash", confDir + "/entrypoint.sh"},
			WaitingFor: wait.ForLog("ISPN080001").WithStartupTimeout(120 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start infinispan EXTERNAL: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

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

func TestExternalAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	certs := generateTLSCerts(t, "admin")
	addr, _ := startInfinispanExternal(t, certs)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uri := fmt.Sprintf("hotrods://%s?trust_ca=%s&client_cert=%s&client_key=%s&sni_host=localhost",
		addr, certs.caFile, certs.clientCertFile, certs.clientKeyFile)

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient EXTERNAL: %v", err)
	}
	defer client.Close()

	cache := client.Cache("ext-test")

	if err := cache.Put(ctx, []byte("ext-key"), []byte("ext-value")); err != nil {
		t.Fatalf("Put with EXTERNAL auth: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("ext-key"))
	if err != nil {
		t.Fatalf("Get with EXTERNAL auth: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if string(val) != "ext-value" {
		t.Errorf("value = %q, want %q", string(val), "ext-value")
	}
}
