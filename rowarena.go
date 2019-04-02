package allocator

import (
	"errors"
	"fmt"
	"unsafe"
)

var allocationLimit = errors.New("allocation limit")

type AOffset struct {
	offset    uint32
	bucketIdx uint8

	arenaMask uint16
}

func (o AOffset) String() string {
	return fmt.Sprintf("offset{mask: %v bucketIdx: %v offset: %v}", o.arenaMask, o.bucketIdx, o.offset)
}

type APtr struct {
	offset    uint32
	bucketIdx uint8

	arenaMask uint16
}

func (p APtr) String() string {
	return fmt.Sprintf("offset{mask: %v bucketIdx: %v offset: %v}", p.arenaMask, p.bucketIdx, p.offset)
}

type RawArena struct {
	buffer        []byte
	offset        int
	availableSize int
}

func (a *RawArena) Alloc(size uintptr, alignment uintptr) (APtr, error) {
	targetSize := int(size)
	if targetSize > a.availableSize {
		return APtr{}, allocationLimit
	}

	targetAlignment := int(alignment)
	paddingSize := a.calculateRequiredPadding(targetAlignment)
	a.offset += paddingSize
	a.availableSize -= paddingSize

	allocationOffset := a.offset
	a.offset += targetSize
	a.availableSize -= targetSize
	return APtr{
		bucketIdx: 0,
		offset:    uint32(allocationOffset),
		arenaMask: 0,
	}, nil
}

func (a *RawArena) CurrentOffset() AOffset {
	return AOffset{
		offset:    uint32(a.offset),
		bucketIdx: uint8(0),
		arenaMask: 0,
	}
}

func (a *RawArena) ToRef(p APtr) unsafe.Pointer {
	return unsafe.Pointer(&a.buffer[int(p.offset)])
}

func (a *RawArena) String() string {
	return fmt.Sprintf("rowestarena{size: %v; offset: %v; available: %v}", len(a.buffer), a.offset, a.availableSize)
}

func (a *RawArena) calculateRequiredPadding(targetAlignment int) int {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (a.offset % targetAlignment)) % targetAlignment
}
