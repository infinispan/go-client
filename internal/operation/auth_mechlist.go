package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type AuthMechListOp struct{}

func (a *AuthMechListOp) RequestOpCode() byte   { return codec.OpAuthMechList }
func (a *AuthMechListOp) ResponseOpCode() byte  { return codec.OpAuthMechListResponse }
func (a *AuthMechListOp) CacheName() []byte     { return []byte{} }
func (a *AuthMechListOp) Flags() int32          { return 0 }
func (a *AuthMechListOp) KeyMediaType() int32   { return 0 }
func (a *AuthMechListOp) ValueMediaType() int32 { return 0 }

func (a *AuthMechListOp) WriteBody(w io.Writer) error {
	return nil
}

func (a *AuthMechListOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	count, err := codec.ReadVInt(r)
	if err != nil {
		return nil, err
	}
	mechs := make([]string, count)
	for i := range count {
		s, err := codec.ReadLPString(r)
		if err != nil {
			return nil, err
		}
		mechs[i] = s
	}
	return mechs, nil
}
