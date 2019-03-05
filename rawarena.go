package allocator

import (
	"fmt"
	"math/rand"
	"reflect"
	"unsafe"
)

const defaultFirstBucketSize int = 16 * 1024

type APtr struct {
	offset    uint32
	bucketIdx uint8

	arenaMask uint16
}

type RawArena struct {
	buckets          []bucket
	currentBucket    bucket
	currentBucketIdx int

	arenaMask uint16
}

func (a *RawArena) Alloc(size uintptr) (APtr, error) {
	targetSize := int(size)
	if targetSize > a.currentBucket.availableSize {
		a.createNewBucket(targetSize)
	}
	allocationOffset := a.currentBucket.offset
	a.currentBucket.offset += targetSize
	a.currentBucket.availableSize -= targetSize
	return APtr{
		bucketIdx: uint8(a.currentBucketIdx),
		offset:    uint32(allocationOffset),
		arenaMask: a.arenaMask,
	}, nil
}

func (a *RawArena) ToRef(p APtr) unsafe.Pointer {
	if p.arenaMask != a.arenaMask {
		panic("pointer isn't part of this arena")
	}
	targetBuffer := a.currentBucket.buffer
	if p.bucketIdx != uint8(a.currentBucketIdx) {
		targetBuffer = a.buckets[p.bucketIdx].buffer
	}
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&targetBuffer))
	return unsafe.Pointer(header.Data + uintptr(p.offset))
}

func (a *RawArena) String() string {
	return fmt.Sprintf("arena{mask: %v}", a.arenaMask)
}

func (a *RawArena) createNewBucket(requiredAvailableSize int) {
	if a.arenaMask == 0 {
		a.arenaMask = uint16(rand.Uint32()) | 1
	}

	minSizeOfNewBucket := max(defaultFirstBucketSize, requiredAvailableSize*2)
	newSize := max(len(a.currentBucket.buffer)*2, minSizeOfNewBucket)
	newBucket := bucket{
		buffer:        make([]byte, newSize),
		availableSize: newSize,
	}
	if a.currentBucket.buffer != nil {
		a.buckets = append(a.buckets, a.currentBucket)
		a.currentBucketIdx += 1
	}
	a.currentBucket = newBucket
}

type bucket struct {
	buffer        []byte
	offset        int
	availableSize int
}

func (b *bucket) String() string {
	return fmt.Sprintf("bucket{size: %v; offset: %v; available: %v}", len(b.buffer), b.offset, b.availableSize)
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
