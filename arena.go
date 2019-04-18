package allocator

import (
	"fmt"
	"unsafe"
)

type EnhancedArenaMetrics struct {
	ArenaMetrics
	CountOfAllocations int
	PaddingOverhead    int
	DataBytes          int
}

type SimpleArena struct {
	target arena

	countOfAllocations int
	paddingOverhead    int
	dataBytes          int
	usedBytes          int
	allocatedBytes     int
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
	arenaMetrics := a.target.Metrics()
	a.usedBytes = arenaMetrics.AllocatedBytes - arenaMetrics.AvailableBytes
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
		"arena{offset: %v countOfAllocations: %v dataBytes: %v usedBytes: %v paddingOverhead %v allocatedBytes %v}",
		offset, a.countOfAllocations, a.dataBytes, a.usedBytes, a.paddingOverhead, a.allocatedBytes,
	)
}

func (a *SimpleArena) Metrics() ArenaMetrics {
	result := ArenaMetrics{
		UsedBytes:      a.usedBytes,
		AllocatedBytes: a.allocatedBytes,
	}
	if a.target != nil {
		targetArenaMetrics := a.target.Metrics()
		result.AvailableBytes = targetArenaMetrics.AvailableBytes
		result.MaxCapacity = targetArenaMetrics.MaxCapacity
	}
	return result
}

func (a *SimpleArena) EnhancedMetrics() EnhancedArenaMetrics {
	return EnhancedArenaMetrics{
		ArenaMetrics:       a.Metrics(),
		CountOfAllocations: a.countOfAllocations,
		PaddingOverhead:    a.paddingOverhead,
		DataBytes:          a.dataBytes,
	}
}

func (a *SimpleArena) init() {
	if a.target == nil {
		a.target = &DynamicArena{}
	}
}
