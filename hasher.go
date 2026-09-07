package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"lukechampine.com/blake3"
)

// Common sentinel errors
var (
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")
	ErrFileTooLarge          = errors.New("file exceeds maximum allowed size for in-memory inspection")
	ErrInvalidPath           = errors.New("invalid or empty file path")
)

// Algorithm represents a supported hashing or checksum algorithm.
type Algorithm string

const (
	AlgoMD5    Algorithm = "MD5"
	AlgoSHA1   Algorithm = "SHA1"
	AlgoSHA256 Algorithm = "SHA256"
	AlgoSHA512 Algorithm = "SHA512"
	AlgoBLAKE3 Algorithm = "BLAKE3"
	AlgoCRC32  Algorithm = "CRC32"
)

// SupportedAlgorithms contains all algorithms enabled for multi-hash operations in GopherHash.
var SupportedAlgorithms = []Algorithm{
	AlgoMD5,
	AlgoSHA1,
	AlgoSHA256,
	AlgoSHA512,
	AlgoBLAKE3,
	AlgoCRC32,
}

var (
	reStandard = regexp.MustCompile(`^([0-9a-fA-F]{8}|[0-9a-fA-F]{32}|[0-9a-fA-F]{40}|[0-9a-fA-F]{64}|[0-9a-fA-F]{128})\s+\*?(.+)$`)
	reOpenSSL  = regexp.MustCompile(`^(?i:(SHA512|SHA256|SHA1|MD5|BLAKE3|CRC32))\s*\((.+)\)\s*=\s*([0-9a-fA-F]{8}|[0-9a-fA-F]{32}|[0-9a-fA-F]{40}|[0-9a-fA-F]{64}|[0-9a-fA-F]{128})$`)
	reSFV      = regexp.MustCompile(`^(.+?)\s+([0-9a-fA-F]{8})$`)
)

// copyBufPool pools 256KB buffers to maximize sequential disk read throughput and minimize GC allocations during file hashing.
const copyBufferSize = 256 * 1024

var copyBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, copyBufferSize)
		return &buf
	},
}

// HashResult represents the result of hashing a file.
type HashResult struct {
	FileName  string  `json:"FileName"`
	Hash      string  `json:"Hash"`
	Algorithm string  `json:"Algorithm"`
	Size      int64   `json:"Size"`
	Duration  float64 `json:"Duration"`
	Error     string  `json:"Error"`
}

// VerificationResult represents the verification of a file against an expected hash.
type VerificationResult struct {
	FileName  string `json:"FileName"`
	Expected  string `json:"Expected"`
	Actual    string `json:"Actual"`
	Algorithm string `json:"Algorithm"`
	Valid     bool   `json:"Valid"`
	Error     string `json:"Error"`
}

// FileCompareResult represents the side-by-side comparison between two files.
type FileCompareResult struct {
	File1     string  `json:"File1"`
	File2     string  `json:"File2"`
	File1Name string  `json:"File1Name"`
	File2Name string  `json:"File2Name"`
	File1Size int64   `json:"File1Size"`
	File2Size int64   `json:"File2Size"`
	Match     bool    `json:"Match"`
	Algorithm string  `json:"Algorithm"`
	Hash1     string  `json:"Hash1"`
	Hash2     string  `json:"Hash2"`
	Duration  float64 `json:"Duration"`
	Error     string  `json:"Error"`
}

// NormalizeAlgorithm normalizes a user-provided algorithm name to its canonical Algorithm representation.
func NormalizeAlgorithm(algo string) (Algorithm, error) {
	switch strings.ToUpper(strings.TrimSpace(algo)) {
	case "MD5":
		return AlgoMD5, nil
	case "SHA1", "SHA-1":
		return AlgoSHA1, nil
	case "SHA256", "SHA-256":
		return AlgoSHA256, nil
	case "SHA512", "SHA-512":
		return AlgoSHA512, nil
	case "BLAKE3", "B3":
		return AlgoBLAKE3, nil
	case "CRC32", "CRC-32":
		return AlgoCRC32, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algo)
	}
}

type formatFunc func() string

// createHasher constructs an io.Writer and formatting callback for a given algorithm.
func createHasher(algo Algorithm) (io.Writer, formatFunc, error) {
	switch algo {
	case AlgoMD5:
		h := md5.New()
		return h, func() string { return fmt.Sprintf("%x", h.Sum(nil)) }, nil
	case AlgoSHA1:
		h := sha1.New()
		return h, func() string { return fmt.Sprintf("%x", h.Sum(nil)) }, nil
	case AlgoSHA256:
		h := sha256.New()
		return h, func() string { return fmt.Sprintf("%x", h.Sum(nil)) }, nil
	case AlgoSHA512:
		h := sha512.New()
		return h, func() string { return fmt.Sprintf("%x", h.Sum(nil)) }, nil
	case AlgoBLAKE3:
		h := blake3.New(32, nil)
		return h, func() string { return fmt.Sprintf("%x", h.Sum(nil)) }, nil
	case AlgoCRC32:
		h := crc32.NewIEEE()
		return h, func() string { return fmt.Sprintf("%08x", h.Sum32()) }, nil
	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algo)
	}
}

// computeSingleHash hashes a file using the specified algorithm with pooled buffer.
func computeSingleHash(path string, algo Algorithm) (string, error) {
	w, format, err := createHasher(algo)
	if err != nil {
		return "", err
	}

	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" || cleanPath == "." {
		return "", ErrInvalidPath
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	bufPtr := copyBufPool.Get().(*[]byte)
	defer copyBufPool.Put(bufPtr)

	if _, err := io.CopyBuffer(w, f, *bufPtr); err != nil {
		return "", err
	}

	return format(), nil
}

// computeMultiHashes computes all specified algorithm hashes in a single disk read pass with pooled buffer.
func computeMultiHashes(path string, algos []Algorithm) (map[Algorithm]string, error) {
	writers := make([]io.Writer, 0, len(algos))
	formatters := make(map[Algorithm]formatFunc, len(algos))

	for _, algo := range algos {
		w, fmtFn, err := createHasher(algo)
		if err != nil {
			return nil, err
		}
		writers = append(writers, w)
		formatters[algo] = fmtFn
	}

	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" || cleanPath == "." {
		return nil, ErrInvalidPath
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	multiWriter := io.MultiWriter(writers...)
	bufPtr := copyBufPool.Get().(*[]byte)
	defer copyBufPool.Put(bufPtr)

	if _, err := io.CopyBuffer(multiWriter, f, *bufPtr); err != nil {
		return nil, err
	}

	results := make(map[Algorithm]string, len(algos))
	for _, algo := range algos {
		results[algo] = formatters[algo]()
	}

	return results, nil
}

// secureCompareHashes performs a case-insensitive, constant-time comparison of two hex hashes to prevent timing attacks.
func secureCompareHashes(h1, h2 string) bool {
	h1 = strings.ToLower(strings.TrimSpace(h1))
	h2 = strings.ToLower(strings.TrimSpace(h2))
	if len(h1) != len(h2) || len(h1) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(h1), []byte(h2)) == 1
}

// HashFile computes the hash of a file for a specific algorithm.
func (a *App) HashFile(path string, algorithm string) HashResult {
	algo, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return HashResult{
			FileName:  path,
			Algorithm: strings.ToUpper(algorithm),
			Error:     err.Error(),
		}
	}

	result := HashResult{
		FileName:  path,
		Algorithm: string(algo),
	}

	start := time.Now()
	hash, err := computeSingleHash(path, algo)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Duration = time.Since(start).Seconds()
	result.Hash = hash

	info, err := os.Stat(path)
	if err == nil {
		result.Size = info.Size()
	}

	return result
}

// HashAll computes all supported algorithms for a file in a single high-performance disk read pass.
func (a *App) HashAll(path string) []HashResult {
	start := time.Now()

	info, statErr := os.Stat(path)
	var fileSize int64
	if statErr == nil {
		fileSize = info.Size()
	}

	hashes, err := computeMultiHashes(path, SupportedAlgorithms)
	totalDuration := time.Since(start).Seconds()

	results := make([]HashResult, 0, len(SupportedAlgorithms))
	for _, algo := range SupportedAlgorithms {
		res := HashResult{
			FileName:  path,
			Algorithm: string(algo),
			Size:      fileSize,
			Duration:  totalDuration,
		}
		if err != nil {
			res.Error = err.Error()
		} else {
			res.Hash = hashes[algo]
		}
		results = append(results, res)
	}

	return results
}

// CompareFiles compares two files using the specified algorithm with concurrent hashing and size short-circuiting.
func (a *App) CompareFiles(path1 string, path2 string, algorithm string) FileCompareResult {
	algo, err := NormalizeAlgorithm(algorithm)
	algoName := strings.ToUpper(algorithm)
	if err == nil {
		algoName = string(algo)
	}

	result := FileCompareResult{
		File1:     path1,
		File2:     path2,
		File1Name: filepath.Base(path1),
		File2Name: filepath.Base(path2),
		Algorithm: algoName,
	}

	if err != nil {
		result.Error = err.Error()
		return result
	}

	start := time.Now()

	info1, err := os.Stat(path1)
	if err != nil {
		result.Error = fmt.Sprintf("File 1 error: %s", err.Error())
		return result
	}
	result.File1Size = info1.Size()

	info2, err := os.Stat(path2)
	if err != nil {
		result.Error = fmt.Sprintf("File 2 error: %s", err.Error())
		return result
	}
	result.File2Size = info2.Size()

	// Performance optimization: Size mismatch guarantee
	// If file sizes differ, cryptographic hashes are guaranteed to differ.
	// We still compute individual hashes concurrently so the user sees both checksums.
	var hash1, hash2 string
	var err1, err2 error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		hash1, err1 = computeSingleHash(path1, algo)
	}()
	go func() {
		defer wg.Done()
		hash2, err2 = computeSingleHash(path2, algo)
	}()
	wg.Wait()

	if err1 != nil {
		result.Error = fmt.Sprintf("Failed to hash %s: %s", filepath.Base(path1), err1.Error())
		return result
	}
	result.Hash1 = hash1

	if err2 != nil {
		result.Error = fmt.Sprintf("Failed to hash %s: %s", filepath.Base(path2), err2.Error())
		return result
	}
	result.Hash2 = hash2

	result.Duration = time.Since(start).Seconds()

	if result.File1Size != result.File2Size {
		result.Match = false
	} else {
		result.Match = secureCompareHashes(hash1, hash2)
	}

	return result
}

// FormatHashResult formats a single hash result as a human-readable string.
func (a *App) FormatHashResult(result HashResult) string {
	if result.Error != "" {
		return fmt.Sprintf("%s\n%s: ERROR - %s", result.FileName, result.Algorithm, result.Error)
	}

	sizeStr := fmt.Sprintf(" (%s in %.3fs)", formatBytes(result.Size), result.Duration)
	return fmt.Sprintf("%s\n%s: %s%s", result.FileName, result.Algorithm, result.Hash, sizeStr)
}

// VerifySingleHash verifies whether a file matches an expected hash string for an algorithm.
func (a *App) VerifySingleHash(file string, algorithm string, expectedHash string) VerificationResult {
	algo, err := NormalizeAlgorithm(algorithm)
	algoName := strings.ToUpper(algorithm)
	if err == nil {
		algoName = string(algo)
	}

	result := VerificationResult{
		FileName:  file,
		Algorithm: algoName,
		Expected:  strings.ToLower(strings.TrimSpace(expectedHash)),
	}

	if err != nil {
		result.Error = err.Error()
		return result
	}

	actualHash, err := computeSingleHash(file, algo)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Actual = strings.ToLower(actualHash)
	result.Valid = secureCompareHashes(result.Expected, result.Actual)
	return result
}

// resolveTargetFilePath safely resolves relative file paths within a checksum file to a sanitized path.
func resolveTargetFilePath(baseDir string, targetName string) string {
	targetName = strings.Trim(strings.TrimSpace(targetName), `"'`)
	if filepath.IsAbs(targetName) {
		return filepath.Clean(targetName)
	}

	if baseDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			baseDir = cwd
		}
	}

	return filepath.Clean(filepath.Join(baseDir, targetName))
}

// inferAlgorithm determines the algorithm from explicit tag, hash length, and file extension.
func inferAlgorithm(explicitTag string, hash string, filenameContext string) Algorithm {
	if explicitTag != "" {
		if algo, err := NormalizeAlgorithm(explicitTag); err == nil {
			return algo
		}
	}

	switch len(hash) {
	case 8:
		return AlgoCRC32
	case 32:
		return AlgoMD5
	case 40:
		return AlgoSHA1
	case 64:
		ext := strings.ToLower(filepath.Ext(filenameContext))
		if ext == ".b3" || ext == ".blake3" {
			return AlgoBLAKE3
		}
		return AlgoSHA256
	case 128:
		return AlgoSHA512
	default:
		return AlgoSHA256
	}
}

// parseChecksumLine extracts hash, filename, and optional algorithm from a checksum manifest line with fast-path optimization.
func parseChecksumLine(line string) (hash, fileName, matchedAlgo string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
		return "", "", "", false
	}

	// Fast-path: Check for OpenSSL format: "ALGO (file) = HASH"
	if strings.Contains(line, "(") && strings.Contains(line, ") = ") {
		if matches := reOpenSSL.FindStringSubmatch(line); len(matches) == 4 {
			return matches[3], matches[2], matches[1], true
		}
	}

	// Fast-path: Check for Standard format: "HASH  FILENAME" or "HASH *FILENAME"
	if matches := reStandard.FindStringSubmatch(line); len(matches) == 3 {
		return matches[1], matches[2], "", true
	}

	// Fast-path: Check for SFV format: "FILENAME HASH" (where hash is 8 hex chars)
	if matches := reSFV.FindStringSubmatch(line); len(matches) == 3 {
		return matches[2], matches[1], "CRC32", true
	}

	return "", "", "", false
}

// verifyScannerLines processes lines from a buffered scanner with bounded memory.
func (a *App) verifyScannerLines(scanner *bufio.Scanner, baseDir string, contextFilename string) []VerificationResult {
	var results []VerificationResult

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		hash, fileName, matchedAlgo, ok := parseChecksumLine(line)
		if !ok {
			results = append(results, VerificationResult{
				FileName: line,
				Error:    "Could not parse checksum line format",
			})
			continue
		}

		resolvedPath := resolveTargetFilePath(baseDir, fileName)
		algo := inferAlgorithm(matchedAlgo, hash, contextFilename)
		result := a.VerifySingleHash(resolvedPath, string(algo), hash)
		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		results = append(results, VerificationResult{
			FileName: contextFilename,
			Error:    fmt.Sprintf("Scanning error: %s", err.Error()),
		})
	}

	return results
}

// VerifyHashFile parses a checksum file (Standard, OpenSSL, or SFV format) and verifies each file using streaming I/O.
func (a *App) VerifyHashFile(hashFile string) []VerificationResult {
	cleanPath := filepath.Clean(strings.TrimSpace(hashFile))
	f, err := os.Open(cleanPath)
	if err != nil {
		return []VerificationResult{
			{
				FileName: hashFile,
				Error:    err.Error(),
			},
		}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Buffer support for long filenames (up to 1MB line length)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	baseDir := filepath.Dir(cleanPath)
	return a.verifyScannerLines(scanner, baseDir, hashFile)
}

// VerifyChecksumText verifies checksum content directly in memory without disk writes.
func (a *App) VerifyChecksumText(content string, baseDir string) []VerificationResult {
	cleanBaseDir := filepath.Clean(strings.TrimSpace(baseDir))
	if cleanBaseDir == "" || cleanBaseDir == "." {
		if cwd, err := os.Getwd(); err == nil {
			cleanBaseDir = cwd
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	return a.verifyScannerLines(scanner, cleanBaseDir, "pasted_checksums.txt")
}

// FormatVerificationResults formats batch verification results into a readable summary.
func (a *App) FormatVerificationResults(results []VerificationResult) string {
	summary := make([]string, 0, len(results))
	passCount, failCount, errCount := 0, 0, 0

	for _, r := range results {
		baseName := filepath.Base(r.FileName)
		switch {
		case r.Error != "":
			summary = append(summary, fmt.Sprintf("✗ ERROR: %s - %s", baseName, r.Error))
			errCount++
		case r.Valid:
			summary = append(summary, fmt.Sprintf("✓ OK: %s", baseName))
			passCount++
		default:
			summary = append(summary, fmt.Sprintf("✗ MISMATCH: %s", baseName))
			failCount++
		}
	}

	sort.Strings(summary)

	var b strings.Builder
	fmt.Fprintf(&b, "=== Verification Results ===\n%d passed, %d failed, %d errors\n\n", passCount, failCount, errCount)
	b.WriteString(strings.Join(summary, "\n"))
	return b.String()
}

// GetVerificationDetails outputs detailed mismatches or expected vs actual values.
func (a *App) GetVerificationDetails(results []VerificationResult) string {
	var b strings.Builder
	for _, r := range results {
		if r.Valid || r.Actual != "" {
			fmt.Fprintf(&b, "  Expected: %s\n  Actual:   %s\n\n", r.Expected, r.Actual)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatBytes(size int64) string {
	if size < 0 {
		return "0 bytes"
	}
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	case size == 1:
		return "1 byte"
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
