package arena

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"unsafe"
)

// DynamicAllocator is the dynamically growable bump pointer allocator.
//
// It can grow it's capacity if needed and can prevent some types of unsafe behaviour by throwing panics.
// Preventable unsafe behaviors are:
//  - ToRef call with arena.Ptr that wasn't allocated by this arena.
//  - ToRef call with arena.Ptr that was allocated before arena DynamicAllocator.Clear call.
//  - Alloc call with unsupported alignment value.
//
// DynamicAllocator has to limits functionality, so for such features, please refer to arena.GenericAllocator.
//
// General advice would be to use arena.GenericAllocator, available in this library,
// and refer to this one only if you really need to.
type DynamicAllocator struct {
	freeListOfClearArenas []RawAllocator

	arenas          []RawAllocator
	currentArena    RawAllocator
	currentArenaIdx int

	usedBytes         int
	allocatedBytes    int
	onHeapAllocations int
	maxCapacity       int

	arenaMask uint16

	zeroPointerTarget [1]byte
}

// NewDynamicAllocator creates an instance of arena.DynamicAllocator.
func NewDynamicAllocator() *DynamicAllocator {
	return &DynamicAllocator{}
}

func NewDynamicAllocatorWithInitialCapacity(size uint) *DynamicAllocator {
	result := &DynamicAllocator{}
	result.grow(int(size))
	return result
}

// Alloc performs allocation within underlying arenas.
//
// It returns arena.Ptr value, which is basically
// an offset and index of arena used for this allocation.
//
// alignment - should be a power of 2 number and can't be 0
// In case of any violations, panic will be thrown.
//
// arena.DynamicAllocator can grow dynamically if required but has to limits functionality,
// so for such features, please refer to arena.GenericAllocator.
//
// arena.Ptr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// arena.Ptr can be converted to unsafe.Pointer by using arena.RawAllocator.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *DynamicAllocator) Alloc(size, alignment uintptr) (Ptr, error) {
	a.init()
	targetSize := uint32(size)
	targetAlignment := uint32(alignment)

	if !isPowerOfTwo(targetAlignment) {
		panic(fmt.Errorf("alignment should be power of 2. actual value: %d", alignment))
	}

	padding := calculatePadding(a.currentArena.offset, targetAlignment)
	resultSize := targetSize + padding
	if resultSize > uint32(len(a.currentArena.buffer))-a.currentArena.offset {
		a.grow(int(resultSize))
	}
	result, allocErr := a.currentArena.Alloc(size, uintptr(targetAlignment))
	if allocErr != nil {
		return Ptr{}, allocErr
	}
	a.usedBytes += int(resultSize)
	result.bucketIdx = uint8(a.currentArenaIdx)
	result.arenaMask = a.arenaMask
	return result, nil
}

// ToRef converts arena.Ptr to unsafe.Pointer.
//
// This method performs bounds check, so it can panic if you pass an arena.Ptr
// allocated by different arena with internal offset bigger than the underlying buffer.
//
// Also, this DynamicAllocator.ToRef has protection and can panic if you try to convert arena.Ptr
// that was allocated by other arena, this is done by comparison of arena.Ptr.arenaMask fields.
//
// We'd suggest calling this method right before using the result pointer to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
func (a *DynamicAllocator) ToRef(p Ptr) unsafe.Pointer {
	if p.arenaMask != a.arenaMask {
		panic("pointer isn't part of this arena")
	}
	targetArena := a.currentArena
	if p.bucketIdx != uint8(a.currentArenaIdx) {
		targetArena = a.arenas[p.bucketIdx]
	}
	if targetArena.buffer == nil && p.offset == 0 {
		return unsafe.Pointer(&a.zeroPointerTarget[0])
	}
	return targetArena.ToRef(p)
}

// CurrentOffset returns the current allocation offset.
// This method can be primarily used to build other allocators on top of arena.DynamicAllocator.
func (a *DynamicAllocator) CurrentOffset() Offset {
	a.init()
	offset := a.currentArena.CurrentOffset()
	offset.p.bucketIdx = uint8(a.currentArenaIdx)
	offset.p.arenaMask = a.arenaMask
	return offset
}

// Clear fills all underlying buffers with zeros and moves offsets to zero.
// Moves all unused arenas to free-list that can be used to prevent future allocations.
//
// Clear invocation also changes the arena.DynamicAllocator.arenaMask
// so it can prevent some "use after free" arena.DynamicAllocator.ToRef calls with arena.Ptr allocated before Clear,
// but it can't catch usages of already converted values.
// To avoid such situations, we'd suggest calling this method right before using the result pointer to eliminate its
// visibility scope and potentially prevent it's escaping to the heap.
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
	a.arenas = a.arenas[:0]

	sort.Slice(a.freeListOfClearArenas, func(i, j int) bool {
		return a.freeListOfClearArenas[i].Metrics().MaxCapacity < a.freeListOfClearArenas[j].Metrics().MaxCapacity
	})

	a.currentArenaIdx = 0
	a.usedBytes = 0
	a.arenaMask = (a.arenaMask + 1) | 1
}

// Metrics provides a snapshot of current allocation statistics,
// that can be used by end-users or other allocators for introspection.
func (a *DynamicAllocator) Metrics() Metrics {
	return Metrics{
		UsedBytes: a.usedBytes,
		// we inline AvailableBytes calculation by hand to avoid full call to a.currentArena.Metrics
		AvailableBytes:           len(a.currentArena.buffer) - int(a.currentArena.offset),
		AllocatedBytes:           a.allocatedBytes,
		MaxCapacity:              a.maxCapacity,
		CountOfOnHeapAllocations: a.onHeapAllocations,
	}
}

// String provides a string snapshot of the current allocation offset.
func (a *DynamicAllocator) String() string {
	a.init()
	return fmt.Sprintf("dynarena{mask: %v offset: %v}", a.arenaMask, a.CurrentOffset())
}

func (a *DynamicAllocator) grow(requiredAvailableSize int) {
	newSize := max(len(a.currentArena.buffer)*2, requiredAvailableSize*2)
	newArena := a.getNewArena(newSize)
	if a.currentArena.buffer != nil {
		a.arenas = append(a.arenas, a.currentArena)
		a.currentArenaIdx++
	}
	a.currentArena = newArena
}

func (a *DynamicAllocator) getNewArena(size int) RawAllocator {
	if a.freeListOfClearArenas == nil {
		newRawArena := NewRawAllocatorWithOptimalSize(uint32(size))
		a.updateAllocationMetrics(len(newRawArena.buffer))
		return *newRawArena
	}

	newArenaFromFreeList, ok := a.tryToPickClearArenaFromFreeList(size)
	if ok {
		return newArenaFromFreeList
	}

	// there will be nothing suitable in free list in future
	// because next sizes will always be bigger than current
	a.freeListOfClearArenas = nil
	newRawArena := NewRawAllocatorWithOptimalSize(uint32(size))
	a.updateAllocationMetrics(len(newRawArena.buffer))
	return *newRawArena
}

func (a *DynamicAllocator) updateAllocationMetrics(allocatedBytes int) {
	a.allocatedBytes += allocatedBytes
	a.onHeapAllocations++
	a.maxCapacity = a.allocatedBytes + (math.MaxInt8-len(a.arenas))*math.MaxUint32
}

func (a *DynamicAllocator) tryToPickClearArenaFromFreeList(size int) (RawAllocator, bool) {
	candidateIdx := sort.Search(len(a.freeListOfClearArenas), func(i int) bool {
		return a.freeListOfClearArenas[i].Metrics().MaxCapacity >= size
	})
	if candidateIdx < len(a.freeListOfClearArenas) {
		newArena := a.freeListOfClearArenas[candidateIdx]
		// clear non-suitable candidates
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
