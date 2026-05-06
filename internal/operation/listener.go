package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type EventType byte

const (
	EventCreated  EventType = iota
	EventModified
	EventRemoved
	EventExpired
)

func (t EventType) String() string {
	switch t {
	case EventCreated:
		return "created"
	case EventModified:
		return "modified"
	case EventRemoved:
		return "removed"
	case EventExpired:
		return "expired"
	default:
		return "unknown"
	}
}

const (
	InterestCreated  byte = 0x01
	InterestModified byte = 0x02
	InterestRemoved  byte = 0x04
	InterestExpired  byte = 0x08
	InterestAll      byte = 0x0F
)

type CacheEntryEvent struct {
	Type    EventType
	Key     []byte
	Version int64
}

type CustomEvent struct {
	Data []byte
}

type AddClientListenerOp struct {
	Cache            string
	ListenerID       []byte
	IncludeState     bool
	Interests        byte
	FilterFactory    string
	ConverterFactory string
	FilterParams     [][]byte
	ConverterParams  [][]byte
}

func (o *AddClientListenerOp) RequestOpCode() byte  { return codec.OpAddClientListener }
func (o *AddClientListenerOp) ResponseOpCode() byte { return codec.OpAddClientListenerResponse }
func (o *AddClientListenerOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *AddClientListenerOp) Flags() int32          { return 0 }
func (o *AddClientListenerOp) KeyMediaType() int32   { return 0 }
func (o *AddClientListenerOp) ValueMediaType() int32 { return 0 }

func (o *AddClientListenerOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPBytes(w, o.ListenerID); err != nil {
		return err
	}
	var state byte
	if o.IncludeState {
		state = 1
	}
	if err := codec.WriteU1(w, state); err != nil {
		return err
	}
	if err := codec.WriteLPString(w, o.FilterFactory); err != nil {
		return err
	}
	if o.FilterFactory != "" {
		if err := codec.WriteU1(w, byte(len(o.FilterParams))); err != nil {
			return err
		}
		for _, p := range o.FilterParams {
			if err := codec.WriteLPBytes(w, p); err != nil {
				return err
			}
		}
	}
	if err := codec.WriteLPString(w, o.ConverterFactory); err != nil {
		return err
	}
	if o.ConverterFactory != "" {
		if err := codec.WriteU1(w, byte(len(o.ConverterParams))); err != nil {
			return err
		}
		for _, p := range o.ConverterParams {
			if err := codec.WriteLPBytes(w, p); err != nil {
				return err
			}
		}
	}
	// useRawData
	if err := codec.WriteU1(w, 1); err != nil {
		return err
	}
	interests := o.Interests
	if interests == 0 {
		interests = InterestAll
	}
	return codec.WriteVInt(w, int32(interests))
}

func (o *AddClientListenerOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}

type RemoveClientListenerOp struct {
	Cache      string
	ListenerID []byte
}

func (o *RemoveClientListenerOp) RequestOpCode() byte  { return codec.OpRemoveClientListener }
func (o *RemoveClientListenerOp) ResponseOpCode() byte { return codec.OpRemoveClientListenerResponse }
func (o *RemoveClientListenerOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *RemoveClientListenerOp) Flags() int32          { return 0 }
func (o *RemoveClientListenerOp) KeyMediaType() int32   { return 0 }
func (o *RemoveClientListenerOp) ValueMediaType() int32 { return 0 }

func (o *RemoveClientListenerOp) WriteBody(w io.Writer) error {
	return codec.WriteLPBytes(w, o.ListenerID)
}

func (o *RemoveClientListenerOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}

func ReadEventListenerID(r io.Reader) ([]byte, error) {
	return codec.ReadLPBytes(r)
}

func DecodeEventBody(opCode byte, r io.Reader) (any, error) {
	isCustom, err := codec.ReadU1(r)
	if err != nil {
		return nil, err
	}
	if _, err := codec.ReadU1(r); err != nil {
		return nil, err
	}

	if isCustom != 0 {
		data, err := codec.ReadLPBytes(r)
		if err != nil {
			return nil, err
		}
		return &CustomEvent{Data: data}, nil
	}

	var evType EventType
	switch opCode {
	case codec.OpCacheEntryCreated:
		evType = EventCreated
	case codec.OpCacheEntryModified:
		evType = EventModified
	case codec.OpCacheEntryRemoved:
		evType = EventRemoved
	case codec.OpCacheEntryExpired:
		evType = EventExpired
	}

	key, err := codec.ReadLPBytes(r)
	if err != nil {
		return nil, err
	}

	var version int64
	if evType == EventCreated || evType == EventModified {
		version, err = codec.ReadLong(r)
		if err != nil {
			return nil, err
		}
	}

	return &CacheEntryEvent{
		Type:    evType,
		Key:     key,
		Version: version,
	}, nil
}

func SkipEventBody(opCode byte, r io.Reader) {
	isCustom, err := codec.ReadU1(r)
	if err != nil {
		return
	}
	if _, err := codec.ReadU1(r); err != nil {
		return
	}
	if isCustom != 0 {
		codec.ReadLPBytes(r)
		return
	}
	if _, err := codec.ReadLPBytes(r); err != nil {
		return
	}
	var evType EventType
	switch opCode {
	case codec.OpCacheEntryCreated:
		evType = EventCreated
	case codec.OpCacheEntryModified:
		evType = EventModified
	}
	if evType == EventCreated || evType == EventModified {
		codec.ReadLong(r)
	}
}

func DecodeEvent(opCode byte, r io.Reader) (any, error) {
	if _, err := ReadEventListenerID(r); err != nil {
		return nil, err
	}
	return DecodeEventBody(opCode, r)
}
