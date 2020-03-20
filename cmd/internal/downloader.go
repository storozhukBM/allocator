package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archiver/v3"
)

type DownloadExecutableOptions struct {
	ExecutableName string
	Version        string

	SkipCache bool

	FileName string
	/*
		Example: "golangci-lint-v{version}-{os}-{arch}.{osArchiveType}"
		Supported template variables:
			- os            - runtime.GOOS
			- arch          - runtime.GOARCH
			- version       - Version
			- osArchiveType - `tar.gz` or `zip` - determined by runtime.GOOS
	*/
	FileNameTemplate string

	ReleaseBinaryUrl string
	/*
		Example: https://github.com/golangci/golangci-lint/releases/download/{version}/golangci-lint-{fileName}
		Supported template variables:
			- os            - runtime.GOOS
			- arch          - runtime.GOARCH
			- version       - Version
			- fileName      - FileName resolved from FileName or FileNameTemplate
			- osArchiveType - `tar.gz` or `zip` - determined by runtime.GOOS
	*/
	ReleaseBinaryUrlTemplate string
	SkipDecompression        bool

	SkipChecksumVerification bool
	FilenameToChecksum       map[string]string
	ChecksumFileURL          string
	/*
		Example: https://github.com/golangci/golangci-lint/releases/download/v{version}/golangci-lint-{version}-checksums.txt
		Supported template variables:
			- os            - runtime.GOOS
			- arch          - runtime.GOARCH
			- version       - Version
			- fileName      - FileName resolved from FileName or FileNameTemplate
	*/
	ChecksumFileURLTemplate string

	DestinationDirectory string

	BinaryPathInside string
	/*
		Example: golangci-lint-{version}-{os}-{arch}/{executableName}{executableExtension}
		Supported template variables:
			- os            - runtime.GOOS
			- arch          - runtime.GOARCH
			- version       - Version
			- fileName      - FileName resolved from FileName or FileNameTemplate
			- osArchiveType - `tar.gz` or `zip` - determined by runtime.GOOS
			- executableName - ExecutableName
			- executableExtension - ".exe" - for windows; "" - for others;
	*/
	BinaryPathInsideTemplate string

	InfoPrinter func(string)
	WarnPrinter func(string)
}

func DownloadExecutable(opts DownloadExecutableOptions) (string, error) {
	inputParamsErr := validateInputParams(opts)
	if inputParamsErr != nil {
		return "", inputParamsErr
	}
	parsedOpts, evaluationErr := evaluateTemplates(opts)
	if evaluationErr != nil {
		return "", evaluationErr
	}
	return downloadWithOpts(parsedOpts)
}

func downloadWithOpts(opts DownloadExecutableOptions) (string, error) {
	currentPath, currentPathErr := os.Getwd()
	if currentPathErr != nil {
		return "", fmt.Errorf("can't determine current work dir path: %v", currentPathErr)
	}
	destination := filepath.Join(currentPath, opts.DestinationDirectory, opts.ExecutableName+osExecutableType())
	if !opts.SkipCache {
		if _, err := os.Stat(destination); err == nil {
			opts.InfoPrinter("skip download. Use file from cache")
			return destination, nil
		}
	}

	downloadedFilePath, downloadErr := downloadBinary(opts)
	if downloadErr != nil {
		return "", fmt.Errorf("can't download: %v", downloadErr)
	}
	filePath, decompressErr := decompressIfNecessary(opts, downloadedFilePath)
	if decompressErr != nil {
		return "", fmt.Errorf("can't decompres. file: %v; error: %v", downloadedFilePath, decompressErr)
	}

	// We could use os.Rename, but if you move files on Windows from one disk to another, like from C: to D: this
	// thing will not work. So we are forced to copy file by hand.
	sourceFile, sourceErr := os.Open(filePath)
	if sourceErr != nil {
		return "", fmt.Errorf("can't open source file %v: %v", filePath, sourceErr)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	mkdirErr := os.MkdirAll(opts.DestinationDirectory, os.ModePerm)
	if mkdirErr != nil {
		return "", fmt.Errorf("can't create dir: %v", mkdirErr)
	}
	destinationFile, destinationErr := os.OpenFile(destination, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if destinationErr != nil {
		return "", fmt.Errorf("can't create destination file %v: %v", destination, destinationErr)
	}
	defer func() {
		_ = destinationFile.Close()
	}()

	_, copyErr := io.Copy(destinationFile, sourceFile)
	if copyErr != nil {
		return "", fmt.Errorf("can't copy file from %v to %v: %v", filePath, destination, copyErr)
	}
	return destination, nil
}

func decompressIfNecessary(opts DownloadExecutableOptions, archivePath string) (string, error) {
	if opts.SkipDecompression {
		return archivePath, nil
	}
	dirPath := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().UnixNano(), 10))
	mkErr := os.Mkdir(dirPath, os.ModePerm)
	if mkErr != nil {
		return "", mkErr
	}
	decompressionErr := archiver.Unarchive(archivePath, dirPath)
	if decompressionErr != nil {
		return "", fmt.Errorf("can't decompress file. File: %v; Error: %v", archivePath, decompressionErr)
	}
	return filepath.Join(dirPath, opts.BinaryPathInside), nil
}

func evaluateTemplates(opts DownloadExecutableOptions) (DownloadExecutableOptions, error) {
	emptyOpts := DownloadExecutableOptions{}
	fileName, fileNameErr := resolveFileName(opts)
	if fileNameErr != nil {
		return emptyOpts, fmt.Errorf("can't resolve fileName: %v", fileNameErr)
	}
	opts.FileName = fileName

	releaseBinaryUrl, binaryErr := resolveReleaseBinaryUrl(opts)
	if binaryErr != nil {
		return emptyOpts, fmt.Errorf("can't resolve binaryUrl: %v", binaryErr)
	}
	opts.ReleaseBinaryUrl = releaseBinaryUrl

	binaryPathInside, binaryPathErr := resolveBinaryPathInside(opts)
	if binaryPathErr != nil {
		return emptyOpts, fmt.Errorf("can't resolve binaryPath: %v", binaryPathErr)
	}
	opts.BinaryPathInside = binaryPathInside
	return opts, nil
}

func validateInputParams(opts DownloadExecutableOptions) error {
	if opts.ExecutableName == "" {
		return fmt.Errorf("executableName can't be empty")
	}
	if opts.Version == "" {
		return fmt.Errorf("version can't be empty")
	}
	return nil
}

func downloadBinary(opts DownloadExecutableOptions) (string, error) {
	opts.InfoPrinter("going to download file from: " + opts.ReleaseBinaryUrl)
	resp, getErr := http.Get(opts.ReleaseBinaryUrl)
	if getErr != nil {
		return "", fmt.Errorf("can't get file. URL: `%v`; Error: %v", opts.ReleaseBinaryUrl, getErr)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("can't get file. URL: `%v`; Code: %v", opts.ReleaseBinaryUrl, resp.Status)
	}
	respBody := resp.Body
	defer respBody.Close()

	destFile, tempFileErr := ioutil.TempFile("", "*-"+opts.FileName)
	if tempFileErr != nil {
		return "", fmt.Errorf("can't store file. URL: `%v`; Error: %v", opts.ReleaseBinaryUrl, tempFileErr)
	}
	defer destFile.Close()

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	runDownloadProgressReporter(ctx, destFile.Name(), resp.ContentLength)

	_, copyErr := io.Copy(destFile, respBody)
	if copyErr != nil {
		return "", fmt.Errorf("can't download file. URL: `%v`; Error: %v", opts.ReleaseBinaryUrl, copyErr)
	}
	return destFile.Name(), nil
}

func runDownloadProgressReporter(ctx context.Context, filePath string, expectedFileSize int64) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				stat, statErr := os.Stat(filePath)
				if statErr != nil {
					continue
				}
				b.Info(fmt.Sprintf("file download: %v%%", stat.Size()/(expectedFileSize/100)))
			case <-ctx.Done():
				return
			}
		}
	}()
}

func resolveBinaryPathInside(opts DownloadExecutableOptions) (string, error) {
	if opts.BinaryPathInside != "" {
		return opts.BinaryPathInside, nil
	}
	if opts.BinaryPathInsideTemplate == "" {
		return "", fmt.Errorf("can't resolve BinaryPathInside from template. Template is empty")
	}
	variablesReplacer := strings.NewReplacer(
		"{os}", runtime.GOOS,
		"{arch}", runtime.GOARCH,
		"{version}", opts.Version,
		"{fileName}", opts.FileName,
		"{osArchiveType}", osArchiveType(),
		"{executableName}", opts.ExecutableName,
		"{executableExtension}", osExecutableType(),
	)
	result := variablesReplacer.Replace(opts.BinaryPathInsideTemplate)
	opts.InfoPrinter(fmt.Sprintf("resolved binary path in destination dir: %s", result))
	return result, nil
}

func resolveReleaseBinaryUrl(opts DownloadExecutableOptions) (string, error) {
	if opts.ReleaseBinaryUrl != "" {
		return opts.ReleaseBinaryUrl, nil
	}
	if opts.ReleaseBinaryUrlTemplate == "" {
		return "", fmt.Errorf("can't resolve ReleaseBinaryUrl from template. Template is empty")
	}
	variablesReplacer := strings.NewReplacer(
		"{os}", runtime.GOOS,
		"{arch}", runtime.GOARCH,
		"{version}", opts.Version,
		"{fileName}", opts.FileName,
		"{osArchiveType}", osArchiveType(),
	)
	result := variablesReplacer.Replace(opts.ReleaseBinaryUrlTemplate)
	opts.InfoPrinter(fmt.Sprintf("resolved binary url: %s", result))
	return result, nil
}

func resolveFileName(opts DownloadExecutableOptions) (string, error) {
	if opts.FileName != "" {
		return opts.FileName, nil
	}
	if opts.FileNameTemplate == "" {
		return "", fmt.Errorf("can't resolve FileName from template. Template is empty")
	}
	variablesReplacer := strings.NewReplacer(
		"{os}", runtime.GOOS,
		"{arch}", runtime.GOARCH,
		"{version}", opts.Version,
		"{osArchiveType}", osArchiveType(),
	)
	result := variablesReplacer.Replace(opts.FileNameTemplate)
	opts.InfoPrinter(fmt.Sprintf("resolved file name: %s", result))
	return result, nil
}

func osArchiveType() string {
	osArchiveType := "tar.gz"
	if runtime.GOOS == "windows" {
		osArchiveType = "zip"
	}
	return osArchiveType
}

func osExecutableType() string {
	result := ""
	if runtime.GOOS == "windows" {
		result = ".exe"
	}
	return result
}
