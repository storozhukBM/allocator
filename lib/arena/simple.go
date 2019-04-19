package arena

import (
	"fmt"
	"unsafe"
)

type allocator interface {
	Alloc(size uintptr, alignment uintptr) (Ptr, error)
	CurrentOffset() Offset
	ToRef(p Ptr) unsafe.Pointer
	Metrics() Metrics
}

type EnhancedMetrics struct {
	Metrics
	CountOfAllocations int
	PaddingOverhead    int
	DataBytes          int
}

type Options struct {
	InitialCapacity uint
}

type Simple struct {
	target allocator

	countOfAllocations int
	paddingOverhead    int
	dataBytes          int
	usedBytes          int
	allocatedBytes     int
}

func New(opts Options) *Simple {
	result := &Simple{}
	if opts.InitialCapacity > 0 {
		result.target = dynamicWithInitialCapacity(opts.InitialCapacity)
	}
	return result
}

func (a *Simple) ToRef(p Ptr) unsafe.Pointer {
	a.init()
	return a.target.ToRef(p)
}

func (a *Simple) Alloc(size, alignment uintptr) (Ptr, error) {
	a.init()
	result, allocErr := a.target.Alloc(size, alignment)
	if allocErr != nil {
		return Ptr{}, allocErr
	}

	targetSize := int(size)
	a.countOfAllocations += 1
	arenaMetrics := a.target.Metrics()
	a.usedBytes = arenaMetrics.AllocatedBytes - arenaMetrics.AvailableBytes
	a.dataBytes += targetSize
	a.paddingOverhead = a.usedBytes - a.dataBytes

	return result, nil
}

func (a *Simple) CurrentOffset() Offset {
	a.init()
	return a.target.CurrentOffset()
}

func (a *Simple) String() string {
	a.init()
	offset := a.target.CurrentOffset()
	return fmt.Sprintf(
		"arena{offset: %v countOfAllocations: %v dataBytes: %v usedBytes: %v paddingOverhead %v allocatedBytes %v}",
		offset, a.countOfAllocations, a.dataBytes, a.usedBytes, a.paddingOverhead, a.allocatedBytes,
	)
}

func (a *Simple) Metrics() Metrics {
	result := Metrics{
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

func (a *Simple) EnhancedMetrics() EnhancedMetrics {
	return EnhancedMetrics{
		Metrics:            a.Metrics(),
		CountOfAllocations: a.countOfAllocations,
		PaddingOverhead:    a.paddingOverhead,
		DataBytes:          a.dataBytes,
	}
}

func (a *Simple) init() {
	if a.target == nil {
		a.target = &Dynamic{}
	}
}
