package allocator

import (
	"fmt"
	"unsafe"
)

type SimpleArena struct {
	target arena

	countOfAllocations int
	paddingOverhead    int
	dataBytes          int
	usedBytes          int
	overallCapacity    int
}

func (a *SimpleArena) ToRef(p APtr) unsafe.Pointer {
	a.init()
	return a.target.ToRef(p)
}

func (a *SimpleArena) Alloc(size, alignment uintptr) (APtr, error) {
	a.init()
	result, allocErr := a.target.Alloc(size, alignment)
	if allocErr != nil {
		return APtr{}, allocErr
	}

	targetSize := int(size)
	a.countOfAllocations += 1
	a.usedBytes = a.target.Capacity() - a.target.AvailableSize()
	a.dataBytes += targetSize
	a.paddingOverhead = a.usedBytes - a.dataBytes

	return result, nil
}

func (a *SimpleArena) CurrentOffset() AOffset {
	a.init()
	return a.target.CurrentOffset()
}

func (a *SimpleArena) String() string {
	a.init()
	offset := a.target.CurrentOffset()
	return fmt.Sprintf(
		"arena{offset: %v countOfAllocations: %v dataBytes: %v usedBytes: %v paddingOverhead %v overallCapacity %v}",
		offset, a.countOfAllocations, a.dataBytes, a.usedBytes, a.paddingOverhead, a.overallCapacity,
	)
}

func (a *SimpleArena) AvailableSize() int {
	if a.target == nil {
		return 0
	}
	return a.target.AvailableSize()
}

func (a *SimpleArena) Capacity() int {
	return a.overallCapacity
}

func (a *SimpleArena) CountOfAllocations() int {
	return a.countOfAllocations
}

func (a *SimpleArena) UsedBytes() int {
	return a.usedBytes
}

func (a *SimpleArena) DataBytes() int {
	return a.dataBytes
}

func (a *SimpleArena) PaddingOverhead() int {
	return a.paddingOverhead
}

func (a *SimpleArena) init() {
	if a.target == nil {
		a.target = &DynamicArena{}
	}
}
