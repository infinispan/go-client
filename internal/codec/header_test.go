package codec

import (
	"bytes"
	"testing"
)

func TestWriteRequestHeader(t *testing.T) {
	var buf bytes.Buffer
	h := &RequestHeader{
		MessageID:          1,
		OpCode:             OpPing,
		CacheName:          []byte{},
		Flags:              0,
		ClientIntelligence: IntelligenceTopologyAware,
		TopologyID:         0,
		KeyMediaType:       0,
		ValueMediaType:     0,
	}
	if err := WriteRequestHeader(&buf, h); err != nil {
		t.Fatalf("WriteRequestHeader: %v", err)
	}
	b := buf.Bytes()
	if b[0] != RequestMagic {
		t.Errorf("magic = 0x%02x, want 0x%02x", b[0], RequestMagic)
	}
	// messageId=1 -> vLong(1) = 0x01
	if b[1] != 0x01 {
		t.Errorf("messageId = 0x%02x, want 0x01", b[1])
	}
	// version = 41
	if b[2] != Version41 {
		t.Errorf("version = %d, want %d", b[2], Version41)
	}
	// opcode = 0x17
	if b[3] != OpPing {
		t.Errorf("opcode = 0x%02x, want 0x%02x", b[3], OpPing)
	}
}

func TestWriteRequestHeaderWithMediaType(t *testing.T) {
	var buf bytes.Buffer
	h := &RequestHeader{
		MessageID:          1,
		OpCode:             OpPut,
		CacheName:          []byte("default"),
		Flags:              0,
		ClientIntelligence: IntelligenceTopologyAware,
		TopologyID:         0,
		KeyMediaType:       MediaIDOctetStream,
		ValueMediaType:     MediaIDOctetStream,
	}
	if err := WriteRequestHeader(&buf, h); err != nil {
		t.Fatalf("WriteRequestHeader: %v", err)
	}
	b := buf.Bytes()
	if b[0] != RequestMagic {
		t.Errorf("magic = 0x%02x, want 0x%02x", b[0], RequestMagic)
	}
}

func TestReadResponseHeaderNoTopology(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteByte(ResponseMagic)              // magic
	_ = WriteVLong(&buf, 42)                      // messageId
	buf.WriteByte(OpGetResponse)              // opcode
	buf.WriteByte(StatusSuccess)              // status
	buf.WriteByte(0)                          // no topology change

	h, err := ReadResponseHeader(&buf, IntelligenceTopologyAware)
	if err != nil {
		t.Fatalf("ReadResponseHeader: %v", err)
	}
	if h.MessageID != 42 {
		t.Errorf("messageId = %d, want 42", h.MessageID)
	}
	if h.OpCode != OpGetResponse {
		t.Errorf("opcode = 0x%02x, want 0x%02x", h.OpCode, OpGetResponse)
	}
	if h.Status != StatusSuccess {
		t.Errorf("status = 0x%02x, want 0x%02x", h.Status, StatusSuccess)
	}
	if h.TopologyUpdate != nil {
		t.Error("expected no topology update")
	}
}

func TestReadResponseHeaderWithTopology(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteByte(ResponseMagic)
	_ = WriteVLong(&buf, 1)
	buf.WriteByte(OpPingResponse)
	buf.WriteByte(StatusSuccess)
	buf.WriteByte(1) // topology change marker

	// topology update
	_ = WriteVInt(&buf, 5)                 // topology id
	_ = WriteVInt(&buf, 2)                 // num servers
	_ = WriteLPString(&buf, "server1")     // host 1
	_ = WriteU2(&buf, 11222)               // port 1
	_ = WriteLPString(&buf, "server2")     // host 2
	_ = WriteU2(&buf, 11223)               // port 2

	h, err := ReadResponseHeader(&buf, IntelligenceTopologyAware)
	if err != nil {
		t.Fatalf("ReadResponseHeader: %v", err)
	}
	if h.TopologyUpdate == nil {
		t.Fatal("expected topology update")
	}
	if h.TopologyUpdate.ID != 5 {
		t.Errorf("topology id = %d, want 5", h.TopologyUpdate.ID)
	}
	if len(h.TopologyUpdate.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(h.TopologyUpdate.Servers))
	}
	if h.TopologyUpdate.Servers[0].Host != "server1" {
		t.Errorf("server 0 host = %q", h.TopologyUpdate.Servers[0].Host)
	}
	if h.TopologyUpdate.Servers[0].Port != 11222 {
		t.Errorf("server 0 port = %d", h.TopologyUpdate.Servers[0].Port)
	}
	if h.TopologyUpdate.Servers[1].Host != "server2" {
		t.Errorf("server 1 host = %q", h.TopologyUpdate.Servers[1].Host)
	}
	if h.TopologyUpdate.Servers[1].Port != 11223 {
		t.Errorf("server 1 port = %d", h.TopologyUpdate.Servers[1].Port)
	}
}

func TestReadResponseHeaderInvalidMagic(t *testing.T) {
	buf := bytes.NewReader([]byte{0xFF, 0x01, 0x04, 0x00, 0x00})
	_, err := ReadResponseHeader(buf, IntelligenceTopologyAware)
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestServerAddressAddr(t *testing.T) {
	a := ServerAddress{Host: "localhost", Port: 11222}
	if a.Addr() != "localhost:11222" {
		t.Errorf("Addr() = %q, want %q", a.Addr(), "localhost:11222")
	}
}
