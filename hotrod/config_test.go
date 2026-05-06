package hotrod

import (
	"testing"
	"time"

	"infinispan.org/go-client/internal/codec"
)

func TestParseURI_Basic(t *testing.T) {
	p, err := parseURI("hotrod://localhost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.tls {
		t.Error("expected tls=false")
	}
	if len(p.servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(p.servers))
	}
	if p.servers[0].host != "localhost" {
		t.Errorf("host = %q, want %q", p.servers[0].host, "localhost")
	}
	if p.servers[0].port != codec.DefaultPort {
		t.Errorf("port = %d, want %d", p.servers[0].port, codec.DefaultPort)
	}
}

func TestParseURI_TLS(t *testing.T) {
	p, err := parseURI("hotrods://secure.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.tls {
		t.Error("expected tls=true")
	}
	if p.servers[0].host != "secure.example.com" {
		t.Errorf("host = %q, want %q", p.servers[0].host, "secure.example.com")
	}
}

func TestParseURI_WithPort(t *testing.T) {
	p, err := parseURI("hotrod://myhost:9999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.servers[0].host != "myhost" {
		t.Errorf("host = %q, want %q", p.servers[0].host, "myhost")
	}
	if p.servers[0].port != 9999 {
		t.Errorf("port = %d, want %d", p.servers[0].port, 9999)
	}
}

func TestParseURI_WithCredentials(t *testing.T) {
	p, err := parseURI("hotrod://admin:secret@localhost:11222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.username != "admin" {
		t.Errorf("username = %q, want %q", p.username, "admin")
	}
	if p.password != "secret" {
		t.Errorf("password = %q, want %q", p.password, "secret")
	}
	if p.servers[0].host != "localhost" {
		t.Errorf("host = %q, want %q", p.servers[0].host, "localhost")
	}
}

func TestParseURI_TLSWithCredentials(t *testing.T) {
	p, err := parseURI("hotrods://user:pass@host1:11222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.tls {
		t.Error("expected tls=true")
	}
	if p.username != "user" {
		t.Errorf("username = %q, want %q", p.username, "user")
	}
	if p.password != "pass" {
		t.Errorf("password = %q, want %q", p.password, "pass")
	}
}

func TestParseURI_DefaultPort(t *testing.T) {
	p, err := parseURI("hotrod://myhost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.servers[0].port != codec.DefaultPort {
		t.Errorf("port = %d, want default %d", p.servers[0].port, codec.DefaultPort)
	}
}

func TestParseURI_InvalidScheme(t *testing.T) {
	_, err := parseURI("http://localhost")
	if err == nil {
		t.Fatal("expected error for invalid scheme")
	}
}

func TestParseURI_EmptyHost(t *testing.T) {
	_, err := parseURI("hotrod://")
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestParseURI_IPv6(t *testing.T) {
	p, err := parseURI("hotrod://[::1]:11222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.servers[0].host != "::1" {
		t.Errorf("host = %q, want %q", p.servers[0].host, "::1")
	}
	if p.servers[0].port != 11222 {
		t.Errorf("port = %d, want %d", p.servers[0].port, 11222)
	}
}

func TestServerAddr_String(t *testing.T) {
	s := serverAddr{host: "myhost", port: 11222}
	if s.String() != "myhost:11222" {
		t.Errorf("String() = %q, want %q", s.String(), "myhost:11222")
	}
}

func TestParseURI_ConnectTimeout(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222?connect_timeout=5000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.connectTimeout != 5000*time.Millisecond {
		t.Errorf("connectTimeout = %v, want %v", p.connectTimeout, 5000*time.Millisecond)
	}
}

func TestParseURI_ConnectTimeoutWithAuth(t *testing.T) {
	p, err := parseURI("hotrod://admin:secret@localhost:11222?connect_timeout=3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.username != "admin" {
		t.Errorf("username = %q, want %q", p.username, "admin")
	}
	if p.connectTimeout != 3000*time.Millisecond {
		t.Errorf("connectTimeout = %v, want %v", p.connectTimeout, 3000*time.Millisecond)
	}
}

func TestParseURI_NoQueryParams(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.connectTimeout != 0 {
		t.Errorf("connectTimeout = %v, want 0", p.connectTimeout)
	}
}

func TestParseURI_InvalidConnectTimeout(t *testing.T) {
	_, err := parseURI("hotrod://localhost:11222?connect_timeout=abc")
	if err == nil {
		t.Fatal("expected error for non-numeric connect_timeout")
	}
}

func TestParseURI_SocketTimeout(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222?socket_timeout=3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.socketTimeout != 3000*time.Millisecond {
		t.Errorf("socketTimeout = %v, want %v", p.socketTimeout, 3000*time.Millisecond)
	}
}

func TestParseURI_InvalidSocketTimeout(t *testing.T) {
	_, err := parseURI("hotrod://localhost:11222?socket_timeout=xyz")
	if err == nil {
		t.Fatal("expected error for non-numeric socket_timeout")
	}
}

func TestParseURI_TCPNoDelay(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222?tcp_no_delay=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.tcpNoDelay == nil || *p.tcpNoDelay != false {
		t.Errorf("tcpNoDelay = %v, want false", p.tcpNoDelay)
	}
}

func TestParseURI_TCPNoDelayTrue(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222?tcp_no_delay=true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.tcpNoDelay == nil || *p.tcpNoDelay != true {
		t.Errorf("tcpNoDelay = %v, want true", p.tcpNoDelay)
	}
}

func TestParseURI_InvalidTCPNoDelay(t *testing.T) {
	_, err := parseURI("hotrod://localhost:11222?tcp_no_delay=maybe")
	if err == nil {
		t.Fatal("expected error for invalid tcp_no_delay")
	}
}

func TestParseURI_TCPKeepAlive(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222?tcp_keep_alive=true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.tcpKeepAlive == nil || *p.tcpKeepAlive != true {
		t.Errorf("tcpKeepAlive = %v, want true", p.tcpKeepAlive)
	}
}

func TestParseURI_InvalidTCPKeepAlive(t *testing.T) {
	_, err := parseURI("hotrod://localhost:11222?tcp_keep_alive=maybe")
	if err == nil {
		t.Fatal("expected error for invalid tcp_keep_alive")
	}
}

func TestParseURI_MultipleProperties(t *testing.T) {
	p, err := parseURI("hotrod://admin:pass@localhost:11222?connect_timeout=5000&socket_timeout=2000&tcp_no_delay=false&tcp_keep_alive=true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.connectTimeout != 5000*time.Millisecond {
		t.Errorf("connectTimeout = %v, want %v", p.connectTimeout, 5000*time.Millisecond)
	}
	if p.socketTimeout != 2000*time.Millisecond {
		t.Errorf("socketTimeout = %v, want %v", p.socketTimeout, 2000*time.Millisecond)
	}
	if p.tcpNoDelay == nil || *p.tcpNoDelay != false {
		t.Errorf("tcpNoDelay = %v, want false", p.tcpNoDelay)
	}
	if p.tcpKeepAlive == nil || *p.tcpKeepAlive != true {
		t.Errorf("tcpKeepAlive = %v, want true", p.tcpKeepAlive)
	}
	if p.username != "admin" {
		t.Errorf("username = %q, want %q", p.username, "admin")
	}
}

func TestParseURI_NoPropertiesDefaults(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.socketTimeout != 0 {
		t.Errorf("socketTimeout = %v, want 0", p.socketTimeout)
	}
	if p.tcpNoDelay != nil {
		t.Errorf("tcpNoDelay = %v, want nil", p.tcpNoDelay)
	}
	if p.tcpKeepAlive != nil {
		t.Errorf("tcpKeepAlive = %v, want nil", p.tcpKeepAlive)
	}
}

func TestParseURI_TrustStore(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?trust_store_file_name=/path/to/ca.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.trustStorePath != "/path/to/ca.pem" {
		t.Errorf("trustStorePath = %q, want %q", p.trustStorePath, "/path/to/ca.pem")
	}
}

func TestParseURI_TrustStoreGoAlias(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?trust_ca=/path/to/ca.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.trustStorePath != "/path/to/ca.pem" {
		t.Errorf("trustStorePath = %q, want %q", p.trustStorePath, "/path/to/ca.pem")
	}
}

func TestParseURI_ClientCert(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?key_store_file_name=/cert.pem&key_store_password=/key.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.clientCertPath != "/cert.pem" {
		t.Errorf("clientCertPath = %q, want %q", p.clientCertPath, "/cert.pem")
	}
	if p.clientKeyPath != "/key.pem" {
		t.Errorf("clientKeyPath = %q, want %q", p.clientKeyPath, "/key.pem")
	}
}

func TestParseURI_ClientCertGoAlias(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?client_cert=/cert.pem&client_key=/key.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.clientCertPath != "/cert.pem" {
		t.Errorf("clientCertPath = %q, want %q", p.clientCertPath, "/cert.pem")
	}
	if p.clientKeyPath != "/key.pem" {
		t.Errorf("clientKeyPath = %q, want %q", p.clientKeyPath, "/key.pem")
	}
}

func TestParseURI_SNIHostName(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?sni_host_name=infinispan.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.sniHostName != "infinispan.example.com" {
		t.Errorf("sniHostName = %q, want %q", p.sniHostName, "infinispan.example.com")
	}
}

func TestParseURI_SNIHostGoAlias(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?sni_host=infinispan.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.sniHostName != "infinispan.example.com" {
		t.Errorf("sniHostName = %q, want %q", p.sniHostName, "infinispan.example.com")
	}
}

func TestParseURI_SSLHostnameValidation(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?ssl_hostname_validation=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.sslHostnameValidation == nil || *p.sslHostnameValidation != false {
		t.Errorf("sslHostnameValidation = %v, want false", p.sslHostnameValidation)
	}
}

func TestParseURI_VerifyHostnameGoAlias(t *testing.T) {
	p, err := parseURI("hotrods://localhost:11222?verify_hostname=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.sslHostnameValidation == nil || *p.sslHostnameValidation != false {
		t.Errorf("sslHostnameValidation = %v, want false", p.sslHostnameValidation)
	}
}

func TestParseURI_InvalidSSLHostnameValidation(t *testing.T) {
	_, err := parseURI("hotrods://localhost:11222?ssl_hostname_validation=nope")
	if err == nil {
		t.Fatal("expected error for invalid ssl_hostname_validation")
	}
}

func TestParseURI_AllTLSProperties(t *testing.T) {
	uri := "hotrods://admin:pass@localhost:11222?" +
		"trust_ca=/ca.pem&client_cert=/cert.pem&client_key=/key.pem&" +
		"sni_host=myhost&verify_hostname=true&connect_timeout=3000"
	p, err := parseURI(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.tls {
		t.Error("expected tls=true")
	}
	if p.trustStorePath != "/ca.pem" {
		t.Errorf("trustStorePath = %q, want %q", p.trustStorePath, "/ca.pem")
	}
	if p.clientCertPath != "/cert.pem" {
		t.Errorf("clientCertPath = %q, want %q", p.clientCertPath, "/cert.pem")
	}
	if p.clientKeyPath != "/key.pem" {
		t.Errorf("clientKeyPath = %q, want %q", p.clientKeyPath, "/key.pem")
	}
	if p.sniHostName != "myhost" {
		t.Errorf("sniHostName = %q, want %q", p.sniHostName, "myhost")
	}
	if p.sslHostnameValidation == nil || *p.sslHostnameValidation != true {
		t.Errorf("sslHostnameValidation = %v, want true", p.sslHostnameValidation)
	}
	if p.connectTimeout != 3000*time.Millisecond {
		t.Errorf("connectTimeout = %v, want %v", p.connectTimeout, 3000*time.Millisecond)
	}
}

func TestParseURI_TLSFieldsDefaultNil(t *testing.T) {
	p, err := parseURI("hotrod://localhost:11222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.trustStorePath != "" {
		t.Errorf("trustStorePath = %q, want empty", p.trustStorePath)
	}
	if p.clientCertPath != "" {
		t.Errorf("clientCertPath = %q, want empty", p.clientCertPath)
	}
	if p.sniHostName != "" {
		t.Errorf("sniHostName = %q, want empty", p.sniHostName)
	}
	if p.sslHostnameValidation != nil {
		t.Errorf("sslHostnameValidation = %v, want nil", p.sslHostnameValidation)
	}
}

func TestParseURI_UnknownProperty(t *testing.T) {
	_, err := parseURI("hotrod://localhost:11222?bogus=1")
	if err == nil {
		t.Fatal("expected error for unknown URI property")
	}
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		input    string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"localhost", "localhost", "", false},
		{"localhost:11222", "localhost", "11222", false},
		{"[::1]", "::1", "", false},
		{"[::1]:11222", "::1", "11222", false},
		{"[bad", "", "", true},
	}
	for _, tt := range tests {
		host, port, err := splitHostPort(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("splitHostPort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if host != tt.wantHost {
			t.Errorf("splitHostPort(%q) host = %q, want %q", tt.input, host, tt.wantHost)
		}
		if port != tt.wantPort {
			t.Errorf("splitHostPort(%q) port = %q, want %q", tt.input, port, tt.wantPort)
		}
	}
}
