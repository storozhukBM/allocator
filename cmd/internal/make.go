package main

import (
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gen2brain/beeep"
	. "github.com/storozhukBM/build"
)

const profileName = `profile.out`
const coverageName = `coverage.out`
const codeGenerationToolName = `allocgen`
const makeExecutable = `make`
const binDirName = `bin`
const golangCiLinterVersion = `1.23.6`

var parallelism = strconv.Itoa(runtime.NumCPU() * runtime.NumCPU())

var b = NewBuild(BuildOptions{})
var commands = []Command{
	{`itself`, b.RunCmd(Go, `build`, `-o`, makeExecutable, `./cmd/internal`)},

	{`build`, b.RunCmd(Go, `build`, `./...`)},
	{`test`, func() { testLib(); testCodeGen() }},
	{`testRace`, testRace},
	{`testLib`, testLib},
	{`testCodeGen`, testCodeGen},

	{`lint`, runLinters},
	{`verify`, func() { testLib(); testCodeGen(); runLinters() }},

	{`generateTestAllocator`, generateTestAllocator},
	{`clean`, clean},
	{`cleanAll`, func() { clean(); cleanExecutables() }},

	{`profileRawAlloc`, profileAllocationBenchmark(`BenchmarkRawAllocator`)},
	{`profileManagedRawAlloc`, profileAllocationBenchmark(`BenchmarkManagedRawAllocator`)},
	{`profileDynamicAlloc`, profileAllocationBenchmark(`BenchmarkDynamicAllocator`)},
	{`profileGenericAlloc`, profileAllocationBenchmark(`BenchmarkManagedGenericAllocatorWithPreAllocWithSubClean`)},

	{`benchAlloc`, b.RunCmd(
		Go, `test`, `-bench=.`,
		`github.com/storozhukBM/allocator/lib/arena/allocation_bench_test`,
	)},
	{`benchAlignment`, b.RunCmd(
		Go, `test`, `-bench=.`, `-benchtime=5s`, `-count=5`,
		`github.com/storozhukBM/allocator/lib/arena/alignment_bench_test`,
	)},

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

	{`buildInlineBounds`, b.ShRunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},
}

func profileAllocationBenchmark(benchmarkName string) func() {
	return func() {
		b.Run(
			Go, `test`, `-run=xxx`, `-bench=`+benchmarkName, `-benchtime=15s`,
			`github.com/storozhukBM/allocator/lib/arena/allocation_bench_test`,
			`-cpuprofile`, profileName,
		)
		b.Run(Go, `tool`, `pprof`, `-web`, profileName)
	}
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
	b.Run(`rm`, `-f`, profileName)
	b.Run(`rm`, `-f`, `allocation_bench_test.test`)
	b.Run(`rm`, `-f`, coverageName)
	b.Run(`rm`, `-f`, codeGenerationToolName)
	b.Run(`rm`, `-f`, `./example/main`)
	// sh run used to expand wildcard
	b.ForceShRun(`rm`, `-f`, `./generator/internal/testdata/etalon/*.alloc.go`)
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
	if runtime.GOOS == "windows" {
		// some linters do not support windows, so we use only default set
		b.Run(
			ciLinterExec, `-j`, parallelism, `run`, `--no-config`,
			`--skip-dirs=cmd`, `--skip-dirs=alignment_bench_test`, `--skip-dirs=allocation_bench_test`,
		)
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
