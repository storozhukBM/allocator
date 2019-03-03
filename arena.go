package allocator

import (
	"fmt"
	"math/rand"
	"reflect"
	"unsafe"
)

const defaultFirstBucketSize int = 16 * 1024

var Nil = APtr{}

type APtr struct {
	offset    uint32
	bucketIdx uint8

	arenaMask uint16
}

func (p APtr) ToRef(arena *Arena) unsafe.Pointer {
	if p == Nil {
		panic("nil pointer conversion")
	}
	if p.arenaMask != arena.arenaMask {
		panic("this pointer isn't part of passed arena")
	}
	targetBuffer := arena.buckets[p.bucketIdx].buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&targetBuffer))
	return unsafe.Pointer(header.Data + uintptr(p.offset))
}

type Arena struct {
	buckets []*bucket

	countOfAllocations int
	usedBytes          int
	overallCapacity    int

	arenaMask uint16
}

func (a *Arena) initIfNecessary() {
	if a.buckets == nil {
		a.buckets = append(a.buckets, &bucket{buffer: make([]byte, defaultFirstBucketSize)})
		a.overallCapacity = defaultFirstBucketSize
		a.arenaMask = uint16(rand.Uint32())
	}
}

func (a *Arena) Alloc(size uintptr) APtr {
	a.initIfNecessary()

	targetSize := int(size)
	bIdx := len(a.buckets) - 1
	b := a.buckets[bIdx]
	if targetSize > b.availableSize() {
		newSize := max(len(b.buffer)*2, targetSize*2)
		newBucket := &bucket{buffer: make([]byte, newSize)}
		a.buckets = append(a.buckets, newBucket)
		bIdx += 1
		b = newBucket

		a.overallCapacity += newSize
	}

	allocationOffset := b.offset
	b.offset += targetSize

	a.countOfAllocations += 1
	a.usedBytes += targetSize

	return APtr{bucketIdx: uint8(bIdx), offset: uint32(allocationOffset), arenaMask: a.arenaMask}
}

func (a *Arena) CountOfAllocations() int {
	return a.countOfAllocations
}

func (a *Arena) UsedBytes() int {
	return a.usedBytes
}

func (a *Arena) OverallCapacity() int {
	return a.overallCapacity
}

func (a *Arena) CountOfBuckets() int {
	return len(a.buckets)
}

type bucket struct {
	buffer []byte
	offset int
}

func (b *bucket) availableSize() int {
	return len(b.buffer) - b.offset
}

func (b *bucket) String() string {
	return fmt.Sprintf("bucket{size: %v; offset: %v; available: %v}", len(b.buffer), b.offset, b.availableSize())
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
