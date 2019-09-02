package testdata_test_test

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/storozhukBM/allocator/generator/internal/testdata"
)

func TestAllocStablePointsVector(t *testing.T) {
	vectorView := testdata.NewStablePointsVectorView(nil)
	{
		vectors, allocErr := vectorView.MakeSlice(1)
		failOnError(t, allocErr)
		eq(t, 1, len(vectors), "len should be 1")
		eq(t, 1, cap(vectors), "cap should be 1")
		vectors[0].Points[0].X = 1
		vectors[0].Points[0].Y = 2
		vectors[0].Points[1].X = 3
		vectors[0].Points[1].Y = 4
		vectors[0].Points[2].X = 5
		vectors[0].Points[2].Y = 6

		expectedVectors := make([]testdata.StablePointsVector, 1)
		expectedVectors[0].Points[0].X = 1
		expectedVectors[0].Points[0].Y = 2
		expectedVectors[0].Points[1].X = 3
		expectedVectors[0].Points[1].Y = 4
		expectedVectors[0].Points[2].X = 5
		expectedVectors[0].Points[2].Y = 6

		eq(t, expectedVectors, vectors, "vectors should be eq")
		eq(t, fmt.Sprint(expectedVectors), fmt.Sprint(vectors), "string representation of vectors should be eq")
		expectPanic(t, func(t *testing.T) {
			vectors[1].Points[0].X = 0
		})
	}
	{
		vectors, allocErr := vectorView.MakeSlice(6)
		failOnError(t, allocErr)
		eq(t, 6, len(vectors), "len should be 6")
		eq(t, 6, cap(vectors), "cap should be 6")
	}
}

func expectPanic(t *testing.T, fToPanic func(t *testing.T)) {
	defer func() {
		p := recover()
		if p == nil {
			failOnError(t, errors.New("expected panic not happened"))
		}
	}()
	fToPanic(t)
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
