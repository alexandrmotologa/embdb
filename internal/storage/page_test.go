package storage

import (
	"bytes"
	"testing"
)

func TestSlottedPageInsertAndGet(t *testing.T) {
	page := NewPage(1, PageTypeData)
	if page.ID() != 1 {
		t.Fatalf("expected page ID 1, got %d", page.ID())
	}
	if page.Type() != PageTypeData {
		t.Fatalf("expected page type %d, got %d", PageTypeData, page.Type())
	}

	doc1 := []byte("hello world vector payload")
	doc2 := []byte("second entry in slotted page")

	slot0, err := page.InsertItem(doc1)
	if err != nil {
		t.Fatalf("failed to insert doc1: %v", err)
	}
	if slot0 != 0 {
		t.Fatalf("expected slot 0, got %d", slot0)
	}

	slot1, err := page.InsertItem(doc2)
	if err != nil {
		t.Fatalf("failed to insert doc2: %v", err)
	}
	if slot1 != 1 {
		t.Fatalf("expected slot 1, got %d", slot1)
	}

	res1, err := page.GetItem(slot0)
	if err != nil {
		t.Fatalf("failed to get item 0: %v", err)
	}
	if !bytes.Equal(res1, doc1) {
		t.Fatalf("got %s, expected %s", string(res1), string(doc1))
	}

	res2, err := page.GetItem(slot1)
	if err != nil {
		t.Fatalf("failed to get item 1: %v", err)
	}
	if !bytes.Equal(res2, doc2) {
		t.Fatalf("got %s, expected %s", string(res2), string(doc2))
	}

	page.UpdateChecksum()
	if !page.VerifyChecksum() {
		t.Fatal("checksum verification failed")
	}

	// Tamper with data and verify failure
	page.Data[100] ^= 0xFF
	if page.VerifyChecksum() {
		t.Fatal("checksum should fail after tampering")
	}
}
