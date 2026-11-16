package hnsw

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestDistanceFunctions(t *testing.T) {
	v1 := []float32{1.0, 0.0, 0.0}
	v2 := []float32{0.0, 1.0, 0.0}
	v3 := []float32{1.0, 0.0, 0.0}

	// Orthogonal vectors
	d12, err := CosineDistance(v1, v2)
	if err != nil {
		t.Fatal(err)
	}
	if d12 < 0.99 || d12 > 1.01 {
		t.Fatalf("expected cosine distance ~1.0, got %f", d12)
	}

	// Identical vectors
	d13, err := CosineDistance(v1, v3)
	if err != nil {
		t.Fatal(err)
	}
	if d13 > 0.001 {
		t.Fatalf("expected cosine distance ~0.0, got %f", d13)
	}

	// L2 distance
	l2_12, err := L2Distance(v1, v2)
	if err != nil {
		t.Fatal(err)
	}
	if l2_12 < 1.41 || l2_12 > 1.42 {
		t.Fatalf("expected L2 distance ~1.414, got %f", l2_12)
	}
}

func TestSQ8Quantization(t *testing.T) {
	vec := []float32{-0.5, 0.0, 0.5, 1.0}
	sq := QuantizeSQ8(vec)
	if len(sq.Data) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(sq.Data))
	}
	deq := sq.Dequantize()
	for i := range vec {
		diff := deq[i] - vec[i]
		if diff < -0.05 || diff > 0.05 {
			t.Fatalf("dequantization error too high at index %d: original %f, got %f", i, vec[i], deq[i])
		}
	}
}

func TestHNSWGraphInsertAndSearch(t *testing.T) {
	dim := 16
	cfg := DefaultConfig(dim)
	cfg.EfConstruction = 50
	cfg.EfSearch = 32
	g := NewGraph(cfg)

	numVectors := 100
	vectors := make([][]float32, numVectors)
	for i := 0; i < numVectors; i++ {
		vec := make([]float32, dim)
		for d := 0; d < dim; d++ {
			vec[d] = rand.Float32()*2.0 - 1.0
		}
		vectors[i] = Normalize(vec)
		if err := g.Insert(uint64(i+1), vectors[i]); err != nil {
			t.Fatalf("failed to insert vector %d: %v", i+1, err)
		}
	}

	if g.Len() != numVectors {
		t.Fatalf("expected %d nodes, got %d", numVectors, g.Len())
	}

	// Search for vector 10
	target := vectors[9]
	results, err := g.Search(target, 5, nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected search results, got 0")
	}

	if results[0].ID != 10 {
		t.Fatalf("expected closest match to be ID 10, got ID %d with distance %f", results[0].ID, results[0].Distance)
	}

	// Filtered search: only even IDs
	filterEven := func(id uint64) bool {
		return id%2 == 0
	}
	filteredResults, err := g.Search(target, 5, filterEven)
	if err != nil {
		t.Fatal(err)
	}
	for _, res := range filteredResults {
		if res.ID%2 != 0 {
			t.Fatalf("expected even ID, got %d", res.ID)
		}
	}

	// Serialization and Deserialization test
	var buf bytes.Buffer
	if err := g.Serialize(&buf); err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	g2 := NewGraph(cfg)
	if err := g2.Deserialize(&buf); err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if g2.Len() != numVectors {
		t.Fatalf("expected %d nodes after deserialize, got %d", numVectors, g2.Len())
	}

	res2, err := g2.Search(target, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res2) == 0 || res2[0].ID != 10 {
		t.Fatalf("expected ID 10 from deserialized graph, got %v", res2)
	}
}
