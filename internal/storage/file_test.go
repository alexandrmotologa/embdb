package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainerLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-container-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.embdb")
	c, err := CreateContainer(dbPath, 384, MetricCosine, QuantizeSQ8)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	h := c.Header()
	if h.VectorDimension != 384 {
		t.Fatalf("expected dim 384, got %d", h.VectorDimension)
	}
	if h.DistanceMetric != MetricCosine {
		t.Fatalf("expected metric cosine, got %d", h.DistanceMetric)
	}
	if h.QuantizationMode != QuantizeSQ8 {
		t.Fatalf("expected SQ8, got %d", h.QuantizationMode)
	}
	if h.TotalPages != 1 {
		t.Fatalf("expected 1 initial page, got %d", h.TotalPages)
	}

	// Allocate page 1
	p1, err := c.AllocatePage(PageTypeData)
	if err != nil {
		t.Fatalf("failed to allocate page: %v", err)
	}
	if p1.ID() != 1 {
		t.Fatalf("expected page ID 1, got %d", p1.ID())
	}

	slot, err := p1.InsertItem([]byte("test item payload"))
	if err != nil {
		t.Fatalf("failed to insert item: %v", err)
	}
	if err := c.WritePage(p1); err != nil {
		t.Fatalf("failed to write page: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("failed to close container: %v", err)
	}

	// Reopen container
	c2, err := OpenContainer(dbPath)
	if err != nil {
		t.Fatalf("failed to open container: %v", err)
	}
	defer c2.Close()

	h2 := c2.Header()
	if h2.TotalPages != 2 {
		t.Fatalf("expected 2 total pages, got %d", h2.TotalPages)
	}

	readP1, err := c2.ReadPage(1)
	if err != nil {
		t.Fatalf("failed to read page 1: %v", err)
	}
	item, err := readP1.GetItem(slot)
	if err != nil {
		t.Fatalf("failed to get item from page 1: %v", err)
	}
	if string(item) != "test item payload" {
		t.Fatalf("unexpected item content: %s", string(item))
	}
}
