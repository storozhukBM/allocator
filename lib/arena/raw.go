package arena

import (
	"fmt"
	"unsafe"
)

const minInternalBufferSize uint32 = 64 * 1024

// RawAllocator is the simplest bump pointer allocator
// that can operate on top of once allocated byte slice.
//
// It has almost none safety checks, and it can't grow dynamically,
// but it is the fastest implementation provided by this library,
// and it is created as a building block that we use to implement other allocators.
//
// All critical path methods like `Alloc` and `ToRef` are designed to be inalienable.
//
// General advice would be to use other more high-level implementations like arena.GenericAllocator,
// available in this library, and refer to this one only if you really need to,
// and you understand all its caveats and potentially unsafe behavior.
type RawAllocator struct {
	buffer []byte
	offset uint32
}

// NewRawAllocator creates an instance of arena.RawAllocator
// and allocates the whole it's underlying buffer from the heap in advance.
func NewRawAllocator(size uint32) *RawAllocator {
	return &RawAllocator{
		buffer: make([]byte, int(size)),
	}
}

// NewRawAllocatorWithOptimalSize creates an instance of arena.RawAllocator
// and allocates the whole it's underlying buffer from the heap in advance.
// This method will figure-out size that will be >= size and will enable certain
// underlying optimizations like vectorized Clear operation.
func NewRawAllocatorWithOptimalSize(size uint32) *RawAllocator {
	targetSize := uint32(max(int(size), int(minInternalBufferSize)))
	additionalPadding := calculatePadding(targetSize, minInternalBufferSize)
	return NewRawAllocator(targetSize + additionalPadding)
}

// Alloc performs allocation within an underlying buffer.
//
// It returns arena.Ptr value, which is basically an offset of the allocated value
// inside the underlying buffer.
//
// alignment - should be a power of 2 number and can't be 0
// Important: this is a raw arena, it will not check violations of this contract.
// Any violations will lead to unpredictable behavior
//
// Alloc can return arena.AllocationLimitError if requested value size
// can't be fitted into the current buffer.
//
// arena.Ptr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// arena.Ptr can be converted to unsafe.Pointer by using arena.RawAllocator.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *RawAllocator) Alloc(size uintptr, alignment uintptr) (Ptr, error) {
	targetSize := uint32(size)
	targetAlignment := uint32(alignment)
	paddingSize := calculatePadding(a.offset, targetAlignment)

	if targetSize+paddingSize > uint32(len(a.buffer))-a.offset {
		return Ptr{}, AllocationLimitError
	}
	a.offset += paddingSize

	allocationOffset := a.offset
	a.offset += targetSize
	return Ptr{offset: allocationOffset}, nil
}

// ToRef converts arena.Ptr to unsafe.Pointer.
//
// This method performs bounds check, so it can panic if you pass an arena.Ptr
// allocated by different arena with internal offset bigger than the underlying buffer.
//
// Also, this RawAllocator.ToRef has no protection from the converting arena.Ptr
// that were allocated by other arenas, so you should be extra careful when using it,
// or please refer to other allocator implementations from this library
// that provide such safety checks.
//
// We'd suggest calling this method right before using the result pointer to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *RawAllocator) ToRef(p Ptr) unsafe.Pointer {
	targetOffset := int(p.offset)
	return unsafe.Pointer(&a.buffer[targetOffset])
}

// CurrentOffset returns the current allocation offset.
// This method can be primarily used to build other allocators on top of arena.RawAllocator.
func (a *RawAllocator) CurrentOffset() Offset {
	return Offset{p: Ptr{offset: a.offset}}
}

// Clear fills the underlying buffer with zeros and moves offset to zero.
//
// It can be a potentially unsafe operation if you try to dereference and/or use arena.Ptr
// that was allocated before the call to Clear method.
// To avoid such situation please refer to other allocator implementations from this library
// that provide additional safety checks.
func (a *RawAllocator) Clear() {
	bytesToClear := a.buffer
	if len(bytesToClear) > 0 {
		padding := calculatePadding(a.offset, minInternalBufferSize)
		idx := min(int(a.offset+padding), len(bytesToClear))
		bytesToClear = bytesToClear[:idx]
	}
	clearBytes(bytesToClear)
	a.offset = 0
}

// Metrics provides a snapshot of current allocation statistics,
// that can be used by end-users or other allocators for introspection.
func (a *RawAllocator) Metrics() Metrics {
	return Metrics{
		UsedBytes:                int(a.offset),
		AvailableBytes:           len(a.buffer) - int(a.offset),
		AllocatedBytes:           len(a.buffer),
		MaxCapacity:              len(a.buffer),
		CountOfOnHeapAllocations: 0,
	}
}

// String provides a string snapshot of the current allocation offset.
func (a *RawAllocator) String() string {
	return fmt.Sprintf("rowarena{%v}", a.CurrentOffset())
}
