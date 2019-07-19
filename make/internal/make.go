package main

import (
	"os"
)

const CoverageName = `coverage.out`

var b = NewBuild(BuildOptions{})

func main() {
	b.Cmd(`build`, func() {
		b.Run(Go, `build`, `./...`)
	})

	b.Cmd(`buildInlineBounds`, func() {
		b.Run(Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`)
	})

	b.Cmd(`test`, func() {
		b.Run(Go, `test`, `./...`)
	})

	b.Cmd(`testDebug`, func() {
		b.Run(Go, `test`, `-v`, `./...`)
	})

	b.Cmd(`coverage`, func() {
		clean()
		b.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+CoverageName, `./lib/arena/...`)
		b.Run(Go, `tool`, `cover`, `-html=`+CoverageName)
	})

	b.Cmd(`clean`, clean)

	b.Build(os.Args[1:])
}

func clean() {
	b.Run(Go, `clean`)
	b.Run(`rm`, `-f`, CoverageName)
}
