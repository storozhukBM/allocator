package arena_test

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"runtime"
	"testing"
	"unsafe"
)

const requiredBytesForBasicTest = 128

type basicArenaCheckingStand struct {
	commonStandState
}

func (s *basicArenaCheckingStand) check(t *testing.T, target allocator) {
	{
		ptr, allocErr := target.Alloc(0, 1)
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		// here we expect 0 as:
		// current_alloc_size | padding | result_size |
		//                 +0 |      +0 |           0 |
		assert(target.Metrics().UsedBytes == 0, "expect used bytes should be 0. instead: %v", target.Metrics())
	}
	{
		ptr, allocErr := target.Alloc(1, 1)
		failOnError(t, allocErr)
		assert(ptr.String() != "", "can't be empty")
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		// here we expect 1 as:
		// current_alloc_size | padding | result_size |
		//                 +0 |      +0 |           0 |
		//                 +1 |      +0 |           1 |
		assert(target.Metrics().UsedBytes == 1, "expect used bytes should be 1. instead: %v", target.Metrics())
	}

	{
		ptr, allocErr := target.Alloc(3, 1)
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		// here we expect 4 as:
		// current_alloc_size | padding | result_size |
		//                 +0 |      +0 |           0 |
		//                 +1 |      +0 |           1 |
		//                 +3 |      +0 |           4 |
		assert(target.Metrics().UsedBytes == 4, "expect used bytes should be 4. instead: %v", target.Metrics())
	}
	{
		ptr, allocErr := target.Alloc(1, 1)
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		// here we expect 5 as:
		// current_alloc_size | padding | result_size |
		//                 +0 |      +0 |           0 |
		//                 +1 |      +0 |           1 |
		//                 +3 |      +0 |           4 |
		//                 +1 |      +0 |           5 |
		assert(target.Metrics().UsedBytes == 5, "expect used bytes should be 5. instead: %v", target.Metrics())
	}
	{
		ptr, testAlignmentErr := target.Alloc(4, 4)
		failOnError(t, testAlignmentErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		// here we expect 12 as:
		// current_alloc_size |    padding      | result_size |
		//                 +0 |      +0         |           0 |
		//                 +1 |      +0         |           1 |
		//                 +3 |      +0         |           4 |
		//                 +1 |      +0         |           5 |
		//                 +4 |      +(0|1|2|3) |          12 |
		assert(target.Metrics().UsedBytes <= 12, "expect used bytes should be less than 12. instead: %v", target.Metrics())
	}
	{
		alloc := arena.NewBytesView(target)

		sizeOfPerson := unsafe.Sizeof(person{})
		alignmentOfPerson := unsafe.Alignof(person{})

		bossPtr, allocErr := target.Alloc(sizeOfPerson, alignmentOfPerson)
		failOnError(t, allocErr)
		boss := (*person)(unsafe.Pointer(target.ToRef(bossPtr)))
		boss.Name, allocErr = alloc.EmbedAsString([]byte("Richard Bahman"))
		boss.Age = 44

		personPtr, allocErr := target.Alloc(sizeOfPerson, alignmentOfPerson)
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, personPtr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)

		ref := target.ToRef(personPtr)
		rawPtr := uintptr(ref)
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			p.Name, allocErr = alloc.EmbedAsString([]byte("John Smith"))
			failOnError(t, allocErr)
			p.Age = 21
			p.Manager = boss
		}
		runtime.GC()
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			assert(p.Name == "John Smith", "unexpected person state: %+v", p)
			assert(p.Age == 21, "unexpected person state: %+v", p)
			assert(p.Manager.Name == "Richard Bahman", "unexpected person state: %+v", p)
			assert(p.Manager.Age == 44, "unexpected person state: %+v", p)
		}
	}
	s.printStandState(t)
}
