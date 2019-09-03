package testdata_test_test

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"testing"
	"unsafe"

	"github.com/storozhukBM/allocator/generator/internal/testdata"
	"github.com/storozhukBM/allocator/lib/arena"
)

func TestUninitializedAlloc(t *testing.T) {
	t.Parallel()
	notInitializedStand := &arenaGenAllocationCheckingStand{}
	notInitializedStand.check(t, nil)
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
	alloc := testdata.NewStablePointsVectorView(target)
	arenaPointsVector, allocErr := alloc.MakeSliceWithCapacity(0, 4)
	failOnError(t, allocErr)
	notEq(t, arenaPointsVector, nil, "new slice can't be empty")

	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector, testdata.StablePointsVector{Points: [3]testdata.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}})
		failOnError(t, allocErr)
		expectedVector := []testdata.StablePointsVector{{Points: [3]testdata.Point{
			{X: 1, Y: 2},
			{X: 3, Y: 4},
			{X: 5, Y: 6},
		}}}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 1, len(arenaPointsVector), "len should be 1")
		eq(t, 4, cap(arenaPointsVector), "cap should be 4")
		t.Logf("vector state: %+v", arenaPointsVector)
	}
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			testdata.StablePointsVector{Points: [3]testdata.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			testdata.StablePointsVector{Points: [3]testdata.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []testdata.StablePointsVector{
			{Points: [3]testdata.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]testdata.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]testdata.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
		}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 3, len(arenaPointsVector), "len should be 3")
		eq(t, 4, cap(arenaPointsVector), "cap should be 4")
		t.Logf("vector state: %+v", arenaPointsVector)
	}
	{
		arenaPointsVector, allocErr = alloc.Append(arenaPointsVector,
			testdata.StablePointsVector{Points: [3]testdata.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			testdata.StablePointsVector{Points: [3]testdata.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
		)
		failOnError(t, allocErr)
		expectedVector := []testdata.StablePointsVector{
			{Points: [3]testdata.Point{
				{X: 1, Y: 2},
				{X: 3, Y: 4},
				{X: 5, Y: 6},
			}},
			{Points: [3]testdata.Point{
				{X: 2, Y: 3},
				{X: 4, Y: 5},
				{X: 6, Y: 7},
			}},
			{Points: [3]testdata.Point{
				{X: 3, Y: 4},
				{X: 5, Y: 6},
				{X: 7, Y: 8},
			}},
			{Points: [3]testdata.Point{
				{X: 0, Y: 1},
				{X: 2, Y: 3},
				{X: 4, Y: 5},
			}},
			{Points: [3]testdata.Point{
				{X: 9, Y: 8},
				{X: 7, Y: 6},
				{X: 5, Y: 4},
			}},
		}
		eq(t, expectedVector, arenaPointsVector, "should be equal")
		eq(t, 5, len(arenaPointsVector), "len should be 5")
		eq(t, true, cap(arenaPointsVector) >= 5, "cap should be >= 5")
		t.Logf("vector state: %+v", arenaPointsVector)
	}
}

//
//type arenaByteAllocationLimitsCheckingStand struct{}
//
//func (s *arenaByteAllocationLimitsCheckingStand) check(t *testing.T, target allocator) {
//	alloc := arena.NewBytesView(target)
//	{
//		arenaBytes, allocErr := alloc.MakeBytes(target.Metrics().AvailableBytes + 1)
//		assert(allocErr != nil, "allocation limit should be triggered")
//		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
//	}
//	{
//		buf := make([]byte, target.Metrics().AvailableBytes+1)
//		arenaBytes, allocErr := alloc.Embed(buf)
//		assert(allocErr != nil, "allocation limit should be triggered")
//		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
//	}
//	{
//		buf := make([]byte, target.Metrics().AvailableBytes+1)
//		arenaStr, allocErr := alloc.EmbedAsString(buf)
//		assert(allocErr != nil, "allocation limit should be triggered")
//		assert(arenaStr == "", "arenaBytes should be empty")
//	}
//	{
//		buf := make([]byte, target.Metrics().AvailableBytes+1)
//		arenaBytes, allocErr := alloc.EmbedAsBytes(buf)
//		assert(allocErr != nil, "allocation limit should be triggered")
//		assert(arenaBytes == nil, "arenaBytes should be empty")
//	}
//	{
//		arenaBytes, allocErr := alloc.MakeBytesWithCapacity(0, target.Metrics().AvailableBytes+1)
//		assert(allocErr != nil, "allocation limit should be triggered")
//		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
//	}
//	{
//		arenaBytes, allocNoErr := alloc.MakeBytesWithCapacity(0, 1)
//		failOnError(t, allocNoErr)
//		assert(arenaBytes != arena.Bytes{}, "arenaBytes shouldn't be empty")
//
//		toAppend := make([]byte, target.Metrics().AvailableBytes+1)
//		arenaBytes, allocErr := alloc.Append(arenaBytes, toAppend...)
//		assert(allocErr != nil, "allocation limit should be triggered")
//		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
//	}
//	{
//		arenaBytes, allocNoErr := alloc.MakeBytesWithCapacity(0, 0)
//		failOnError(t, allocNoErr)
//		assert(arenaBytes != arena.Bytes{}, "arenaBytes shouldn't be empty")
//
//		toAppend := make([]byte, target.Metrics().AvailableBytes+1)
//		arenaBytes, allocErr := alloc.Append(arenaBytes, toAppend...)
//		assert(allocErr != nil, "allocation limit should be triggered")
//		assert(arenaBytes == arena.Bytes{}, "bytes should be empty")
//	}
//}

func expectPanic(t *testing.T, fToPanic func(t *testing.T)) {
	defer func() {
		p := recover()
		if p == nil {
			failOnError(t, errors.New("expected panic not happened"))
		}
	}()
	fToPanic(t)
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
