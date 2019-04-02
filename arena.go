package allocator

import (
	"fmt"
	"unsafe"
)

type Arena struct {
	target           DynamicArena
	lastAllocatedPrt APtr

	countOfAllocations int
	paddingOverhead    int
	dataBytes          int
	usedBytes          int
	overallCapacity    int
}

func (a *Arena) ToRef(p APtr) unsafe.Pointer {
	return a.target.ToRef(p)
}

func (a *Arena) Alloc(size, alignment uintptr) (APtr, error) {
	aPtrNil := APtr{}
	result, allocErr := a.target.Alloc(size, alignment)
	if allocErr != nil {
		return APtr{}, allocErr
	}
	if result.bucketIdx != a.lastAllocatedPrt.bucketIdx || a.lastAllocatedPrt == aPtrNil {
		a.overallCapacity += len(a.target.currentArena.buffer)
	}

	targetSize := int(size)
	a.countOfAllocations += 1
	a.usedBytes = a.overallCapacity - a.target.currentArena.availableSize
	a.dataBytes += targetSize
	a.paddingOverhead = a.usedBytes - a.dataBytes
	a.lastAllocatedPrt = result

	return result, nil
}

func (a *Arena) CurrentOffset() AOffset {
	return a.target.CurrentOffset()
}

func (a *Arena) String() string {
	return fmt.Sprintf(
		"arena{mask: %v countOfAllocations: %v dataBytes: %v usedBytes: %v paddingOverhead %v overallCapacity %v countOfBuckets: %v}",
		a.target.arenaMask, a.countOfAllocations, a.dataBytes, a.usedBytes, a.paddingOverhead, a.overallCapacity, a.CountOfBuckets(),
	)
}

func (a *Arena) CountOfAllocations() int {
	return a.countOfAllocations
}

func (a *Arena) UsedBytes() int {
	return a.usedBytes
}

func (a *Arena) DataBytes() int {
	return a.dataBytes
}

func (a *Arena) PaddingOverhead() int {
	return a.paddingOverhead
}

func (a *Arena) OverallCapacity() int {
	return a.overallCapacity
}

func (a *Arena) CountOfBuckets() int {
	return int(a.lastAllocatedPrt.bucketIdx) + 1
}
