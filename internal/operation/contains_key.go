package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type ContainsKeyOp struct {
	Cache   string
	Key     []byte
	OpFlags int32
}

func (o *ContainsKeyOp) RequestOpCode() byte   { return codec.OpContainsKey }
func (o *ContainsKeyOp) ResponseOpCode() byte  { return codec.OpContainsKeyResponse }
func (o *ContainsKeyOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *ContainsKeyOp) Flags() int32          { return o.OpFlags }
func (o *ContainsKeyOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *ContainsKeyOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *ContainsKeyOp) KeyBytes() []byte      { return o.Key }

func (o *ContainsKeyOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, o.Key)
}

func (o *ContainsKeyOp) DecodeResponse(status byte, _ io.Reader) (any, error) {
	return status != codec.StatusKeyDoesNotExist, nil
}
