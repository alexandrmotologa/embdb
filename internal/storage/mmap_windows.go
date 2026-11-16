//go:build windows

package storage

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

type MmapReader struct {
	file   *os.File
	handle syscall.Handle
	data   []byte
	size   int64
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

	hMap, err := syscall.CreateFileMapping(
		syscall.Handle(file.Fd()),
		nil,
		syscall.PAGE_READONLY,
		0,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}
	if hMap == 0 {
		return nil, errors.New("storage: failed to create file mapping handle")
	}

	addr, err := syscall.MapViewOfFile(
		hMap,
		syscall.FILE_MAP_READ,
		0,
		0,
		0,
	)
	if err != nil {
		syscall.CloseHandle(hMap)
		return nil, err
	}

	data := unsafe.Slice((*byte)(unsafe.Pointer(addr)), size)

	return &MmapReader{
		file:   file,
		handle: hMap,
		data:   data,
		size:   size,
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
		addr := uintptr(unsafe.Pointer(&m.data[0]))
		syscall.UnmapViewOfFile(addr)
		m.data = nil
	}
	if m.handle != 0 {
		syscall.CloseHandle(m.handle)
		m.handle = 0
	}
	return nil
}
