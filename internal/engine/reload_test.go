package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPersistentReload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-reload-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "reload.embdb")
	db, err := OpenDB(dbPath, 8, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 5; i++ {
		err := db.Insert(&Document{
			Key:    fmt.Sprintf("item_%d", i),
			Text:   fmt.Sprintf("Document text number %d", i),
			Vector: []float32{1, 0, 0, 0, 0, 0, 0, 0},
		})
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}

	if db.Stats().TotalDocuments != 5 {
		t.Fatalf("expected 5 docs in memory, got %d", db.Stats().TotalDocuments)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen
	db2, err := OpenDB(dbPath, 8, 0, 0)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer db2.Close()

	stats := db2.Stats()
	if stats.TotalDocuments != 5 {
		t.Fatalf("expected 5 docs after reopen, got %d (pages: %d)", stats.TotalDocuments, stats.TotalPages)
	}
}
