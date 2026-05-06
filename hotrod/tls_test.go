package hotrod

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func generateTestCerts(t *testing.T, dir string) (caFile, certFile, keyFile string) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caFile = filepath.Join(dir, "ca.pem")
	writePEM(t, caFile, "CERTIFICATE", caDER)

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Test Client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, "client.pem")
	writePEM(t, certFile, "CERTIFICATE", clientDER)

	keyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatal(err)
	}
	keyFile = filepath.Join(dir, "client-key.pem")
	writePEM(t, keyFile, "EC PRIVATE KEY", keyDER)

	return
}

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

func TestBuildTLSConfig_NoTLS(t *testing.T) {
	parsed := &parsedURI{}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg != nil {
		t.Error("expected nil tls config when no TLS is configured")
	}
}

func TestBuildTLSConfig_SchemeOnly(t *testing.T) {
	parsed := &parsedURI{tls: true}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil tls config")
	}
}

func TestBuildTLSConfig_TrustStore(t *testing.T) {
	dir := t.TempDir()
	caFile, _, _ := generateTestCerts(t, dir)

	parsed := &parsedURI{tls: true, trustStorePath: caFile}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
}

func TestBuildTLSConfig_TrustStoreNotFound(t *testing.T) {
	parsed := &parsedURI{tls: true, trustStorePath: "/nonexistent/ca.pem"}
	cfg := &clientConfig{}
	_, err := buildTLSConfig(parsed, cfg)
	if err == nil {
		t.Fatal("expected error for missing trust store")
	}
}

func TestBuildTLSConfig_TrustStoreInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.pem")
	os.WriteFile(badFile, []byte("not a pem"), 0644)

	parsed := &parsedURI{tls: true, trustStorePath: badFile}
	cfg := &clientConfig{}
	_, err := buildTLSConfig(parsed, cfg)
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestBuildTLSConfig_ClientCert(t *testing.T) {
	dir := t.TempDir()
	_, certFile, keyFile := generateTestCerts(t, dir)

	parsed := &parsedURI{tls: true, clientCertPath: certFile, clientKeyPath: keyFile}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(tlsCfg.Certificates))
	}
}

func TestBuildTLSConfig_ClientCertWithoutKey(t *testing.T) {
	dir := t.TempDir()
	_, certFile, _ := generateTestCerts(t, dir)

	parsed := &parsedURI{tls: true, clientCertPath: certFile}
	cfg := &clientConfig{}
	_, err := buildTLSConfig(parsed, cfg)
	if err == nil {
		t.Fatal("expected error when client_cert is set without client_key")
	}
}

func TestBuildTLSConfig_SNI(t *testing.T) {
	parsed := &parsedURI{tls: true, sniHostName: "infinispan.example.com"}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg.ServerName != "infinispan.example.com" {
		t.Errorf("ServerName = %q, want %q", tlsCfg.ServerName, "infinispan.example.com")
	}
}

func TestBuildTLSConfig_SkipVerify(t *testing.T) {
	f := false
	parsed := &parsedURI{tls: true, sslHostnameValidation: &f}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify=true when hostname validation is disabled")
	}
}

func TestBuildTLSConfig_VerifyEnabled(t *testing.T) {
	tr := true
	parsed := &parsedURI{tls: true, sslHostnameValidation: &tr}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify=false when hostname validation is enabled")
	}
}

func TestBuildTLSConfig_OptionOverridesURI(t *testing.T) {
	dir := t.TempDir()
	caFile, certFile, keyFile := generateTestCerts(t, dir)

	parsed := &parsedURI{tls: true, sniHostName: "from-uri"}
	cfg := &clientConfig{
		trustStorePath: caFile,
		clientCertPath: certFile,
		clientKeyPath:  keyFile,
		sniHostName:    "from-option",
	}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg.ServerName != "from-option" {
		t.Errorf("ServerName = %q, want %q (option should override URI)", tlsCfg.ServerName, "from-option")
	}
	if tlsCfg.RootCAs == nil {
		t.Error("expected RootCAs from option")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Errorf("expected 1 client cert from option, got %d", len(tlsCfg.Certificates))
	}
}

func TestBuildTLSConfig_WithTLSBase(t *testing.T) {
	parsed := &parsedURI{tls: true, sniHostName: "myhost"}
	cfg := &clientConfig{
		tls: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg.MinVersion != tls.VersionTLS13 {
		t.Error("expected MinVersion from WithTLS base config to be preserved")
	}
	if tlsCfg.ServerName != "myhost" {
		t.Errorf("ServerName = %q, want %q", tlsCfg.ServerName, "myhost")
	}
}

func TestBuildTLSConfig_AutoEnableTLS(t *testing.T) {
	dir := t.TempDir()
	caFile, _, _ := generateTestCerts(t, dir)

	parsed := &parsedURI{tls: false, trustStorePath: caFile}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg == nil {
		t.Fatal("expected TLS to be auto-enabled when trust store is set")
	}
	if tlsCfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
}

func TestBuildTLSConfig_FullMTLS(t *testing.T) {
	dir := t.TempDir()
	caFile, certFile, keyFile := generateTestCerts(t, dir)

	f := false
	parsed := &parsedURI{
		tls:                   true,
		trustStorePath:        caFile,
		clientCertPath:        certFile,
		clientKeyPath:         keyFile,
		sniHostName:           "infinispan.local",
		sslHostnameValidation: &f,
	}
	cfg := &clientConfig{}
	tlsCfg, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tlsCfg.RootCAs == nil {
		t.Error("expected RootCAs")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Errorf("expected 1 client cert, got %d", len(tlsCfg.Certificates))
	}
	if tlsCfg.ServerName != "infinispan.local" {
		t.Errorf("ServerName = %q, want %q", tlsCfg.ServerName, "infinispan.local")
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify=true")
	}
}
