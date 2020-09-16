package main

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	. "github.com/storozhukBM/build"
	. "github.com/storozhukBM/downloader"
)

const golangCiLinterVersion = `1.31.0`

const profileName = `profile.out`
const coverageName = `coverage.out`

const binDirName = `bin`
const makeExecutable = `make`
const codeGenerationToolName = `allocgen`

const arenaModule = `github.com/storozhukBM/allocator/lib/arena`
const generatorModule = `github.com/storozhukBM/allocator/generator`

var parallelism = strconv.Itoa(2 * runtime.NumCPU())

var b = NewBuild(BuildOptions{})
var commands = []Command{
	{`itself`, b.RunCmd(Go, `build`, `-o`, makeExecutable, `make`)},

	{`test`, func() { testLib(); testCodeGen() }},
	{`testRace`, testRace},
	{`testLib`, testLib},
	{`testCodeGen`, testCodeGen},

	{`lint`, runLinters},
	{`verify`, func() { testLib(); testCodeGen(); runLinters() }},

	{`generateTestAllocator`, generateTestAllocator},
	{`clean`, clean},
	{`cleanAll`, func() { clean(); cleanExecutables() }},

	{`pprof`, b.RunCmd(Go, `tool`, `pprof`, profileName)},
	{`profileRawAlloc`, profileAllocBench(`BenchmarkRawAllocator`)},
	{`profileManagedRawAlloc`, profileAllocBench(`BenchmarkManagedRawAllocator`)},
	{`profileDynamicAlloc`, profileAllocBench(`BenchmarkDynamicAllocator`)},
	{`profileManagedDynWithPreAlloc`, profileAllocBench(`BenchmarkManagedDynamicAllocatorWithPreAlloc`)},
	{`profileGenericAlloc`, profileAllocBench(`BenchmarkGenericAllocatorWithSubClean`)},

	{`benchAlloc`, b.RunCmd(
		Go, `test`, `-bench=.`, `-count=5`, arenaModule+`/allocation_bench_test`,
	)},
	{`benchAlignment`, b.RunCmd(
		Go, `test`, `-bench=.`, `-benchtime=5s`, `-count=5`, arenaModule+`/alignment_bench_test`,
	)},

	{`coverage`, func() {
		clean()
		b.Run(Go, `test`, `-coverpkg=./...`, `-coverprofile=`+coverageName, arenaModule+`/arena_test/...`)
		b.Run(Go, `tool`, `cover`, `-html=`+coverageName)
	}},
	{`coverageCodeGen`, func() {
		clean()
		generateTestAllocator()
		b.Run(
			Go, `test`, `-coverpkg=./...`, `-coverprofile=`+coverageName,
			generatorModule+`/internal/testdata/testdata_test`,
		)
		b.Run(Go, `tool`, `cover`, `-html=`+coverageName)
	}},

	{`buildInlineBounds`, b.ShRunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, arenaModule+`/...`,
	)},
}

func profileAllocBench(benchmarkName string) func() {
	return func() {
		b.Run(
			Go, `test`, `-run=xxx`, `-bench=`+benchmarkName, `-benchtime=15s`,
			arenaModule+`/allocation_bench_test`,
			`-cpuprofile`, profileName,
		)
		b.Run(Go, `tool`, `pprof`, `-web`, profileName)
	}
}

func generateTestAllocator() {
	defer b.AddTarget("🏗  generate test allocator")()
	b.Run(Go, `run`, `./generator/main.go`,
		`-type`, `StablePointsVector`,
		`-dir`, `./generator/internal/testdata/etalon/`,
	)
}

func testLib() {
	defer b.AddTarget("🧪 test library code")()
	defer forceClean()
	b.Run(Go, `test`, `-parallel`, parallelism, arenaModule+`/...`)
	generateTestAllocator()
	defer b.AddTarget("🎯 test generated code")()
	b.Run(Go, `test`, `-parallel`, parallelism, generatorModule+`/internal/testdata/testdata_test`)
}

func testCodeGen() {
	defer b.AddTarget("🔦 test code generator itself")()
	defer forceClean()
	b.Run(Go, `test`, `-parallel`, parallelism, `github.com/storozhukBM/allocator/generator/...`)
}

func testRace() {
	defer b.AddTarget("🧪 test library code")()
	defer forceClean()
	b.Run(Go, `test`, `-race`, arenaModule+`/...`)
	generateTestAllocator()
	defer b.AddTarget("🔦 test generated code")()
	b.Run(Go, `test`, `-race`, generatorModule+`/internal/testdata/testdata_test`)
	defer forceClean()
	b.Run(Go, `test`, `-race`, generatorModule+`/...`)
}

func clean() {
	b.Once(`cleanOnce`, func() { forceClean() })
}

func forceClean() {
	defer b.AddTarget("🧹 clean")()
	b.Run(`rm`, `-f`, profileName)
	b.Run(`rm`, `-f`, `allocation_bench_test.test`)
	b.Run(`rm`, `-f`, coverageName)
	b.Run(`rm`, `-f`, `./lib/arena/`+coverageName)
	b.Run(`rm`, `-f`, codeGenerationToolName)
	b.Run(`rm`, `-f`, `./example/main`)
	// sh run used to expand wildcard
	b.ForceShRun(`rm`, `-f`, `./generator/internal/testdata/etalon/*.alloc.go`)
}

func cleanExecutables() {
	defer b.AddTarget("🧻 clean executables")()
	b.Run(`rm`, `-f`, makeExecutable)
	b.Run(`rm`, `-rf`, binDirName)
}

func runLinters() {
	defer b.AddTarget("🕵️  run linters")()
	ciLinterExec, downloadErr := downloadCILinter()
	if downloadErr != nil {
		b.AddError(downloadErr)
		return
	}
	if runtime.GOOS == "windows" {
		// some linters do not support windows, so we use only default set
		runLint := func(targetDir string) {
			patchedExec := strings.ReplaceAll(ciLinterExec, `\`, `/`)
			b.ShRun(
				`cd`, targetDir, `&&`, patchedExec, `-j`, parallelism, `run`, `--no-config`,
				`--skip-dirs=cmd`, `--skip-dirs=alignment_bench_test`, `--skip-dirs=allocation_bench_test`,
			)
		}
		runLint(`./lib/arena`)
		runLint(`./generator`)
		return
	}
	runLint := func(targetDir string) {
		b.ShRun(`cd`, targetDir, `&&`, ciLinterExec, `-j`, parallelism, `run`)
	}
	runLint(`./lib/arena`)
	runLint(`./generator`)
}

func downloadCILinter() (string, error) {
	urlTemplate := "https://github.com/golangci/golangci-lint/releases/download/v{version}/{fileName}"
	filePath, downloadErr := DownloadExecutable(DownloadExecutableOptions{
		ExecutableName:           "golangci-lint",
		Version:                  golangCiLinterVersion,
		FileNameTemplate:         "golangci-lint-{version}-{os}-{arch}.{osArchiveType}",
		ReleaseBinaryUrlTemplate: urlTemplate,
		ChecksumFileContent:      golangcilintChecksumFile,
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
		_ = beeep.Notify("Failure ❌", "Allocator build failure: "+buildErr.Error(), "")
		os.Exit(-1)
	}
	if time.Since(buildStart).Seconds() > 5 {
		_ = beeep.Notify("Success ✅", "Allocator build success", "")
	}
}

const golangcilintChecksumFile = `
1b50abb58fca75e4fb354359501c816d0f8d3bfa474c56d0bcec38162ebcdb3b  golangci-lint-1.31.0-freebsd-amd64.tar.gz
317a401ebc91eae49205ab71ad3c81bc511b7722da31adcbca2e4dbad1661bc3  golangci-lint-1.31.0-linux-s390x.deb
36ab03fb97771e2189a98b438b3d7cbbc6a46995841213783a14c89574b88753  golangci-lint-1.31.0-linux-amd64.rpm
48ad2b2e47b051976667bee17a36af10f1753981a664d62b04a9cf90fb6d1adb  golangci-lint-1.31.0-linux-s390x.rpm
5a18d18d9614ba8cda1d287e3544ebdc4605ecf13aa50ff648feeeeedfaf3f72  golangci-lint-1.31.0-linux-ppc64le.tar.gz
6ce46e1c107e7a5a130f3c5630fea8b4a092932a9bce88f01291da47eaa1e5cb  golangci-lint-1.31.0-linux-386.deb
6ce6b1d3207a63256d82fbbac80bb9e85d7705ec1a408f005dfe324457c54966  golangci-lint-1.31.0-windows-amd64.zip
72578d27b5632f51f6b464fe528ec2fa08ab6190f0670328f2ec8ae745b7a68b  golangci-lint-1.31.0-linux-ppc64le.rpm
7bd083d4b7db3dc331d2cc2a81922638ab4a5bba0a9b42f1d49b29e2d4e96546  golangci-lint-1.31.0-linux-amd64.deb
86b571c1f0fb1dae477aecaa9db44c68a6293a876ace220846ad7419018696a6  golangci-lint-1.31.0-linux-arm64.deb
8e363aac5d9c5fa72c84fbda0451d9e129a0830144cc59473cf70953179df103  golangci-lint-1.31.0-linux-armv7.rpm
94bc443e01817ff14c76afcce8e67ba05221f24919f1b690224e34d80b92ce24  golangci-lint-1.31.0-linux-armv6.rpm
9a5d47b51442d68b718af4c7350f4406cdc087e2236a5b9ae52f37aebede6cb3  golangci-lint-1.31.0-linux-amd64.tar.gz
9fff85f4649d17d18ebbcb775fec05de42a83e08787af1850f8f5f8dd4c066e9  golangci-lint-1.31.0-linux-arm64.tar.gz
a68a7d845034352a3ccb0886efdc488e6383144a9c245d041dde799580b671e3  golangci-lint-1.31.0-linux-386.tar.gz
ae32d4676dd6dd3e08665df22e9b3ec1d85511412a94cf5f397c8fe2a0d4ed1a  golangci-lint-1.31.0-linux-arm64.rpm
b43c4ab76c6edb4c3fde1d5764e1952e3bf017d264ef671047faa741a34b24c3  golangci-lint-1.31.0-freebsd-386.tar.gz
bd3534eabc22b1a04745b0efa1c122865173fbc8c3b70e77272da209bfe02228  golangci-lint-1.31.0-linux-386.rpm
c8d93e2c6f2acb9fe843387909ab2097f7d5fcda4ec04f050b90a30e593aac10  golangci-lint-1.31.0-freebsd-armv7.tar.gz
d93cb2696bb263c273e76594d8eaa0f866e3b24954a0b8e3859c0b58abf4ec8f  golangci-lint-1.31.0-linux-ppc64le.deb
d9ce8edd34994abead106f6a58337ca9396c8d0b1352bb19f7cd731ea9ea2021  golangci-lint-1.31.0-linux-armv6.tar.gz
dbd12f2f64b801c2f737725bd05e0208d4383a42df00ff0ccc43a288937bbf1d  golangci-lint-1.31.0-linux-armv6.deb
df5eaf84c96b3800a3f3642b7e418e8b96bb71d380bb7b5e6acf14de7baf93fe  golangci-lint-1.31.0-freebsd-armv6.tar.gz
e60257d148bd655438622853b28b9ca65d20919d62115ce87ccb45d136592643  golangci-lint-1.31.0-linux-armv7.tar.gz
ec9940b5a7dbc54fee75006ec67475243be645557bb3d3145beacd6289c8b654  golangci-lint-1.31.0-linux-s390x.tar.gz
f3afeb6ad6964615e2b238f56cc2e5b32464f2f302a4f3ccf5874a04170c287a  golangci-lint-1.31.0-darwin-amd64.tar.gz
fb5c25fa4471a74b5606be480bb083a06b15e834ef1293719b05d27e162e0bd6  golangci-lint-1.31.0-linux-armv7.deb
ffee09c855a44e55e4d1242791eb5935f734fadf7b7b02c8d1eecbeeb7f868a1  golangci-lint-1.31.0-windows-386.zip
`
