package main

const CoverageName = `coverage.out`
const CodeGenerationToolName = `allocgen`

var B = NewBuild(BuildOptions{})
var Commands = []Command{
	{Name: `build`, Body: B.RunCmd(
		Go, `build`, `./...`,
	)},

	{Name: `buildInlineBounds`, Body: B.ShRunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},

	{Name: `clean`, Body: clean},
	{Name: `testLib`, Body: testLib},
	{Name: `testCodeGen`, Body: testCodeGen},
	{Name: `test`, Body: func() { testLib(); testCodeGen() }},
	{Name: `generateTestAllocator`, Body: generateTestAllocator},

	{Name: `coverage`, Body: func() {
		clean()
		B.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./lib/arena/...`)
		B.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	}},

	{Name: `coverageCodeGen`, Body: func() {
		clean()
		generateTestAllocator()
		B.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./generator/internal/testdata/testdata_test/...`)
		B.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	}},
}

func generateTestAllocator() {
	B.Run(Go, `build`, `-o`, CodeGenerationToolName, `./generator/main.go`)
	B.Run(
		`./`+CodeGenerationToolName,
		`-type`, `StablePointsVector`,
		`-dir`, `./generator/internal/testdata/etalon/`,
	)
}

func testLib() {
	defer forceClean()
	B.Run(Go, `test`, `./lib/...`)
	generateTestAllocator()
	B.Run(Go, `test`, `./generator/internal/testdata/testdata_test/...`)
}

func testCodeGen() {
	defer forceClean()
	B.Run(Go, `test`, `./generator/...`)
}

func clean() {
	B.Once(`cleanOnce`, func() { forceClean() })
}

func forceClean() {
	B.Run(Go, `clean`)
	B.Run(`rm`, `-f`, CoverageName)
	B.Run(`rm`, `-f`, CodeGenerationToolName)
	B.Run(`rm`, `-f`, `./example/main`)
	// sh run used to expand wildcard
	B.ForceShRun(`rm`, `-f`, `./generator/internal/testdata/etalon/*.alloc.go`)
}

func main() {
	B.Register(Commands)
	B.BuildFromOsArgs()
}
