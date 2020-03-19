package arena_test

import (
	"encoding/json"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type arenaDynamicGrowthStand struct{}

func (s *arenaDynamicGrowthStand) check(t *testing.T, target allocator) {
	s.allocateDifferentObjects(t, target)
	alloc := arena.NewBytesView(target)

	var personTarget arena.Bytes
	{
		boss := &person{Name: "Richard Bahman", Age: 44}
		p := &person{Name: "John Smith", Age: 21, Manager: boss}
		arenaBuffer := arena.NewBuffer(target)
		encodingErr := json.NewEncoder(arenaBuffer).Encode(p)
		failOnError(t, encodingErr)
		personTarget = arenaBuffer.ArenaBytes()
	}

	s.allocateDifferentObjects(t, target)
	runtime.GC()

	{
		var p person
		unmarshalErr := json.Unmarshal(alloc.BytesToRef(personTarget), &p)
		failOnError(t, unmarshalErr)
		assert(p.Name == "John Smith", "unexpected person state: %+v", p)
		assert(p.Age == 21, "unexpected person state: %+v", p)
		assert(p.Manager.Name == "Richard Bahman", "unexpected person state: %+v", p)
		assert(p.Manager.Age == 44, "unexpected person state: %+v", p)
	}
	for i := 0; i < 3; i++ {
		target.Clear()
		func() {
			defer func() {
				wrongArenaToRefPanic := recover()
				assert(wrongArenaToRefPanic != nil, "toRef on cleared arena should trigger panic")
			}()
			alloc.BytesToRef(personTarget)
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
	for i := 0; i < scaleFactor*scaleFactor*scaleFactor; i++ {
		_, allocErr := target.Alloc(genRandomSize(), genRandomAlignment())
		failOnError(t, allocErr)
		if rand.Float32() < 0.01 {
			personPtr, allocErr := target.Alloc(unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
			failOnError(t, allocErr)
			ref := target.ToRef(personPtr)
			p := (*person)(ref)
			p.Name = "John " + strconv.Itoa(rand.Int())
			p.Age = uint(rand.Uint32())
			allocations = append(allocations, allocatedPerson{ptr: personPtr, person: *p})
		}
	}

	for _, alloc := range allocations {
		ref := target.ToRef(alloc.ptr)
		assert(ref != nil, "ref should be resolvable")
		p := (*person)(ref)
		assert(p.Name == alloc.person.Name, "unexpected person state: %+v; %+v", p, alloc)
		assert(p.Age == alloc.person.Age, "unexpected person state: %+v; %+v", p, alloc)
	}
	t.Logf("after allocation: %v", target.Metrics())
}

func genRandomSize() uintptr {
	size := uintptr(rand.Intn(1024))
	modifier := rand.Float32()
	switch {
	case modifier < 0.1:
		size *= 1024
		return size
	case modifier < 0.05:
		size *= 1024 * 1024
		return size
	case modifier < 0.003:
		size *= 1024 * 1024 * 1024
		return size
	default:
		return size
	}
}

func genRandomAlignment() uintptr {
	alignments := []uintptr{1, 8, 16, 32}
	i := rand.Intn(3)
	return alignments[i]
}
