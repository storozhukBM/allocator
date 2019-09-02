package main

const CoverageName = `coverage.out`

var B = NewBuild(BuildOptions{})
var Commands = []Command{
	{Name: `build`, Body: B.RunCmd(
		Go, `build`, `./...`,
	)},

	{Name: `buildInlineBounds`, Body: B.RunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},

	{Name: `test`, Body: B.RunCmd(
		Go, `test`, `./...`,
	)},

	{Name: `testDebug`, Body: B.RunCmd(
		Go, `test`, `-v`, `./...`,
	)},

	{Name: `coverage`, Body: func() {
		clean()
		B.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./lib/arena/...`)
		B.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	}},

	{Name: `coverage_gen`, Body: func() {
		clean()
		B.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./generator/internal/testdata/testdata_test/...`)
		B.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	}},

	{Name: `clean`, Body: clean},
}

func clean() {
	B.Once(`cleanOnce`, func() {
		B.Run(Go, `clean`)
		B.Run(`rm`, `-f`, CoverageName)
	})
}

func main() {
	B.Register(Commands)
	B.BuildFromOsArgs()
}
