package arena

import (
	"fmt"
	"unsafe"
)

// RawAllocator is the simplest bump pointer allocator
// that can operate on top of once allocated byte slice.
//
// It has almost none safety checks, and it can't grow dynamically,
// but it is the fastest implementation provided by this library,
// and it is created as a building block that we use to implement other allocators.
//
// All critical path methods like `Alloc` and `ToRef` are designed to be inalienable.
//
// General advice would be to use other more high-level implementations available in this library
// and refer to this one only if you really need to,
// and you understand all its caveats and potentially unsafe behavior.
type RawAllocator struct {
	buffer []byte
	offset int
}

// NewRawAllocator creates an instance of arena.RawAllocator
// and allocates the whole it's underlying buffer from the heap in advance.
func NewRawAllocator(size uint) *RawAllocator {
	return &RawAllocator{
		buffer: make([]byte, int(size)),
	}
}

// Alloc performs allocation within an underlying buffer.
//
// It returns arena.Ptr value, which is basically an offset of the allocated value
// inside the underlying buffer.
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

func (a *RawAllocator) CurrentOffset() Offset {
	return Offset{p: Ptr{offset: uint32(a.offset)}}
}

func (a *RawAllocator) Clear() {
	clearBytes(a.buffer)
	a.offset = 0
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

func (a *RawAllocator) String() string {
	return fmt.Sprintf("rowarena{%v}", a.CurrentOffset())
}
