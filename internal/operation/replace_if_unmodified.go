package operation

import (
	"io"
	"time"

	"infinispan.org/go-client/internal/codec"
)

type ReplaceIfUnmodifiedOp struct {
	Cache    string
	Key      []byte
	Value    []byte
	Version  int64
	Lifespan time.Duration
	MaxIdle  time.Duration
	OpFlags  int32
}

func (o *ReplaceIfUnmodifiedOp) RequestOpCode() byte   { return codec.OpReplaceIfUnmodified }
func (o *ReplaceIfUnmodifiedOp) ResponseOpCode() byte  { return codec.OpReplaceIfUnmodifiedResponse }
func (o *ReplaceIfUnmodifiedOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *ReplaceIfUnmodifiedOp) Flags() int32          { return o.OpFlags }
func (o *ReplaceIfUnmodifiedOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *ReplaceIfUnmodifiedOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *ReplaceIfUnmodifiedOp) KeyBytes() []byte      { return o.Key }

func (o *ReplaceIfUnmodifiedOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	tu := codec.EncodeTimeUnits(o.Lifespan, o.MaxIdle)
	if err := tu.Write(w); err != nil {
		return err
	}
	if err := codec.WriteLong(w, o.Version); err != nil {
		return err
	}
	return codec.WriteLPBytes(w, o.Value)
}

func (o *ReplaceIfUnmodifiedOp) DecodeResponse(status byte, r io.Reader) (any, error) {
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

type CASResponse struct {
	Success       bool
	PreviousValue []byte
}
