package etalon_test_test

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"testing"
	"unsafe"

	"github.com/storozhukBM/allocator/generator/internal/testdata/etalon"
	"github.com/storozhukBM/allocator/lib/arena"
)

const bytesRequiredForBasicTest = 580

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

func TestSimpleArenaWithInitialCapacity(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{InitialCapacity: 64})

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
	t.Logf("Point: size: %+v; alignment: %+v", unsafe.Sizeof(etalon.Point{}), unsafe.Alignof(etalon.Point{}))
	t.Logf("[3]Point: size: %+v; alignment: %+v", unsafe.Sizeof([3]etalon.Point{}), unsafe.Alignof([3]etalon.Point{}))

	alloc := etalon.NewStablePointsVectorView(target)
	arenaPointsVector, allocErr := alloc.MakeSliceWithCapacity(0, 4)
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
		arenaPointsVector, allocErr = alloc.MakeSlice(1)
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
}

type arenaGenAllocationLimitCheckingStand struct{}

func (s *arenaGenAllocationLimitCheckingStand) check(t *testing.T, target testAllocator) {
	t.Logf("alloc metrics: %+v", target.Metrics())
	alloc := etalon.NewStablePointsVectorView(target)
	sizeOfVector := int(unsafe.Sizeof(etalon.StablePointsVector{}))
	{
		vector, allocErr := alloc.MakeSlice((target.Metrics().AvailableBytes / sizeOfVector) + 1)
		expectErr(t, allocErr)
		eq(t, true, vector == nil, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
	{
		vector, allocErr := alloc.MakeSliceWithCapacity(0, (target.Metrics().AvailableBytes/sizeOfVector)+1)
		expectErr(t, allocErr)
		eq(t, true, vector == nil, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
	{
		vector, allocNoErr := alloc.MakeSliceWithCapacity(0, 1)
		failOnError(t, allocNoErr)
		eq(t, true, vector != nil, "vector should be empty")

		toAppend := make([]etalon.StablePointsVector, (target.Metrics().AvailableBytes/sizeOfVector)+1)
		newVec, allocErr := alloc.Append(vector, toAppend...)
		expectErr(t, allocErr)
		eq(t, true, newVec == nil, "vector should be empty")
		t.Logf("alloc metrics: %+v", target.Metrics())
	}
	{
		vector, allocNoErr := alloc.MakeSliceWithCapacity(0, 0)
		failOnError(t, allocNoErr)
		eq(t, true, vector != nil, "vector should be empty")

		toAppend := make([]etalon.StablePointsVector, (target.Metrics().AvailableBytes/sizeOfVector)+1)
		newVec, allocErr := alloc.Append(vector, toAppend...)
		expectErr(t, allocErr)
		eq(t, true, newVec == nil, "vector should be empty")
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
	t.Errorf("objects are not equal. `%+v` != `%+v`", expected, actual)
	t.Errorf("exp: `%+v`\n", expected)
	t.Errorf("act: `%+v`\n", actual)
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
