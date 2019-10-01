package arena

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"unsafe"
)

const defaultFirstBucketSize int = 16 * 1024

type DynamicAllocator struct {
	freeListOfClearArenas []RawAllocator

	arenas          []RawAllocator
	currentArena    RawAllocator
	currentArenaIdx int

	allocatedBytes    int
	usedBytes         int
	onHeapAllocations int

	arenaMask uint16
}

func NewDynamicAllocator() *DynamicAllocator {
	return &DynamicAllocator{}
}

func dynamicWithInitialCapacity(size uint) *DynamicAllocator {
	result := &DynamicAllocator{}
	result.grow(int(size))
	return result
}

func (a *DynamicAllocator) Alloc(size, alignment uintptr) (Ptr, error) {
	a.init()
	targetSize := int(size)
	targetAlignment := max(int(alignment), 1)
	padding := calculateRequiredPadding(a.currentArena.CurrentOffset(), targetAlignment)
	if targetSize+padding > len(a.currentArena.buffer)-a.currentArena.offset {
		a.grow(targetSize + padding)
	}
	result, allocErr := a.currentArena.Alloc(size, uintptr(targetAlignment))
	if allocErr != nil {
		return Ptr{}, allocErr
	}
	a.usedBytes += targetSize + padding
	result.bucketIdx = uint8(a.currentArenaIdx)
	result.arenaMask = a.arenaMask
	return result, nil
}

func (a *DynamicAllocator) Clear() {
	if len(a.currentArena.buffer) > 0 {
		a.currentArena.Clear()
		a.freeListOfClearArenas = append(a.freeListOfClearArenas, a.currentArena)
	}
	a.currentArena = RawAllocator{}

	for _, ar := range a.arenas {
		if len(ar.buffer) > 0 {
			ar.Clear()
			a.freeListOfClearArenas = append(a.freeListOfClearArenas, ar)
		}
	}
	a.arenas = nil

	sort.Slice(a.freeListOfClearArenas, func(i, j int) bool {
		return a.freeListOfClearArenas[i].Metrics().MaxCapacity < a.freeListOfClearArenas[j].Metrics().MaxCapacity
	})

	a.currentArena = a.freeListOfClearArenas[0]
	if len(a.freeListOfClearArenas) > 1 {
		a.freeListOfClearArenas[0] = RawAllocator{}
		a.freeListOfClearArenas = a.freeListOfClearArenas[1:]
	}

	a.currentArenaIdx = 0
	a.usedBytes = 0
	a.arenaMask = (a.arenaMask + 1) | 1
}

func (a *DynamicAllocator) CurrentOffset() Offset {
	a.init()
	offset := a.currentArena.CurrentOffset()
	offset.p.bucketIdx = uint8(a.currentArenaIdx)
	offset.p.arenaMask = a.arenaMask
	return offset
}

func (a *DynamicAllocator) ToRef(p Ptr) unsafe.Pointer {
	if p.arenaMask != a.arenaMask {
		panic("pointer isn't part of this arena")
	}
	targetArena := a.currentArena
	if p.bucketIdx != uint8(a.currentArenaIdx) {
		targetArena = a.arenas[p.bucketIdx]
	}
	return targetArena.ToRef(p)
}

func (a *DynamicAllocator) String() string {
	a.init()
	return fmt.Sprintf("dynarena{mask: %v offset: %v}", a.arenaMask, a.CurrentOffset())
}

func (a *DynamicAllocator) Metrics() Metrics {
	currentArenaMetrics := a.currentArena.Metrics()
	return Metrics{
		UsedBytes:                a.usedBytes,
		AvailableBytes:           currentArenaMetrics.AvailableBytes,
		AllocatedBytes:           a.allocatedBytes,
		MaxCapacity:              a.allocatedBytes + (math.MaxInt8-len(a.arenas))*math.MaxUint32,
		CountOfOnHeapAllocations: a.onHeapAllocations,
	}
}

func (a *DynamicAllocator) grow(requiredAvailableSize int) {
	minSizeOfNewArena := max(defaultFirstBucketSize, requiredAvailableSize*2)
	newSize := max(len(a.currentArena.buffer)*2, minSizeOfNewArena)
	newArena := a.getNewArena(newSize)
	if a.currentArena.buffer != nil {
		a.arenas = append(a.arenas, a.currentArena)
		a.currentArenaIdx += 1
	}
	a.currentArena = newArena
}

func (a *DynamicAllocator) getNewArena(size int) RawAllocator {
	if a.freeListOfClearArenas == nil {
		newRawArena := NewRawAllocator(uint(size))
		a.allocatedBytes += size
		a.onHeapAllocations += 1
		return *newRawArena
	}

	newArenaFromFreeList, ok := a.tryToPickClearArenaFromFreeList(size)
	if ok {
		return newArenaFromFreeList
	}

	// there will be nothing suitable in free list in future
	// because next sizes will always be bigger than current
	a.freeListOfClearArenas = nil
	newRawArena := NewRawAllocator(uint(size))
	a.allocatedBytes += size
	a.onHeapAllocations += 1
	return *newRawArena
}

func (a *DynamicAllocator) tryToPickClearArenaFromFreeList(size int) (RawAllocator, bool) {
	candidateIdx := sort.Search(len(a.freeListOfClearArenas), func(i int) bool {
		return a.freeListOfClearArenas[i].Metrics().MaxCapacity >= size
	})
	if candidateIdx < len(a.freeListOfClearArenas) {
		newArena := a.freeListOfClearArenas[candidateIdx]
		// clear nonsuitable candidates
		for idx := range a.freeListOfClearArenas {
			a.freeListOfClearArenas[idx] = RawAllocator{}
			if idx == candidateIdx {
				break
			}
		}
		if candidateIdx+1 != len(a.freeListOfClearArenas) {
			a.freeListOfClearArenas = a.freeListOfClearArenas[candidateIdx+1:]
		}
		return newArena, true
	}
	return RawAllocator{}, false
}

func (a *DynamicAllocator) init() {
	if a.arenaMask == 0 {
		a.arenaMask = uint16(rand.Uint32()) | 1
	}
}
