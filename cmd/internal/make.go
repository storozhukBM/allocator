package main

import (
	"fmt"
	archiver "github.com/mholt/archiver/v3"
	. "github.com/storozhukBM/build"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

const coverageName = `coverage.out`
const codeGenerationToolName = `allocgen`
const binDirName = `bin`
const linterName = `golangci-lint`
const linterVersion = `v1.23.3`

var parallelism = strconv.Itoa(runtime.NumCPU() * 4)

var b = NewBuild(BuildOptions{})
var commands = []Command{
	{`build`, b.RunCmd(Go, `build`, `./...`)},

	{`buildInlineBounds`, b.ShRunCmd(
		Go, `build`, `-gcflags='-m -d=ssa/check_bce/debug=1'`, `./...`,
	)},

	{`clean`, clean},
	{`cleanAll`, func() { clean(); cleanExecutables() }},
	{`testLib`, testLib},
	{`testCodeGen`, testCodeGen},
	{`test`, func() { testLib(); testCodeGen() }},
	{`generateTestAllocator`, generateTestAllocator},

	{`lint`, cilint},

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

func cleanExecutables() {
	b.Run(`rm`, `-rf`, binDirName)
}

func cilint() {
	executableFileName := linterName
	if runtime.GOOS == "windows" {
		executableFileName += ".exe"
	}
	versionInUrl := linterVersion[1:]
	targetFileName := linterFileNameByVersionAndRuntime(versionInUrl)
	executable := filepath.Join(binDirName, targetFileName, executableFileName)

	if _, err := os.Stat(executable); os.IsNotExist(err) {
		resultExecutable, downloadErr := downloadAndCompileLinter()
		if downloadErr != nil {
			b.AddError(downloadErr)
			return
		}
		if executable != resultExecutable {
			b.AddError(fmt.Errorf(
				"wrong exec version; expected: %v; actual: %v",
				executable, resultExecutable,
			))
			return
		}
	}

	b.Run(executable, `-j`, parallelism, `run`)
}

func downloadAndCompileLinter() (string, error) {
	versionInUrl := linterVersion[1:]
	filePath, downloadErr := downloadLinter()
	if downloadErr != nil {
		return "", downloadErr
	}

	executableFile := linterName
	if runtime.GOOS == "windows" {
		executableFile += ".exe"
	}

	decompressionErr := archiver.Unarchive(filePath, binDirName)
	if decompressionErr != nil {
		return "", fmt.Errorf("can't decompress file. File: %v; Error: %v", filePath, decompressionErr)
	}
	targetFileName := linterFileNameByVersionAndRuntime(versionInUrl)

	return filepath.Join(binDirName, targetFileName, executableFile), nil
}

func downloadLinter() (string, error) {
	versionInUrl := linterVersion[1:]
	archiveType := "tar.gz"
	if runtime.GOOS == "windows" {
		archiveType = "zip"
	}
	targetFileName := linterFileNameByVersionAndRuntime(versionInUrl)
	downloadUrl := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/"+
			"releases/download/%s/%s.%s",
		linterVersion, targetFileName, archiveType,
	)
	fmt.Printf("Going to download linter: %s\n", downloadUrl)

	resp, getErr := http.Get(downloadUrl)
	if getErr != nil {
		return "", fmt.Errorf("can't get linter. URL: `%v`; Error: %v", downloadUrl, getErr)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("can't get linter. URL: `%v`; Code: %v", downloadUrl, resp.Status)
	}
	respBody := resp.Body
	defer respBody.Close()

	destFile, tempFileErr := ioutil.TempFile("", "*."+archiveType)
	if tempFileErr != nil {
		return "", fmt.Errorf("can't store linter. URL: `%v`; Error: %v", downloadUrl, tempFileErr)
	}
	defer destFile.Close()

	_, copyErr := io.Copy(destFile, respBody)
	if copyErr != nil {
		return "", fmt.Errorf("can't download linter. URL: `%v`; Error: %v", downloadUrl, copyErr)
	}
	return destFile.Name(), nil
}

func linterFileNameByVersionAndRuntime(versionInUrl string) string {
	targetFileName := fmt.Sprintf("golangci-lint-%s-%s-%s", versionInUrl, runtime.GOOS, runtime.GOARCH)
	return targetFileName
}

func main() {
	b.Register(commands)
	b.BuildFromOsArgs()
}
