package generator

import (
	"fmt"
	"runtime/debug"
	"testing"
	"unsafe"
)

func TestGeneratorForOne(t *testing.T) {
	t.Skip()
	failOnError(t, RunGeneratorForTypes("./testdata", []string{"Point"}))
}

func TestGeneratorFor(t *testing.T) {
	t.Skip()
	validTypes := []string{
		"Point",
		"coordinate",
		"Circle",
		"CircleColor",
		"StablePointsVector",
	}
	for _, validType := range validTypes {
		t.Run("_valid_"+validType, func(t *testing.T) {
			failOnError(t, RunGeneratorForTypes("./testdata", []string{validType}))
		})
	}

	invalidTypes := []string{
		"CirclePtr",
		"CircleCirclePtr",
		"coordinates",
		"PointsVector",
		"StableCirclePtrVector",
	}
	for _, invalidType := range invalidTypes {
		t.Run("_invalid_"+invalidType, func(t *testing.T) {
			expectErr(t, RunGeneratorForTypes("./testdata", []string{invalidType}))
		})
	}
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
