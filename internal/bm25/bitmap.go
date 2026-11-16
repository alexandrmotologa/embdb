package bm25

import (
	"math/bits"
)

// Bitset is a memory-efficient sparse-capable 64-bit word bitset.
type Bitset struct {
	words []uint64
}

func NewBitset() *Bitset {
	return &Bitset{words: make([]uint64, 0)}
}

func (b *Bitset) Set(id uint64) {
	wordIdx := id / 64
	bitIdx := id % 64

	if int(wordIdx) >= len(b.words) {
		newWords := make([]uint64, wordIdx+1)
		copy(newWords, b.words)
		b.words = newWords
	}
	b.words[wordIdx] |= (1 << bitIdx)
}

func (b *Bitset) Contains(id uint64) bool {
	wordIdx := id / 64
	bitIdx := id % 64

	if int(wordIdx) >= len(b.words) {
		return false
	}
	return (b.words[wordIdx] & (1 << bitIdx)) != 0
}

func (b *Bitset) Count() int {
	var total int
	for _, w := range b.words {
		total += bits.OnesCount64(w)
	}
	return total
}

func (b *Bitset) And(other *Bitset) *Bitset {
	minLen := len(b.words)
	if len(other.words) < minLen {
		minLen = len(other.words)
	}

	res := &Bitset{words: make([]uint64, minLen)}
	for i := 0; i < minLen; i++ {
		res.words[i] = b.words[i] & other.words[i]
	}
	return res
}

func (b *Bitset) Or(other *Bitset) *Bitset {
	maxLen := len(b.words)
	if len(other.words) > maxLen {
		maxLen = len(other.words)
	}

	res := &Bitset{words: make([]uint64, maxLen)}
	for i := 0; i < len(b.words); i++ {
		res.words[i] |= b.words[i]
	}
	for i := 0; i < len(other.words); i++ {
		res.words[i] |= other.words[i]
	}
	return res
}

func (b *Bitset) ToSlice() []uint64 {
	var res []uint64
	for wIdx, word := range b.words {
		if word == 0 {
			continue
		}
		for bIdx := 0; bIdx < 64; bIdx++ {
			if (word & (1 << bIdx)) != 0 {
				res = append(res, uint64(wIdx*64+bIdx))
			}
		}
	}
	return res
}
