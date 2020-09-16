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

const golangCiLinterVersion = `1.30.0`

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
0701ede88347aa0ad7bd5c2bcc22d7d7b64ae7e9828abd3c4fe5a8c57c4f7b5a  golangci-lint-1.30.0-linux-s390x.rpm
191f21f971054b6a016027c73f7643402a9b3322fe42be3cc1533605a67de21a  golangci-lint-1.30.0-linux-amd64.deb
258e51decfe286e2f3b985ef2eece6197dc61f1e50bc1ca4986d23776a673737  golangci-lint-1.30.0-darwin-amd64.tar.gz
2dd52d77d9dc0aff73a85938a12e53f4a3688cf19e9ca52632629771a5fd664c  golangci-lint-1.30.0-linux-386.rpm
2f0c6c747f03bf75594bc34969d9f5e919a9becea7d5fa23599f63150a960c2c  golangci-lint-1.30.0-windows-amd64.zip
320156934ee15977b03da44f2fed88d3f6ad3480d08adea3fe9e999cc026c248  golangci-lint-1.30.0-linux-arm64.deb
34bf2ce67182b9f74c69ed1768c327ff140dd6029e46498c12bd88a25dd809a2  golangci-lint-1.30.0-freebsd-armv7.tar.gz
484e0ab3644068585528f0c0d35c6fe97451eade659cb12ff5f14c1d4c5f36ba  golangci-lint-1.30.0-linux-amd64.rpm
4db9f2ee472d02167e26dc4a4afb8880b58fc8b409c713314d89d5f24b76d8be  golangci-lint-1.30.0-windows-386.zip
5ab313c203522b8ef0fca4a86f03c21647552eb126682f6bb6e0c2c27519806a  golangci-lint-1.30.0-linux-s390x.tar.gz
5c91512b4120620513b3e551e996ec2d0d476d0feb503c6e932c8336949cdc74  golangci-lint-1.30.0-linux-ppc64le.rpm
63e43bddd08485e3c652d399d11486874427fd4cdd4ef26d3b96168d63e1450c  golangci-lint-1.30.0-linux-armv7.deb
669ef3611963b44dc5b6cbf3709b5dd56c9d1dab11562309a5cdefcf4096eed0  golangci-lint-1.30.0-linux-armv6.tar.gz
74fbcf4110ec10b0ddcc1779cfbe11a83ce1b7b83920d8eefa195cc93e3c11f3  golangci-lint-1.30.0-linux-386.tar.gz
9556cba775505e270b1e6c9549fe6673bf3c36f1bda3aefe2e253749f2f9cecb  golangci-lint-1.30.0-linux-386.deb
a9c1d14d84a687fb8e67c2c613533ec3e2802b5baf95ca82ad528f788cb6d61f  golangci-lint-1.30.0-linux-arm64.tar.gz
b0cb8001137e1a3e465204b9176e136c29399e94f4b494020471b2e8559a28b6  golangci-lint-1.30.0-linux-ppc64le.tar.gz
b542d5cabe71649a8ae57dbb929c0d43c5b41d2aef8395aa1b028ff34ebd7676  golangci-lint-1.30.0-linux-arm64.rpm
b60feec3da0dfce79aef36c05dada77bf2aa98f07ad64cf4027bd36ec8db4a4e  golangci-lint-1.30.0-freebsd-amd64.tar.gz
bd9b633796537dea78d0e6003da7b5f78fe17f7903dc967c3e216411549a53f7  golangci-lint-1.30.0-linux-armv6.deb
bf1947b191dbbca974a59f3897c999964113c3d3d6098b54e719011d7bcf3db0  golangci-lint-1.30.0-freebsd-armv6.tar.gz
c8e8fc5753e74d2eb489ad428dbce219eb9907799a57c02bcd8b80b4b98c60d4  golangci-lint-1.30.0-linux-amd64.tar.gz
cd6deee3c903261f6222991526128637656a4a9ae987a0d54bd4d9b8be6a2162  golangci-lint-1.30.0-linux-s390x.deb
d15ed16baa62d89ccdf3a6189219a1f4e8820721d9a1e2e361fb92c2525f324e  golangci-lint-1.30.0-linux-armv7.tar.gz
d87ccd0b50ffdcff03f48a02099144b3668bfee295ee08e838262aa4a801df57  golangci-lint-1.30.0-linux-armv7.rpm
d9e2fd2d2807d121b5936d419a9027555954b48173e52c743064ebf474f30008  golangci-lint-1.30.0-linux-ppc64le.deb
e0977ea0542e8a42fe8e91ec32911c6385d4f8f1f23c2be7bca5bece079b10eb  golangci-lint-1.30.0-freebsd-386.tar.gz
edb2cf91cf6478d5d096fa88c7b13f236b078756bf4f9a6de63a8daf51a716b4  golangci-lint-1.30.0-linux-armv6.rpm
`
