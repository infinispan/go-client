package operation

import (
	"fmt"
	"io"

	"infinispan.org/go-client/internal/codec"
)

type IterationStartOp struct {
	Cache           string
	BatchSize       int32
	IncludeMetadata bool
}

func (o *IterationStartOp) RequestOpCode() byte   { return codec.OpIterationStart }
func (o *IterationStartOp) ResponseOpCode() byte  { return codec.OpIterationStartResponse }
func (o *IterationStartOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *IterationStartOp) Flags() int32          { return 0 }
func (o *IterationStartOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *IterationStartOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }

func (o *IterationStartOp) WriteBody(w io.Writer) error {
	if err := codec.WriteSignedVInt(w, -1); err != nil {
		return err
	}
	if err := codec.WriteSignedVInt(w, -1); err != nil {
		return err
	}
	if err := codec.WriteVInt(w, o.BatchSize); err != nil {
		return err
	}
	meta := byte(0)
	if o.IncludeMetadata {
		meta = 1
	}
	return codec.WriteU1(w, meta)
}

func (o *IterationStartOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	id, err := codec.ReadLPBytes(r)
	if err != nil {
		return nil, fmt.Errorf("read iteration id: %w", err)
	}
	return string(id), nil
}

type IterEntry struct {
	Key      []byte
	Value    []byte
	Metadata *codec.EntryMetadata
}

type IterationNextResponse struct {
	Entries []IterEntry
	HasMore bool
}

type IterationNextOp struct {
	Cache           string
	IterationID     string
	IncludeMetadata bool
}

func (o *IterationNextOp) RequestOpCode() byte   { return codec.OpIterationNext }
func (o *IterationNextOp) ResponseOpCode() byte  { return codec.OpIterationNextResponse }
func (o *IterationNextOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *IterationNextOp) Flags() int32          { return 0 }
func (o *IterationNextOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *IterationNextOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }

func (o *IterationNextOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, []byte(o.IterationID))
}

func (o *IterationNextOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status == codec.StatusInvalidIteration {
		return nil, fmt.Errorf("invalid iteration: server cursor expired or unknown")
	}

	if _, err := codec.ReadLPBytes(r); err != nil {
		return nil, fmt.Errorf("read finished segments: %w", err)
	}

	entriesSize, err := codec.ReadVInt(r)
	if err != nil {
		return nil, fmt.Errorf("read entries size: %w", err)
	}
	if entriesSize == 0 {
		return &IterationNextResponse{HasMore: false}, nil
	}

	if _, err := codec.ReadVInt(r); err != nil {
		return nil, fmt.Errorf("read projections size: %w", err)
	}

	entries := make([]IterEntry, entriesSize)
	for i := int32(0); i < entriesSize; i++ {
		metaFlag, err := codec.ReadU1(r)
		if err != nil {
			return nil, fmt.Errorf("read entry %d meta flag: %w", i, err)
		}

		var md *codec.EntryMetadata
		if metaFlag == 1 {
			md, err = readIterMetadata(r)
			if err != nil {
				return nil, fmt.Errorf("read entry %d metadata: %w", i, err)
			}
		}

		key, err := codec.ReadLPBytes(r)
		if err != nil {
			return nil, fmt.Errorf("read entry %d key: %w", i, err)
		}
		value, err := codec.ReadLPBytes(r)
		if err != nil {
			return nil, fmt.Errorf("read entry %d value: %w", i, err)
		}

		entries[i] = IterEntry{Key: key, Value: value, Metadata: md}
	}

	return &IterationNextResponse{Entries: entries, HasMore: true}, nil
}

func readIterMetadata(r io.Reader) (*codec.EntryMetadata, error) {
	flags, err := codec.ReadU1(r)
	if err != nil {
		return nil, err
	}
	var md codec.EntryMetadata
	if flags&0x01 == 0 {
		if md.Created, err = codec.ReadLong(r); err != nil {
			return nil, err
		}
		if md.Lifespan, err = codec.ReadVInt(r); err != nil {
			return nil, err
		}
	} else {
		md.Lifespan = -1
	}
	if flags&0x02 == 0 {
		if md.LastUsed, err = codec.ReadLong(r); err != nil {
			return nil, err
		}
		if md.MaxIdle, err = codec.ReadVInt(r); err != nil {
			return nil, err
		}
	} else {
		md.MaxIdle = -1
	}
	if md.Version, err = codec.ReadLong(r); err != nil {
		return nil, err
	}
	return &md, nil
}

type IterationEndOp struct {
	Cache       string
	IterationID string
}

func (o *IterationEndOp) RequestOpCode() byte   { return codec.OpIterationEnd }
func (o *IterationEndOp) ResponseOpCode() byte  { return codec.OpIterationEndResponse }
func (o *IterationEndOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *IterationEndOp) Flags() int32          { return 0 }
func (o *IterationEndOp) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *IterationEndOp) ValueMediaType() int32 { return codec.MediaIDOctetStream }

func (o *IterationEndOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, []byte(o.IterationID))
}

func (o *IterationEndOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}
