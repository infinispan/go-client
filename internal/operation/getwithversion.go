package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type GetWithVersionResponse struct {
	Value   []byte
	Version int64
	Found   bool
}

type GetWithVersionOp struct {
	Cache   string
	Key     []byte
	OpFlags int32
}

func (g *GetWithVersionOp) RequestOpCode() byte   { return codec.OpGetWithVersion }
func (g *GetWithVersionOp) ResponseOpCode() byte  { return codec.OpGetWithVersionResponse }
func (g *GetWithVersionOp) CacheName() []byte     { return []byte(g.Cache) }
func (g *GetWithVersionOp) Flags() int32          { return g.OpFlags }
func (g *GetWithVersionOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (g *GetWithVersionOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (g *GetWithVersionOp) KeyBytes() []byte      { return g.Key }

func (g *GetWithVersionOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, g.Key)
}

func (g *GetWithVersionOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusKeyDoesNotExist {
		return &GetWithVersionResponse{Found: false}, nil
	}
	version, err := codec.ReadLong(r)
	if err != nil {
		return nil, err
	}
	value, err := codec.ReadLPBytes(r)
	if err != nil {
		return nil, err
	}
	return &GetWithVersionResponse{Value: value, Version: version, Found: true}, nil
}
