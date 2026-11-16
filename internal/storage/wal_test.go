package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWALCommitAndRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-wal-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.wal")
	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}

	// Transaction 1: committed
	tx1 := uint64(100)
	if err := wal.WriteRecord(&WalRecord{
		Type:    WalRecordInsert,
		TxID:    tx1,
		PageID:  1,
		SlotID:  0,
		Payload: []byte("doc from committed transaction 1"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := wal.WriteRecord(&WalRecord{
		Type: WalRecordCommit,
		TxID: tx1,
	}); err != nil {
		t.Fatal(err)
	}

	// Transaction 2: uncommitted (aborted/crashed)
	tx2 := uint64(101)
	if err := wal.WriteRecord(&WalRecord{
		Type:    WalRecordInsert,
		TxID:    tx2,
		PageID:  1,
		SlotID:  1,
		Payload: []byte("doc from uncommitted transaction 2"),
	}); err != nil {
		t.Fatal(err)
	}

	// Recover
	records, err := wal.Recover()
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 recovered record, got %d", len(records))
	}
	if string(records[0].Payload) != "doc from committed transaction 1" {
		t.Fatalf("unexpected payload: %s", string(records[0].Payload))
	}

	// Reset WAL
	if err := wal.Reset(); err != nil {
		t.Fatal(err)
	}
	recoveredAfterReset, err := wal.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(recoveredAfterReset) != 0 {
		t.Fatalf("expected 0 records after reset, got %d", len(recoveredAfterReset))
	}

	if err := wal.Close(); err != nil {
		t.Fatal(err)
	}
}
