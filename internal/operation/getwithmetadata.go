package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type GetWithMetadataResponse struct {
	Value    []byte
	Metadata *codec.EntryMetadata
	Found    bool
}

type GetWithMetadataOp struct {
	Cache   string
	Key     []byte
	KeyMT   int32
	ValueMT int32
	OpFlags int32
}

func (g *GetWithMetadataOp) RequestOpCode() byte  { return codec.OpGetWithMetadata }
func (g *GetWithMetadataOp) ResponseOpCode() byte { return codec.OpGetWithMetadataResponse }
func (g *GetWithMetadataOp) CacheName() []byte    { return []byte(g.Cache) }
func (g *GetWithMetadataOp) Flags() int32         { return g.OpFlags }

func (g *GetWithMetadataOp) KeyMediaType() int32 {
	if g.KeyMT != 0 {
		return g.KeyMT
	}
	return codec.MediaIDOctetStream
}

func (g *GetWithMetadataOp) ValueMediaType() int32 {
	if g.ValueMT != 0 {
		return g.ValueMT
	}
	return codec.MediaIDOctetStream
}

func (g *GetWithMetadataOp) KeyBytes() []byte { return g.Key }

func (g *GetWithMetadataOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, g.Key)
}

func (g *GetWithMetadataOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusKeyDoesNotExist {
		return &GetWithMetadataResponse{Found: false}, nil
	}
	md, value, err := codec.ReadMetadata(r)
	if err != nil {
		return nil, err
	}
	return &GetWithMetadataResponse{Value: value, Metadata: md, Found: true}, nil
}
