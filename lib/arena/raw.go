package arena

import (
	"fmt"
	"unsafe"
)

type RawAllocator struct {
	buffer []byte
	offset int
}

func NewRawAllocator(size uint) *RawAllocator {
	return &RawAllocator{
		buffer: make([]byte, int(size)),
	}
}

func (a *RawAllocator) Alloc(size uintptr, alignment uintptr) (Ptr, error) {
	targetSize := int(size)
	targetAlignment := int(alignment)

	paddingSize := calculateRequiredPadding(a.CurrentOffset(), targetAlignment)
	if targetSize+paddingSize > len(a.buffer)-a.offset {
		return Ptr{}, AllocationLimitError
	}
	a.offset += paddingSize

	allocationOffset := a.offset
	a.offset += targetSize
	return Ptr{offset: uint32(allocationOffset)}, nil
}

func (a *RawAllocator) Clear() {
	clearBytes(a.buffer)
	a.offset = 0
}

func (a *RawAllocator) CurrentOffset() Offset {
	return Offset{p: Ptr{offset: uint32(a.offset)}}
}

func (a *RawAllocator) ToRef(p Ptr) unsafe.Pointer {
	targetOffset := int(p.offset) % len(a.buffer)
	return unsafe.Pointer(&a.buffer[targetOffset])
}

func (a *RawAllocator) String() string {
	return fmt.Sprintf("rowarena{%v}", a.CurrentOffset())
}

func (a *RawAllocator) Metrics() Metrics {
	return Metrics{
		UsedBytes:                a.offset,
		AvailableBytes:           len(a.buffer) - a.offset,
		AllocatedBytes:           len(a.buffer),
		MaxCapacity:              len(a.buffer),
		CountOfOnHeapAllocations: 0,
	}
}
