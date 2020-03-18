package etalon_test_test

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"unsafe"

	"github.com/storozhukBM/allocator/generator/internal/testdata/etalon"
	"github.com/storozhukBM/allocator/lib/arena"
)

const bytesRequiredForBasicTest = 1184

func TestUninitializedAlloc(t *testing.T) {
	t.Parallel()
	s := &arenaGenAllocationCheckingStand{}
	s.check(t, nil)
}

func TestSimpleArenaWithoutConstructor(t *testing.T) {
	t.Parallel()
	a := &arena.GenericAllocator{}
	s := &arenaGenAllocationCheckingStand{}
	s.check(t, a)
	t.Logf("alloc metrics: %+v", a.Metrics())
}

func TestSimpleArenaWithVerySmallInitialCapacity(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{InitialCapacity: 12})

	s := &arenaGenAllocationCheckingStand{}
	s.check(t, a)
	t.Logf("alloc metrics: %+v", a.Metrics())
}

func TestSimpleArenaWithInitialCapacity(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{InitialCapacity: 64})

	s := &arenaGenAllocationCheckingStand{}
	s.check(t, a)
	t.Logf("alloc metrics: %+v", a.Metrics())
}

func TestDynamicArena(t *testing.T) {
	t.Parallel()

	a := arena.NewDynamicAllocator()
	for i := 0; i < 100; i++ {
		s := &arenaGenAllocationCheckingStand{}
		s.check(t, a)
	}
	t.Logf("alloc metrics: %+v", a.Metrics())

	a.Clear()
	t.Logf("alloc metrics: %+v", a.Metrics())

	s := &arenaGenAllocationCheckingStand{}
	s.check(t, a)
	t.Logf("alloc metrics: %+v", a.Metrics())
}

func TestSimpleArenaWithInitialCapacityAndAllocLimit(t *testing.T) {
	t.Parallel()

	a := arena.NewGenericAllocator(arena.Options{
		InitialCapacity:        bytesRequiredForBasicTest,
		AllocationLimitInBytes: 2 * bytesRequiredForBasicTest,
	})
	s := &arenaGenAllocationCheckingStand{}
	s.check(t, a)
	t.Logf("alloc metrics: %+v", a.Metrics())

	ls := &arenaGenAllocationLimitCheckingStand{}
	ls.check(t, a)
	t.Logf("alloc metrics: %+v", a.Metrics())
}

type testAllocator interface {
	Alloc(size, alignment uintptr) (arena.Ptr, error)
	ToRef(ptr arena.Ptr) unsafe.Pointer
	CurrentOffset() arena.Offset
	Metrics() arena.Metrics
	Clear()
}

type arenaGenAllocationCheckingStand struct{}

func (s *arenaGenAllocationCheckingStand) check(t *testing.T, target testAllocator) {
	alloc := etalon.NewStablePointsVectorView(target)
	s.verifySliceAlloc(t, target, alloc)
	s.verifyBufferAlloc(t, target, alloc)
	s.verifySingleItemAllocation(t, alloc)
}

func (s *arenaGenAllocationCheckingStand) verifyBufferAlloc(
	t *testing.T, target testAllocator,
	view *etalon.StablePointsVectorView,
) {
	alloc := view.Buffer
	arenaPointsVector, allocErr := alloc.MakeWithCapacity(0, 4)
	failOnError(t, allocErr)
	notEq(t, arenaPointsVector, nil, "new slice can't be empty")
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector, etalon.StablePointsVector{Points: [3]etalon.
			Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}})
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{{Points: [3]etalon.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}}}
		eq(t, expectedVector, alloc.ToRef(arenaPointsVector), "should be equal")
		eq(t, 1, arenaPointsVector.Len(), "len should be 1")
		eq(t, 4, arenaPointsVector.Cap(), "cap should be 4")
	}
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
		}
		eq(t, expectedVector, alloc.ToRef(arenaPointsVector), "should be equal")
		eq(t, 3, arenaPointsVector.Len(), "len should be 3")
		eq(t, 4, arenaPointsVector.Cap(), "cap should be 4")
	}
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
			{Points: [3]etalon.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			{Points: [3]etalon.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
		}
		eq(t, expectedVector, alloc.ToRef(arenaPointsVector), "should be equal")
		eq(t, 5, arenaPointsVector.Len(), "len should be 5")
		eq(t, true, arenaPointsVector.Cap() >= 5, "cap should be >= 5")
	}
	if target == nil {
		return
	}
	{
		// This call required to disable "subsequent allocations" optimisation
		// and observe actual reallocation of the whole buffer
		_, ptrAllocErr := target.Alloc(1, 1)
		failOnError(t, ptrAllocErr)

		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 1, Y: 2},
				{X: 1, Y: 2},
			}},
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 2, Y: 3},
				{X: 2, Y: 3},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
			{Points: [3]etalon.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			{Points: [3]etalon.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 1, Y: 2},
				{X: 1, Y: 2},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 2, Y: 3},
				{X: 2, Y: 3},
			}},
		}
		eq(t, expectedVector, alloc.ToRef(arenaPointsVector), "should be equal")
		eq(t, 7, arenaPointsVector.Len(), "len should be 7")
		eq(t, true, arenaPointsVector.Cap() >= 7, "cap should be >= 7")
	}
	{
		for l := -1; l < arenaPointsVector.Len()+1; l++ {
			for h := -1; h < arenaPointsVector.Len()+1; h++ {
				s.checkSubSlice(t, view, arenaPointsVector, l, h)
			}
		}
	}
	{
		for i := -1; i < arenaPointsVector.Len()+1; i++ {
			s.checkGet(t, view, arenaPointsVector, i)
		}
	}
	{
		arenaPointsVector, allocErr = alloc.Make(1)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{{Points: [3]etalon.Point{
			{X: 0, Y: 0},
			{X: 0, Y: 0},
			{X: 0, Y: 0},
		}}}
		eq(t, expectedVector, alloc.ToRef(arenaPointsVector), "should be equal")
		eq(t, 1, arenaPointsVector.Len(), "len should be 1")
		eq(t, 1, arenaPointsVector.Cap(), "cap should be 1")
	}
	{
		arenaPointsVector, allocErr := alloc.Append(
			etalon.StablePointsVectorBuffer{},
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{{Points: [3]etalon.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}}}
		eq(t, expectedVector, alloc.ToRef(arenaPointsVector), "should be equal")
		eq(t, 1, arenaPointsVector.Len(), "len should be 1")
		eq(t, true, arenaPointsVector.Cap() >= 1, "cap should be >= 1")
	}
}

func (s *arenaGenAllocationCheckingStand) verifySliceAlloc(
	t *testing.T, target testAllocator,
	view *etalon.StablePointsVectorView,
) {
	alloc := view.Slice
	arenaPointsVector, allocErr := alloc.MakeWithCapacity(0, 4)
	failOnError(t, allocErr)
	notEq(t, arenaPointsVector, nil, "new slice can't be empty")
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector, etalon.StablePointsVector{Points: [3]etalon.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}})
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{{Points: [3]etalon.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}}}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 1, len(arenaPointsVector), "len should be 1")
		eq(t, 4, cap(arenaPointsVector), "cap should be 4")
	}
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
		}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 3, len(arenaPointsVector), "len should be 3")
		eq(t, 4, cap(arenaPointsVector), "cap should be 4")
	}
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
			{Points: [3]etalon.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			{Points: [3]etalon.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
		}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 5, len(arenaPointsVector), "len should be 5")
		eq(t, true, cap(arenaPointsVector) >= 5, "cap should be >= 5")
	}
	if target == nil {
		return
	}
	{
		// This call required to disable "subsequent allocations" optimisation
		// and observe actual reallocation of the whole slice
		_, ptrAllocErr := target.Alloc(1, 1)
		failOnError(t, ptrAllocErr)

		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 1, Y: 2},
				{X: 1, Y: 2},
			}},
			etalon.StablePointsVector{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 2, Y: 3},
				{X: 2, Y: 3},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]etalon.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
			{Points: [3]etalon.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			{Points: [3]etalon.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
			{Points: [3]etalon.Point{
				{X: 1, Y: 2},
				{X: 1, Y: 2},
				{X: 1, Y: 2},
			}},
			{Points: [3]etalon.Point{
				{X: 2, Y: 3},
				{X: 2, Y: 3},
				{X: 2, Y: 3},
			}},
		}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 7, len(arenaPointsVector), "len should be 7")
		eq(t, true, cap(arenaPointsVector) >= 7, "cap should be >= 7")
	}
	{
		arenaPointsVector, allocErr = alloc.Make(1)
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{{Points: [3]etalon.Point{
			{X: 0, Y: 0},
			{X: 0, Y: 0},
			{X: 0, Y: 0},
		}}}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 1, len(arenaPointsVector), "len should be 1")
		eq(t, 1, cap(arenaPointsVector), "cap should be 1")
	}
	{
		arenaPointsVector, allocErr := alloc.Append(nil, etalon.StablePointsVector{Points: [3]etalon.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}})
		failOnError(t, allocErr)
		expectedVector := []etalon.StablePointsVector{{Points: [3]etalon.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}}}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 1, len(arenaPointsVector), "len should be 1")
		eq(t, true, cap(arenaPointsVector) >= 1, "cap should be >= 1")
	}
}

func (s *arenaGenAllocationCheckingStand) verifySingleItemAllocation(t *testing.T,
	view *etalon.StablePointsVectorView) {
	alloc := view.Ptr
	{
		pointsVectorPtr, allocErr := alloc.New()
		failOnError(t, allocErr)
		notEq(t, etalon.StablePointsVectorPtr{}, pointsVectorPtr, "must not be eq")
		eq(t, etalon.StablePointsVector{}, *alloc.ToRef(pointsVectorPtr), "must be eq")
		eq(t, etalon.StablePointsVector{}, alloc.DeRef(pointsVectorPtr), "must be eq")

		vectorRefFirst := alloc.ToRef(pointsVectorPtr)
		vectorRefFirst.Points[0].X = 11
		vectorRefFirst.Points[0].Y = 11
		vectorRefFirst.Points[1].X = 21
		vectorRefFirst.Points[1].Y = 21
		vectorRefFirst.Points[2].X = 13
		vectorRefFirst.Points[2].Y = 13

		expectedVector := etalon.StablePointsVector{Points: [3]etalon.Point{
			{X: 11, Y: 11},
			{X: 21, Y: 21},
			{X: 13, Y: 13},
		}}
		eq(t, expectedVector, *alloc.ToRef(pointsVectorPtr), "should be equal")
		eq(t, expectedVector, alloc.DeRef(pointsVectorPtr), "should be equal")
	}
	{
		pointsVectorPtr, allocErr := alloc.New()
		failOnError(t, allocErr)
		notEq(t, etalon.StablePointsVectorPtr{}, pointsVectorPtr, "must not be eq")
		eq(t, etalon.StablePointsVector{}, *alloc.ToRef(pointsVectorPtr), "must be eq")
		eq(t, etalon.StablePointsVector{}, alloc.DeRef(pointsVectorPtr), "must be eq")

		expectedVector := etalon.StablePointsVector{Points: [3]etalon.Point{
			{X: 11, Y: 11},
			{X: 21, Y: 21},
			{X: 13, Y: 13},
		}}
		*alloc.ToRef(pointsVectorPtr) = expectedVector
		eq(t, expectedVector, *alloc.ToRef(pointsVectorPtr), "should be equal")
		eq(t, expectedVector, alloc.DeRef(pointsVectorPtr), "should be equal")
	}
	{
		initVector := etalon.StablePointsVector{Points: [3]etalon.Point{
			{X: 44, Y: 44},
			{X: 65, Y: 32},
			{X: 65, Y: 32},
		}}
		pointsVectorPtr, allocErr := alloc.Embed(initVector)
		failOnError(t, allocErr)
		notEq(t, etalon.StablePointsVectorPtr{}, pointsVectorPtr, "must not be eq")
		eq(t, initVector, *alloc.ToRef(pointsVectorPtr), "must be eq")
		eq(t, initVector, alloc.DeRef(pointsVectorPtr), "must be eq")

		expectedVector := etalon.StablePointsVector{Points: [3]etalon.Point{
			{X: 11, Y: 11},
			{X: 21, Y: 21},
			{X: 13, Y: 13},
		}}
		*alloc.ToRef(pointsVectorPtr) = expectedVector
		eq(t, expectedVector, *alloc.ToRef(pointsVectorPtr), "should be equal")
		eq(t, expectedVector, alloc.DeRef(pointsVectorPtr), "should be equal")
		eq(t, int32(65), initVector.Points[2].X, "should be equal")
	}
}

func (s *arenaGenAllocationCheckingStand) checkGet(
	t *testing.T,
	alloc *etalon.StablePointsVectorView, buffer etalon.StablePointsVectorBuffer,
	idx int,
) {
	realSlice := alloc.Buffer.ToRef(buffer)
	expected := s.idx(realSlice, idx)
	actual := s.idxBuffer(alloc, buffer, idx)

	outOfBoundsError := "runtime error: index out of range ["
	if strings.HasPrefix(expected, outOfBoundsError) && strings.HasPrefix(actual, outOfBoundsError) {
		return
	}
	eq(t, expected, actual, "should be the same")
}

func (s *arenaGenAllocationCheckingStand) idxBuffer(
	alloc *etalon.StablePointsVectorView, buffer etalon.StablePointsVectorBuffer,
	idx int,
) (result string) {
	defer func() {
		err := recover()
		if err != nil {
			result = fmt.Sprintf("%v", err)
			return
		}
	}()
	value := alloc.Ptr.DeRef(buffer.Get(idx))
	return fmt.Sprintf("val: `%v`", value)
}

func (s *arenaGenAllocationCheckingStand) idx(
	slice []etalon.StablePointsVector, idx int,
) (result string) {
	defer func() {
		err := recover()
		if err != nil {
			result = fmt.Sprintf("%v", err)
			return
		}
	}()
	value := slice[idx]
	return fmt.Sprintf("val: `%v`", value)
}

func (s *arenaGenAllocationCheckingStand) checkSubSlice(
	t *testing.T,
	alloc *etalon.StablePointsVectorView, buffer etalon.StablePointsVectorBuffer,
	low int, high int,
) {
	realSlice := alloc.Buffer.ToRef(buffer)
	expected := s.subSlice(realSlice, low, high)
	actual := s.subSliceBuffer(alloc, buffer, low, high)

	outOfBoundsError := "runtime error: slice bounds out of range ["
	if strings.HasPrefix(expected, outOfBoundsError) && strings.HasPrefix(actual, outOfBoundsError) {
		return
	}
	eq(t, expected, actual, "should be the same")
}

func (s *arenaGenAllocationCheckingStand) subSliceBuffer(
	alloc *etalon.StablePointsVectorView, buffer etalon.StablePointsVectorBuffer,
	low int, high int,
) (result string) {
	defer func() {
		err := recover()
		if err != nil {
			result = fmt.Sprintf("%v", err)
			return
		}
	}()
	slice := alloc.Buffer.ToRef(buffer.SubSlice(low, high))
	return fmt.Sprintf("len: %v; cap: %v; val: `%v`", len(slice), cap(slice), slice)
}

func (s *arenaGenAllocationCheckingStand) subSlice(
	slice []etalon.StablePointsVector, low int, high int,
) (result string) {
	defer func() {
		err := recover()
		if err != nil {
			result = fmt.Sprintf("%v", err)
			return
		}
	}()
	subSlice := slice[low:high]
	return fmt.Sprintf("len: %v; cap: %v; val: `%v`", len(subSlice), cap(subSlice), subSlice)
}

type arenaGenAllocationLimitCheckingStand struct{}

func (s *arenaGenAllocationLimitCheckingStand) check(t *testing.T, target testAllocator) {
	t.Logf("alloc metrics: %+v", target.Metrics())
	alloc := etalon.NewStablePointsVectorView(target)
	sizeOfVector := int(unsafe.Sizeof(etalon.StablePointsVector{}))
	{
		vector, allocErr := alloc.Slice.Make((target.Metrics().AvailableBytes / sizeOfVector) + 1)
		expectErr(t, allocErr)
		eq(t, true, vector == nil, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
	{
		vector, allocErr := alloc.Slice.MakeWithCapacity(0, (target.Metrics().AvailableBytes/sizeOfVector)+1)
		expectErr(t, allocErr)
		eq(t, true, vector == nil, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
	{
		vector, allocNoErr := alloc.Slice.MakeWithCapacity(0, 1)
		failOnError(t, allocNoErr)
		eq(t, true, vector != nil, "vector should be empty")

		toAppend := make([]etalon.StablePointsVector, (target.Metrics().AvailableBytes/sizeOfVector)+1)
		newVec, allocErr := alloc.Slice.Append(vector, toAppend...)
		expectErr(t, allocErr)
		eq(t, true, newVec == nil, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
	{
		vector, allocNoErr := alloc.Slice.MakeWithCapacity(0, 0)
		failOnError(t, allocNoErr)
		eq(t, true, vector != nil, "vector should be empty")

		toAppend := make([]etalon.StablePointsVector, (target.Metrics().AvailableBytes/sizeOfVector)+1)
		newVec, allocErr := alloc.Slice.Append(vector, toAppend...)
		expectErr(t, allocErr)
		eq(t, true, newVec == nil, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}

	vector, allocErr := alloc.Slice.Make(target.Metrics().AvailableBytes / sizeOfVector)
	failOnError(t, allocErr)
	notEq(t, nil, vector, "should be nil")
	{
		vector, allocErr := alloc.Ptr.New()
		expectErr(t, allocErr)
		eq(t, etalon.StablePointsVectorPtr{}, vector, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
	{
		vector, allocErr := alloc.Ptr.Embed(etalon.StablePointsVector{Points: [3]etalon.Point{{X: 12}}})
		expectErr(t, allocErr)
		eq(t, etalon.StablePointsVectorPtr{}, vector, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
}

func notEq(t *testing.T, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		return
	}
	t.Errorf("objects are equal. `%+v` != `%+v`", expected, actual)
	t.Errorf(msg, args...)
	debug.PrintStack()
	t.FailNow()
}

func eq(t *testing.T, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	if reflect.DeepEqual(expected, actual) {
		return
	}
	t.Errorf("objects are not equal. %T(`%+v`) != %T(`%+v`)", expected, expected, actual, actual)
	t.Errorf("exp: %T(`%+v`)\n", expected, expected)
	t.Errorf("act: %T(`%+v`)\n", actual, actual)
	t.Errorf(msg, args...)
	debug.PrintStack()
	t.FailNow()
}

func expectErr(t *testing.T, e error) {
	if e == nil {
		t.Error("error expected for this type")
		debug.PrintStack()
		t.FailNow()
	}
	fmt.Println(e)
}

func failOnError(t *testing.T, e error) {
	if e != nil {
		t.Error(e)
		debug.PrintStack()
		t.FailNow()
	}
}
