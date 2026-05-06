package codec

import (
	"fmt"
	"io"
)

type RequestHeader struct {
	MessageID          int64
	OpCode             byte
	Version            byte
	CacheName          []byte
	Flags              int32
	ClientIntelligence byte
	TopologyID         int32
	KeyMediaType       int32
	ValueMediaType     int32
}

type ResponseHeader struct {
	MessageID      int64
	OpCode         byte
	Status         byte
	TopologyUpdate *TopologyUpdate
}

type TopologyUpdate struct {
	ID                  int32
	Servers             []ServerAddress
	HashFunctionVersion byte
	NumSegments         int
	SegmentOwners       [][]int
}

type ServerAddress struct {
	Host string
	Port uint16
}

func (a ServerAddress) Addr() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

func WriteRequestHeader(w io.Writer, h *RequestHeader) error {
	if err := WriteU1(w, RequestMagic); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}
	if err := WriteVLong(w, h.MessageID); err != nil {
		return fmt.Errorf("write message id: %w", err)
	}
	version := h.Version
	if version == 0 {
		version = Version41
	}
	if err := WriteU1(w, version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}
	if err := WriteU1(w, h.OpCode); err != nil {
		return fmt.Errorf("write opcode: %w", err)
	}
	if err := WriteLPBytes(w, h.CacheName); err != nil {
		return fmt.Errorf("write cache name: %w", err)
	}
	if err := WriteVInt(w, h.Flags); err != nil {
		return fmt.Errorf("write flags: %w", err)
	}
	if err := WriteU1(w, h.ClientIntelligence); err != nil {
		return fmt.Errorf("write client intelligence: %w", err)
	}
	if err := WriteVInt(w, h.TopologyID); err != nil {
		return fmt.Errorf("write topology id: %w", err)
	}
	if err := writeMediaTypeField(w, h.KeyMediaType); err != nil {
		return fmt.Errorf("write key media type: %w", err)
	}
	if err := writeMediaTypeField(w, h.ValueMediaType); err != nil {
		return fmt.Errorf("write value media type: %w", err)
	}
	// v4.0+ otherParams: empty map
	if err := WriteVInt(w, 0); err != nil {
		return fmt.Errorf("write other params: %w", err)
	}
	return nil
}

func writeMediaTypeField(w io.Writer, mediaType int32) error {
	if mediaType > 0 {
		return WriteMediaTypePredefined(w, mediaType)
	}
	return WriteMediaTypeNone(w)
}

func ReadResponseHeader(r io.Reader, clientIntelligence byte) (*ResponseHeader, error) {
	magic, err := ReadU1(r)
	if err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if magic != ResponseMagic {
		return nil, fmt.Errorf("invalid response magic: 0x%02x", magic)
	}
	msgID, err := ReadVLong(r)
	if err != nil {
		return nil, fmt.Errorf("read message id: %w", err)
	}
	opcode, err := ReadU1(r)
	if err != nil {
		return nil, fmt.Errorf("read opcode: %w", err)
	}
	status, err := ReadU1(r)
	if err != nil {
		return nil, fmt.Errorf("read status: %w", err)
	}
	marker, err := ReadU1(r)
	if err != nil {
		return nil, fmt.Errorf("read topology change marker: %w", err)
	}

	h := &ResponseHeader{
		MessageID: msgID,
		OpCode:    opcode,
		Status:    status,
	}

	if marker != 0 {
		topo, err := readTopologyUpdate(r, clientIntelligence)
		if err != nil {
			return nil, fmt.Errorf("read topology update: %w", err)
		}
		h.TopologyUpdate = topo
	}
	return h, nil
}

func readTopologyUpdate(r io.Reader, clientIntelligence byte) (*TopologyUpdate, error) {
	topoID, err := ReadVInt(r)
	if err != nil {
		return nil, fmt.Errorf("read topology id: %w", err)
	}
	numServers, err := ReadVInt(r)
	if err != nil {
		return nil, fmt.Errorf("read num servers: %w", err)
	}
	servers := make([]ServerAddress, numServers)
	for i := int32(0); i < numServers; i++ {
		host, err := ReadLPString(r)
		if err != nil {
			return nil, fmt.Errorf("read server %d host: %w", i, err)
		}
		port, err := ReadU2(r)
		if err != nil {
			return nil, fmt.Errorf("read server %d port: %w", i, err)
		}
		servers[i] = ServerAddress{Host: host, Port: port}
	}

	topo := &TopologyUpdate{ID: topoID, Servers: servers}

	if clientIntelligence == IntelligenceHashDistAware {
		hashVersion, err := ReadU1(r)
		if err != nil {
			return nil, fmt.Errorf("read hash function version: %w", err)
		}
		topo.HashFunctionVersion = hashVersion

		numSegments, err := ReadVInt(r)
		if err != nil {
			return nil, fmt.Errorf("read num segments: %w", err)
		}
		topo.NumSegments = int(numSegments)

		if hashVersion > 0 {
			segmentOwners := make([][]int, numSegments)
			for i := int32(0); i < numSegments; i++ {
				numOwners, err := ReadU1(r)
				if err != nil {
					return nil, fmt.Errorf("read segment %d num owners: %w", i, err)
				}
				owners := make([]int, numOwners)
				for j := byte(0); j < numOwners; j++ {
					idx, err := ReadVInt(r)
					if err != nil {
						return nil, fmt.Errorf("read segment %d owner %d: %w", i, j, err)
					}
					owners[j] = int(idx)
				}
				segmentOwners[i] = owners
			}
			topo.SegmentOwners = segmentOwners
		}
	}

	return topo, nil
}

func ReadErrorMessage(r io.Reader) (string, error) {
	return ReadLPString(r)
}
