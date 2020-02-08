package main

import (
	. "github.com/storozhukBM/build"
	"runtime"
	"strconv"
)

const coverageName = `coverage.out`
const codeGenerationToolName = `allocgen`

var parallelism = strconv.Itoa(runtime.NumCPU() * 4)

var b = NewBuild(BuildOptions{})
var commands = []Command{
	{`build`, b.RunCmd(
		Go, `build`, `./...`,
	)},

	{`buildInlineBounds`, b.ShRunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},

	{`clean`, clean},
	{`testLib`, testLib},
	{`testCodeGen`, testCodeGen},
	{`test`, func() { testLib(); testCodeGen() }},
	{`generateTestAllocator`, generateTestAllocator},

	{`coverage`, func() {
		clean()
		b.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+coverageName, `./lib/arena/...`)
		b.Run(Go, `tool`, `cover`, `-html=`+coverageName)
	}},

	{`coverageCodeGen`, func() {
		clean()
		generateTestAllocator()
		b.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+coverageName, `./generator/internal/testdata/testdata_test/...`)
		b.Run(Go, `tool`, `cover`, `-html=`+coverageName)
	}},
}

func generateTestAllocator() {
	b.Run(Go, `build`, `-o`, codeGenerationToolName, `./generator/main.go`)
	b.Run(
		`./`+codeGenerationToolName,
		`-type`, `StablePointsVector`,
		`-dir`, `./generator/internal/testdata/etalon/`,
	)
}

func testLib() {
	defer forceClean()
	b.Run(Go, `test`, `-parallel`, parallelism, `./lib/...`)
	generateTestAllocator()
	b.Run(Go, `test`, `-parallel`, parallelism, `./generator/internal/testdata/testdata_test/...`)
}

func testCodeGen() {
	defer forceClean()
	b.Run(Go, `test`, `-parallel`, parallelism, `./generator/...`)
}

func clean() {
	b.Once(`cleanOnce`, func() { forceClean() })
}

func forceClean() {
	b.Run(Go, `clean`, `./...`)
	b.Run(`rm`, `-f`, coverageName)
	b.Run(`rm`, `-f`, codeGenerationToolName)
	b.Run(`rm`, `-f`, `./example/main`)
	// sh run used to expand wildcard
	b.ForceShRun(`rm`, `-f`, `./generator/internal/testdata/etalon/*.alloc.go`)
}

func main() {
	b.Register(commands)
	b.BuildFromOsArgs()
}
