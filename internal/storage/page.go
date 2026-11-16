package storage

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const (
	PageSize = 4096

	PageTypeHeader uint16 = 0x0001
	PageTypeData   uint16 = 0x0002
	PageTypeIndex  uint16 = 0x0003
	PageTypeFree   uint16 = 0x0004

	PageHeaderSize = 18 // PageID(4) + Type(2) + ItemCount(2) + Upper(2) + Lower(2) + Flags(2) + Checksum(4)
	SlotSize       = 4  // Offset(2) + Length(2)
)

var (
	ErrPageFull       = errors.New("storage: page has insufficient space")
	ErrSlotNotFound   = errors.New("storage: slot index out of range")
	ErrCorruptChecksum= errors.New("storage: page crc32 checksum mismatch")
)

// Page represents a 4KB slotted page.
type Page struct {
	Data [PageSize]byte
}

// NewPage initializes a blank page with given ID and Type.
func NewPage(pageID uint32, pageType uint16) *Page {
	p := &Page{}
	binary.LittleEndian.PutUint32(p.Data[0:4], pageID)
	binary.LittleEndian.PutUint16(p.Data[4:6], pageType)
	binary.LittleEndian.PutUint16(p.Data[6:8], 0)                 // ItemCount = 0
	binary.LittleEndian.PutUint16(p.Data[8:10], PageHeaderSize)   // Upper: end of slot directory (grows up)
	binary.LittleEndian.PutUint16(p.Data[10:12], PageSize)       // Lower: start of item payload (grows down)
	binary.LittleEndian.PutUint16(p.Data[12:14], 0)               // Flags
	binary.LittleEndian.PutUint32(p.Data[14:18], 0)               // Checksum
	return p
}

func (p *Page) ID() uint32 {
	return binary.LittleEndian.Uint32(p.Data[0:4])
}

func (p *Page) Type() uint16 {
	return binary.LittleEndian.Uint16(p.Data[4:6])
}

func (p *Page) ItemCount() uint16 {
	return binary.LittleEndian.Uint16(p.Data[6:8])
}

func (p *Page) FreeSpace() int {
	upper := int(binary.LittleEndian.Uint16(p.Data[8:10]))
	lower := int(binary.LittleEndian.Uint16(p.Data[10:12]))
	space := lower - upper
	if space < SlotSize {
		return 0
	}
	return space - SlotSize
}

// InsertItem inserts payload into the slotted page.
func (p *Page) InsertItem(payload []byte) (uint16, error) {
	itemLen := len(payload)
	needed := itemLen + SlotSize
	upper := int(binary.LittleEndian.Uint16(p.Data[8:10]))
	lower := int(binary.LittleEndian.Uint16(p.Data[10:12]))

	if lower-upper < needed {
		return 0, ErrPageFull
	}

	newLower := lower - itemLen
	copy(p.Data[newLower:lower], payload)

	slotIdx := binary.LittleEndian.Uint16(p.Data[6:8])
	slotOffset := upper

	binary.LittleEndian.PutUint16(p.Data[slotOffset:slotOffset+2], uint16(newLower))
	binary.LittleEndian.PutUint16(p.Data[slotOffset+2:slotOffset+4], uint16(itemLen))

	binary.LittleEndian.PutUint16(p.Data[6:8], slotIdx+1)
	binary.LittleEndian.PutUint16(p.Data[8:10], uint16(upper+SlotSize))
	binary.LittleEndian.PutUint16(p.Data[10:12], uint16(newLower))

	return slotIdx, nil
}

// GetItem retrieves payload by slot index.
func (p *Page) GetItem(slotIdx uint16) ([]byte, error) {
	count := p.ItemCount()
	if slotIdx >= count {
		return nil, ErrSlotNotFound
	}

	slotOffset := PageHeaderSize + int(slotIdx)*SlotSize
	itemOffset := binary.LittleEndian.Uint16(p.Data[slotOffset : slotOffset+2])
	itemLen := binary.LittleEndian.Uint16(p.Data[slotOffset+2 : slotOffset+4])

	if itemOffset == 0 && itemLen == 0 {
		return nil, nil // Deleted item
	}

	res := make([]byte, itemLen)
	copy(res, p.Data[itemOffset:itemOffset+itemLen])
	return res, nil
}

// UpdateChecksum calculates and records CRC32 of page content.
func (p *Page) UpdateChecksum() {
	binary.LittleEndian.PutUint32(p.Data[14:18], 0)
	checksum := crc32.ChecksumIEEE(p.Data[:])
	binary.LittleEndian.PutUint32(p.Data[14:18], checksum)
}

// VerifyChecksum validates the CRC32 integrity.
func (p *Page) VerifyChecksum() bool {
	recorded := binary.LittleEndian.Uint32(p.Data[14:18])
	var temp [PageSize]byte
	copy(temp[:], p.Data[:])
	binary.LittleEndian.PutUint32(temp[14:18], 0)
	actual := crc32.ChecksumIEEE(temp[:])
	return recorded == actual
}
