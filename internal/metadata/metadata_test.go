package metadata

import (
	"testing"
)

func TestBTreeIndex(t *testing.T) {
	btree := NewBTreeIndex()
	btree.Insert(FromNumber(10), 1)
	btree.Insert(FromNumber(25), 2)
	btree.Insert(FromNumber(50), 3)
	btree.Insert(FromNumber(25), 4)

	// Exact match
	matches := btree.FindExact(FromNumber(25))
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for number 25, got %d", len(matches))
	}

	// Range match: [20, 60]
	rangeMatches := btree.FindRange(FromNumber(20), FromNumber(60), true, true)
	if len(rangeMatches) != 3 { // doc 2, 4, 3
		t.Fatalf("expected 3 matches in range [20, 60], got %d", len(rangeMatches))
	}
}

func TestPredicateFiltering(t *testing.T) {
	doc1 := map[string]any{
		"category": "legal",
		"year":     2024,
		"tags":     []string{"nda", "confidential"},
		"active":   true,
	}

	doc2 := map[string]any{
		"category": "finance",
		"year":     2021,
		"tags":     []string{"tax", "audit"},
		"active":   false,
	}

	p1 := Eq("category", "legal")
	if !p1.Evaluate(doc1) || p1.Evaluate(doc2) {
		t.Fatal("Eq failed")
	}

	p2 := Gt("year", 2022)
	if !p2.Evaluate(doc1) || p2.Evaluate(doc2) {
		t.Fatal("Gt failed")
	}

	p3 := And(
		Eq("category", "legal"),
		Gt("year", 2020),
		Eq("active", true),
	)
	if !p3.Evaluate(doc1) {
		t.Fatal("Composite AND failed on doc1")
	}

	pContains := Predicate{
		Field: "tags",
		Op:    OpContains,
		Value: "nda",
	}
	if !pContains.Evaluate(doc1) || pContains.Evaluate(doc2) {
		t.Fatal("Contains failed on tags")
	}
}
