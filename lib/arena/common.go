package arena

import (
	"fmt"
)

// Error type used by the library to declare error constants.
type Error string

// Error method that implements error interface.
func (e Error) Error() string {
	return string(e)
}

// AllocationLimitError typically returned if
// allocator can't afford the requested allocation.
const AllocationLimitError = Error("allocation limit")

// AllocationInvalidArgumentError typically returned if
// you passed an invalid argument to the allocation method.
const AllocationInvalidArgumentError = Error("allocation argument is invalid")

// Ptr is a struct, which is basically represents an offset of the allocated value
// inside one of the arenas.
//
// arena.Ptr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// arena.Ptr can be converted to unsafe.Pointer by using arena allocator ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type Ptr struct {
	offset    uintptr
	bucketIdx uint8

	arenaMask uint16
}

// String provides a string snapshot of the current arena.Ptr.
func (p Ptr) String() string {
	return fmt.Sprintf("{mask: %v bucketIdx: %v offset: %v}", p.arenaMask, p.bucketIdx, p.offset)
}

//Offset is a arena.Ptr that can't be converted to unsafe.Pointer
//or used as any kind of reference.
//
//This struct can be primarily used to build other allocators on top of low-level arenas
//and can help to pre-calculate resulting padding or offset before performing the actual allocation.
type Offset struct {
	p Ptr
}

// String provides a string snapshot of the current arena.Offset.
func (o Offset) String() string {
	return o.p.String()
}

// Stats is a struct that represents a snapshot of essential allocation statistics,
// that can be used by end-users or other allocators for introspection.
type Stats struct {
	UsedBytes                int // count of bytes actually allocated and used inside an arena
	AllocatedBytes           int // count of bytes that are allocated inside the general heap
	CountOfOnHeapAllocations int // count of allocations performed inside the general heap
}

// String provides a string snapshot of the Metrics state.
func (s Stats) String() string {
	return fmt.Sprintf(
		"{UsedBytes: %v AllocatedBytes %v CountOfOnHeapAllocations %v}",
		s.UsedBytes, s.AllocatedBytes, s.CountOfOnHeapAllocations,
	)
}

// Metrics is a struct that represents a snapshot of current allocation statistics,
// that can be used by end-users or other allocators for introspection.
type Metrics struct {
	Stats
	AvailableBytes int // count of bytes that are reserved from the general heap, but aren't used
	MaxCapacity    int // count of bytes that potentially can be allocated using specific arena
}

// String provides a string snapshot of the Metrics state.
func (p Metrics) String() string {
	return fmt.Sprintf(
		"{UsedBytes: %v AvailableBytes: %v AllocatedBytes %v MaxCapacity %v CountOfOnHeapAllocations %v}",
		p.UsedBytes, p.AvailableBytes, p.AllocatedBytes, p.MaxCapacity, p.CountOfOnHeapAllocations,
	)
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}

func isPowerOfTwo(x uintptr) bool {
	return x != 0 && (x&(x-1)) == 0
}

func calculatePadding(offset uintptr, targetAlignment uintptr) uintptr {
	mask := targetAlignment - 1
	return (targetAlignment - (offset & mask)) & mask
}

func clearBytes(buf []byte) {
	if len(buf) == 0 {
		return
	}
	// this pattern will be recognized by compiler and optimized
	for i := range buf {
		buf[i] = 0
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
