package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type GetOp struct {
	Cache   string
	Key     []byte
	KeyMT   int32
	ValueMT int32
	OpFlags int32
}

type GetResponse struct {
	Value []byte
	Found bool
}

func (g *GetOp) RequestOpCode() byte  { return codec.OpGet }
func (g *GetOp) ResponseOpCode() byte { return codec.OpGetResponse }
func (g *GetOp) CacheName() []byte    { return []byte(g.Cache) }
func (g *GetOp) Flags() int32         { return g.OpFlags }

func (g *GetOp) KeyMediaType() int32 {
	if g.KeyMT != 0 {
		return g.KeyMT
	}
	return codec.MediaIDOctetStream
}

func (g *GetOp) ValueMediaType() int32 {
	if g.ValueMT != 0 {
		return g.ValueMT
	}
	return codec.MediaIDOctetStream
}

func (g *GetOp) KeyBytes() []byte { return g.Key }

func (g *GetOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, g.Key)
}

func (g *GetOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusKeyDoesNotExist {
		return &GetResponse{Found: false}, nil
	}
	value, err := codec.ReadLPBytes(r)
	if err != nil {
		return nil, err
	}
	return &GetResponse{Value: value, Found: true}, nil
}
