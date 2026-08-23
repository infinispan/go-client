package operation

import (
	"fmt"
	"io"

	"infinispan.org/go-client/internal/codec"
)

// CounterType distinguishes strong and weak counters.
type CounterType byte

const (
	CounterStrong CounterType = 0
	CounterWeak   CounterType = 1
)

// CounterStorage controls whether a counter survives restarts.
type CounterStorage byte

const (
	StorageVolatile   CounterStorage = 0
	StoragePersistent CounterStorage = 1
)

// CounterState represents the state of a counter value relative to its bounds.
type CounterState byte

const (
	CounterStateValid      CounterState = 0
	CounterStateLowerBound CounterState = 1
	CounterStateUpperBound CounterState = 2
)

// CounterConfiguration holds the configuration of a counter.
type CounterConfiguration struct {
	Type             CounterType
	Bounded          bool
	Storage          CounterStorage
	ConcurrencyLevel int32
	LowerBound       int64
	UpperBound       int64
	InitialValue     int64
}

func (cc *CounterConfiguration) encode(w io.Writer) error {
	var flags byte
	if cc.Type == CounterWeak {
		flags |= 0x01
	}
	if cc.Bounded {
		flags |= 0x02
	}
	if cc.Storage == StoragePersistent {
		flags |= 0x04
	}
	if err := codec.WriteU1(w, flags); err != nil {
		return err
	}
	if cc.Type == CounterWeak {
		if err := codec.WriteVInt(w, cc.ConcurrencyLevel); err != nil {
			return err
		}
	}
	if cc.Bounded {
		if err := codec.WriteLong(w, cc.LowerBound); err != nil {
			return err
		}
		if err := codec.WriteLong(w, cc.UpperBound); err != nil {
			return err
		}
	}
	return codec.WriteLong(w, cc.InitialValue)
}

func decodeCounterConfiguration(r io.Reader) (*CounterConfiguration, error) {
	flags, err := codec.ReadU1(r)
	if err != nil {
		return nil, err
	}
	cc := &CounterConfiguration{}
	if flags&0x01 != 0 {
		cc.Type = CounterWeak
	}
	cc.Bounded = flags&0x02 != 0
	if flags&0x04 != 0 {
		cc.Storage = StoragePersistent
	}
	if cc.Type == CounterWeak {
		cc.ConcurrencyLevel, err = codec.ReadVInt(r)
		if err != nil {
			return nil, err
		}
	}
	if cc.Bounded {
		cc.LowerBound, err = codec.ReadLong(r)
		if err != nil {
			return nil, err
		}
		cc.UpperBound, err = codec.ReadLong(r)
		if err != nil {
			return nil, err
		}
	}
	cc.InitialValue, err = codec.ReadLong(r)
	if err != nil {
		return nil, err
	}
	return cc, nil
}

// CounterEvent represents a counter value change event from the server.
type CounterEvent struct {
	Name     string
	OldState CounterState
	NewState CounterState
	OldValue int64
	NewValue int64
}

func DecodeCounterEventBody(r io.Reader) (*CounterEvent, []byte, error) {
	name, err := codec.ReadLPString(r)
	if err != nil {
		return nil, nil, err
	}
	listenerID, err := codec.ReadLPBytes(r)
	if err != nil {
		return nil, nil, err
	}
	stateFlags, err := codec.ReadU1(r)
	if err != nil {
		return nil, nil, err
	}
	oldValue, err := codec.ReadLong(r)
	if err != nil {
		return nil, nil, err
	}
	newValue, err := codec.ReadLong(r)
	if err != nil {
		return nil, nil, err
	}
	return &CounterEvent{
		Name:     name,
		OldState: CounterState(stateFlags & 0x03),
		NewState: CounterState((stateFlags >> 2) & 0x03),
		OldValue: oldValue,
		NewValue: newValue,
	}, listenerID, nil
}

func SkipCounterEventBody(r io.Reader) {
	_, _ = codec.ReadLPString(r)
	_, _ = codec.ReadLPBytes(r)
	_, _ = codec.ReadU1(r)
	_, _ = codec.ReadLong(r)
	_, _ = codec.ReadLong(r)
}

// --- CounterCreateOp ---

type CounterCreateOp struct {
	Name   string
	Config *CounterConfiguration
}

func (o *CounterCreateOp) RequestOpCode() byte   { return codec.OpCounterCreate }
func (o *CounterCreateOp) ResponseOpCode() byte  { return codec.OpCounterCreateResponse }
func (o *CounterCreateOp) CacheName() []byte     { return []byte{} }
func (o *CounterCreateOp) Flags() int32          { return 0 }
func (o *CounterCreateOp) KeyMediaType() int32   { return 0 }
func (o *CounterCreateOp) ValueMediaType() int32 { return 0 }

func (o *CounterCreateOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, o.Name); err != nil {
		return err
	}
	return o.Config.encode(w)
}

func (o *CounterCreateOp) DecodeResponse(status byte, _ io.Reader) (any, error) {
	return status == codec.StatusSuccess, nil
}

// --- CounterGetConfigurationOp ---

type CounterGetConfigurationOp struct {
	Name string
}

func (o *CounterGetConfigurationOp) RequestOpCode() byte { return codec.OpCounterGetConfiguration }
func (o *CounterGetConfigurationOp) ResponseOpCode() byte {
	return codec.OpCounterGetConfigurationResponse
}
func (o *CounterGetConfigurationOp) CacheName() []byte     { return []byte{} }
func (o *CounterGetConfigurationOp) Flags() int32          { return 0 }
func (o *CounterGetConfigurationOp) KeyMediaType() int32   { return 0 }
func (o *CounterGetConfigurationOp) ValueMediaType() int32 { return 0 }

func (o *CounterGetConfigurationOp) WriteBody(w io.Writer) error {
	return codec.WriteLPString(w, o.Name)
}

func (o *CounterGetConfigurationOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status != codec.StatusSuccess {
		return nil, nil
	}
	return decodeCounterConfiguration(r)
}

// --- CounterIsDefinedOp ---

type CounterIsDefinedOp struct {
	Name string
}

func (o *CounterIsDefinedOp) RequestOpCode() byte   { return codec.OpCounterIsDefined }
func (o *CounterIsDefinedOp) ResponseOpCode() byte  { return codec.OpCounterIsDefinedResponse }
func (o *CounterIsDefinedOp) CacheName() []byte     { return []byte{} }
func (o *CounterIsDefinedOp) Flags() int32          { return 0 }
func (o *CounterIsDefinedOp) KeyMediaType() int32   { return 0 }
func (o *CounterIsDefinedOp) ValueMediaType() int32 { return 0 }

func (o *CounterIsDefinedOp) WriteBody(w io.Writer) error {
	return codec.WriteLPString(w, o.Name)
}

func (o *CounterIsDefinedOp) DecodeResponse(status byte, _ io.Reader) (any, error) {
	return status == codec.StatusSuccess, nil
}

// --- CounterAddAndGetOp ---

type CounterAddAndGetOp struct {
	Name  string
	Delta int64
}

func (o *CounterAddAndGetOp) RequestOpCode() byte   { return codec.OpCounterAddAndGet }
func (o *CounterAddAndGetOp) ResponseOpCode() byte  { return codec.OpCounterAddAndGetResponse }
func (o *CounterAddAndGetOp) CacheName() []byte     { return []byte{} }
func (o *CounterAddAndGetOp) Flags() int32          { return 0 }
func (o *CounterAddAndGetOp) KeyMediaType() int32   { return 0 }
func (o *CounterAddAndGetOp) ValueMediaType() int32 { return 0 }

func (o *CounterAddAndGetOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, o.Name); err != nil {
		return err
	}
	return codec.WriteLong(w, o.Delta)
}

func (o *CounterAddAndGetOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	// When successful, read and return the new counter value
	if status == codec.StatusSuccess {
		return codec.ReadLong(r)
	}
	// Status 0x04 means counter bound was reached - operation rejected
	if status == codec.StatusCounterBoundReached {
		return nil, fmt.Errorf("counter bound reached")
	}
	// For other non-success statuses, return nil (will be handled by caller)
	return nil, nil
}

// --- CounterResetOp ---

type CounterResetOp struct {
	Name string
}

func (o *CounterResetOp) RequestOpCode() byte   { return codec.OpCounterReset }
func (o *CounterResetOp) ResponseOpCode() byte  { return codec.OpCounterResetResponse }
func (o *CounterResetOp) CacheName() []byte     { return []byte{} }
func (o *CounterResetOp) Flags() int32          { return 0 }
func (o *CounterResetOp) KeyMediaType() int32   { return 0 }
func (o *CounterResetOp) ValueMediaType() int32 { return 0 }

func (o *CounterResetOp) WriteBody(w io.Writer) error {
	return codec.WriteLPString(w, o.Name)
}

func (o *CounterResetOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}

// --- CounterGetOp ---

type CounterGetOp struct {
	Name string
}

func (o *CounterGetOp) RequestOpCode() byte   { return codec.OpCounterGet }
func (o *CounterGetOp) ResponseOpCode() byte  { return codec.OpCounterGetResponse }
func (o *CounterGetOp) CacheName() []byte     { return []byte{} }
func (o *CounterGetOp) Flags() int32          { return 0 }
func (o *CounterGetOp) KeyMediaType() int32   { return 0 }
func (o *CounterGetOp) ValueMediaType() int32 { return 0 }

func (o *CounterGetOp) WriteBody(w io.Writer) error {
	return codec.WriteLPString(w, o.Name)
}

func (o *CounterGetOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status != codec.StatusSuccess {
		return nil, nil
	}
	return codec.ReadLong(r)
}

// --- CounterCASResponse ---

type CounterCASResult struct {
	Success  bool
	OldValue int64
}

// --- CounterCasOp ---

type CounterCasOp struct {
	Name   string
	Expect int64
	Update int64
}

func (o *CounterCasOp) RequestOpCode() byte   { return codec.OpCounterCAS }
func (o *CounterCasOp) ResponseOpCode() byte  { return codec.OpCounterCASResponse }
func (o *CounterCasOp) CacheName() []byte     { return []byte{} }
func (o *CounterCasOp) Flags() int32          { return 0 }
func (o *CounterCasOp) KeyMediaType() int32   { return 0 }
func (o *CounterCasOp) ValueMediaType() int32 { return 0 }

func (o *CounterCasOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, o.Name); err != nil {
		return err
	}
	if err := codec.WriteLong(w, o.Expect); err != nil {
		return err
	}
	return codec.WriteLong(w, o.Update)
}

func (o *CounterCasOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status != codec.StatusSuccess {
		return nil, nil
	}
	val, err := codec.ReadLong(r)
	if err != nil {
		return nil, err
	}
	return &CounterCASResult{
		Success:  val == o.Expect,
		OldValue: val,
	}, nil
}

// --- CounterGetAndSetOp ---

type CounterGetAndSetOp struct {
	Name  string
	Value int64
}

func (o *CounterGetAndSetOp) RequestOpCode() byte   { return codec.OpCounterGetAndSet }
func (o *CounterGetAndSetOp) ResponseOpCode() byte  { return codec.OpCounterGetAndSetResponse }
func (o *CounterGetAndSetOp) CacheName() []byte     { return []byte{} }
func (o *CounterGetAndSetOp) Flags() int32          { return 0 }
func (o *CounterGetAndSetOp) KeyMediaType() int32   { return 0 }
func (o *CounterGetAndSetOp) ValueMediaType() int32 { return 0 }

func (o *CounterGetAndSetOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, o.Name); err != nil {
		return err
	}
	return codec.WriteLong(w, o.Value)
}

func (o *CounterGetAndSetOp) DecodeResponse(status byte, r io.Reader) (any, error) {
	if status != codec.StatusSuccess {
		return nil, nil
	}
	return codec.ReadLong(r)
}

// --- CounterRemoveOp ---

type CounterRemoveOp struct {
	Name string
}

func (o *CounterRemoveOp) RequestOpCode() byte   { return codec.OpCounterRemove }
func (o *CounterRemoveOp) ResponseOpCode() byte  { return codec.OpCounterRemoveResponse }
func (o *CounterRemoveOp) CacheName() []byte     { return []byte{} }
func (o *CounterRemoveOp) Flags() int32          { return 0 }
func (o *CounterRemoveOp) KeyMediaType() int32   { return 0 }
func (o *CounterRemoveOp) ValueMediaType() int32 { return 0 }

func (o *CounterRemoveOp) WriteBody(w io.Writer) error {
	return codec.WriteLPString(w, o.Name)
}

func (o *CounterRemoveOp) DecodeResponse(status byte, _ io.Reader) (any, error) {
	// Return true if removal was successful
	return status == codec.StatusSuccess, nil
}

// --- CounterAddListenerOp ---

type CounterAddListenerOp struct {
	Name       string
	ListenerID []byte
}

func (o *CounterAddListenerOp) RequestOpCode() byte   { return codec.OpCounterAddListener }
func (o *CounterAddListenerOp) ResponseOpCode() byte  { return codec.OpCounterAddListenerResponse }
func (o *CounterAddListenerOp) CacheName() []byte     { return []byte{} }
func (o *CounterAddListenerOp) Flags() int32          { return 0 }
func (o *CounterAddListenerOp) KeyMediaType() int32   { return 0 }
func (o *CounterAddListenerOp) ValueMediaType() int32 { return 0 }

func (o *CounterAddListenerOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, o.Name); err != nil {
		return err
	}
	return codec.WriteLPBytes(w, o.ListenerID)
}

func (o *CounterAddListenerOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}

// --- CounterRemoveListenerOp ---

type CounterRemoveListenerOp struct {
	Name       string
	ListenerID []byte
}

func (o *CounterRemoveListenerOp) RequestOpCode() byte   { return codec.OpCounterRemoveListener }
func (o *CounterRemoveListenerOp) ResponseOpCode() byte  { return codec.OpCounterRemoveListenerResponse }
func (o *CounterRemoveListenerOp) CacheName() []byte     { return []byte{} }
func (o *CounterRemoveListenerOp) Flags() int32          { return 0 }
func (o *CounterRemoveListenerOp) KeyMediaType() int32   { return 0 }
func (o *CounterRemoveListenerOp) ValueMediaType() int32 { return 0 }

func (o *CounterRemoveListenerOp) WriteBody(w io.Writer) error {
	if err := codec.WriteLPString(w, o.Name); err != nil {
		return err
	}
	return codec.WriteLPBytes(w, o.ListenerID)
}

func (o *CounterRemoveListenerOp) DecodeResponse(_ byte, _ io.Reader) (any, error) {
	return nil, nil
}

// --- CounterGetNamesOp ---

type CounterGetNamesOp struct{}

func (o *CounterGetNamesOp) RequestOpCode() byte   { return codec.OpCounterGetNames }
func (o *CounterGetNamesOp) ResponseOpCode() byte  { return codec.OpCounterGetNamesResponse }
func (o *CounterGetNamesOp) CacheName() []byte     { return []byte{} }
func (o *CounterGetNamesOp) Flags() int32          { return 0 }
func (o *CounterGetNamesOp) KeyMediaType() int32   { return 0 }
func (o *CounterGetNamesOp) ValueMediaType() int32 { return 0 }

func (o *CounterGetNamesOp) WriteBody(_ io.Writer) error {
	return nil
}

func (o *CounterGetNamesOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	count, err := codec.ReadVInt(r)
	if err != nil {
		return nil, err
	}
	names := make([]string, count)
	for i := int32(0); i < count; i++ {
		names[i], err = codec.ReadLPString(r)
		if err != nil {
			return nil, err
		}
	}
	return names, nil
}
