package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version defines the application semantic version.
// This can be overridden at compile-time via: -ldflags "-X main.Version=x.y.z"
var Version = "1.0.0"

// MaxTextFileSize defines the maximum allowed size (10 MB) for reading text/manifest files into memory.
const MaxTextFileSize int64 = 10 * 1024 * 1024

// App defines the application lifecycle and desktop dialog bindings.
type App struct {
	ctx       context.Context
	tempMu    sync.Mutex
	tempFiles []string
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		tempFiles: make([]string, 0, 8),
	}
}

// GetVersion returns the current application version string.
func (a *App) GetVersion() string {
	return Version
}

// startup is invoked when the Wails desktop application launches.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is invoked when the Wails desktop application terminates.
func (a *App) shutdown(ctx context.Context) {
	a.cleanupTempFiles()
}

// cleanupTempFiles deletes any temporary files tracked during the application lifetime.
func (a *App) cleanupTempFiles() {
	a.tempMu.Lock()
	defer a.tempMu.Unlock()
	for _, f := range a.tempFiles {
		_ = os.Remove(f)
	}
	a.tempFiles = a.tempFiles[:0]
}

// OpenFile displays a native file selection dialog and returns the chosen file path.
func (a *App) OpenFile() (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
	if err != nil {
		return "", err
	}
	return file, nil
}

// WriteTempFile writes content to a temporary checksum file for batch verification.
// All temporary files created here are tracked and automatically cleaned up on shutdown.
func (a *App) WriteTempFile(content string) (string, error) {
	file, err := os.CreateTemp("", "gopherhash_*.sha256")
	if err != nil {
		return "", err
	}
	fileName := file.Name()

	_, err = file.WriteString(content)
	if err != nil {
		file.Close()
		os.Remove(fileName)
		return "", err
	}
	file.Close()

	a.tempMu.Lock()
	a.tempFiles = append(a.tempFiles, fileName)
	a.tempMu.Unlock()

	return fileName, nil
}

// SelectChecksumFile displays a native file selection dialog filtered for supported checksum formats.
func (a *App) SelectChecksumFile() (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Checksum File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Checksum Files (*.sha256, *.md5, *.sha1, *.sha512, *.b3, *.blake3, *.crc32, *.sfv, *.txt)", Pattern: "*.sha256;*.md5;*.sha1;*.sha512;*.b3;*.blake3;*.crc;*.crc32;*.sfv;*.txt;*.chk;*.checksum"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	return file, nil
}

// ReadFileContent reads text content from a specified file path with a 10MB memory safety guard.
func (a *App) ReadFileContent(path string) (string, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" || cleanPath == "." {
		return "", ErrInvalidPath
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}

	if info.Size() > MaxTextFileSize {
		return "", fmt.Errorf("%w: file size %d bytes exceeds limit of %d bytes", ErrFileTooLarge, info.Size(), MaxTextFileSize)
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Limit reader to MaxTextFileSize + 1 to detect concurrent file growth
	limitReader := io.LimitReader(f, MaxTextFileSize+1)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return "", err
	}

	if int64(len(data)) > MaxTextFileSize {
		return "", fmt.Errorf("%w: file size exceeds limit of %d bytes", ErrFileTooLarge, MaxTextFileSize)
	}

	return string(data), nil
}
