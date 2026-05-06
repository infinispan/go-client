package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

type ScramSHA256 struct {
	username     string
	password     string
	state        scramState
	clientNonce  string
	serverNonce  string
	salt         []byte
	iterations   int
	authMessage  string
	saltedPwd    []byte
	serverKey    []byte
}

type scramState int

const (
	scramStateNew scramState = iota
	scramStateChallengeReceived
	scramStateComplete
)

func NewScramSHA256(username, password string) *ScramSHA256 {
	return &ScramSHA256{username: username, password: password}
}

func (s *ScramSHA256) Name() string            { return "SCRAM-SHA-256" }
func (s *ScramSHA256) HasInitialResponse() bool { return true }
func (s *ScramSHA256) IsComplete() bool         { return s.state == scramStateComplete }
func (s *ScramSHA256) Username() string         { return s.username }
func (s *ScramSHA256) Password() string         { return s.password }

func (s *ScramSHA256) InitialResponse() ([]byte, error) {
	nonce := make([]byte, 18)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	s.clientNonce = base64.StdEncoding.EncodeToString(nonce)
	// client-first-message: gs2-header + client-first-message-bare
	msg := fmt.Sprintf("n,,n=%s,r=%s", saslPrep(s.username), s.clientNonce)
	return []byte(msg), nil
}

func (s *ScramSHA256) EvaluateChallenge(challenge []byte) ([]byte, error) {
	switch s.state {
	case scramStateNew:
		return s.processServerFirst(string(challenge))
	case scramStateChallengeReceived:
		return s.processServerFinal(string(challenge))
	default:
		return nil, fmt.Errorf("unexpected SCRAM state")
	}
}

func (s *ScramSHA256) processServerFirst(serverFirst string) ([]byte, error) {
	parts := parseScramAttributes(serverFirst)
	var ok bool
	s.serverNonce, ok = parts["r"]
	if !ok {
		return nil, fmt.Errorf("server-first: missing nonce")
	}
	if !strings.HasPrefix(s.serverNonce, s.clientNonce) {
		return nil, fmt.Errorf("server-first: nonce doesn't start with client nonce")
	}
	saltB64, ok := parts["s"]
	if !ok {
		return nil, fmt.Errorf("server-first: missing salt")
	}
	var err error
	s.salt, err = base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("server-first: decode salt: %w", err)
	}
	iterStr, ok := parts["i"]
	if !ok {
		return nil, fmt.Errorf("server-first: missing iteration count")
	}
	s.iterations, err = strconv.Atoi(iterStr)
	if err != nil {
		return nil, fmt.Errorf("server-first: parse iterations: %w", err)
	}

	s.saltedPwd = pbkdf2SHA256([]byte(saslPrep(s.password)), s.salt, s.iterations, 32)
	clientKey := hmacSHA256(s.saltedPwd, []byte("Client Key"))
	storedKey := sha256Sum(clientKey)
	s.serverKey = hmacSHA256(s.saltedPwd, []byte("Server Key"))

	clientFirstBare := fmt.Sprintf("n=%s,r=%s", saslPrep(s.username), s.clientNonce)
	clientFinalNoProof := fmt.Sprintf("c=%s,r=%s", base64.StdEncoding.EncodeToString([]byte("n,,")), s.serverNonce)
	s.authMessage = clientFirstBare + "," + serverFirst + "," + clientFinalNoProof

	clientSignature := hmacSHA256(storedKey, []byte(s.authMessage))
	clientProof := xorBytes(clientKey, clientSignature)

	clientFinal := clientFinalNoProof + ",p=" + base64.StdEncoding.EncodeToString(clientProof)

	s.state = scramStateChallengeReceived
	return []byte(clientFinal), nil
}

func (s *ScramSHA256) processServerFinal(serverFinal string) ([]byte, error) {
	parts := parseScramAttributes(serverFinal)
	if errMsg, ok := parts["e"]; ok {
		return nil, fmt.Errorf("server-final error: %s", errMsg)
	}
	verifierB64, ok := parts["v"]
	if !ok {
		return nil, fmt.Errorf("server-final: missing verifier")
	}
	serverSignature := hmacSHA256(s.serverKey, []byte(s.authMessage))
	expectedVerifier := base64.StdEncoding.EncodeToString(serverSignature)
	if verifierB64 != expectedVerifier {
		return nil, fmt.Errorf("server signature verification failed")
	}
	s.state = scramStateComplete
	return nil, nil
}

func parseScramAttributes(s string) map[string]string {
	result := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		if idx := strings.IndexByte(part, '='); idx >= 0 {
			result[part[:idx]] = part[idx+1:]
		}
	}
	return result
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func xorBytes(a, b []byte) []byte {
	result := make([]byte, len(a))
	for i := range a {
		result[i] = a[i] ^ b[i]
	}
	return result
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	numBlocks := (keyLen + sha256.Size - 1) / sha256.Size
	dk := make([]byte, 0, numBlocks*sha256.Size)
	for block := 1; block <= numBlocks; block++ {
		dk = append(dk, pbkdf2Block(password, salt, iterations, block)...)
	}
	return dk[:keyLen]
}

func pbkdf2Block(password, salt []byte, iterations, blockIndex int) []byte {
	h := hmac.New(sha256.New, password)
	h.Write(salt)
	h.Write([]byte{byte(blockIndex >> 24), byte(blockIndex >> 16), byte(blockIndex >> 8), byte(blockIndex)})
	u := h.Sum(nil)
	result := make([]byte, len(u))
	copy(result, u)
	for i := 1; i < iterations; i++ {
		h.Reset()
		h.Write(u)
		u = h.Sum(u[:0])
		for j := range result {
			result[j] ^= u[j]
		}
	}
	return result
}

func saslPrep(s string) string {
	return s
}
