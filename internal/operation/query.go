package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type QueryOp struct {
	Cache   string
	Request []byte // pre-encoded WrappedMessage bytes
}

func (o *QueryOp) RequestOpCode() byte  { return codec.OpQuery }
func (o *QueryOp) ResponseOpCode() byte { return codec.OpQueryResponse }
func (o *QueryOp) CacheName() []byte    { return []byte(o.Cache) }
func (o *QueryOp) Flags() int32         { return 0 }

func (o *QueryOp) KeyMediaType() int32   { return codec.MediaIDProtostream }
func (o *QueryOp) ValueMediaType() int32 { return codec.MediaIDProtostream }

func (o *QueryOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, o.Request)
}

func (o *QueryOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	return codec.ReadLPBytes(r)
}
