package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type AddBloomNearCacheListenerOp struct {
	Cache           string
	ListenerID      []byte
	BloomFilterBits int32
}

func (o *AddBloomNearCacheListenerOp) RequestOpCode() byte {
	return codec.OpAddBloomNearCacheListener
}
func (o *AddBloomNearCacheListenerOp) ResponseOpCode() byte {
	return codec.OpAddBloomNearCacheListenerResponse
}
func (o *AddBloomNearCacheListenerOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *AddBloomNearCacheListenerOp) Flags() int32          { return 0 }
func (o *AddBloomNearCacheListenerOp) KeyMediaType() int32   { return 0 }
func (o *AddBloomNearCacheListenerOp) ValueMediaType() int32 { return 0 }

func (o *AddBloomNearCacheListenerOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.ListenerID); err != nil {
		return err
	}
	return codec.WriteVInt(w, o.BloomFilterBits)
}

func (o *AddBloomNearCacheListenerOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}

type UpdateBloomFilterOp struct {
	Cache     string
	BloomBits []byte
}

func (o *UpdateBloomFilterOp) RequestOpCode() byte {
	return codec.OpUpdateBloomFilter
}
func (o *UpdateBloomFilterOp) ResponseOpCode() byte {
	return codec.OpUpdateBloomFilterResponse
}
func (o *UpdateBloomFilterOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *UpdateBloomFilterOp) Flags() int32          { return 0 }
func (o *UpdateBloomFilterOp) KeyMediaType() int32   { return 0 }
func (o *UpdateBloomFilterOp) ValueMediaType() int32 { return 0 }

func (o *UpdateBloomFilterOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, o.BloomBits)
}

func (o *UpdateBloomFilterOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}
