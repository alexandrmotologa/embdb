package storage

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"sync"
	"time"
)

var (
	MagicBytes = []byte("EMBDB_V1")

	ErrInvalidMagic       = errors.New("storage: invalid container magic header")
	ErrUnsupportedVersion = errors.New("storage: unsupported container version")
	ErrInvalidPageID      = errors.New("storage: page id exceeds total pages")
)

const (
	MetricCosine     uint32 = 0
	MetricL2         uint32 = 1
	MetricDotProduct uint32 = 2

	QuantizeNone uint32 = 0
	QuantizeFP16 uint32 = 1
	QuantizeSQ8  uint32 = 2
)

// Header holds database container metadata.
type Header struct {
	Magic            [8]byte
	PageSize         uint32
	Version          uint32
	DistanceMetric   uint32
	QuantizationMode uint32
	VectorDimension  uint32
	TotalPages       uint32
	TotalDocuments   uint64
	HnswRootPage     uint32
	Bm25RootPage     uint32
	MetaRootPage     uint32
	GraphRootPage    uint32
	FreeListHead     uint32
	CreatedAt        int64
	ModifiedAt       int64
	Checksum         uint32
}

// Container represents an open .embdb single-file storage.
type Container struct {
	mu     sync.RWMutex
	file   *os.File
	path   string
	wal    *WAL
	header Header
}

// CreateContainer creates a new .embdb container file on disk.
func CreateContainer(path string, dim uint32, metric uint32, quant uint32) (*Container, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	walPath := path + "-wal"
	wal, err := OpenWAL(walPath)
	if err != nil {
		file.Close()
		return nil, err
	}

	now := time.Now().Unix()
	c := &Container{
		file: file,
		path: path,
		wal:  wal,
		header: Header{
			PageSize:         PageSize,
			Version:          1,
			DistanceMetric:   metric,
			QuantizationMode: quant,
			VectorDimension:  dim,
			TotalPages:       1, // Page 0 is header page
			TotalDocuments:   0,
			CreatedAt:        now,
			ModifiedAt:       now,
		},
	}
	copy(c.header.Magic[:], MagicBytes)

	// Write Page 0 (Header Page)
	headerPage := NewPage(0, PageTypeHeader)
	c.serializeHeader(headerPage.Data[:])
	headerPage.UpdateChecksum()

	if _, err := file.WriteAt(headerPage.Data[:], 0); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

// OpenContainer opens an existing .embdb file.
func OpenContainer(path string) (*Container, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	headerPage := &Page{}
	if _, err := file.ReadAt(headerPage.Data[:], 0); err != nil {
		file.Close()
		return nil, err
	}

	if !headerPage.VerifyChecksum() {
		file.Close()
		return nil, ErrCorruptChecksum
	}

	walPath := path + "-wal"
	wal, err := OpenWAL(walPath)
	if err != nil {
		file.Close()
		return nil, err
	}

	c := &Container{
		file: file,
		path: path,
		wal:  wal,
	}

	if err := c.deserializeHeader(headerPage.Data[:]); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Container) Header() Header {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.header
}

const (
	HeaderOffset = 32
)

func (c *Container) serializeHeader(buf []byte) {
	hBuf := buf[HeaderOffset:]
	copy(hBuf[0:8], c.header.Magic[:])
	binary.LittleEndian.PutUint32(hBuf[8:12], c.header.PageSize)
	binary.LittleEndian.PutUint32(hBuf[12:16], c.header.Version)
	binary.LittleEndian.PutUint32(hBuf[16:20], c.header.DistanceMetric)
	binary.LittleEndian.PutUint32(hBuf[20:24], c.header.QuantizationMode)
	binary.LittleEndian.PutUint32(hBuf[24:28], c.header.VectorDimension)
	binary.LittleEndian.PutUint32(hBuf[28:32], c.header.TotalPages)
	binary.LittleEndian.PutUint64(hBuf[32:40], c.header.TotalDocuments)
	binary.LittleEndian.PutUint32(hBuf[40:44], c.header.HnswRootPage)
	binary.LittleEndian.PutUint32(hBuf[44:48], c.header.Bm25RootPage)
	binary.LittleEndian.PutUint32(hBuf[48:52], c.header.MetaRootPage)
	binary.LittleEndian.PutUint32(hBuf[52:56], c.header.GraphRootPage)
	binary.LittleEndian.PutUint32(hBuf[56:60], c.header.FreeListHead)
	binary.LittleEndian.PutUint64(hBuf[60:68], uint64(c.header.CreatedAt))
	binary.LittleEndian.PutUint64(hBuf[68:76], uint64(c.header.ModifiedAt))

	// Checksum of the header block (first 76 bytes)
	c.header.Checksum = crc32.ChecksumIEEE(hBuf[0:76])
	binary.LittleEndian.PutUint32(hBuf[76:80], c.header.Checksum)
}

func (c *Container) deserializeHeader(buf []byte) error {
	hBuf := buf[HeaderOffset:]
	copy(c.header.Magic[:], hBuf[0:8])
	if string(c.header.Magic[:]) != string(MagicBytes) {
		return ErrInvalidMagic
	}

	c.header.PageSize = binary.LittleEndian.Uint32(hBuf[8:12])
	c.header.Version = binary.LittleEndian.Uint32(hBuf[12:16])
	if c.header.Version != 1 {
		return ErrUnsupportedVersion
	}

	c.header.DistanceMetric = binary.LittleEndian.Uint32(hBuf[16:20])
	c.header.QuantizationMode = binary.LittleEndian.Uint32(hBuf[20:24])
	c.header.VectorDimension = binary.LittleEndian.Uint32(hBuf[24:28])
	c.header.TotalPages = binary.LittleEndian.Uint32(hBuf[28:32])
	c.header.TotalDocuments = binary.LittleEndian.Uint64(hBuf[32:40])
	c.header.HnswRootPage = binary.LittleEndian.Uint32(hBuf[40:44])
	c.header.Bm25RootPage = binary.LittleEndian.Uint32(hBuf[44:48])
	c.header.MetaRootPage = binary.LittleEndian.Uint32(hBuf[48:52])
	c.header.GraphRootPage = binary.LittleEndian.Uint32(hBuf[52:56])
	c.header.FreeListHead = binary.LittleEndian.Uint32(hBuf[56:60])
	c.header.CreatedAt = int64(binary.LittleEndian.Uint64(hBuf[60:68]))
	c.header.ModifiedAt = int64(binary.LittleEndian.Uint64(hBuf[68:76]))
	c.header.Checksum = binary.LittleEndian.Uint32(hBuf[76:80])

	expectedChecksum := crc32.ChecksumIEEE(hBuf[0:76])
	if c.header.Checksum != expectedChecksum {
		return ErrCorruptChecksum
	}

	return nil
}

// SetTotalDocuments updates the total document count in the header.
func (c *Container) SetTotalDocuments(count uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.header.TotalDocuments = count
	c.header.ModifiedAt = time.Now().Unix()

	headerPage := NewPage(0, PageTypeHeader)
	c.serializeHeader(headerPage.Data[:])
	headerPage.UpdateChecksum()
	_, err := c.file.WriteAt(headerPage.Data[:], 0)
	return err
}

// AllocatePage creates a new page in the container.
func (c *Container) AllocatePage(pageType uint16) (*Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	newPageID := c.header.TotalPages
	page := NewPage(newPageID, pageType)
	page.UpdateChecksum()

	offset := int64(newPageID) * PageSize
	if _, err := c.file.WriteAt(page.Data[:], offset); err != nil {
		return nil, err
	}

	c.header.TotalPages++
	c.header.ModifiedAt = time.Now().Unix()

	// Persist updated header page 0
	headerPage := NewPage(0, PageTypeHeader)
	c.serializeHeader(headerPage.Data[:])
	headerPage.UpdateChecksum()
	if _, err := c.file.WriteAt(headerPage.Data[:], 0); err != nil {
		return nil, err
	}

	return page, nil
}

// ReadPage reads a page by pageID.
func (c *Container) ReadPage(pageID uint32) (*Page, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if pageID >= c.header.TotalPages {
		return nil, ErrInvalidPageID
	}

	page := &Page{}
	offset := int64(pageID) * PageSize
	if _, err := c.file.ReadAt(page.Data[:], offset); err != nil {
		if err == io.EOF {
			return nil, ErrInvalidPageID
		}
		return nil, err
	}

	if !page.VerifyChecksum() {
		return nil, ErrCorruptChecksum
	}

	return page, nil
}

// WritePage writes an updated page to disk.
func (c *Container) WritePage(page *Page) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if page.ID() >= c.header.TotalPages {
		return ErrInvalidPageID
	}

	page.UpdateChecksum()
	offset := int64(page.ID()) * PageSize
	if _, err := c.file.WriteAt(page.Data[:], offset); err != nil {
		return err
	}

	c.header.ModifiedAt = time.Now().Unix()
	return nil
}

// Sync flushes container and WAL to disk.
func (c *Container) Sync() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.file.Sync(); err != nil {
		return err
	}
	return nil
}

func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	if c.wal != nil {
		if err := c.wal.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.file != nil {
		if err := c.file.Sync(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := c.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
