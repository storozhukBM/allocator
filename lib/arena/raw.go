package arena

import (
	"fmt"
	"unsafe"
)

type Raw struct {
	buffer        []byte
	offset        int
	availableSize int
}

func NewRawArena(size uint) *Raw {
	return &Raw{
		buffer:        make([]byte, int(size)),
		availableSize: int(size),
	}
}

func (a *Raw) Alloc(size uintptr, alignment uintptr) (Ptr, error) {
	targetAlignment := int(alignment)
	paddingSize := calculateRequiredPadding(a.CurrentOffset(), targetAlignment)
	targetSize := int(size)
	if targetSize+paddingSize > a.availableSize {
		return Ptr{}, AllocationLimitError
	}

	a.offset += paddingSize
	a.availableSize -= paddingSize

	allocationOffset := a.offset
	a.offset += targetSize
	a.availableSize -= targetSize
	return Ptr{offset: uint32(allocationOffset)}, nil
}

func (a *Raw) Clear() {
	clearBytes(a.buffer)
	a.offset = 0
	a.availableSize = len(a.buffer)
}

func (a *Raw) CurrentOffset() Offset {
	return Offset{p: Ptr{offset: uint32(a.offset)}}
}

func (a *Raw) ToRef(p Ptr) unsafe.Pointer {
	targetOffset := int(p.offset) % len(a.buffer)
	return unsafe.Pointer(&a.buffer[targetOffset])
}

func (a *Raw) String() string {
	return fmt.Sprintf("rowarena{%v}", a.CurrentOffset())
}

func (a *Raw) Metrics() Metrics {
	return Metrics{
		UsedBytes:      a.offset,
		AvailableBytes: a.availableSize,
		AllocatedBytes: len(a.buffer),
		MaxCapacity:    len(a.buffer),
	}
}
