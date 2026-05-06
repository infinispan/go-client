package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type GetAllEntry struct {
	Key   []byte
	Value []byte
}

type GetAllOp struct {
	Cache string
	Keys  [][]byte
}

func (o *GetAllOp) RequestOpCode() byte   { return codec.OpGetAll }
func (o *GetAllOp) ResponseOpCode() byte  { return codec.OpGetAllResponse }
func (o *GetAllOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *GetAllOp) Flags() int32          { return 0 }
func (o *GetAllOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *GetAllOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }

func (o *GetAllOp) WriteBody(w io.Writer) error {
	if err := codec.WriteVInt(w, int32(len(o.Keys))); err != nil {
		return err
	}
	for _, k := range o.Keys {
		if err := codec.WriteLPBytes(w, k); err != nil {
			return err
		}
	}
	return nil
}

func (o *GetAllOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	count, err := codec.ReadVInt(r)
	if err != nil {
		return nil, err
	}
	entries := make([]GetAllEntry, count)
	for i := int32(0); i < count; i++ {
		key, err := codec.ReadLPBytes(r)
		if err != nil {
			return nil, err
		}
		value, err := codec.ReadLPBytes(r)
		if err != nil {
			return nil, err
		}
		entries[i] = GetAllEntry{Key: key, Value: value}
	}
	return entries, nil
}
