package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func encodeCreatedEvent(listenerID, key []byte, version int64) []byte {
	var buf bytes.Buffer
	codec.WriteLPBytes(&buf, listenerID)
	codec.WriteU1(&buf, 0) // not custom
	codec.WriteU1(&buf, 0) // not retried
	codec.WriteLPBytes(&buf, key)
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = byte(version)
		version >>= 8
	}
	buf.Write(b)
	return buf.Bytes()
}

func encodeRemovedEvent(listenerID, key []byte) []byte {
	var buf bytes.Buffer
	codec.WriteLPBytes(&buf, listenerID)
	codec.WriteU1(&buf, 0) // not custom
	codec.WriteU1(&buf, 0) // not retried
	codec.WriteLPBytes(&buf, key)
	return buf.Bytes()
}

func TestDecodeCreatedEvent(t *testing.T) {
	id := []byte{1, 2, 3, 4}
	key := []byte("my-key")
	data := encodeCreatedEvent(id, key, 42)

	decoded, err := DecodeEvent(codec.OpCacheEntryCreated, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	ev := decoded.(*CacheEntryEvent)
	if ev.Type != EventCreated {
		t.Errorf("Type = %v, want EventCreated", ev.Type)
	}
	if string(ev.Key) != "my-key" {
		t.Errorf("Key = %q, want %q", ev.Key, "my-key")
	}
	if ev.Version != 42 {
		t.Errorf("Version = %d, want 42", ev.Version)
	}
}

func TestDecodeModifiedEvent(t *testing.T) {
	id := []byte{5, 6}
	key := []byte("mod-key")
	data := encodeCreatedEvent(id, key, 99) // same format as created

	decoded, err := DecodeEvent(codec.OpCacheEntryModified, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	ev := decoded.(*CacheEntryEvent)
	if ev.Type != EventModified {
		t.Errorf("Type = %v, want EventModified", ev.Type)
	}
	if string(ev.Key) != "mod-key" {
		t.Errorf("Key = %q, want %q", ev.Key, "mod-key")
	}
	if ev.Version != 99 {
		t.Errorf("Version = %d, want 99", ev.Version)
	}
}

func TestDecodeRemovedEvent(t *testing.T) {
	id := []byte{7, 8}
	key := []byte("del-key")
	data := encodeRemovedEvent(id, key)

	decoded, err := DecodeEvent(codec.OpCacheEntryRemoved, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	ev := decoded.(*CacheEntryEvent)
	if ev.Type != EventRemoved {
		t.Errorf("Type = %v, want EventRemoved", ev.Type)
	}
	if string(ev.Key) != "del-key" {
		t.Errorf("Key = %q, want %q", ev.Key, "del-key")
	}
	if ev.Version != 0 {
		t.Errorf("Version = %d, want 0", ev.Version)
	}
}

func TestDecodeExpiredEvent(t *testing.T) {
	id := []byte{9}
	key := []byte("exp-key")
	data := encodeRemovedEvent(id, key) // same format as removed

	decoded, err := DecodeEvent(codec.OpCacheEntryExpired, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	ev := decoded.(*CacheEntryEvent)
	if ev.Type != EventExpired {
		t.Errorf("Type = %v, want EventExpired", ev.Type)
	}
	if string(ev.Key) != "exp-key" {
		t.Errorf("Key = %q, want %q", ev.Key, "exp-key")
	}
}

func TestDecodeCustomEvent(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteLPBytes(&buf, []byte{1, 2})
	codec.WriteU1(&buf, 1) // is_custom
	codec.WriteU1(&buf, 0) // not retried
	codec.WriteLPBytes(&buf, []byte("custom-data"))

	decoded, err := DecodeEvent(codec.OpCacheEntryCreated, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	ce, ok := decoded.(*CustomEvent)
	if !ok {
		t.Fatalf("expected *CustomEvent, got %T", decoded)
	}
	if string(ce.Data) != "custom-data" {
		t.Errorf("Data = %q, want %q", ce.Data, "custom-data")
	}
}

func TestAddClientListenerOpEncode(t *testing.T) {
	op := &AddClientListenerOp{
		Cache:        "test-cache",
		ListenerID:   []byte{0x01, 0x02, 0x03},
		IncludeState: true,
		Interests:    InterestCreated | InterestRemoved,
	}

	if op.RequestOpCode() != codec.OpAddClientListener {
		t.Errorf("RequestOpCode = 0x%02x, want 0x%02x", op.RequestOpCode(), codec.OpAddClientListener)
	}
	if op.ResponseOpCode() != codec.OpAddClientListenerResponse {
		t.Errorf("ResponseOpCode = 0x%02x, want 0x%02x", op.ResponseOpCode(), codec.OpAddClientListenerResponse)
	}
	if string(op.CacheName()) != "test-cache" {
		t.Errorf("CacheName = %q, want %q", string(op.CacheName()), "test-cache")
	}

	var buf bytes.Buffer
	if err := op.WriteBody(&buf); err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(buf.Bytes())

	// listener ID
	id, _ := codec.ReadLPBytes(r)
	if !bytes.Equal(id, []byte{0x01, 0x02, 0x03}) {
		t.Errorf("listenerID = %v, want [1,2,3]", id)
	}
	// include state
	state, _ := codec.ReadU1(r)
	if state != 1 {
		t.Errorf("includeState = %d, want 1", state)
	}
	// filter factory (empty)
	ff, _ := codec.ReadLPString(r)
	if ff != "" {
		t.Errorf("filterFactory = %q, want empty", ff)
	}
	// converter factory (empty)
	cf, _ := codec.ReadLPString(r)
	if cf != "" {
		t.Errorf("converterFactory = %q, want empty", cf)
	}
	// useRawData
	raw, _ := codec.ReadU1(r)
	if raw != 1 {
		t.Errorf("useRawData = %d, want 1", raw)
	}
	// interests
	interests, _ := codec.ReadVInt(r)
	if interests != int32(InterestCreated|InterestRemoved) {
		t.Errorf("interests = 0x%02x, want 0x%02x", interests, InterestCreated|InterestRemoved)
	}
}

func TestRemoveClientListenerOpEncode(t *testing.T) {
	op := &RemoveClientListenerOp{
		Cache:      "test-cache",
		ListenerID: []byte{0xAA, 0xBB},
	}

	var buf bytes.Buffer
	if err := op.WriteBody(&buf); err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(buf.Bytes())

	id, _ := codec.ReadLPBytes(r)
	if !bytes.Equal(id, []byte{0xAA, 0xBB}) {
		t.Errorf("listenerID = %v, want [AA,BB]", id)
	}
}

func TestAddClientListenerOpEncodeWithFactory(t *testing.T) {
	param0 := []byte{0x01, 0x02}
	param1 := []byte{0x03}
	op := &AddClientListenerOp{
		Cache:            "cq-cache",
		ListenerID:       []byte{0x10},
		IncludeState:     true,
		Interests:        InterestAll,
		FilterFactory:    "my-filter",
		ConverterFactory: "my-converter",
		FilterParams:     [][]byte{param0, param1},
		ConverterParams:  [][]byte{param0},
	}

	var buf bytes.Buffer
	if err := op.WriteBody(&buf); err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(buf.Bytes())

	// listener ID
	codec.ReadLPBytes(r)
	// include state
	state, _ := codec.ReadU1(r)
	if state != 1 {
		t.Errorf("includeState = %d, want 1", state)
	}
	// filter factory name
	ff, _ := codec.ReadLPString(r)
	if ff != "my-filter" {
		t.Errorf("filterFactory = %q, want %q", ff, "my-filter")
	}
	// filter param count
	fpCount, _ := codec.ReadU1(r)
	if fpCount != 2 {
		t.Errorf("filterParams count = %d, want 2", fpCount)
	}
	fp0, _ := codec.ReadLPBytes(r)
	if !bytes.Equal(fp0, param0) {
		t.Errorf("filterParams[0] = %x, want %x", fp0, param0)
	}
	fp1, _ := codec.ReadLPBytes(r)
	if !bytes.Equal(fp1, param1) {
		t.Errorf("filterParams[1] = %x, want %x", fp1, param1)
	}
	// converter factory name
	cf, _ := codec.ReadLPString(r)
	if cf != "my-converter" {
		t.Errorf("converterFactory = %q, want %q", cf, "my-converter")
	}
	// converter param count
	cpCount, _ := codec.ReadU1(r)
	if cpCount != 1 {
		t.Errorf("converterParams count = %d, want 1", cpCount)
	}
	cp0, _ := codec.ReadLPBytes(r)
	if !bytes.Equal(cp0, param0) {
		t.Errorf("converterParams[0] = %x, want %x", cp0, param0)
	}
	// useRawData
	raw, _ := codec.ReadU1(r)
	if raw != 1 {
		t.Errorf("useRawData = %d, want 1", raw)
	}
	// interests
	interests, _ := codec.ReadVInt(r)
	if interests != int32(InterestAll) {
		t.Errorf("interests = 0x%02x, want 0x%02x", interests, InterestAll)
	}
}
