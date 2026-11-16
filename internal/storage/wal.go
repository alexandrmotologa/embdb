package storage

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

const (
	WalRecordInsert     uint8 = 0x01
	WalRecordUpdate     uint8 = 0x02
	WalRecordDelete     uint8 = 0x03
	WalRecordCommit     uint8 = 0x04
	WalRecordCheckpoint uint8 = 0x05

	WalRecordHeaderSize = 19 // Size(4) + Type(1) + TxID(8) + PageID(4) + SlotID(2)
)

var (
	ErrWalCorrupt = errors.New("storage: wal record corrupted")
)

type WalRecord struct {
	Type    uint8
	TxID    uint64
	PageID  uint32
	SlotID  uint16
	Payload []byte
}

type WAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func OpenWAL(path string) (*WAL, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		file.Close()
		return nil, err
	}
	return &WAL{
		file: file,
		path: path,
	}, nil
}

// WriteRecord appends a record with CRC32 checksum to the WAL.
func (w *WAL) WriteRecord(rec *WalRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	payloadLen := len(rec.Payload)
	totalLen := uint32(WalRecordHeaderSize + payloadLen + 4) // +4 for CRC32

	buf := make([]byte, totalLen)
	binary.LittleEndian.PutUint32(buf[0:4], totalLen)
	buf[4] = rec.Type
	binary.LittleEndian.PutUint64(buf[5:13], rec.TxID)
	binary.LittleEndian.PutUint32(buf[13:17], rec.PageID)
	binary.LittleEndian.PutUint16(buf[17:19], rec.SlotID)

	if payloadLen > 0 {
		copy(buf[19:19+payloadLen], rec.Payload)
	}

	checksum := crc32.ChecksumIEEE(buf[:totalLen-4])
	binary.LittleEndian.PutUint32(buf[totalLen-4:], checksum)

	if _, err := w.file.Write(buf); err != nil {
		return err
	}
	return w.file.Sync()
}

// Recover reads all valid committed records from the WAL.
func (w *WAL) Recover() ([]*WalRecord, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var records []*WalRecord
	txRecords := make(map[uint64][]*WalRecord)
	committedTxs := make(map[uint64]bool)

	headerBuf := make([]byte, 4)
	for {
		_, err := io.ReadFull(w.file, headerBuf)
		if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		totalLen := binary.LittleEndian.Uint32(headerBuf)
		if totalLen < WalRecordHeaderSize+4 {
			break // Corrupted or truncated end
		}

		recordBuf := make([]byte, totalLen)
		copy(recordBuf[0:4], headerBuf)
		if _, err := io.ReadFull(w.file, recordBuf[4:]); err != nil {
			break // Incomplete record at EOF
		}

		expectedChecksum := binary.LittleEndian.Uint32(recordBuf[totalLen-4:])
		actualChecksum := crc32.ChecksumIEEE(recordBuf[:totalLen-4])
		if expectedChecksum != actualChecksum {
			return nil, ErrWalCorrupt
		}

		rec := &WalRecord{
			Type:   recordBuf[4],
			TxID:   binary.LittleEndian.Uint64(recordBuf[5:13]),
			PageID: binary.LittleEndian.Uint32(recordBuf[13:17]),
			SlotID: binary.LittleEndian.Uint16(recordBuf[17:19]),
		}

		payloadLen := int(totalLen) - WalRecordHeaderSize - 4
		if payloadLen > 0 {
			rec.Payload = make([]byte, payloadLen)
			copy(rec.Payload, recordBuf[19:19+payloadLen])
		}

		if rec.Type == WalRecordCommit {
			committedTxs[rec.TxID] = true
		} else {
			txRecords[rec.TxID] = append(txRecords[rec.TxID], rec)
		}
	}

	for txID, recList := range txRecords {
		if committedTxs[txID] {
			records = append(records, recList...)
		}
	}

	// Move cursor back to append mode
	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	return records, nil
}

// Reset truncates the WAL after a successful checkpoint.
func (w *WAL) Reset() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Truncate(0); err != nil {
		return err
	}
	_, err := w.file.Seek(0, io.SeekStart)
	return err
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
