package main

import (
	"github.com/gen2brain/beeep"
	. "github.com/storozhukBM/build"
	"os"
	"runtime"
	"strconv"
	"time"
)

const coverageName = `coverage.out`
const codeGenerationToolName = `allocgen`
const makeExecutable = `make`
const binDirName = `bin`
const golangCiLinterVersion = `1.23.6`

var parallelism = strconv.Itoa(runtime.NumCPU() * runtime.NumCPU())

var b = NewBuild(BuildOptions{})
var commands = []Command{
	{`build`, b.RunCmd(Go, `build`, `./...`)},
	{`buildInlineBounds`, b.ShRunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},

	{`itself`, b.RunCmd(Go, `build`, `-o`, makeExecutable, `./cmd/internal`)},

	{`clean`, clean},
	{`cleanAll`, func() { clean(); cleanExecutables() }},

	{`lint`, runLinters},
	{`testLib`, testLib},
	{`testCodeGen`, testCodeGen},
	{`testRace`, testRace},
	{`test`, func() { testLib(); testCodeGen() }},
	{`verify`, func() { testLib(); testCodeGen(); runLinters() }},

	{`generateTestAllocator`, generateTestAllocator},

	{`coverage`, func() {
		clean()
		b.Run(
			Go, `test`, `-coverpkg=./...`, `-coverprofile=`+coverageName,
			`./lib/arena/...`,
		)
		b.Run(Go, `tool`, `cover`, `-html=`+coverageName)
	}},
	{`coverageCodeGen`, func() {
		clean()
		generateTestAllocator()
		b.Run(
			Go, `test`, `-coverpkg=./...`, `-coverprofile=`+coverageName,
			`./generator/internal/testdata/testdata_test/...`,
		)
		b.Run(Go, `tool`, `cover`, `-html=`+coverageName)
	}},
}

func generateTestAllocator() {
	defer b.AddTarget("generate test allocator")()
	b.Run(Go, `run`, `./generator/main.go`,
		`-type`, `StablePointsVector`,
		`-dir`, `./generator/internal/testdata/etalon/`,
	)
}

func testLib() {
	defer b.AddTarget("test library code")()
	defer forceClean()
	b.Run(Go, `test`, `-parallel`, parallelism, `./lib/...`)
	generateTestAllocator()
	defer b.AddTarget("test generated code")()
	b.Run(Go, `test`, `-parallel`, parallelism, `./generator/internal/testdata/testdata_test/...`)
}

func testCodeGen() {
	defer b.AddTarget("test code generator itself")()
	defer forceClean()
	b.Run(Go, `test`, `-parallel`, parallelism, `./generator/...`)
}

func testRace() {
	defer b.AddTarget("test library code")()
	defer forceClean()
	b.Run(Go, `test`, `-race`, `./lib/...`)
	generateTestAllocator()
	defer b.AddTarget("test generated code")()
	b.Run(Go, `test`, `-race`, `./generator/internal/testdata/testdata_test/...`)
	defer forceClean()
	b.Run(Go, `test`, `-race`, `./generator/...`)
}

func clean() {
	b.Once(`cleanOnce`, func() { forceClean() })
}

func forceClean() {
	defer b.AddTarget("clean")()
	b.Run(Go, `clean`, `./...`)
	b.Run(`rm`, `-f`, coverageName)
	b.Run(`rm`, `-f`, codeGenerationToolName)
	b.Run(`rm`, `-f`, `./example/main`)
	// sh run used to expand wildcard
	b.ForceRun(`sh`, `-c`, `rm -f ./generator/internal/testdata/etalon/*.alloc.go`)
}

func cleanExecutables() {
	defer b.AddTarget("clean executables")()
	b.Run(`rm`, `-f`, makeExecutable)
	b.Run(`rm`, `-rf`, binDirName)
}

func runLinters() {
	defer b.AddTarget("run linters")()
	ciLinterExec, downloadErr := downloadCILinter()
	if downloadErr != nil {
		b.AddError(downloadErr)
		return
	}
	b.Run(ciLinterExec, `-j`, parallelism, `run`)
}

func downloadCILinter() (string, error) {
	urlTemplate := "https://github.com/golangci/golangci-lint/releases/download/v{version}/{fileName}"
	filePath, downloadErr := DownloadExecutable(DownloadExecutableOptions{
		ExecutableName:           "golangci-lint",
		Version:                  golangCiLinterVersion,
		FileNameTemplate:         "golangci-lint-{version}-{os}-{arch}.{osArchiveType}",
		ReleaseBinaryUrlTemplate: urlTemplate,
		SkipChecksumVerification: true,
		DestinationDirectory:     "bin/linters/",
		BinaryPathInsideTemplate: "golangci-lint-{version}-{os}-{arch}/{executableName}{executableExtension}",
		InfoPrinter:              b.Info,
		WarnPrinter:              b.Warn,
	})
	return filePath, downloadErr
}

func main() {
	b.Register(commands)
	buildStart := time.Now()
	buildErr := b.BuildFromOsArgs()
	if buildErr != nil {
		_ = beeep.Notify("Failure", "Allocator build failure: "+buildErr.Error(), "")
		os.Exit(-1)
	}
	if time.Since(buildStart).Seconds() > 5 {
		_ = beeep.Notify("Success", "Allocator build success", "")
	}
}
