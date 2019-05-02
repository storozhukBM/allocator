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

func (p EnhancedMetrics) String() string {
	return fmt.Sprintf(
		"{UsedBytes: %v AvailableBytes: %v AllocatedBytes %v MaxCapacity %v CountOfAllocations: %v PaddingOverhead: %v DataBytes: %v}",
		p.UsedBytes, p.AvailableBytes, p.AllocatedBytes, p.MaxCapacity, p.CountOfAllocations, p.PaddingOverhead, p.DataBytes,
	)
}

type AllocResult struct {
	Size             uintptr
	Alignment        uintptr
	ResultingMetrics EnhancedMetrics
}

type Options struct {
	InitialCapacity        uint
	AllocationLimitInBytes uint
}

type Simple struct {
	target allocator

	allocationLimitInBytes int

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
	if opts.AllocationLimitInBytes > 0 {
		result.allocationLimitInBytes = int(opts.AllocationLimitInBytes)
	}
	return result
}

func SubAllocator(target allocator, opts Options) *Simple {
	if target == nil {
		target = New(opts)
	}
	result := &Simple{target: target}
	if opts.AllocationLimitInBytes > 0 {
		result.allocationLimitInBytes = int(opts.AllocationLimitInBytes)
	}
	return result
}

func (a *Simple) ToRef(p Ptr) unsafe.Pointer {
	a.init()
	return a.target.ToRef(p)
}

func (a *Simple) Alloc(size, alignment uintptr) (Ptr, error) {
	a.init()
	targetAlignment := max(int(alignment), 1)
	targetSize := int(size)
	targetPadding := calculateRequiredPadding(a.target.CurrentOffset(), targetAlignment)

	if a.allocationLimitInBytes > 0 && a.usedBytes+targetSize+targetPadding >= a.allocationLimitInBytes {
		return Ptr{}, allocationLimit
	}

	beforeCallMetrics := a.target.Metrics()
	result, allocErr := a.target.Alloc(size, alignment)
	if allocErr != nil {
		return Ptr{}, allocErr
	}
	afterCallMetrics := a.target.Metrics()

	a.countOfAllocations += 1
	a.usedBytes += afterCallMetrics.UsedBytes - beforeCallMetrics.UsedBytes
	a.dataBytes += targetSize
	a.paddingOverhead = a.usedBytes - a.dataBytes
	a.allocatedBytes += afterCallMetrics.AllocatedBytes - beforeCallMetrics.AllocatedBytes

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
		"arena{offset: %v metrics: %v}",
		offset, a.EnhancedMetrics(),
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
	if a.allocationLimitInBytes > 0 {
		result.MaxCapacity = a.allocationLimitInBytes
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
