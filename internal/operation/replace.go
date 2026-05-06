package operation

import (
	"io"
	"time"

	"infinispan.org/go-client/internal/codec"
)

type ReplaceOp struct {
	Cache     string
	Key       []byte
	Value     []byte
	Lifespan  time.Duration
	MaxIdle   time.Duration
	MediaType int32
	OpFlags   int32
}

func (o *ReplaceOp) RequestOpCode() byte  { return codec.OpReplace }
func (o *ReplaceOp) ResponseOpCode() byte { return codec.OpReplaceResponse }
func (o *ReplaceOp) CacheName() []byte    { return []byte(o.Cache) }
func (o *ReplaceOp) Flags() int32         { return o.OpFlags }

func (o *ReplaceOp) KeyMediaType() int32 {
	if o.MediaType != 0 {
		return o.MediaType
	}
	return codec.MediaIDOctetStream
}

func (o *ReplaceOp) ValueMediaType() int32 {
	if o.MediaType != 0 {
		return o.MediaType
	}
	return codec.MediaIDOctetStream
}

func (o *ReplaceOp) KeyBytes() []byte { return o.Key }

func (o *ReplaceOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	tu := codec.EncodeTimeUnits(o.Lifespan, o.MaxIdle)
	if err := tu.Write(w); err != nil {
		return err
	}
	return codec.WriteLPBytes(w, o.Value)
}

func (o *ReplaceOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusSuccessWithPrevious || status == codec.StatusNotExecWithPrevious {
		prev, err := codec.ReadMetadataValue(r)
		if err != nil {
			return nil, err
		}
		return &CASResponse{
			Success:       status == codec.StatusSuccess || status == codec.StatusSuccessWithPrevious,
			PreviousValue: prev,
		}, nil
	}
	return &CASResponse{
		Success: status == codec.StatusSuccess,
	}, nil
}
