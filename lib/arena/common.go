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
// allocator can afford the requested allocation.
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
// arena.Ptr can be converted to unsafe.Pointer by using arena.RawAllocator.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type Ptr struct {
	offset    uint32
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

// Metrics is a struct that represents a snapshot of current allocation statistics,
// that can be used by end-users or other allocators for introspection.
type Metrics struct {
	UsedBytes                int // count of bytes actually allocated and used inside an arena
	AvailableBytes           int // count of bytes that are reserved from the general heap, but aren't used
	AllocatedBytes           int // count of bytes that are allocated inside the general heap
	MaxCapacity              int // count of bytes that potentially can be allocated using specific arena
	CountOfOnHeapAllocations int // count of allocations performed inside the general heap
}

// String provides a string snapshot of the Metrics state.
func (p Metrics) String() string {
	return fmt.Sprintf(
		"{UsedBytes: %v AvailableBytes: %v AllocatedBytes %v MaxCapacity %v CountOfOnHeapAllocations %v}",
		p.UsedBytes, p.AvailableBytes, p.AllocatedBytes, p.MaxCapacity, p.CountOfOnHeapAllocations,
	)
}

func calculateRequiredPadding(o Offset, targetAlignment int) int {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (int(o.p.offset) % targetAlignment)) % targetAlignment
}

func clearBytes(buf []byte) {
	if len(buf) == 0 {
		return
	}
	buf[0] = 0
	for bufPart := 1; bufPart < len(buf); bufPart *= 2 {
		copy(buf[bufPart:], buf[:bufPart])
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
