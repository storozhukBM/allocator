package arena

import (
	"fmt"
	"unsafe"
)

const minInternalBufferSize uintptr = 64 * 1024

// RawAllocator is the simplest bump pointer allocator
// that can operate on top of once allocated byte slice.
//
// It has almost none safety checks, and it can't grow dynamically,
// but it is the fastest implementation provided by this library,
// and it is created as a building block that we use to implement other allocators.
//
// All critical path methods like `Alloc`, `AllocUnaligned` and `ToRef` are designed to be inalienable.
//
// General advice would be to use other more high-level implementations like arena.GenericAllocator,
// available in this library, and refer to this one only if you really need to,
// and you understand all its caveats and potentially unsafe behavior.
type RawAllocator struct {
	startPtr unsafe.Pointer // strong reference to actual byte slice
	endPtr   uintptr
	offset   uintptr
}

// NewRawAllocator creates an instance of arena.RawAllocator
// and allocates the whole it's underlying buffer from the heap in advance.
func NewRawAllocator(size uint32) *RawAllocator {
	bytes := make([]byte, int(size))
	startPtr := unsafe.Pointer(&bytes[0])
	return &RawAllocator{
		startPtr: startPtr,
		endPtr:   uintptr(unsafe.Pointer(&bytes[size-1])),
		offset:   uintptr(startPtr),
	}
}

// NewRawAllocatorWithOptimalSize creates an instance of arena.RawAllocator
// and allocates the whole it's underlying buffer from the heap in advance.
// This method will figure-out size that will be >= size and will enable certain
// underlying optimizations like vectorized Clear operation.
func NewRawAllocatorWithOptimalSize(size uint32) *RawAllocator {
	targetSize := uintptr(max(int(size), int(minInternalBufferSize)))
	additionalPadding := calculatePadding(targetSize, minInternalBufferSize)
	return NewRawAllocator(uint32(targetSize + additionalPadding))
}

// AllocUnaligned performs allocation within an underlying buffer, but without automatic alignment.
// This method is more performant and can be used to allocate memory with an alignment of 1 byte
// or to create a dedicated method that will allocate only object with the same alignment,
// so there are no additional padding calculations required.
//
// IMPORTANT, this method is potentially UNSAFE to use, please if you use it,
// try to test/run your program with race detector or `-d=checkptr` flag.
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
// arena.Ptr can be converted to unsafe.Pointer by using arena allocator ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *RawAllocator) AllocUnaligned(size uintptr) (Ptr, error) {
	if a.offset+size > a.endPtr {
		return Ptr{}, AllocationLimitError
	}
	result := Ptr{offset: a.offset}
	a.offset += size
	return result, nil
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
// arena.Ptr can be converted to unsafe.Pointer by using arena allocator ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *RawAllocator) Alloc(size uintptr, alignment uintptr) (Ptr, error) {
	paddingSize := calculatePadding(a.offset, alignment)
	if a.offset+size+paddingSize > a.endPtr {
		return Ptr{}, AllocationLimitError
	}
	a.offset += paddingSize
	result := Ptr{offset: a.offset}
	a.offset += size
	return result, nil
}

// ToRef converts arena.Ptr to unsafe.Pointer.
//
// UNSAFE CAUTION This method doesn't perform bounds check. CAUTION UNSAFE
//
// Also, this RawAllocator.ToRef has no protection from the converting arena.Ptr
// that were allocated by other arenas, so you should be extra careful when using it,
// or please refer to other allocator implementations from this library
// that provide such safety checks.
//
// We'd suggest calling this method right before using the result pointer to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
//go:nocheckptr
func (a *RawAllocator) ToRef(p Ptr) unsafe.Pointer {
	return unsafe.Pointer(p.offset)
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
	sliceHdr := sliceHeader{
		Data: uintptr(a.startPtr),
		Len:  a.len(),
		Cap:  a.len(),
	}
	bytesToClear := *(*[]byte)(unsafe.Pointer(&sliceHdr))
	if len(bytesToClear) > 0 {
		sliceOffset := a.idx()
		padding := calculatePadding(sliceOffset, minInternalBufferSize)
		idx := min(int(sliceOffset+padding), len(bytesToClear))
		bytesToClear = bytesToClear[:idx]
	}
	clearBytes(bytesToClear)
	a.offset = uintptr(a.startPtr)
}

// Stats provides a snapshot of essential allocation statistics,
// that can be used by end-users or other allocators for introspection.
func (a *RawAllocator) Stats() Stats {
	return Stats{
		UsedBytes:                int(a.offset - uintptr(a.startPtr)),
		AllocatedBytes:           int(a.endPtr-uintptr(a.startPtr)) + 1,
		CountOfOnHeapAllocations: 0,
	}
}

// Metrics provides a snapshot of current allocation statistics,
// that can be used by end-users or other allocators for introspection.
func (a *RawAllocator) Metrics() Metrics {
	return Metrics{
		Stats:          a.Stats(),
		AvailableBytes: int(a.endPtr - a.offset),
		MaxCapacity:    int(a.endPtr-uintptr(a.startPtr)) + 1,
	}
}

// String provides a string snapshot of the current allocation offset.
func (a *RawAllocator) String() string {
	return fmt.Sprintf("rowarena{%v}", a.CurrentOffset())
}

func (a *RawAllocator) availableBytes() int {
	return int(a.endPtr - a.offset)
}

func (a *RawAllocator) idx() uintptr {
	return a.offset - uintptr(a.startPtr)
}

func (a *RawAllocator) len() int {
	return int(a.endPtr-uintptr(a.startPtr)) + 1
}
