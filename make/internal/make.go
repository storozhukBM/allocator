package main

import (
	"os"
)

const CoverageName = `coverage.out`

var b = NewBuild(BuildOptions{})
var commands = []Command{
	{`build`, b.RunCmd(
		Go, `build`, `./...`,
	)},

	{`buildInlineBounds`, b.RunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},

	{`test`, b.RunCmd(
		Go, `test`, `./...`,
	)},

	{`testDebug`, b.RunCmd(
		Go, `test`, `-v`, `./...`,
	)},

	{`coverage`, func() {
		clean()
		b.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./lib/arena/...`)
		b.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	}},

	{`clean`, clean},
}

func clean() {
	b.Run(Go, `clean`)
	b.Run(`rm -f`, CoverageName)
}

func main() {
	b.Register(commands)
	b.Build(os.Args[1:])
}
