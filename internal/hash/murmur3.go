package hash

import "encoding/binary"

const defaultSeed uint64 = 9001

func Hash(key []byte) int32 {
	return HashWithSeed(key, defaultSeed)
}

func HashWithSeed(key []byte, seed uint64) int32 {
	h1 := murmurHash3x64_64(key, seed)
	return int32(h1 >> 32)
}

func murmurHash3x64_64(key []byte, seed uint64) uint64 {
	h1 := uint64(0x9368e53c2f6af274) ^ seed
	h2 := uint64(0x586dcd208f7cd3fd) ^ seed
	c1 := uint64(0x87c37b91114253d5)
	c2 := uint64(0x4cf5ad432745937f)

	nblocks := len(key) / 16
	for i := range nblocks {
		k1 := binary.LittleEndian.Uint64(key[i*16:])
		k2 := binary.LittleEndian.Uint64(key[i*16+8:])
		h1, h2, c1, c2 = bmix(h1, h2, k1, k2, c1, c2)
	}

	var k1, k2 uint64
	tail := nblocks * 16

	switch len(key) & 15 {
	case 15:
		k2 ^= uint64(signExtend(key[tail+14])) << 48
		fallthrough
	case 14:
		k2 ^= uint64(signExtend(key[tail+13])) << 40
		fallthrough
	case 13:
		k2 ^= uint64(signExtend(key[tail+12])) << 32
		fallthrough
	case 12:
		k2 ^= uint64(signExtend(key[tail+11])) << 24
		fallthrough
	case 11:
		k2 ^= uint64(signExtend(key[tail+10])) << 16
		fallthrough
	case 10:
		k2 ^= uint64(signExtend(key[tail+9])) << 8
		fallthrough
	case 9:
		k2 ^= uint64(signExtend(key[tail+8]))
		fallthrough
	case 8:
		k1 ^= uint64(signExtend(key[tail+7])) << 56
		fallthrough
	case 7:
		k1 ^= uint64(signExtend(key[tail+6])) << 48
		fallthrough
	case 6:
		k1 ^= uint64(signExtend(key[tail+5])) << 40
		fallthrough
	case 5:
		k1 ^= uint64(signExtend(key[tail+4])) << 32
		fallthrough
	case 4:
		k1 ^= uint64(signExtend(key[tail+3])) << 24
		fallthrough
	case 3:
		k1 ^= uint64(signExtend(key[tail+2])) << 16
		fallthrough
	case 2:
		k1 ^= uint64(signExtend(key[tail+1])) << 8
		fallthrough
	case 1:
		k1 ^= uint64(signExtend(key[tail]))
		h1, h2, c1, c2 = bmix(h1, h2, k1, k2, c1, c2)
	}
	_ = c1
	_ = c2

	h2 ^= uint64(len(key))

	h1 += h2
	h2 += h1

	h1 = fmix(h1)
	h2 = fmix(h2)

	h1 += h2
	_ = h2

	return h1
}

// signExtend replicates Java's (long) byte cast, which sign-extends.
func signExtend(b byte) int64 {
	return int64(int8(b))
}

func bmix(h1, h2, k1, k2, c1, c2 uint64) (uint64, uint64, uint64, uint64) {
	k1 *= c1
	k1 = (k1 << 23) | (k1 >> 41)
	k1 *= c2
	h1 ^= k1
	h1 += h2

	h2 = (h2 << 41) | (h2 >> 23)

	k2 *= c2
	k2 = (k2 << 23) | (k2 >> 41)
	k2 *= c1
	h2 ^= k2
	h2 += h1

	h1 = h1*3 + 0x52dce729
	h2 = h2*3 + 0x38495ab5

	c1 = c1*5 + 0x7b7d159c
	c2 = c2*5 + 0x6bce6396

	return h1, h2, c1, c2
}

func fmix(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33
	return k
}
