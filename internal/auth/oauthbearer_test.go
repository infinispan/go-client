package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestOAuthBearerName(t *testing.T) {
	m := NewOAuthBearer("tok")
	if m.Name() != "OAUTHBEARER" {
		t.Errorf("Name() = %q, want OAUTHBEARER", m.Name())
	}
}

func TestOAuthBearerInitialResponse(t *testing.T) {
	token := "eyJhbGciOiJSUzI1NiJ9.test"
	m := NewOAuthBearer(token)

	if !m.HasInitialResponse() {
		t.Fatal("HasInitialResponse() = false")
	}

	resp, err := m.InitialResponse()
	if err != nil {
		t.Fatal(err)
	}

	// RFC 7628: "n,," + 0x01 + "auth=Bearer " + token + 0x01 + 0x01
	expected := []byte("n,,\x01auth=Bearer " + token + "\x01\x01")
	if !bytes.Equal(resp, expected) {
		t.Errorf("InitialResponse() = %q, want %q", resp, expected)
	}

	if !m.IsComplete() {
		t.Error("expected IsComplete() = true after InitialResponse")
	}
}

func TestOAuthBearerEvaluateChallengeReturnsError(t *testing.T) {
	m := NewOAuthBearer("tok")
	resp, err := m.EvaluateChallenge([]byte(`{"status":"invalid_token"}`))
	if err == nil {
		t.Fatal("expected error from EvaluateChallenge")
	}
	if !strings.Contains(err.Error(), "OAUTHBEARER authentication failed") {
		t.Errorf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0] != 0x01 {
		t.Errorf("expected [0x01] response, got %v", resp)
	}
}

func TestOAuthBearerToken(t *testing.T) {
	m := NewOAuthBearer("my-token")
	if m.Token() != "my-token" {
		t.Errorf("Token() = %q, want %q", m.Token(), "my-token")
	}
}
