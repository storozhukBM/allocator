package main

const CoverageName = `coverage.out`
const CodeGenerationToolName = `allocgen`

var B = NewBuild(BuildOptions{})
var Commands = []Command{
	{Name: `build`, Body: B.RunCmd(
		Go, `build`, `./...`,
	)},

	{Name: `buildInlineBounds`, Body: B.RunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},

	{Name: `test`, Body: func() {
		testCodeGen()
		testLib()
	}},

	{Name: `testLib`, Body: testLib},

	{Name: `testCodeGen`, Body: testCodeGen},

	{Name: `coverage`, Body: func() {
		clean()
		B.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./lib/arena/...`)
		B.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	}},

	{Name: `coverageCodeGen`, Body: func() {
		clean()
		B.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./generator/internal/testdata/testdata_test/...`)
		B.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	}},

	{Name: `clean`, Body: clean},
}

func testLib() {
	B.Run(Go, `test`, `./lib/...`)
}

func testCodeGen() {
	clean()
	B.Run(Go, `build`, `-o`, CodeGenerationToolName, `./generator/main.go`)
	B.Run(
		`./`+CodeGenerationToolName,
		`-type`, `StablePointsVector`,
		`-dir`, `./generator/internal/testdata/`,
	)
	//	B.Run(`cp`, `./generator/internal/testdata/testdata_test/*`, `./generator/internal/testdata/`)
	B.Run(Go, `test`, `./generator/internal/testdata/testdata_test/...`)
}

func clean() {
	B.Once(`cleanOnce`, func() {
		B.Run(Go, `clean`)
		B.Run(`rm`, `-f`, CoverageName)
		B.Run(`rm`, `-f`, CodeGenerationToolName)
		B.Run(`rm`, `-f`, `./generator/internal/testdata/*.alloc.go`)
	})
}

func main() {
	B.Register(Commands)
	B.BuildFromOsArgs()
}
