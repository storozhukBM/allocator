package allocator

import (
	"fmt"
	"math"
	"math/rand"
	"unsafe"
)

const defaultFirstBucketSize int = 16 * 1024

type DynamicArena struct {
	arenas          []RawArena
	currentArena    RawArena
	currentArenaIdx int

	allocatedBytes int

	arenaMask uint16
}

func (a *DynamicArena) Alloc(size, alignment uintptr) (APtr, error) {
	targetSize := int(size)
	targetAlignment := max(int(alignment), 1)
	padding := calculateRequiredPadding(a.currentArena.CurrentOffset(), targetAlignment)
	if targetSize+padding > a.currentArena.availableSize {
		a.grow(targetSize + padding)
	}
	result, allocErr := a.currentArena.Alloc(size, alignment)
	if allocErr != nil {
		return APtr{}, allocErr
	}
	result.bucketIdx = uint8(a.currentArenaIdx)
	result.arenaMask = a.arenaMask
	return result, nil
}

func (a *DynamicArena) CurrentOffset() AOffset {
	offset := a.currentArena.CurrentOffset()
	offset.p.bucketIdx = uint8(a.currentArenaIdx)
	offset.p.arenaMask = a.arenaMask
	return offset
}

func (a *DynamicArena) ToRef(p APtr) unsafe.Pointer {
	if p.arenaMask != a.arenaMask {
		panic("pointer isn't part of this arena")
	}
	targetArena := a.currentArena
	if p.bucketIdx != uint8(a.currentArenaIdx) {
		targetArena = a.arenas[p.bucketIdx]
	}
	if targetArena.buffer == nil {
		return nil
	}
	return targetArena.ToRef(p)
}

func (a *DynamicArena) String() string {
	return fmt.Sprintf("arena{mask: %v}", a.arenaMask)
}

func (a *DynamicArena) Metrics() ArenaMetrics {
	currentArenaMetrics := a.currentArena.Metrics()
	return ArenaMetrics{
		UsedBytes:      a.allocatedBytes - currentArenaMetrics.AvailableBytes,
		AvailableBytes: currentArenaMetrics.AvailableBytes,
		AllocatedBytes: a.allocatedBytes,
		MaxCapacity:    a.allocatedBytes + (math.MaxInt8-len(a.arenas))*math.MaxUint32,
	}
}

func (a *DynamicArena) grow(requiredAvailableSize int) {
	a.init()
	minSizeOfNewArena := max(defaultFirstBucketSize, requiredAvailableSize*2)
	newSize := max(len(a.currentArena.buffer)*2, minSizeOfNewArena)
	newArena := RawArena{
		buffer:        make([]byte, newSize),
		availableSize: newSize,
	}
	if a.currentArena.buffer != nil {
		a.arenas = append(a.arenas, a.currentArena)
		a.currentArenaIdx += 1
	}
	a.allocatedBytes += newSize
	a.currentArena = newArena
}

func (a *DynamicArena) init() {
	if a.arenaMask == 0 {
		a.arenaMask = uint16(rand.Uint32()) | 1
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
