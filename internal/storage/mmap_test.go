package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMmapReader(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-mmap-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.embdb")
	c, err := CreateContainer(dbPath, 128, MetricCosine, QuantizeNone)
	if err != nil {
		t.Fatal(err)
	}

	p1, err := c.AllocatePage(PageTypeData)
	if err != nil {
		t.Fatal(err)
	}
	slot, err := p1.InsertItem([]byte("mmap test content"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.WritePage(p1); err != nil {
		t.Fatal(err)
	}
	if err := c.Sync(); err != nil {
		t.Fatal(err)
	}

	// Open mmap
	mmap, err := OpenMmap(c.file)
	if err != nil {
		t.Fatalf("failed to open mmap: %v", err)
	}
	defer mmap.Close()

	if mmap.size != 2*PageSize {
		t.Fatalf("expected mmap size %d, got %d", 2*PageSize, mmap.size)
	}

	page1, err := mmap.ReadPage(1)
	if err != nil {
		t.Fatalf("failed to read page 1 via mmap: %v", err)
	}

	item, err := page1.GetItem(slot)
	if err != nil {
		t.Fatalf("failed to get item: %v", err)
	}
	if string(item) != "mmap test content" {
		t.Fatalf("unexpected content: %s", string(item))
	}

	c.Close()
}
