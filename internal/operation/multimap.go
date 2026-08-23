package operation

import (
	"io"
	"time"

	"infinispan.org/go-client/internal/codec"
)

func writeSupportsDuplicates(w io.Writer, v bool) error {
	if v {
		return codec.WriteU1(w, 1)
	}
	return codec.WriteU1(w, 0)
}

func decodeBoolResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusKeyDoesNotExist {
		return false, nil
	}
	b, err := codec.ReadU1(r)
	if err != nil {
		return nil, err
	}
	return b == 1, nil
}

var infiniteTimeUnits = codec.EncodeTimeUnits(-1, -1)

// --- MultimapGetOp ---

type MultimapGetOp struct {
	Cache              string
	Key                []byte
	SupportsDuplicates bool
}

func (o *MultimapGetOp) RequestOpCode() byte   { return codec.OpMultimapGet }
func (o *MultimapGetOp) ResponseOpCode() byte  { return codec.OpMultimapGetResponse }
func (o *MultimapGetOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapGetOp) Flags() int32          { return 0 }
func (o *MultimapGetOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *MultimapGetOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *MultimapGetOp) KeyBytes() []byte      { return o.Key }

func (o *MultimapGetOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapGetOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusKeyDoesNotExist {
		return [][]byte{}, nil
	}
	count, err := codec.ReadVInt(r)
	if err != nil {
		return nil, err
	}
	values := make([][]byte, count)
	for i := int32(0); i < count; i++ {
		v, err := codec.ReadLPBytes(r)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	return values, nil
}

// --- MultimapGetWithMetadataOp ---

type MultimapGetWithMetadataResponse struct {
	Values   [][]byte
	Metadata *codec.EntryMetadata
	Found    bool
}

type MultimapGetWithMetadataOp struct {
	Cache              string
	Key                []byte
	SupportsDuplicates bool
}

func (o *MultimapGetWithMetadataOp) RequestOpCode() byte { return codec.OpMultimapGetWithMetadata }
func (o *MultimapGetWithMetadataOp) ResponseOpCode() byte {
	return codec.OpMultimapGetWithMetadataResponse
}
func (o *MultimapGetWithMetadataOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapGetWithMetadataOp) Flags() int32          { return 0 }
func (o *MultimapGetWithMetadataOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *MultimapGetWithMetadataOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *MultimapGetWithMetadataOp) KeyBytes() []byte      { return o.Key }

func (o *MultimapGetWithMetadataOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapGetWithMetadataOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusKeyDoesNotExist {
		return &MultimapGetWithMetadataResponse{Found: false}, nil
	}
	md, _, err := codec.ReadMetadata(r)
	if err != nil {
		return nil, err
	}
	count, err := codec.ReadVInt(r)
	if err != nil {
		return nil, err
	}
	values := make([][]byte, count)
	for i := int32(0); i < count; i++ {
		v, err := codec.ReadLPBytes(r)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	return &MultimapGetWithMetadataResponse{Values: values, Metadata: md, Found: true}, nil
}

// --- MultimapPutOp ---

type MultimapPutOp struct {
	Cache              string
	Key                []byte
	Value              []byte
	Lifespan           time.Duration
	MaxIdle            time.Duration
	SupportsDuplicates bool
}

func (o *MultimapPutOp) RequestOpCode() byte   { return codec.OpMultimapPut }
func (o *MultimapPutOp) ResponseOpCode() byte  { return codec.OpMultimapPutResponse }
func (o *MultimapPutOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapPutOp) Flags() int32          { return 0 }
func (o *MultimapPutOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *MultimapPutOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *MultimapPutOp) KeyBytes() []byte      { return o.Key }

func (o *MultimapPutOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	tu := codec.EncodeTimeUnits(o.Lifespan, o.MaxIdle)
	if err := tu.Write(w); err != nil {
		return err
	}
	if err := codec.WriteLPBytes(w, o.Value); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapPutOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}

// --- MultimapRemoveKeyOp ---

type MultimapRemoveKeyOp struct {
	Cache              string
	Key                []byte
	SupportsDuplicates bool
}

func (o *MultimapRemoveKeyOp) RequestOpCode() byte   { return codec.OpMultimapRemoveKey }
func (o *MultimapRemoveKeyOp) ResponseOpCode() byte  { return codec.OpMultimapRemoveKeyResponse }
func (o *MultimapRemoveKeyOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapRemoveKeyOp) Flags() int32          { return 0 }
func (o *MultimapRemoveKeyOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *MultimapRemoveKeyOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *MultimapRemoveKeyOp) KeyBytes() []byte      { return o.Key }

func (o *MultimapRemoveKeyOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapRemoveKeyOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	return decodeBoolResponse(status, r)
}

// --- MultimapRemoveEntryOp ---

type MultimapRemoveEntryOp struct {
	Cache              string
	Key                []byte
	Value              []byte
	SupportsDuplicates bool
}

func (o *MultimapRemoveEntryOp) RequestOpCode() byte   { return codec.OpMultimapRemoveEntry }
func (o *MultimapRemoveEntryOp) ResponseOpCode() byte  { return codec.OpMultimapRemoveEntryResponse }
func (o *MultimapRemoveEntryOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapRemoveEntryOp) Flags() int32          { return 0 }
func (o *MultimapRemoveEntryOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *MultimapRemoveEntryOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *MultimapRemoveEntryOp) KeyBytes() []byte      { return o.Key }

func (o *MultimapRemoveEntryOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	if err := infiniteTimeUnits.Write(w); err != nil {
		return err
	}
	if err := codec.WriteLPBytes(w, o.Value); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapRemoveEntryOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	return decodeBoolResponse(status, r)
}

// --- MultimapSizeOp ---

type MultimapSizeOp struct {
	Cache              string
	SupportsDuplicates bool
}

func (o *MultimapSizeOp) RequestOpCode() byte   { return codec.OpMultimapSize }
func (o *MultimapSizeOp) ResponseOpCode() byte  { return codec.OpMultimapSizeResponse }
func (o *MultimapSizeOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapSizeOp) Flags() int32          { return 0 }
func (o *MultimapSizeOp) KeyMediaType() int32   { return 0 }
func (o *MultimapSizeOp) ValueMediaType() int32 { return 0 }

func (o *MultimapSizeOp) WriteBody(w io.Writer) error {
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapSizeOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	return codec.ReadVLong(r)
}

// --- MultimapContainsEntryOp ---

type MultimapContainsEntryOp struct {
	Cache              string
	Key                []byte
	Value              []byte
	SupportsDuplicates bool
}

func (o *MultimapContainsEntryOp) RequestOpCode() byte   { return codec.OpMultimapContainsEntry }
func (o *MultimapContainsEntryOp) ResponseOpCode() byte  { return codec.OpMultimapContainsEntryResponse }
func (o *MultimapContainsEntryOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapContainsEntryOp) Flags() int32          { return 0 }
func (o *MultimapContainsEntryOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *MultimapContainsEntryOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *MultimapContainsEntryOp) KeyBytes() []byte      { return o.Key }

func (o *MultimapContainsEntryOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	if err := infiniteTimeUnits.Write(w); err != nil {
		return err
	}
	if err := codec.WriteLPBytes(w, o.Value); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapContainsEntryOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	return decodeBoolResponse(status, r)
}

// --- MultimapContainsKeyOp ---

type MultimapContainsKeyOp struct {
	Cache              string
	Key                []byte
	SupportsDuplicates bool
}

func (o *MultimapContainsKeyOp) RequestOpCode() byte   { return codec.OpMultimapContainsKey }
func (o *MultimapContainsKeyOp) ResponseOpCode() byte  { return codec.OpMultimapContainsKeyResponse }
func (o *MultimapContainsKeyOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapContainsKeyOp) Flags() int32          { return 0 }
func (o *MultimapContainsKeyOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *MultimapContainsKeyOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }
func (o *MultimapContainsKeyOp) KeyBytes() []byte      { return o.Key }

func (o *MultimapContainsKeyOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.Key); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapContainsKeyOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	return decodeBoolResponse(status, r)
}

// --- MultimapContainsValueOp ---

type MultimapContainsValueOp struct {
	Cache              string
	Value              []byte
	SupportsDuplicates bool
}

func (o *MultimapContainsValueOp) RequestOpCode() byte   { return codec.OpMultimapContainsValue }
func (o *MultimapContainsValueOp) ResponseOpCode() byte  { return codec.OpMultimapContainsValueResponse }
func (o *MultimapContainsValueOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *MultimapContainsValueOp) Flags() int32          { return 0 }
func (o *MultimapContainsValueOp) KeyMediaType() int32   { return 0 }
func (o *MultimapContainsValueOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }

func (o *MultimapContainsValueOp) WriteBody(w io.Writer) error {
	if err := infiniteTimeUnits.Write(w); err != nil {
		return err
	}
	if err := codec.WriteLPBytes(w, o.Value); err != nil {
		return err
	}
	return writeSupportsDuplicates(w, o.SupportsDuplicates)
}

func (o *MultimapContainsValueOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	return decodeBoolResponse(status, r)
}
