package arena_test

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

type arenaMaskCheckingStand struct {
	commonStandState
}

func (s *arenaMaskCheckingStand) check(t *testing.T, target allocator) {
	separateArena := arena.NewGenericAllocator(arena.Options{})
	subArena := arena.NewSubAllocator(target, arena.Options{})

	ptr, allocErr := target.Alloc(0, 1)
	failOnError(t, allocErr)
	s.checkPointerIsUnique(t, ptr)
	s.checkOffsetIsUnique(t, target.CurrentOffset())
	s.checkMetricsAreUnique(t, target.Metrics())
	s.checkEnhancedMetricsAreUnique(t, target)
	s.checkArenaStrIsUnique(t, target)

	func() {
		defer func() {
			wrongArenaToRefPanic := recover()
			assert(
				wrongArenaToRefPanic != nil,
				"toRef on different arena should trigger panic. arena: %v; ptr: %v",
				separateArena, ptr,
			)
		}()
		separateArena.ToRef(ptr)
	}()

	func() {
		defer func() {
			wrongArenaToRefPanic := recover()
			assert(
				wrongArenaToRefPanic != nil,
				"toRef on sub arena should trigger panic. arena: %v; ptr: %v",
				subArena, ptr,
			)
		}()
		subArena.ToRef(ptr)
	}()

	ref := target.ToRef(ptr)
	assert(ref != nil, "ref can't be nil")

	target.Clear()
	assert(target.Metrics().UsedBytes == 0, "expect used bytes should be 0. instead: %v", target.Metrics())
	_, testAlignmentErr := target.Alloc(20, 4)
	failOnError(t, testAlignmentErr)
	assert(target.Metrics().UsedBytes == 20, "expect used bytes should be 4. instead: %v", target.Metrics())

	func() {
		defer func() {
			wrongArenaToRefPanic := recover()
			assert(wrongArenaToRefPanic != nil, "toRef on cleared arena should trigger panic")
		}()
		target.ToRef(ptr)
	}()
}
