package codec

import (
	"bytes"
	"testing"
)

func TestReadResponseHeaderWithHashTopology(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteByte(ResponseMagic)
	_ = WriteVLong(&buf, 1)
	buf.WriteByte(OpPutResponse)
	buf.WriteByte(StatusSuccess)
	buf.WriteByte(1) // topology change marker

	// topology update
	_ = WriteVInt(&buf, 10)            // topology id
	_ = WriteVInt(&buf, 3)             // 3 servers
	_ = WriteLPString(&buf, "node-a")  // host 0
	_ = WriteU2(&buf, 11222)           // port 0
	_ = WriteLPString(&buf, "node-b")  // host 1
	_ = WriteU2(&buf, 11222)           // port 1
	_ = WriteLPString(&buf, "node-c")  // host 2
	_ = WriteU2(&buf, 11222)           // port 2

	// hash distribution data (only read when clientIntelligence == HASH_DIST_AWARE)
	buf.WriteByte(3)   // hash function version
	_ = WriteVInt(&buf, 4) // 4 segments

	// segment 0: 2 owners (node 0, node 1)
	buf.WriteByte(2)
	_ = WriteVInt(&buf, 0)
	_ = WriteVInt(&buf, 1)
	// segment 1: 2 owners (node 1, node 2)
	buf.WriteByte(2)
	_ = WriteVInt(&buf, 1)
	_ = WriteVInt(&buf, 2)
	// segment 2: 2 owners (node 2, node 0)
	buf.WriteByte(2)
	_ = WriteVInt(&buf, 2)
	_ = WriteVInt(&buf, 0)
	// segment 3: 1 owner (node 0)
	buf.WriteByte(1)
	_ = WriteVInt(&buf, 0)

	h, err := ReadResponseHeader(&buf, IntelligenceHashDistAware)
	if err != nil {
		t.Fatalf("ReadResponseHeader: %v", err)
	}
	if h.TopologyUpdate == nil {
		t.Fatal("expected topology update")
	}
	topo := h.TopologyUpdate
	if topo.ID != 10 {
		t.Errorf("topology id = %d, want 10", topo.ID)
	}
	if len(topo.Servers) != 3 {
		t.Fatalf("servers = %d, want 3", len(topo.Servers))
	}
	if topo.HashFunctionVersion != 3 {
		t.Errorf("hash version = %d, want 3", topo.HashFunctionVersion)
	}
	if topo.NumSegments != 4 {
		t.Errorf("num segments = %d, want 4", topo.NumSegments)
	}
	if len(topo.SegmentOwners) != 4 {
		t.Fatalf("segment owners len = %d, want 4", len(topo.SegmentOwners))
	}

	// Verify segment 0 owners
	if len(topo.SegmentOwners[0]) != 2 {
		t.Fatalf("seg 0 owners = %d, want 2", len(topo.SegmentOwners[0]))
	}
	if topo.SegmentOwners[0][0] != 0 || topo.SegmentOwners[0][1] != 1 {
		t.Errorf("seg 0 owners = %v, want [0,1]", topo.SegmentOwners[0])
	}

	// Verify segment 3 owners
	if len(topo.SegmentOwners[3]) != 1 || topo.SegmentOwners[3][0] != 0 {
		t.Errorf("seg 3 owners = %v, want [0]", topo.SegmentOwners[3])
	}
}

func TestReadResponseHeaderTopologyAwareSkipsHash(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteByte(ResponseMagic)
	_ = WriteVLong(&buf, 1)
	buf.WriteByte(OpPingResponse)
	buf.WriteByte(StatusSuccess)
	buf.WriteByte(1) // topology change marker

	_ = WriteVInt(&buf, 5)                 // topology id
	_ = WriteVInt(&buf, 1)                 // 1 server
	_ = WriteLPString(&buf, "server1")     // host
	_ = WriteU2(&buf, 11222)               // port

	// No hash data follows — topology-aware doesn't read it

	h, err := ReadResponseHeader(&buf, IntelligenceTopologyAware)
	if err != nil {
		t.Fatalf("ReadResponseHeader: %v", err)
	}
	if h.TopologyUpdate == nil {
		t.Fatal("expected topology update")
	}
	if h.TopologyUpdate.HashFunctionVersion != 0 {
		t.Errorf("hash version = %d, want 0", h.TopologyUpdate.HashFunctionVersion)
	}
	if h.TopologyUpdate.SegmentOwners != nil {
		t.Error("expected nil segment owners for topology-aware")
	}
}
