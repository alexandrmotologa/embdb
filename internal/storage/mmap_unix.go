//go:build !windows

package storage

import (
	"os"
	"syscall"
)

type MmapReader struct {
	file *os.File
	data []byte
	size int64
}

func OpenMmap(file *os.File) (*MmapReader, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return &MmapReader{file: file, size: 0, data: nil}, nil
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	return &MmapReader{
		file: file,
		data: data,
		size: size,
	}, nil
}

func (m *MmapReader) Bytes() []byte {
	return m.data
}

func (m *MmapReader) ReadPage(pageID uint32) (*Page, error) {
	offset := int64(pageID) * PageSize
	if offset+PageSize > m.size {
		return nil, ErrInvalidPageID
	}
	page := &Page{}
	copy(page.Data[:], m.data[offset:offset+PageSize])
	if !page.VerifyChecksum() {
		return nil, ErrCorruptChecksum
	}
	return page, nil
}

func (m *MmapReader) Close() error {
	if m.data != nil && len(m.data) > 0 {
		err := syscall.Munmap(m.data)
		m.data = nil
		return err
	}
	return nil
}
