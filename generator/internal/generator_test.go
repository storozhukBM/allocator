package generator

import (
	"fmt"
	"io/ioutil"
	"runtime/debug"
	"strings"
	"testing"
)

func TestGeneratorForPoint(t *testing.T) {
	t.Parallel()
	failOnError(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"Point"}))
	compareOutputFiles(t, "Point")
}

func TestGeneratorForCoordinate(t *testing.T) {
	t.Parallel()
	failOnError(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"coordinate"}))
	compareOutputFiles(t, "coordinate")
}

func TestGeneratorForCircle(t *testing.T) {
	t.Parallel()
	failOnError(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"Circle"}))
	compareOutputFiles(t, "Circle")
}

func TestGeneratorForCircleColor(t *testing.T) {
	t.Parallel()
	failOnError(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"CircleColor"}))
	compareOutputFiles(t, "CircleColor")
}

func TestGeneratorForStablePointsVector(t *testing.T) {
	t.Parallel()
	failOnError(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"StablePointsVector"}))
	compareOutputFiles(t, "StablePointsVector")
}

func TestGeneratorForInvalidCirclePtr(t *testing.T) {
	t.Parallel()
	expectErr(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"CircleWithPointer"}))
}

func TestGeneratorForInvalidCircleCirclePtr(t *testing.T) {
	t.Parallel()
	expectErr(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"EmbeddedCircleWithPointer"}))
}

func TestGeneratorForInvalidCoordinates(t *testing.T) {
	t.Parallel()
	expectErr(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"coordinates"}))
}

func TestGeneratorForInvalidPointsVector(t *testing.T) {
	t.Parallel()
	expectErr(t, NewGenerator().RunGeneratorForTypes("./testdata/etalon/", []string{"PointsVector"}))
}

func TestGeneratorForInvalidFixedCircleCirclePtrVector(t *testing.T) {
	t.Parallel()
	expectErr(t, NewGenerator().RunGeneratorForTypes(
		"./testdata/etalon/", []string{"FixedEmbeddedCircleWithPointerVector"},
	))
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
	actualStr := string(actual)
	expectedStr := string(expected)
	if actualStr != expectedStr {
		for i := 0; i < len(actualStr); i++ {
			if actualStr[i] != expectedStr[i] {
				t.Errorf("i: %d; exp char: %s; act char %v", i, string(actualStr[i]), string(expectedStr[i]))
			}
		}

		t.Errorf(
			"actual `%s` and expected `%s` files are different. exp len: %v; act len: %v\nexp: `%s`\nact: `%s`\n",
			actualOutputFile, expectedOutputFile, len(expectedStr), len(actualStr), expectedStr, actualStr,
		)
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
