package metadata

import (
	"sort"
	"sync"
)

type ValueType int

const (
	TypeString ValueType = iota
	TypeNumber
	TypeBool
)

type Value struct {
	Type   ValueType
	StrVal string
	NumVal float64
	BoolVal bool
}

func FromString(s string) Value {
	return Value{Type: TypeString, StrVal: s}
}

func FromNumber(n float64) Value {
	return Value{Type: TypeNumber, NumVal: n}
}

func FromBool(b bool) Value {
	return Value{Type: TypeBool, BoolVal: b}
}

func (v Value) Compare(other Value) int {
	if v.Type != other.Type {
		if v.Type < other.Type {
			return -1
		}
		return 1
	}
	switch v.Type {
	case TypeString:
		if v.StrVal < other.StrVal {
			return -1
		} else if v.StrVal > other.StrVal {
			return 1
		}
		return 0
	case TypeNumber:
		if v.NumVal < other.NumVal {
			return -1
		} else if v.NumVal > other.NumVal {
			return 1
		}
		return 0
	case TypeBool:
		if !v.BoolVal && other.BoolVal {
			return -1
		} else if v.BoolVal && !other.BoolVal {
			return 1
		}
		return 0
	default:
		return 0
	}
}

type entry struct {
	val   Value
	docID uint64
}

// BTreeIndex provides ordered indexing over scalar document metadata.
type BTreeIndex struct {
	mu      sync.RWMutex
	entries []entry
}

func NewBTreeIndex() *BTreeIndex {
	return &BTreeIndex{
		entries: make([]entry, 0),
	}
}

func (b *BTreeIndex) Insert(val Value, docID uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	newEntry := entry{val: val, docID: docID}
	idx := sort.Search(len(b.entries), func(i int) bool {
		cmp := b.entries[i].val.Compare(val)
		if cmp == 0 {
			return b.entries[i].docID >= docID
		}
		return cmp > 0
	})

	b.entries = append(b.entries, entry{})
	copy(b.entries[idx+1:], b.entries[idx:])
	b.entries[idx] = newEntry
}

func (b *BTreeIndex) FindExact(val Value) []uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	start := sort.Search(len(b.entries), func(i int) bool {
		return b.entries[i].val.Compare(val) >= 0
	})

	var res []uint64
	for i := start; i < len(b.entries) && b.entries[i].val.Compare(val) == 0; i++ {
		res = append(res, b.entries[i].docID)
	}
	return res
}

func (b *BTreeIndex) FindRange(minVal, maxVal Value, includeMin, includeMax bool) []uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	start := sort.Search(len(b.entries), func(i int) bool {
		cmp := b.entries[i].val.Compare(minVal)
		if includeMin {
			return cmp >= 0
		}
		return cmp > 0
	})

	var res []uint64
	for i := start; i < len(b.entries); i++ {
		cmpMax := b.entries[i].val.Compare(maxVal)
		if includeMax && cmpMax > 0 {
			break
		}
		if !includeMax && cmpMax >= 0 {
			break
		}
		res = append(res, b.entries[i].docID)
	}
	return res
}
