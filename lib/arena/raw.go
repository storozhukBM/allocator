package arena

import (
	"fmt"
	"unsafe"
)

type Raw struct {
	buffer []byte
	offset int
}

func NewRawArena(size uint) *Raw {
	return &Raw{
		buffer: make([]byte, int(size)),
	}
}

func (a *Raw) Alloc(size uintptr, alignment uintptr) (Ptr, error) {
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

func (a *Raw) Clear() {
	clearBytes(a.buffer)
	a.offset = 0
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
		UsedBytes:                a.offset,
		AvailableBytes:           len(a.buffer) - a.offset,
		AllocatedBytes:           len(a.buffer),
		MaxCapacity:              len(a.buffer),
		CountOfOnHeapAllocations: 0,
	}
}
