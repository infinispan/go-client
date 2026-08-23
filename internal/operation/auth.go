package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type AuthOp struct {
	Mechanism    string
	ResponseData []byte
}

type AuthResponse struct {
	Completed bool
	Challenge []byte
}

func (a *AuthOp) RequestOpCode() byte   { return codec.OpAuth }
func (a *AuthOp) ResponseOpCode() byte  { return codec.OpAuthResponse }
func (a *AuthOp) CacheName() []byte     { return []byte{} }
func (a *AuthOp) Flags() int32          { return 0 }
func (a *AuthOp) KeyMediaType() int32   { return 0 }
func (a *AuthOp) ValueMediaType() int32 { return 0 }

func (a *AuthOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, a.Mechanism); err != nil {
		return err
	}
	return codec.WriteLPBytes(w, a.ResponseData)
}

func (a *AuthOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	completed, err := codec.ReadU1(r)
	if err != nil {
		return nil, err
	}
	challenge, err := codec.ReadLPBytes(r)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		Completed: completed != 0,
		Challenge: challenge,
	}, nil
}
