package hash

import "math"

type ConsistentHash struct {
	segmentOwners [][]string
	numSegments   int
	segmentSize   int
}

func NewConsistentHash(numSegments int, segmentOwners [][]string) *ConsistentHash {
	return &ConsistentHash{
		segmentOwners: segmentOwners,
		numSegments:   numSegments,
		segmentSize:   int(math.Ceil(float64(math.MaxInt32+1) / float64(numSegments))),
	}
}

func (ch *ConsistentHash) PrimaryOwner(key []byte) string {
	normalizedHash := int(Hash(key)) & math.MaxInt32
	segment := normalizedHash / ch.segmentSize
	if segment >= ch.numSegments {
		segment = ch.numSegments - 1
	}
	owners := ch.segmentOwners[segment]
	if len(owners) == 0 {
		return ""
	}
	return owners[0]
}

func (ch *ConsistentHash) NumSegments() int {
	return ch.numSegments
}

func (ch *ConsistentHash) Owners() [][]string {
	return ch.segmentOwners
}
