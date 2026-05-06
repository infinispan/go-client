package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type PingOp struct{}

func (p *PingOp) RequestOpCode() byte  { return codec.OpPing }
func (p *PingOp) ResponseOpCode() byte { return codec.OpPingResponse }
func (p *PingOp) CacheName() []byte    { return []byte{} }
func (p *PingOp) Flags() int32         { return 0 }
func (p *PingOp) KeyMediaType() int32   { return 0 }
func (p *PingOp) ValueMediaType() int32 { return 0 }

func (p *PingOp) WriteBody(w io.Writer) error {
	return nil
}

func (p *PingOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status != codec.StatusSuccess {
		return nil, nil
	}
	// Protocol >= 3.0 ping response: keyMediaType + valueMediaType + serverVersion + opCount + opcodes
	if err := codec.SkipMediaType(r); err != nil {
		return nil, err
	}
	if err := codec.SkipMediaType(r); err != nil {
		return nil, err
	}
	// server version (u1)
	if _, err := codec.ReadU1(r); err != nil {
		return nil, err
	}
	// supported opcodes count
	opCount, err := codec.ReadVInt(r)
	if err != nil {
		return nil, err
	}
	for i := int32(0); i < opCount; i++ {
		if _, err := codec.ReadU2(r); err != nil {
			return nil, err
		}
	}
	return true, nil
}
