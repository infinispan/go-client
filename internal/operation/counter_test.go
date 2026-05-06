package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestCounterGetDecodeResponse(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteLong(&buf, 99)

	result, err := (&CounterGetOp{Name: "c1"}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	val := result.(int64)
	if val != 99 {
		t.Errorf("value = %d, want 99", val)
	}
}

func TestCounterGetDecodeNotDefined(t *testing.T) {
	result, err := (&CounterGetOp{Name: "c1"}).DecodeResponse(codec.StatusKeyDoesNotExist, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil for undefined counter, got %v", result)
	}
}

func TestCounterAddAndGetDecodeResponse(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteLong(&buf, 42)

	result, err := (&CounterAddAndGetOp{Name: "c1", Delta: 10}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	val := result.(int64)
	if val != 42 {
		t.Errorf("value = %d, want 42", val)
	}
}

func TestCounterGetAndSetDecodeResponse(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteLong(&buf, 7)

	result, err := (&CounterGetAndSetOp{Name: "c1", Value: 100}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	val := result.(int64)
	if val != 7 {
		t.Errorf("value = %d, want 7", val)
	}
}

func TestCounterCasDecodeSuccess(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteLong(&buf, 10)

	result, err := (&CounterCasOp{Name: "c1", Expect: 10, Update: 20}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	cas := result.(*CounterCASResult)
	if !cas.Success {
		t.Error("expected Success=true when returned value equals Expect")
	}
	if cas.OldValue != 10 {
		t.Errorf("OldValue = %d, want 10", cas.OldValue)
	}
}

func TestCounterCasDecodeFailure(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteLong(&buf, 5)

	result, err := (&CounterCasOp{Name: "c1", Expect: 10, Update: 20}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	cas := result.(*CounterCASResult)
	if cas.Success {
		t.Error("expected Success=false when returned value differs from Expect")
	}
	if cas.OldValue != 5 {
		t.Errorf("OldValue = %d, want 5", cas.OldValue)
	}
}

func TestCounterCreateDecodeCreated(t *testing.T) {
	result, err := (&CounterCreateOp{Name: "c1", Config: &CounterConfiguration{}}).DecodeResponse(codec.StatusSuccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != true {
		t.Error("expected true for new counter")
	}
}

func TestCounterCreateDecodeAlreadyExists(t *testing.T) {
	result, err := (&CounterCreateOp{Name: "c1", Config: &CounterConfiguration{}}).DecodeResponse(codec.StatusNotExecuted, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != false {
		t.Error("expected false for existing counter")
	}
}

func TestCounterIsDefinedDecodeTrue(t *testing.T) {
	result, err := (&CounterIsDefinedOp{Name: "c1"}).DecodeResponse(codec.StatusSuccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != true {
		t.Error("expected true")
	}
}

func TestCounterIsDefinedDecodeFalse(t *testing.T) {
	result, err := (&CounterIsDefinedOp{Name: "c1"}).DecodeResponse(codec.StatusNotExecuted, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != false {
		t.Error("expected false")
	}
}

func TestCounterGetConfigurationDecode(t *testing.T) {
	var buf bytes.Buffer
	// flags: strong(0) | bounded(0x02) | persistent(0x04) = 0x06
	codec.WriteU1(&buf, 0x06)
	// no ConcurrencyLevel (strong counter)
	codec.WriteLong(&buf, -100) // LowerBound
	codec.WriteLong(&buf, 100)  // UpperBound
	codec.WriteLong(&buf, 0)    // InitialValue

	result, err := (&CounterGetConfigurationOp{Name: "c1"}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	cc := result.(*CounterConfiguration)
	if cc.Type != CounterStrong {
		t.Errorf("Type = %d, want Strong(0)", cc.Type)
	}
	if !cc.Bounded {
		t.Error("expected Bounded=true")
	}
	if cc.Storage != StoragePersistent {
		t.Error("expected StoragePersistent")
	}
	if cc.LowerBound != -100 {
		t.Errorf("LowerBound = %d, want -100", cc.LowerBound)
	}
	if cc.UpperBound != 100 {
		t.Errorf("UpperBound = %d, want 100", cc.UpperBound)
	}
	if cc.InitialValue != 0 {
		t.Errorf("InitialValue = %d, want 0", cc.InitialValue)
	}
}

func TestCounterGetConfigurationDecodeWeak(t *testing.T) {
	var buf bytes.Buffer
	// flags: weak(0x01) | volatile(0) = 0x01
	codec.WriteU1(&buf, 0x01)
	codec.WriteVInt(&buf, 16) // ConcurrencyLevel
	codec.WriteLong(&buf, 5) // InitialValue

	result, err := (&CounterGetConfigurationOp{Name: "c1"}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	cc := result.(*CounterConfiguration)
	if cc.Type != CounterWeak {
		t.Errorf("Type = %d, want Weak(1)", cc.Type)
	}
	if cc.Bounded {
		t.Error("expected Bounded=false")
	}
	if cc.Storage != StorageVolatile {
		t.Error("expected StorageVolatile")
	}
	if cc.ConcurrencyLevel != 16 {
		t.Errorf("ConcurrencyLevel = %d, want 16", cc.ConcurrencyLevel)
	}
	if cc.InitialValue != 5 {
		t.Errorf("InitialValue = %d, want 5", cc.InitialValue)
	}
}

func TestCounterGetConfigurationDecodeNotDefined(t *testing.T) {
	result, err := (&CounterGetConfigurationOp{Name: "c1"}).DecodeResponse(codec.StatusKeyDoesNotExist, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil for undefined counter, got %v", result)
	}
}

func TestCounterConfigurationRoundTrip(t *testing.T) {
	original := &CounterConfiguration{
		Type:         CounterWeak,
		Bounded:      true,
		Storage:      StoragePersistent,
		ConcurrencyLevel: 8,
		LowerBound:   -50,
		UpperBound:   50,
		InitialValue: 10,
	}

	var buf bytes.Buffer
	if err := original.encode(&buf); err != nil {
		t.Fatal(err)
	}

	decoded, err := decodeCounterConfiguration(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type = %d, want %d", decoded.Type, original.Type)
	}
	if decoded.Bounded != original.Bounded {
		t.Errorf("Bounded = %v, want %v", decoded.Bounded, original.Bounded)
	}
	if decoded.Storage != original.Storage {
		t.Errorf("Storage = %d, want %d", decoded.Storage, original.Storage)
	}
	if decoded.ConcurrencyLevel != original.ConcurrencyLevel {
		t.Errorf("ConcurrencyLevel = %d, want %d", decoded.ConcurrencyLevel, original.ConcurrencyLevel)
	}
	if decoded.LowerBound != original.LowerBound {
		t.Errorf("LowerBound = %d, want %d", decoded.LowerBound, original.LowerBound)
	}
	if decoded.UpperBound != original.UpperBound {
		t.Errorf("UpperBound = %d, want %d", decoded.UpperBound, original.UpperBound)
	}
	if decoded.InitialValue != original.InitialValue {
		t.Errorf("InitialValue = %d, want %d", decoded.InitialValue, original.InitialValue)
	}
}

func TestDecodeCounterEventBody(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteLPString(&buf, "myCounter")
	codec.WriteLPBytes(&buf, []byte{0x01, 0x02, 0x03})
	// state flags: oldState=0 (Valid), newState=2 (UpperBound) → (2<<2)|0 = 0x08
	codec.WriteU1(&buf, 0x08)
	codec.WriteLong(&buf, 99)  // oldValue
	codec.WriteLong(&buf, 100) // newValue

	ev, listenerID, err := DecodeCounterEventBody(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Name != "myCounter" {
		t.Errorf("Name = %q, want %q", ev.Name, "myCounter")
	}
	if len(listenerID) != 3 || listenerID[0] != 0x01 {
		t.Errorf("unexpected listenerID: %v", listenerID)
	}
	if ev.OldState != CounterStateValid {
		t.Errorf("OldState = %d, want Valid(0)", ev.OldState)
	}
	if ev.NewState != CounterStateUpperBound {
		t.Errorf("NewState = %d, want UpperBound(2)", ev.NewState)
	}
	if ev.OldValue != 99 {
		t.Errorf("OldValue = %d, want 99", ev.OldValue)
	}
	if ev.NewValue != 100 {
		t.Errorf("NewValue = %d, want 100", ev.NewValue)
	}
}

func TestCounterResetDecodeResponse(t *testing.T) {
	result, err := (&CounterResetOp{Name: "c1"}).DecodeResponse(codec.StatusSuccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil result for reset, got %v", result)
	}
}

func TestCounterRemoveDecodeResponse(t *testing.T) {
	result, err := (&CounterRemoveOp{Name: "c1"}).DecodeResponse(codec.StatusSuccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil result for remove, got %v", result)
	}
}

func TestCounterGetNamesDecodeResponse(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteVInt(&buf, 3)
	codec.WriteLPString(&buf, "counter-a")
	codec.WriteLPString(&buf, "counter-b")
	codec.WriteLPString(&buf, "counter-c")

	result, err := (&CounterGetNamesOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	names := result.([]string)
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "counter-a" || names[1] != "counter-b" || names[2] != "counter-c" {
		t.Errorf("names = %v, want [counter-a counter-b counter-c]", names)
	}
}

func TestCounterGetNamesDecodeEmpty(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteVInt(&buf, 0)

	result, err := (&CounterGetNamesOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	names := result.([]string)
	if len(names) != 0 {
		t.Errorf("expected empty names, got %d", len(names))
	}
}
