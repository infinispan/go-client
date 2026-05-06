package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type ExecParam struct {
	Name  string
	Value []byte
}

type ExecOp struct {
	TaskName string
	Params   []ExecParam
}

func (o *ExecOp) RequestOpCode() byte  { return codec.OpExec }
func (o *ExecOp) ResponseOpCode() byte { return codec.OpExecResponse }
func (o *ExecOp) CacheName() []byte    { return []byte{} }
func (o *ExecOp) Flags() int32         { return 0 }
func (o *ExecOp) KeyMediaType() int32  { return 0 }
func (o *ExecOp) ValueMediaType() int32 { return 0 }

func (o *ExecOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, o.TaskName); err != nil {
		return err
	}
	if err := codec.WriteVInt(w, int32(len(o.Params))); err != nil {
		return err
	}
	for _, p := range o.Params {
		if err := codec.WriteLPString(w, p.Name); err != nil {
			return err
		}
		if err := codec.WriteLPBytes(w, p.Value); err != nil {
			return err
		}
	}
	return nil
}

func (o *ExecOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	return codec.ReadLPBytes(r)
}
