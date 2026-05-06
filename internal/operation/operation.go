package operation

import (
	"io"
)

type Operation interface {
	RequestOpCode() byte
	ResponseOpCode() byte
	CacheName() []byte
	Flags() int32
	KeyMediaType() int32
	ValueMediaType() int32
	WriteBody(w io.Writer) error
	DecodeResponse(status byte, r io.Reader) (any, error)
}

type KeyedOperation interface {
	Operation
	KeyBytes() []byte
}
