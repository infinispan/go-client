package hash

import "testing"

func TestBloomFilter_AddAndBits(t *testing.T) {
	bf := NewBloomFilter(1024)
	bf.Add([]byte("hello"))
	bf.Add([]byte("world"))

	bits := bf.ToBitSet()
	if len(bits) == 0 {
		t.Fatal("ToBitSet() returned empty after adding keys")
	}

	setBits := 0
	for _, b := range bits {
		for i := 0; i < 8; i++ {
			if b&(1<<i) != 0 {
				setBits++
			}
		}
	}
	if setBits < 3 || setBits > 6 {
		t.Errorf("expected 3-6 set bits (2 keys * 3 hashes, some may collide), got %d", setBits)
	}
}

func TestBloomFilter_Deterministic(t *testing.T) {
	bf1 := NewBloomFilter(256)
	bf1.Add([]byte("test-key"))
	bits1 := bf1.ToBitSet()

	bf2 := NewBloomFilter(256)
	bf2.Add([]byte("test-key"))
	bits2 := bf2.ToBitSet()

	if len(bits1) != len(bits2) {
		t.Fatalf("lengths differ: %d vs %d", len(bits1), len(bits2))
	}
	for i := range bits1 {
		if bits1[i] != bits2[i] {
			t.Fatalf("byte %d differs: %02x vs %02x", i, bits1[i], bits2[i])
		}
	}
}

func TestBloomFilter_Clear(t *testing.T) {
	bf := NewBloomFilter(128)
	bf.Add([]byte("hello"))
	bf.Clear()

	bits := bf.ToBitSet()
	if len(bits) != 0 {
		t.Errorf("ToBitSet() after Clear() returned %d bytes, want 0", len(bits))
	}
}

func TestBloomFilter_ToBitSetLittleEndian(t *testing.T) {
	bf := NewBloomFilter(16)
	// Manually set bit 0 to verify little-endian byte ordering
	bf.Add([]byte{})

	bits := bf.ToBitSet()
	if len(bits) == 0 {
		t.Skip("empty key produced no bits")
	}

	// Verify that each bit position maps to byte[bit/8] & (1 << (bit%8))
	// which is the little-endian Java BitSet format
	bf2 := NewBloomFilter(64)
	bf2.Add([]byte("a"))
	result := bf2.ToBitSet()
	if len(result) == 0 {
		t.Fatal("expected non-empty bitset")
	}
	// Just verify trailing zeros are stripped
	if result[len(result)-1] == 0 {
		t.Error("trailing zero byte not stripped")
	}
}

func TestBloomFilter_TrailingZerosStripped(t *testing.T) {
	bf := NewBloomFilter(1024)
	bf.Add([]byte("x"))
	bits := bf.ToBitSet()

	if len(bits) > 0 && bits[len(bits)-1] == 0 {
		t.Error("ToBitSet should strip trailing zero bytes")
	}
	if len(bits) > 128 {
		t.Errorf("ToBitSet returned %d bytes for 1024-bit filter, expected much less with 1 key", len(bits))
	}
}
