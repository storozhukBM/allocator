package arena

import (
	"fmt"
	"math/rand"
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
		"{UsedBytes: %v AvailableBytes: %v AllocatedBytes %v MaxCapacity %v CountOfOnHeapAllocations %v CountOfAllocations: %v PaddingOverhead: %v DataBytes: %v}",
		p.UsedBytes, p.AvailableBytes, p.AllocatedBytes, p.MaxCapacity, p.CountOfOnHeapAllocations, p.CountOfAllocations, p.PaddingOverhead, p.DataBytes,
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
	target    allocator
	arenaMask uint16

	allocationLimitInBytes int

	countOfAllocations int
	paddingOverhead    int
	dataBytes          int
	usedBytes          int
	allocatedBytes     int
	onHeapAllocations  int
}

func New(opts Options) *Simple {
	result := &Simple{}
	if opts.InitialCapacity > 0 {
		result.target = dynamicWithInitialCapacity(opts.InitialCapacity)
		result.allocatedBytes += result.target.Metrics().AllocatedBytes
	}
	if opts.AllocationLimitInBytes > 0 {
		result.allocationLimitInBytes = int(opts.AllocationLimitInBytes)
	}
	result.init()
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
	result.init()
	return result
}

func (a *Simple) ToRef(p Ptr) unsafe.Pointer {
	if p.arenaMask != a.arenaMask {
		panic("pointer isn't part of this arena")
	}

	if a.target == nil {
		return nil
	}
	p.arenaMask = a.target.CurrentOffset().p.arenaMask
	return a.target.ToRef(p)
}

func (a *Simple) Alloc(size, alignment uintptr) (Ptr, error) {
	a.init()
	targetAlignment := max(int(alignment), 1)
	targetSize := int(size)
	targetPadding := calculateRequiredPadding(a.target.CurrentOffset(), targetAlignment)

	if a.allocationLimitInBytes > 0 && a.usedBytes+targetSize+targetPadding > a.allocationLimitInBytes {
		return Ptr{}, AllocationLimitError
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
	a.onHeapAllocations += afterCallMetrics.CountOfOnHeapAllocations - beforeCallMetrics.CountOfOnHeapAllocations

	result.arenaMask = a.arenaMask
	return result, nil
}

func (a *Simple) Clear() {
	a.target = nil

	a.arenaMask = (a.arenaMask + 1) | 1
	a.paddingOverhead = 0
	a.dataBytes = 0
	a.usedBytes = 0
}

func (a *Simple) CurrentOffset() Offset {
	a.init()
	result := a.target.CurrentOffset()
	result.p.arenaMask = a.arenaMask
	return result
}

func (a *Simple) String() string {
	a.init()
	return fmt.Sprintf("arena{mask: %v target: %v}", a.arenaMask, a.target)
}

func (a *Simple) Metrics() Metrics {
	result := Metrics{
		UsedBytes:                a.usedBytes,
		AllocatedBytes:           a.allocatedBytes,
		CountOfOnHeapAllocations: a.onHeapAllocations,
	}
	if a.target != nil {
		targetArenaMetrics := a.target.Metrics()
		result.AvailableBytes = targetArenaMetrics.AvailableBytes
		result.MaxCapacity = targetArenaMetrics.MaxCapacity
	}
	if a.allocationLimitInBytes > 0 {
		result.MaxCapacity = a.allocationLimitInBytes
		result.AvailableBytes = min(result.AvailableBytes, a.allocationLimitInBytes) - result.UsedBytes
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
	if a.arenaMask == 0 {
		// here we can give guarantees that sub-arena mask will differ from parent arena
		modifier := uint16(rand.Uint32()) | 1
		a.arenaMask = (a.target.CurrentOffset().p.arenaMask + modifier) | 1
	}
}
