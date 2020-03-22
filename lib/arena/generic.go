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
	Clear()
}

// GenericAllocator is the wrapper on top of any other allocator that provides
// additional functionality, enhanced metrics, and safety features.
// It can be configured using arena.Options struct.
// By default, it creates arena.DynamicAllocator underneath, but you can path any other,
// already created allocator using arena.NewSubAllocator method.
//
// Preventable unsafe behaviors are:
//  - ToRef call with arena.Ptr that wasn't allocated by this arena.
//  - ToRef call with arena.Ptr that was allocated before arena.DynamicAllocator.Clear call.
//  - Alloc call with unsupported alignment value.
//
// GenericAllocator also has limits functionality so that you can specify upper allocation limits.
//
// General advice would be to use this GenericAllocator by default,
// and refer to other implementations only if you really need to.
type GenericAllocator struct {
	target    allocator
	arenaMask uint16

	delegateClear          bool
	allocationLimitInBytes int

	countOfAllocations int
	paddingOverhead    int
	dataBytes          int
	usedBytes          int
	allocatedBytes     int
	onHeapAllocations  int
}

// Options is a structure used to configure arena.GenericAllocator.
//
// You can configure:
//  - InitialCapacity - initial capacity of the underlying allocator,
//    if not specified we will use the default capacity of the underlying allocator.
//  - AllocationLimitInBytes - upper limit for allocations,
//    if not specified we will use the limit of the underlying allocator.
//  - DelegateClearToUnderlyingAllocator - delegate Clear call,
//    this option changes behaviour of Clear method, so it calls Clear on underlying allocator,
//    for additional details please refer to Clear method documentation.
type Options struct {
	InitialCapacity                    uint
	AllocationLimitInBytes             uint
	DelegateClearToUnderlyingAllocator bool
}

// NewGenericAllocator creates an instance of the arena.GenericAllocator
// configured by Options.
// For possible configuration options, please refer to arena.Options documentation.
//
// If you are OK with all arena.Options defaults, please pass the empty Options struct.
func NewGenericAllocator(opts Options) *GenericAllocator {
	result := &GenericAllocator{delegateClear: opts.DelegateClearToUnderlyingAllocator}
	if opts.InitialCapacity > 0 {
		result.target = NewDynamicAllocatorWithInitialCapacity(opts.InitialCapacity)
		result.allocatedBytes += result.target.Metrics().AllocatedBytes
	}
	if opts.AllocationLimitInBytes > 0 {
		result.allocationLimitInBytes = int(opts.AllocationLimitInBytes)
	}
	result.init()
	return result
}

// NewSubAllocator creates an allocator view on top of any other allocator instance.
//
// It can be used to distinguish allocators between different functional scopes
// because a new sub-allocator has a different arena mask and separate metrics.
// You can also set separate limits for the new sub-allocator.
//
// Sub-allocator delegates almost all operations to its underlying allocator called target.
func NewSubAllocator(target allocator, opts Options) *GenericAllocator {
	if target == nil {
		target = NewGenericAllocator(opts)
	}
	result := &GenericAllocator{target: target, delegateClear: opts.DelegateClearToUnderlyingAllocator}
	if opts.AllocationLimitInBytes > 0 {
		result.allocationLimitInBytes = int(opts.AllocationLimitInBytes)
	}
	result.init()
	return result
}

// Alloc performs allocation within the underlying target allocator.
//
// It returns arena.Ptr value, which is basically
// an offset and index of the target arena used for this allocation.
//
// alignment - should be a power of 2 number and can't be 0
// In case of any violations, panic will be thrown.
//
// arena.GenericAllocator has "limits" functionality, so it checks
// if a future allocation can violate specified allocationLimitInBytes
// and returns arena.AllocationLimitError, if so.
//
// arena.Ptr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// arena.Ptr can be converted to unsafe.Pointer by using arena.RawAllocator.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *GenericAllocator) Alloc(size, alignment uintptr) (Ptr, error) {
	a.init()
	targetSize := int(size)
	targetAlignment := uint32(alignment)

	if !isPowerOfTwo(targetAlignment) {
		panic(fmt.Errorf("alignment should be power of 2. actual value: %d", alignment))
	}
	targetPadding := calculatePadding(a.target.CurrentOffset().p.offset, targetAlignment)

	if a.allocationLimitInBytes > 0 && a.usedBytes+targetSize+int(targetPadding) > a.allocationLimitInBytes {
		return Ptr{}, AllocationLimitError
	}

	beforeCallMetrics := a.target.Metrics()
	result, allocErr := a.target.Alloc(size, alignment)
	if allocErr != nil {
		return Ptr{}, allocErr
	}
	afterCallMetrics := a.target.Metrics()

	a.countOfAllocations++
	a.usedBytes += afterCallMetrics.UsedBytes - beforeCallMetrics.UsedBytes
	a.dataBytes += targetSize
	a.paddingOverhead = a.usedBytes - a.dataBytes
	a.allocatedBytes += afterCallMetrics.AllocatedBytes - beforeCallMetrics.AllocatedBytes
	a.onHeapAllocations += afterCallMetrics.CountOfOnHeapAllocations - beforeCallMetrics.CountOfOnHeapAllocations

	result.arenaMask = a.arenaMask
	return result, nil
}

// ToRef converts arena.Ptr to unsafe.Pointer.
//
// This method performs bounds check, so it can panic if you pass an arena.Ptr
// allocated by different arena with internal offset bigger than the underlying buffer.
//
// Also, this GenericAllocator.ToRef has protection and can panic if you try to convert arena.Ptr
// that was allocated by other arena, this is done by comparison of arena.Ptr.arenaMask fields.
//
// We'd suggest calling this method right before using the result pointer to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *GenericAllocator) ToRef(p Ptr) unsafe.Pointer {
	if p.arenaMask != a.arenaMask {
		panic("pointer isn't part of this arena")
	}

	if a.target == nil {
		return nil
	}
	p.arenaMask = a.target.CurrentOffset().p.arenaMask
	return a.target.ToRef(p)
}

// CurrentOffset returns the current allocation offset.
// This method can be primarily used to build other allocators on top of arena.GenericAllocator.
func (a *GenericAllocator) CurrentOffset() Offset {
	a.init()
	result := a.target.CurrentOffset()
	result.p.arenaMask = a.arenaMask
	return result
}

// Clear gets rid of data in current allocator, clears metrics, and makes it available for re-use.
// According to DelegateClearToUnderlyingAllocator option, it will either call Clear on underlying allocator
// or simply gets rid of it, so it will create a new target during the first call after Clear.
//
// Clear invocation also changes the arena.GenericAllocator.arenaMask
// so it can prevent some "use after free" arena.GenericAllocator.ToRef calls with arena.Ptr allocated before Clear,
// but it can't catch usages of already converted values.
// To avoid such situations, we'd suggest calling this method right before using the result pointer to eliminate its
// visibility scope and potentially prevent it's escaping to the heap.
func (a *GenericAllocator) Clear() {
	if a.delegateClear {
		a.target.Clear()
	} else {
		a.target = nil
	}

	a.arenaMask = (a.arenaMask + 1) | 1
	a.paddingOverhead = 0
	a.dataBytes = 0
	a.usedBytes = 0
}

// Metrics provides a snapshot of current allocation statistics,
// that can be used by end-users or other allocators for introspection.
func (a *GenericAllocator) Metrics() Metrics {
	if a.target == nil {
		return Metrics{}
	}
	targetArenaMetrics := a.target.Metrics()
	result := Metrics{
		UsedBytes:                a.usedBytes,
		AvailableBytes:           targetArenaMetrics.AvailableBytes,
		AllocatedBytes:           a.allocatedBytes,
		MaxCapacity:              targetArenaMetrics.MaxCapacity,
		CountOfOnHeapAllocations: a.onHeapAllocations,
	}
	if a.allocationLimitInBytes > 0 {
		result.MaxCapacity = a.allocationLimitInBytes
		result.AvailableBytes = min(a.allocationLimitInBytes, targetArenaMetrics.AllocatedBytes) - result.UsedBytes
	}
	return result
}

// EnhancedMetrics provides few additional values and metrics besides the usual arena.Metrics.
type EnhancedMetrics struct {
	Metrics
	CountOfAllocations int // simple counter of Alloc calls
	PaddingOverhead    int // count of bytes used for alignment padding
	DataBytes          int // count of bytes used specifically for useful data
}

// String provides a string snapshot of the EnhancedMetrics state.
func (p EnhancedMetrics) String() string {
	return fmt.Sprintf(
		"{UsedBytes: %v AvailableBytes: %v AllocatedBytes %v MaxCapacity %v "+
			"CountOfOnHeapAllocations %v CountOfAllocations: %v PaddingOverhead: %v DataBytes: %v}",
		p.UsedBytes, p.AvailableBytes, p.AllocatedBytes, p.MaxCapacity,
		p.CountOfOnHeapAllocations, p.CountOfAllocations, p.PaddingOverhead, p.DataBytes,
	)
}

// EnhancedMetrics provides a snapshot of detailed allocation statistics,
// that can be used by end-users or other allocators for introspection.
func (a *GenericAllocator) EnhancedMetrics() EnhancedMetrics {
	return EnhancedMetrics{
		Metrics:            a.Metrics(),
		CountOfAllocations: a.countOfAllocations,
		PaddingOverhead:    a.paddingOverhead,
		DataBytes:          a.dataBytes,
	}
}

// String provides a string snapshot of the current allocation offset.
func (a *GenericAllocator) String() string {
	a.init()
	return fmt.Sprintf("arena{mask: %v target: %v}", a.arenaMask, a.target)
}

func (a *GenericAllocator) init() {
	if a.target == nil {
		a.target = &DynamicAllocator{}
	}
	if a.arenaMask == 0 {
		// here we can give guarantees that sub-arena mask will differ from parent arena
		modifier := uint16(rand.Uint32()) | 1
		a.arenaMask = (a.target.CurrentOffset().p.arenaMask + modifier) | 1
	}
}
