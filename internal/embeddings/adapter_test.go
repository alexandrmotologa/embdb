package embeddings

import (
	"context"
	"testing"
)

func TestDeterministicAdapter(t *testing.T) {
	adapter := NewDeterministicAdapter(128)
	if adapter.Dimension() != 128 {
		t.Fatalf("expected dim 128, got %d", adapter.Dimension())
	}

	ctx := context.Background()
	text1 := "hybrid vector database search"
	text2 := "hybrid vector database search"
	text3 := "completely unrelated culinary recipe"

	v1, err := adapter.Embed(ctx, text1)
	if err != nil {
		t.Fatal(err)
	}
	if len(v1) != 128 {
		t.Fatalf("expected vector length 128, got %d", len(v1))
	}

	v2, err := adapter.Embed(ctx, text2)
	if err != nil {
		t.Fatal(err)
	}

	// Must be perfectly deterministic
	for i := 0; i < 128; i++ {
		if v1[i] != v2[i] {
			t.Fatalf("deterministic embeddings did not match at index %d", i)
		}
	}

	v3, err := adapter.Embed(ctx, text3)
	if err != nil {
		t.Fatal(err)
	}

	// Different text must have different vector
	diff := false
	for i := 0; i < 128; i++ {
		if v1[i] != v3[i] {
			diff = true
			break
		}
	}
	if !diff {
		t.Fatal("different texts produced identical embeddings")
	}
}
