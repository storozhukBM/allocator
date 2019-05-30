package arena_test

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"unsafe"
)

type arenaDynamicGrowthStand struct {
	commonStandState
}

func (s *arenaDynamicGrowthStand) check(t *testing.T, target allocator) {
	s.allocateDifferentObjects(t, target)

	var personPtrTarget arena.Ptr
	{
		boss := &person{name: "Richard Bahman", age: 44}

		personPtr, allocErr := target.Alloc(unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
		failOnError(t, allocErr)
		ref := target.ToRef(personPtr)
		rawPtr := uintptr(ref)
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			p.name = "John Smith"
			p.age = 21
			p.manager = boss
		}
		personPtrTarget = personPtr
	}

	s.allocateDifferentObjects(t, target)
	runtime.GC()

	{
		ref := target.ToRef(personPtrTarget)
		rawPtr := uintptr(ref)
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			assert(p.name == "John Smith", "unexpected person state: %+v", p)
			assert(p.age == 21, "unexpected person state: %+v", p)
			assert(p.manager.name == "Richard Bahman", "unexpected person state: %+v", p)
			assert(p.manager.age == 44, "unexpected person state: %+v", p)
		}
	}
	for i := 0; i < 3; i++ {
		target.Clear()
		func() {
			defer func() {
				wrongArenaToRefPanic := recover()
				assert(wrongArenaToRefPanic != nil, "toRef on cleared arena should trigger panic")
			}()
			target.ToRef(personPtrTarget)
		}()
		afterClearAllocatedBytes := target.Metrics().AllocatedBytes
		iterations := 0
		for target.Metrics().AllocatedBytes == afterClearAllocatedBytes {
			s.allocateDifferentObjects(t, target)
			iterations++
		}
		t.Logf("allocation cycles before a new bucket get allocated: %v", iterations)
	}
}

func (s *arenaDynamicGrowthStand) allocateDifferentObjects(t *testing.T, target allocator) {
	t.Logf("before allocation: %v", target.Metrics())
	type allocatedPerson struct {
		ptr    arena.Ptr
		person person
	}
	allocations := make([]allocatedPerson, 0, 100)
	scaleFactor := rand.Intn(9) + 1
	for i := 0; i < 1000*scaleFactor; i++ {
		_, allocErr := target.Alloc(genRandomSize(), genRandomAlignment())
		failOnError(t, allocErr)
		if rand.Float32() < 0.01 {
			personPtr, allocErr := target.Alloc(unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
			failOnError(t, allocErr)
			ref := target.ToRef(personPtr)
			rawPtr := uintptr(ref)
			p := (*person)(unsafe.Pointer(rawPtr))
			p.name = "John " + strconv.Itoa(rand.Int())
			p.age = uint(rand.Uint32())
			allocations = append(allocations, allocatedPerson{ptr: personPtr, person: *p})
		}
	}

	for _, alloc := range allocations {
		ref := target.ToRef(alloc.ptr)
		assert(ref != nil, "ref should be resolvable")
		rawPtr := uintptr(ref)
		p := (*person)(unsafe.Pointer(rawPtr))
		assert(p.name == alloc.person.name, "unexpected person state: %+v; %+v", p, alloc)
		assert(p.age == alloc.person.age, "unexpected person state: %+v; %+v", p, alloc)
	}
	t.Logf("after allocation: %v", target.Metrics())
}

func genRandomSize() uintptr {
	size := uintptr(rand.Intn(1024))
	modifier := rand.Float32()
	if modifier < 0.1 {
		size *= 1024
		return size
	} else if modifier < 0.05 {
		size *= 1024 * 1024
		return size
	} else if modifier < 0.003 {
		size *= 1024 * 1024 * 1024
		return size
	}
	return size
}

func genRandomAlignment() uintptr {
	alignments := []uintptr{1, 8, 16, 32}
	i := rand.Intn(3)
	return alignments[i]
}
