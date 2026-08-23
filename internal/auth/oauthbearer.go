package auth

import "fmt"

// OAuthBearer implements the OAUTHBEARER SASL mechanism (RFC 7628).
type OAuthBearer struct {
	token    string
	complete bool
}

// NewOAuthBearer creates an OAUTHBEARER mechanism with the given bearer token.
func NewOAuthBearer(token string) *OAuthBearer {
	return &OAuthBearer{token: token}
}

func (o *OAuthBearer) Name() string            { return "OAUTHBEARER" }
func (o *OAuthBearer) HasInitialResponse() bool { return true }
func (o *OAuthBearer) Complete() bool         { return o.complete }
func (o *OAuthBearer) Token() string            { return o.token }

func (o *OAuthBearer) InitialResponse() ([]byte, error) {
	o.complete = true
	// RFC 7628 §3.1: gs2-header kvsep "auth=Bearer " token kvsep kvsep
	// gs2-header = "n,,"
	// kvsep      = 0x01
	resp := make([]byte, 0, 3+1+13+len(o.token)+1+1)
	resp = append(resp, "n,,"...)
	resp = append(resp, 0x01)
	resp = append(resp, "auth=Bearer "...)
	resp = append(resp, o.token...)
	resp = append(resp, 0x01, 0x01)
	return resp, nil
}

func (o *OAuthBearer) EvaluateChallenge(challenge []byte) ([]byte, error) {
	// Server rejected the token; acknowledge with 0x01 per RFC 7628 §3.2.3.
	return []byte{0x01}, fmt.Errorf("OAUTHBEARER authentication failed: %s", challenge)
}
