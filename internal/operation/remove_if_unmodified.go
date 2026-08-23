package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type RemoveIfUnmodifiedOp struct {
	Cache   string
	Key     []byte
	Version int64
	OpFlags int32
}

func (o *RemoveIfUnmodifiedOp) RequestOpCode() byte   { return codec.OpRemoveIfUnmodified }
func (o *RemoveIfUnmodifiedOp) ResponseOpCode() byte  { return codec.OpRemoveIfUnmodifiedResponse }
func (o *RemoveIfUnmodifiedOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *RemoveIfUnmodifiedOp) Flags() int32          { return o.OpFlags }
func (o *RemoveIfUnmodifiedOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *RemoveIfUnmodifiedOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *RemoveIfUnmodifiedOp) KeyBytes() []byte      { return o.Key }

func (o *RemoveIfUnmodifiedOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	return codec.WriteLong(w, o.Version)
}

func (o *RemoveIfUnmodifiedOp) DecodeResponse(status byte, r io.Reader) (any, error) {
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
