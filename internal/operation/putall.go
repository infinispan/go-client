package operation

import (
	"io"
	"time"

	"infinispan.org/go-client/internal/codec"
)

type PutAllEntry struct {
	Key   []byte
	Value []byte
}

type PutAllOp struct {
	Cache    string
	Entries  []PutAllEntry
	Lifespan time.Duration
	MaxIdle  time.Duration
}

func (o *PutAllOp) RequestOpCode() byte   { return codec.OpPutAll }
func (o *PutAllOp) ResponseOpCode() byte  { return codec.OpPutAllResponse }
func (o *PutAllOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *PutAllOp) Flags() int32          { return 0 }
func (o *PutAllOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *PutAllOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }

func (o *PutAllOp) WriteBody(w io.Writer) error {
	tu := codec.EncodeTimeUnits(o.Lifespan, o.MaxIdle)
	if err := tu.Write(w); err != nil {
		return err
	}
	if err := codec.WriteVInt(w, int32(len(o.Entries))); err != nil {
		return err
	}
	for _, e := range o.Entries {
		if err := codec.WriteLPBytes(w, e.Key); err != nil {
			return err
		}
		if err := codec.WriteLPBytes(w, e.Value); err != nil {
			return err
		}
	}
	return nil
}

func (o *PutAllOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}
