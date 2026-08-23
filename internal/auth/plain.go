package auth

import "fmt"

type Plain struct {
	username string
	password string
	complete bool
}

func NewPlain(username, password string) *Plain {
	return &Plain{username: username, password: password}
}

func (p *Plain) Name() string             { return "PLAIN" }
func (p *Plain) HasInitialResponse() bool { return true }
func (p *Plain) Complete() bool           { return p.complete }
func (p *Plain) Username() string         { return p.username }
func (p *Plain) Password() string         { return p.password }

func (p *Plain) InitialResponse() ([]byte, error) {
	p.complete = true
	// RFC 4616: [authzid] NUL authcid NUL passwd
	resp := make([]byte, 0, 1+len(p.username)+1+len(p.password))
	resp = append(resp, 0)
	resp = append(resp, []byte(p.username)...)
	resp = append(resp, 0)
	resp = append(resp, []byte(p.password)...)
	return resp, nil
}

func (p *Plain) EvaluateChallenge(challenge []byte) ([]byte, error) {
	return nil, fmt.Errorf("PLAIN does not expect a challenge")
}
