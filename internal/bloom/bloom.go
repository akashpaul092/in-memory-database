package bloom

import (
	"hash/fnv"
	"math"
)

// Filter is a Bloom filter for probabilistic set membership.
type Filter struct {
	bits   []uint64
	hashN  int
	length int
}

// NewBloomFilter creates a Bloom filter with the given expected keys and false positive rate.
func NewBloomFilter(expectedKeys int, falsePositiveRate float64) *Filter {
	// m = -n*ln(p)/(ln(2)^2)
	m := int(-float64(expectedKeys) * math.Log(falsePositiveRate) / (math.Log(2) * math.Log(2)))
	if m < 1 {
		m = 1
	}
	// k = m/n * ln(2)
	k := int(float64(m) / float64(expectedKeys) * math.Log(2))
	if k < 1 {
		k = 1
	}
	if k > 10 {
		k = 10
	}

	// Each uint64 holds 64 bits
	numWords := (m + 63) / 64
	if numWords < 1 {
		numWords = 1
	}

	return &Filter{
		bits:   make([]uint64, numWords),
		hashN:  k,
		length: m,
	}
}

// Add adds a key to the filter.
func (f *Filter) Add(key string) {
	for i := 0; i < f.hashN; i++ {
		h := f.hash(key, uint32(i))
		idx := h % uint64(f.length)
		wordIdx := idx / 64
		bitIdx := idx % 64
		f.bits[wordIdx] |= 1 << bitIdx
	}
}

// Contains returns true if the key might be in the set, false if it definitely is not.
func (f *Filter) Contains(key string) bool {
	for i := 0; i < f.hashN; i++ {
		h := f.hash(key, uint32(i))
		idx := h % uint64(f.length)
		wordIdx := idx / 64
		bitIdx := idx % 64
		if f.bits[wordIdx]&(1<<bitIdx) == 0 {
			return false
		}
	}
	return true
}

func (f *Filter) hash(key string, seed uint32) uint64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	h.Write([]byte{byte(seed), byte(seed >> 8), byte(seed >> 16), byte(seed >> 24)})
	return h.Sum64()
}
