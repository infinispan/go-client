package hotrod

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"infinispan.org/go-client/internal/codec"
)

type serverAddr struct {
	host string
	port int
}

func (s serverAddr) String() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

type parsedURI struct {
	tls                   bool
	username              string
	password              string
	servers               []serverAddr
	connectTimeout        time.Duration
	socketTimeout         time.Duration
	tcpNoDelay            *bool
	tcpKeepAlive          *bool
	trustStorePath        string
	clientCertPath        string
	clientKeyPath         string
	sniHostName           string
	sslHostnameValidation *bool
	saslMechanism         string
	token                 string
}

func parseURI(rawURI string) (*parsedURI, error) {
	result := &parsedURI{}

	switch {
	case strings.HasPrefix(rawURI, "hotrods://"):
		result.tls = true
		rawURI = "hotrods" + rawURI[len("hotrods"):]
	case strings.HasPrefix(rawURI, "hotrod://"):
		rawURI = "hotrod" + rawURI[len("hotrod"):]
	default:
		return nil, fmt.Errorf("unsupported scheme, expected hotrod:// or hotrods://")
	}

	tempURI := strings.Replace(rawURI, "hotrod://", "http://", 1)
	tempURI = strings.Replace(tempURI, "hotrods://", "https://", 1)

	u, err := url.Parse(tempURI)
	if err != nil {
		return nil, fmt.Errorf("parse URI: %w", err)
	}

	if u.User != nil {
		result.username = u.User.Username()
		result.password, _ = u.User.Password()
	}

	// net/url already parsed the user info out of Host
	hostPart := u.Host

	hosts := strings.Split(hostPart, ",")
	for _, h := range hosts {
		if h == "" {
			continue
		}
		host, portStr, err := splitHostPort(h)
		if err != nil {
			return nil, fmt.Errorf("parse host %q: %w", h, err)
		}
		port := codec.DefaultPort
		if portStr != "" {
			port, err = strconv.Atoi(portStr)
			if err != nil {
				return nil, fmt.Errorf("parse port %q: %w", portStr, err)
			}
		}
		result.servers = append(result.servers, serverAddr{host: host, port: port})
	}

	if len(result.servers) == 0 {
		return nil, fmt.Errorf("no servers specified")
	}

	if err := result.parseQuery(u.Query()); err != nil {
		return nil, err
	}

	return result, nil
}

func (p *parsedURI) parseQuery(q url.Values) error {
	for key := range q {
		val := q.Get(key)
		switch key {
		case "connect_timeout":
			ms, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid connect_timeout %q: %w", val, err)
			}
			p.connectTimeout = time.Duration(ms) * time.Millisecond
		case "socket_timeout":
			ms, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid socket_timeout %q: %w", val, err)
			}
			p.socketTimeout = time.Duration(ms) * time.Millisecond
		case "tcp_no_delay":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid tcp_no_delay %q: %w", val, err)
			}
			p.tcpNoDelay = &b
		case "tcp_keep_alive":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid tcp_keep_alive %q: %w", val, err)
			}
			p.tcpKeepAlive = &b
		case "trust_store_file_name", "trust_ca":
			p.trustStorePath = val
		case "key_store_file_name", "client_cert":
			p.clientCertPath = val
		case "key_store_password", "client_key":
			p.clientKeyPath = val
		case "sni_host_name", "sni_host":
			p.sniHostName = val
		case "ssl_hostname_validation", "verify_hostname":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid %s %q: %w", key, val, err)
			}
			p.sslHostnameValidation = &b
		case "sasl_mechanism":
			p.saslMechanism = val
		case "token":
			p.token = val
		default:
			return fmt.Errorf("unknown URI property: %s", key)
		}
	}
	return nil
}

func splitHostPort(s string) (host, port string, err error) {
	if len(s) > 0 && s[0] == '[' {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", "", fmt.Errorf("invalid IPv6 address: %s", s)
		}
		if end+1 == len(s) {
			return s[1:end], "", nil
		}
		if s[end+1] == ':' {
			return s[1:end], s[end+2:], nil
		}
		return "", "", fmt.Errorf("invalid address: %s", s)
	}
	lastColon := strings.LastIndex(s, ":")
	if lastColon < 0 {
		return s, "", nil
	}
	return s[:lastColon], s[lastColon+1:], nil
}
