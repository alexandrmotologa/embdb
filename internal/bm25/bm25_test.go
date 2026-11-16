package bm25

import (
	"bytes"
	"testing"
)

func TestTokenizer(t *testing.T) {
	text := "The quick brown fox is running fast towards modern vector databases!"
	tokens := Tokenize(text)

	// 'The', 'is' should be eliminated as stop-words.
	// 'running' -> 'run' (stemmed).
	// 'databases' -> 'databas' (stemmed).
	foundRun := false
	for _, tok := range tokens {
		if tok == "run" {
			foundRun = true
		}
		if tok == "the" || tok == "is" {
			t.Fatalf("stop word '%s' was not removed", tok)
		}
	}
	if !foundRun {
		t.Fatal("expected stemmed token 'run' to be present")
	}
}

func TestBitsetOperations(t *testing.T) {
	b1 := NewBitset()
	b1.Set(1)
	b1.Set(5)
	b1.Set(100)

	b2 := NewBitset()
	b2.Set(5)
	b2.Set(50)
	b2.Set(100)

	and := b1.And(b2)
	if !and.Contains(5) || !and.Contains(100) {
		t.Fatal("bitset AND failed")
	}
	if and.Contains(1) || and.Contains(50) {
		t.Fatal("bitset AND included non-common elements")
	}

	or := b1.Or(b2)
	if or.Count() != 4 {
		t.Fatalf("expected 4 unique elements in OR, got %d", or.Count())
	}
}

func TestBM25SearchAndSerialization(t *testing.T) {
	idx := NewIndex()

	idx.IndexDocument(1, "PostgreSQL relational database systems with ACID transactions")
	idx.IndexDocument(2, "Vector search engine with HNSW graphs and semantic embeddings")
	idx.IndexDocument(3, "Hybrid search combining lexical BM25 and dense vector embeddings")
	idx.IndexDocument(4, "SQLite single-file embedded database engine written in C")

	if idx.TotalDocs() != 4 {
		t.Fatalf("expected 4 docs, got %d", idx.TotalDocs())
	}

	// Search for semantic vector
	results := idx.Search("semantic embeddings vector", 5, nil)
	if len(results) == 0 {
		t.Fatal("expected search results, got 0")
	}

	// Doc 2 or 3 should be top result
	topID := results[0].DocID
	if topID != 2 && topID != 3 {
		t.Fatalf("expected doc 2 or 3 as top result, got doc %d", topID)
	}

	// Filtered search: only doc 3
	filtered := idx.Search("embeddings", 5, func(id uint64) bool {
		return id == 3
	})
	if len(filtered) != 1 || filtered[0].DocID != 3 {
		t.Fatalf("expected only doc 3, got %v", filtered)
	}

	// Serialization test
	var buf bytes.Buffer
	if err := idx.Serialize(&buf); err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	idx2 := NewIndex()
	if err := idx2.Deserialize(&buf); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	if idx2.TotalDocs() != 4 {
		t.Fatalf("expected 4 docs after deserialize, got %d", idx2.TotalDocs())
	}

	res2 := idx2.Search("semantic embeddings vector", 2, nil)
	if len(res2) == 0 || (res2[0].DocID != 2 && res2[0].DocID != 3) {
		t.Fatalf("expected same top result after deserialize, got %v", res2)
	}
}
