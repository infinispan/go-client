package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type SizeOp struct {
	Cache string
}

func (o *SizeOp) RequestOpCode() byte   { return codec.OpSize }
func (o *SizeOp) ResponseOpCode() byte  { return codec.OpSizeResponse }
func (o *SizeOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *SizeOp) Flags() int32          { return 0 }
func (o *SizeOp) KeyMediaType() int32   { return 0 }
func (o *SizeOp) ValueMediaType() int32 { return 0 }

func (o *SizeOp) WriteBody(_ io.Writer) error {
	return nil
}

func (o *SizeOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	return codec.ReadVInt(r)
}
