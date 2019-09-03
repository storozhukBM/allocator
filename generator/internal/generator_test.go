package generator

import (
	"fmt"
	"runtime/debug"
	"testing"
	"unsafe"
)

func TestGeneratorForPoint(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata", []string{"Point"}))
}

func TestGeneratorForCoordinate(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata", []string{"coordinate"}))
}

func TestGeneratorForCircle(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata", []string{"Circle"}))
}

func TestGeneratorForCircleColor(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata", []string{"CircleColor"}))
}

func TestGeneratorForStablePointsVector(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata", []string{"StablePointsVector"}))
}

func TestGeneratorForInvalidCirclePtr(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata", []string{"CirclePtr"}))
}

func TestGeneratorForInvalidCircleCirclePtr(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata", []string{"CircleCirclePtr"}))
}

func TestGeneratorForInvalidCoordinates(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata", []string{"coordinates"}))
}

func TestGeneratorForInvalidPointsVector(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata", []string{"PointsVector"}))
}

func TestGeneratorForInvalidFixedCircleCirclePtrVector(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata", []string{"FixedCircleCirclePtrVector"}))
}

type some struct {
	a int
	b byte
}

func TestName(t *testing.T) {
	fmt.Println(unsafe.Sizeof(some{}))
	fmt.Println(unsafe.Alignof(some{}))

	ss := make([]some, 3)
	fmt.Println(unsafe.Sizeof(ss))
	fmt.Println(unsafe.Alignof(ss))
	fmt.Printf("%v\n", uintptr(unsafe.Pointer(&ss[0])))
	fmt.Printf("%v\n", uintptr(unsafe.Pointer(&ss[1])))
	fmt.Printf("%v\n", uintptr(unsafe.Pointer(&ss[2])))
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
