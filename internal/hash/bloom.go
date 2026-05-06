package hash

var bloomSeeds = [3]uint64{239, 1847, 2719}

type BloomFilter struct {
	bitsToUse int
	bits      []byte
}

func NewBloomFilter(bitsToUse int) *BloomFilter {
	nbytes := (bitsToUse + 7) / 8
	return &BloomFilter{
		bitsToUse: bitsToUse,
		bits:      make([]byte, nbytes),
	}
}

func (bf *BloomFilter) Add(key []byte) {
	for _, seed := range bloomSeeds {
		h := HashWithSeed(key, seed)
		if h < 0 {
			h = -h
		}
		bit := int(h) % bf.bitsToUse
		bf.bits[bit/8] |= 1 << (bit % 8)
	}
}

func (bf *BloomFilter) Clear() {
	for i := range bf.bits {
		bf.bits[i] = 0
	}
}

// ToBitSet returns the bloom filter bits as a little-endian byte array
// compatible with Java's BitSet.toByteArray() — trailing zero bytes are stripped.
func (bf *BloomFilter) ToBitSet() []byte {
	last := len(bf.bits) - 1
	for last >= 0 && bf.bits[last] == 0 {
		last--
	}
	if last < 0 {
		return []byte{}
	}
	out := make([]byte, last+1)
	copy(out, bf.bits[:last+1])
	return out
}
