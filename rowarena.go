package allocator

import (
	"errors"
	"fmt"
	"unsafe"
)

var allocationLimit = errors.New("allocation limit")

type APtr struct {
	offset    uint32
	bucketIdx uint8

	arenaMask uint16
}

func (p APtr) String() string {
	return fmt.Sprintf("{mask: %v bucketIdx: %v offset: %v}", p.arenaMask, p.bucketIdx, p.offset)
}

type AOffset struct {
	p APtr
}

func (o AOffset) String() string {
	return o.p.String()
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
	paddingSize := calculateRequiredPadding(a.CurrentOffset(), targetAlignment)
	a.offset += paddingSize
	a.availableSize -= paddingSize

	allocationOffset := a.offset
	a.offset += targetSize
	a.availableSize -= targetSize
	return APtr{offset: uint32(allocationOffset)}, nil
}

func (a *RawArena) CurrentOffset() AOffset {
	return AOffset{p: APtr{offset: uint32(a.offset)}}
}

func (a *RawArena) ToRef(p APtr) unsafe.Pointer {
	return unsafe.Pointer(&a.buffer[int(p.offset)])
}

func (a *RawArena) String() string {
	return fmt.Sprintf("rowestarena{size: %v; offset: %v; available: %v}", len(a.buffer), a.offset, a.availableSize)
}

func (a *RawArena) AvailableSize() int {
	return a.availableSize
}

func (a *RawArena) Capacity() int {
	return len(a.buffer)
}
