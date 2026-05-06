package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type ClearOp struct {
	Cache string
}

func (o *ClearOp) RequestOpCode() byte   { return codec.OpClear }
func (o *ClearOp) ResponseOpCode() byte  { return codec.OpClearResponse }
func (o *ClearOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *ClearOp) Flags() int32          { return 0 }
func (o *ClearOp) KeyMediaType() int32   { return 0 }
func (o *ClearOp) ValueMediaType() int32 { return 0 }

func (o *ClearOp) WriteBody(_ io.Writer) error {
	return nil
}

func (o *ClearOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}
