package operation

import (
	"io"
	"time"

	"infinispan.org/go-client/internal/codec"
)

type PutOp struct {
	Cache    string
	Key      []byte
	Value    []byte
	Lifespan time.Duration
	MaxIdle  time.Duration
	KeyMT    int32
	ValueMT  int32
	OpFlags  int32
}

type PutResponse struct {
	PreviousValue []byte
}

func (p *PutOp) RequestOpCode() byte  { return codec.OpPut }
func (p *PutOp) ResponseOpCode() byte { return codec.OpPutResponse }
func (p *PutOp) CacheName() []byte    { return []byte(p.Cache) }
func (p *PutOp) Flags() int32         { return p.OpFlags }

func (p *PutOp) KeyMediaType() int32 {
	if p.KeyMT != 0 {
		return p.KeyMT
	}
	return codec.MediaIDOctetStream
}

func (p *PutOp) ValueMediaType() int32 {
	if p.ValueMT != 0 {
		return p.ValueMT
	}
	return codec.MediaIDOctetStream
}

func (p *PutOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, p.Key); err != nil {
		return err
	}
	tu := codec.EncodeTimeUnits(p.Lifespan, p.MaxIdle)
	if err := tu.Write(w); err != nil {
		return err
	}
	return codec.WriteLPBytes(w, p.Value)
}

func (p *PutOp) KeyBytes() []byte { return p.Key }

func (p *PutOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusSuccessWithPrevious || status == codec.StatusNotExecWithPrevious {
		prev, err := codec.ReadMetadataValue(r)
		if err != nil {
			return nil, err
		}
		return &PutResponse{PreviousValue: prev}, nil
	}
	return nil, nil
}
