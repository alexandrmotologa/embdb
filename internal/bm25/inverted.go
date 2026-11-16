package bm25

import (
	"encoding/binary"
	"io"
	"math"
	"sort"
	"sync"
)

type Posting struct {
	DocID         uint64
	TermFrequency uint32
}

type Match struct {
	DocID uint64
	Score float32
}

type Index struct {
	mu          sync.RWMutex
	postings    map[string][]Posting
	bitsets     map[string]*Bitset
	docLengths  map[uint64]uint32
	totalDocs   uint64
	totalLength uint64
	k1          float64
	b           float64
}

func NewIndex() *Index {
	return &Index{
		postings:   make(map[string][]Posting),
		bitsets:    make(map[string]*Bitset),
		docLengths: make(map[uint64]uint32),
		k1:         1.2,
		b:          0.75,
	}
}

// IndexDocument indexes a text document with given docID.
func (idx *Index) IndexDocument(docID uint64, text string) {
	tokens := Tokenize(text)
	docLen := uint32(len(tokens))
	if docLen == 0 {
		return
	}

	tfMap := make(map[string]uint32)
	for _, t := range tokens {
		tfMap[t]++
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.docLengths[docID] = docLen
	idx.totalDocs++
	idx.totalLength += uint64(docLen)

	for term, tf := range tfMap {
		idx.postings[term] = append(idx.postings[term], Posting{
			DocID:         docID,
			TermFrequency: tf,
		})

		bs := idx.bitsets[term]
		if bs == nil {
			bs = NewBitset()
			idx.bitsets[term] = bs
		}
		bs.Set(docID)
	}
}

func (idx *Index) AvgDocLength() float64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if idx.totalDocs == 0 {
		return 0
	}
	return float64(idx.totalLength) / float64(idx.totalDocs)
}

func (idx *Index) TotalDocs() uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.totalDocs
}

// Search calculates BM25 scores for matching documents, applying an optional predicate filter.
func (idx *Index) Search(query string, k int, filter func(id uint64) bool) []Match {
	queryTokens := Tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.totalDocs == 0 {
		return nil
	}

	avgdl := float64(idx.totalLength) / float64(idx.totalDocs)
	nTotal := float64(idx.totalDocs)

	// Accumulate scores per document
	docScores := make(map[uint64]float64)

	for _, term := range queryTokens {
		postingsList, ok := idx.postings[term]
		if !ok || len(postingsList) == 0 {
			continue
		}

		df := float64(len(postingsList))
		// Robertson-Sparck Jones IDF formula with add-1 smoothing
		idf := math.Log((nTotal-df+0.5)/(df+0.5) + 1.0)
		if idf < 0 {
			idf = 0.0001
		}

		for _, p := range postingsList {
			if filter != nil && !filter(p.DocID) {
				continue
			}

			tf := float64(p.TermFrequency)
			docLen := float64(idx.docLengths[p.DocID])

			numerator := tf * (idx.k1 + 1.0)
			denominator := tf + idx.k1*(1.0-idx.b+idx.b*(docLen/avgdl))

			termScore := idf * (numerator / denominator)
			docScores[p.DocID] += termScore
		}
	}

	if len(docScores) == 0 {
		return nil
	}

	matches := make([]Match, 0, len(docScores))
	for docID, score := range docScores {
		matches = append(matches, Match{
			DocID: docID,
			Score: float32(score),
		})
	}

	// Sort descending by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if k > 0 && len(matches) > k {
		matches = matches[:k]
	}

	return matches
}

// Serialize writes the BM25 index to a binary stream.
func (idx *Index) Serialize(w io.Writer) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if err := binary.Write(w, binary.LittleEndian, idx.totalDocs); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, idx.totalLength); err != nil {
		return err
	}

	// Write document lengths
	docCount := uint32(len(idx.docLengths))
	if err := binary.Write(w, binary.LittleEndian, docCount); err != nil {
		return err
	}
	for id, dLen := range idx.docLengths {
		if err := binary.Write(w, binary.LittleEndian, id); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, dLen); err != nil {
			return err
		}
	}

	// Write terms and postings
	termCount := uint32(len(idx.postings))
	if err := binary.Write(w, binary.LittleEndian, termCount); err != nil {
		return err
	}

	for term, plist := range idx.postings {
		termBytes := []byte(term)
		termLen := uint16(len(termBytes))
		if err := binary.Write(w, binary.LittleEndian, termLen); err != nil {
			return err
		}
		if _, err := w.Write(termBytes); err != nil {
			return err
		}

		pCount := uint32(len(plist))
		if err := binary.Write(w, binary.LittleEndian, pCount); err != nil {
			return err
		}
		for _, p := range plist {
			if err := binary.Write(w, binary.LittleEndian, p.DocID); err != nil {
				return err
			}
			if err := binary.Write(w, binary.LittleEndian, p.TermFrequency); err != nil {
				return err
			}
		}
	}

	return nil
}

// Deserialize loads the BM25 index from a binary stream.
func (idx *Index) Deserialize(r io.Reader) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if err := binary.Read(r, binary.LittleEndian, &idx.totalDocs); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &idx.totalLength); err != nil {
		return err
	}

	var docCount uint32
	if err := binary.Read(r, binary.LittleEndian, &docCount); err != nil {
		return err
	}

	idx.docLengths = make(map[uint64]uint32, docCount)
	for i := uint32(0); i < docCount; i++ {
		var id uint64
		var dLen uint32
		if err := binary.Read(r, binary.LittleEndian, &id); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &dLen); err != nil {
			return err
		}
		idx.docLengths[id] = dLen
	}

	var termCount uint32
	if err := binary.Read(r, binary.LittleEndian, &termCount); err != nil {
		return err
	}

	idx.postings = make(map[string][]Posting, termCount)
	idx.bitsets = make(map[string]*Bitset, termCount)

	for i := uint32(0); i < termCount; i++ {
		var termLen uint16
		if err := binary.Read(r, binary.LittleEndian, &termLen); err != nil {
			return err
		}
		termBuf := make([]byte, termLen)
		if _, err := io.ReadFull(r, termBuf); err != nil {
			return err
		}
		term := string(termBuf)

		var pCount uint32
		if err := binary.Read(r, binary.LittleEndian, &pCount); err != nil {
			return err
		}

		plist := make([]Posting, pCount)
		bs := NewBitset()
		for j := uint32(0); j < pCount; j++ {
			var docID uint64
			var tf uint32
			if err := binary.Read(r, binary.LittleEndian, &docID); err != nil {
				return err
			}
			if err := binary.Read(r, binary.LittleEndian, &tf); err != nil {
				return err
			}
			plist[j] = Posting{DocID: docID, TermFrequency: tf}
			bs.Set(docID)
		}

		idx.postings[term] = plist
		idx.bitsets[term] = bs
	}

	return nil
}
