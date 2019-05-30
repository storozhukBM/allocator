package arena_test

import (
	"runtime"
	"testing"
	"unsafe"
)

const requiredBytesForBasicTest = 48

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
		// current_alloc_size | padding | result_size |
		//                 +0 |      +0 |           0 |
		//                 +1 |      +0 |           1 |
		//                 +3 |      +0 |           4 |
		//                 +1 |      +0 |           5 |
		//                 +4 |      +3 |          12 |
		assert(target.Metrics().UsedBytes == 12, "expect used bytes should be 12. instead: %v", target.Metrics())
	}
	{
		boss := &person{name: "Richard Bahman", age: 44}

		personPtr, allocErr := target.Alloc(unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
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
			p.name = "John Smith"
			p.age = 21
			p.manager = boss
		}
		runtime.GC()
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			assert(p.name == "John Smith", "unexpected person state: %+v", p)
			assert(p.age == 21, "unexpected person state: %+v", p)
			assert(p.manager.name == "Richard Bahman", "unexpected person state: %+v", p)
			assert(p.manager.age == 44, "unexpected person state: %+v", p)
		}
	}
	s.printStandState(t)
}
