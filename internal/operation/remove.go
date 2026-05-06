package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type RemoveOp struct {
	Cache     string
	Key       []byte
	MediaType int32
	OpFlags   int32
}

type RemoveResponse struct {
	Existed       bool
	PreviousValue []byte
}

func (r *RemoveOp) RequestOpCode() byte  { return codec.OpRemove }
func (r *RemoveOp) ResponseOpCode() byte { return codec.OpRemoveResponse }
func (r *RemoveOp) CacheName() []byte    { return []byte(r.Cache) }
func (r *RemoveOp) Flags() int32         { return r.OpFlags }

func (r *RemoveOp) KeyMediaType() int32 {
	if r.MediaType != 0 {
		return r.MediaType
	}
	return codec.MediaIDOctetStream
}

func (r *RemoveOp) ValueMediaType() int32 {
	if r.MediaType != 0 {
		return r.MediaType
	}
	return codec.MediaIDOctetStream
}

func (r *RemoveOp) KeyBytes() []byte { return r.Key }

func (r *RemoveOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, r.Key)
}

func (r *RemoveOp) DecodeResponse(status byte, r2 io.Reader) (any, error) {
	if status == codec.StatusSuccessWithPrevious {
		prev, err := codec.ReadMetadataValue(r2)
		if err != nil {
			return nil, err
		}
		return &RemoveResponse{Existed: true, PreviousValue: prev}, nil
	}
	return &RemoveResponse{Existed: status != codec.StatusKeyDoesNotExist}, nil
}
