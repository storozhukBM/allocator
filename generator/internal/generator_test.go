package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"runtime/debug"
	"strings"
	"testing"
)

func TestGeneratorForPoint(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata/etalon/", []string{"Point"}))
	compareOutputFiles(t, "Point")
}

func TestGeneratorForCoordinate(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata/etalon/", []string{"coordinate"}))
	compareOutputFiles(t, "coordinate")
}

func TestGeneratorForCircle(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata/etalon/", []string{"Circle"}))
	compareOutputFiles(t, "Circle")
}

func TestGeneratorForCircleColor(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata/etalon/", []string{"CircleColor"}))
	compareOutputFiles(t, "CircleColor")
}

func TestGeneratorForStablePointsVector(t *testing.T) {
	t.Parallel()
	failOnError(t, RunGeneratorForTypes("./testdata/etalon/", []string{"StablePointsVector"}))
	compareOutputFiles(t, "StablePointsVector")
}

func TestGeneratorForInvalidCirclePtr(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata/etalon/", []string{"CircleWithPointer"}))
}

func TestGeneratorForInvalidCircleCirclePtr(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata/etalon/", []string{"EmbeddedCircleWithPointer"}))
}

func TestGeneratorForInvalidCoordinates(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata/etalon/", []string{"coordinates"}))
}

func TestGeneratorForInvalidPointsVector(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata/etalon/", []string{"PointsVector"}))
}

func TestGeneratorForInvalidFixedCircleCirclePtrVector(t *testing.T) {
	t.Parallel()
	expectErr(t, RunGeneratorForTypes("./testdata/etalon/", []string{"FixedEmbeddedCircleWithPointerVector"}))
}

func compareOutputFiles(t *testing.T, targetType string) {
	expectedOutputFile := "./testdata/expected/" + strings.ToLower(targetType) + ".alloc.go"
	actualOutputFile := "./testdata/etalon/" + strings.ToLower(targetType) + ".alloc.go"
	expected, err := ioutil.ReadFile(expectedOutputFile)
	if err != nil {
		t.Errorf("can't read expected file: %s", err.Error())
	}
	actual, err := ioutil.ReadFile(actualOutputFile)
	if err != nil {
		t.Errorf("can't read actual file: %s", err.Error())
	}
	if !bytes.Equal(actual, expected) {
		t.Errorf("actual `%s` and expected `%s` files are different", actualOutputFile, expectedOutputFile)
	}
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
