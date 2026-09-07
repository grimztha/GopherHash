package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeHashSHA256(t *testing.T) {
	// Create test file
	f, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()
	result := app.HashFile(f.Name(), "sha256")

	// Known SHA256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if result.Hash != expected {
		t.Errorf("expected %s, got %s", expected, result.Hash)
	}
}

func TestComputeHashMD5(t *testing.T) {
	f, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()
	result := app.HashFile(f.Name(), "md5")

	expected := "5eb63bbbe01eeed093cb22bb8f5acdc3"
	if result.Hash != expected {
		t.Errorf("expected %s, got %s", expected, result.Hash)
	}
}

func TestComputeHashMissingFile(t *testing.T) {
	app := NewApp()
	result := app.HashFile("nonexistent_file_xyz.txt", "sha256")
	if result.Error == "" {
		t.Error("expected error for missing file")
	}
}

func TestVerifySingleHash(t *testing.T) {
	f, err := os.CreateTemp("", "test_verify_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()

	// Valid hash
	resValid := app.VerifySingleHash(f.Name(), "sha256", "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
	if !resValid.Valid {
		t.Errorf("expected valid hash, got mismatch")
	}

	// Invalid hash
	resInvalid := app.VerifySingleHash(f.Name(), "sha256", "0000000000000000000000000000000000000000000000000000000000000000")
	if resInvalid.Valid {
		t.Errorf("expected mismatch, got valid")
	}
}

func TestVerifyHashFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hash_test_dir_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// File with spaces
	fileWithSpaces := tempDir + "/test file with spaces.txt"
	if err := os.WriteFile(fileWithSpaces, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Second file
	file2 := tempDir + "/simple.txt"
	if err := os.WriteFile(file2, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create hash file testing standard sha256sum and OpenSSL format, plus BLAKE3 and CRC32
	hashFileContent := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9  " + fileWithSpaces + "\n" +
		"5eb63bbbe01eeed093cb22bb8f5acdc3 *" + file2 + "\n" +
		"SHA256 (" + file2 + ") = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\n" +
		"BLAKE3 (" + file2 + ") = d74981efa70a0c880b8d8c1985d075dbcbf679b99a5f9914e5aaf96b831a9e24\n" +
		"CRC32 (" + file2 + ") = 0d4a1185\n" +
		"0d4a1185  " + file2 + "\n"

	hashFilePath := tempDir + "/checksums.txt"
	if err := os.WriteFile(hashFilePath, []byte(hashFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	results := app.VerifyHashFile(hashFilePath)

	if len(results) != 6 {
		t.Fatalf("expected 6 results, got %d", len(results))
	}

	for i, r := range results {
		if r.Error != "" {
			t.Errorf("result %d returned error: %s", i, r.Error)
		}
		if !r.Valid {
			t.Errorf("result %d failed verification: expected %s, got %s", i, r.Expected, r.Actual)
		}
	}

	summary := app.FormatVerificationResults(results)
	if summary == "" {
		t.Errorf("expected non-empty summary")
	}
}

func TestComputeHashSHA1(t *testing.T) {
	f, err := os.CreateTemp("", "test_sha1_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()
	result := app.HashFile(f.Name(), "sha1")

	// Known SHA1 of "hello world"
	expected := "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"
	if result.Hash != expected {
		t.Errorf("expected %s, got %s", expected, result.Hash)
	}
}

func TestComputeHashSHA512(t *testing.T) {
	f, err := os.CreateTemp("", "test_sha512_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()
	result := app.HashFile(f.Name(), "sha512")

	// Known SHA512 of "hello world"
	expected := "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f"
	if result.Hash != expected {
		t.Errorf("expected %s, got %s", expected, result.Hash)
	}
}

func TestComputeHashBlake3(t *testing.T) {
	f, err := os.CreateTemp("", "test_blake3_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()
	result := app.HashFile(f.Name(), "blake3")

	// Known BLAKE3 of "hello world"
	expected := "d74981efa70a0c880b8d8c1985d075dbcbf679b99a5f9914e5aaf96b831a9e24"
	if result.Hash != expected {
		t.Errorf("expected %s, got %s", expected, result.Hash)
	}
	if result.Algorithm != "BLAKE3" {
		t.Errorf("expected algorithm BLAKE3, got %s", result.Algorithm)
	}
}

func TestComputeHashCRC32(t *testing.T) {
	f, err := os.CreateTemp("", "test_crc32_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()
	result := app.HashFile(f.Name(), "crc32")

	// Known CRC32 (IEEE) of "hello world"
	expected := "0d4a1185"
	if result.Hash != expected {
		t.Errorf("expected %s, got %s", expected, result.Hash)
	}
	if result.Algorithm != "CRC32" {
		t.Errorf("expected algorithm CRC32, got %s", result.Algorithm)
	}
}

func TestHashAll(t *testing.T) {
	f, err := os.CreateTemp("", "test_hashall_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello world")
	f.Close()

	app := NewApp()
	results := app.HashAll(f.Name())

	if len(results) != 6 {
		t.Fatalf("expected 6 hash results, got %d", len(results))
	}
	expectedAlgos := map[string]bool{
		"MD5": false, "SHA1": false, "SHA256": false, "SHA512": false, "BLAKE3": false, "CRC32": false,
	}
	for _, r := range results {
		if r.Error != "" {
			t.Errorf("HashAll returned error for %s: %s", r.Algorithm, r.Error)
		}
		if r.Hash == "" {
			t.Errorf("HashAll returned empty hash for %s", r.Algorithm)
		}
		expectedAlgos[r.Algorithm] = true
	}
	for algo, found := range expectedAlgos {
		if !found {
			t.Errorf("HashAll missing expected algorithm %s", algo)
		}
	}
}

func TestVerifyBlake3File(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blake3_test_dir_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	file1 := tempDir + "/test.txt"
	if err := os.WriteFile(file1, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	hashFileContent := "d74981efa70a0c880b8d8c1985d075dbcbf679b99a5f9914e5aaf96b831a9e24  " + file1 + "\n"
	hashFilePath := tempDir + "/checksums.b3"
	if err := os.WriteFile(hashFilePath, []byte(hashFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	results := app.VerifyHashFile(hashFilePath)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Valid {
		t.Errorf("expected valid, got mismatch: %s vs %s (error: %s)", results[0].Expected, results[0].Actual, results[0].Error)
	}
	if results[0].Algorithm != "BLAKE3" {
		t.Errorf("expected BLAKE3, got %s", results[0].Algorithm)
	}
}

func TestVerifyCRC32SFVFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sfv_test_dir_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	file1 := tempDir + "/test.txt"
	if err := os.WriteFile(file1, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// SFV format comments with ';' and lines with 'filename CRC'
	hashFileContent := "; Generated by SFV\n" + file1 + " 0d4a1185\n"
	hashFilePath := tempDir + "/checksums.sfv"
	if err := os.WriteFile(hashFilePath, []byte(hashFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	results := app.VerifyHashFile(hashFilePath)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Valid {
		t.Errorf("expected valid, got mismatch: %s vs %s (error: %s)", results[0].Expected, results[0].Actual, results[0].Error)
	}
	if results[0].Algorithm != "CRC32" {
		t.Errorf("expected CRC32, got %s", results[0].Algorithm)
	}
}

func TestCompareFilesMatch(t *testing.T) {
	f1, err := os.CreateTemp("", "test_compare1_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f1.Name())
	f1.WriteString("identical file content")
	f1.Close()

	f2, err := os.CreateTemp("", "test_compare2_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f2.Name())
	f2.WriteString("identical file content")
	f2.Close()

	app := NewApp()
	res := app.CompareFiles(f1.Name(), f2.Name(), "sha256")

	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	if !res.Match {
		t.Errorf("expected files to match, got mismatch")
	}
	if res.Hash1 != res.Hash2 {
		t.Errorf("expected hashes to be identical, got %s vs %s", res.Hash1, res.Hash2)
	}
}

func TestCompareFilesMismatch(t *testing.T) {
	f1, err := os.CreateTemp("", "test_compare_mismatch1_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f1.Name())
	f1.WriteString("file content alpha")
	f1.Close()

	f2, err := os.CreateTemp("", "test_compare_mismatch2_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f2.Name())
	f2.WriteString("file content beta")
	f2.Close()

	app := NewApp()
	res := app.CompareFiles(f1.Name(), f2.Name(), "sha256")

	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	if res.Match {
		t.Errorf("expected files to mismatch, got match")
	}
	if res.Hash1 == res.Hash2 {
		t.Errorf("expected hashes to differ, got %s == %s", res.Hash1, res.Hash2)
	}
}

func TestCompareFilesBlake3AndCRC32(t *testing.T) {
	f1, err := os.CreateTemp("", "test_compare_b3_1_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f1.Name())
	f1.WriteString("blake3 test content")
	f1.Close()

	f2, err := os.CreateTemp("", "test_compare_b3_2_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f2.Name())
	f2.WriteString("blake3 test content")
	f2.Close()

	app := NewApp()
	resB3 := app.CompareFiles(f1.Name(), f2.Name(), "blake3")
	if !resB3.Match {
		t.Errorf("expected BLAKE3 compare match")
	}
	if resB3.Algorithm != "BLAKE3" {
		t.Errorf("expected algorithm BLAKE3, got %s", resB3.Algorithm)
	}

	resCRC := app.CompareFiles(f1.Name(), f2.Name(), "crc32")
	if !resCRC.Match {
		t.Errorf("expected CRC32 compare match")
	}
	if resCRC.Algorithm != "CRC32" {
		t.Errorf("expected algorithm CRC32, got %s", resCRC.Algorithm)
	}
}

func TestNormalizeAlgorithm(t *testing.T) {
	tests := []struct {
		input    string
		expected Algorithm
		wantErr  bool
	}{
		{"md5", AlgoMD5, false},
		{"MD5", AlgoMD5, false},
		{"sha1", AlgoSHA1, false},
		{"SHA-1", AlgoSHA1, false},
		{"sha256", AlgoSHA256, false},
		{"SHA-256", AlgoSHA256, false},
		{"sha512", AlgoSHA512, false},
		{"SHA-512", AlgoSHA512, false},
		{"blake3", AlgoBLAKE3, false},
		{"B3", AlgoBLAKE3, false},
		{"crc32", AlgoCRC32, false},
		{"CRC-32", AlgoCRC32, false},
		{"unknown_algo", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		algo, err := NormalizeAlgorithm(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeAlgorithm(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("NormalizeAlgorithm(%q) unexpected error: %v", tt.input, err)
			}
			if algo != tt.expected {
				t.Errorf("NormalizeAlgorithm(%q) = %q, expected %q", tt.input, algo, tt.expected)
			}
		}
	}
}

func TestHashAllSinglePassEquivalence(t *testing.T) {
	f, err := os.CreateTemp("", "test_multi_pass_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("GopherHash high performance single-pass multi-hashing verification payload 1234567890!")
	f.Close()

	app := NewApp()
	multiResults := app.HashAll(f.Name())

	if len(multiResults) != len(SupportedAlgorithms) {
		t.Fatalf("expected %d results, got %d", len(SupportedAlgorithms), len(multiResults))
	}

	multiMap := make(map[string]string)
	for _, r := range multiResults {
		if r.Error != "" {
			t.Fatalf("HashAll error for %s: %s", r.Algorithm, r.Error)
		}
		multiMap[r.Algorithm] = r.Hash
	}

	// Compare with individual single-hash calls
	for _, algo := range SupportedAlgorithms {
		single := app.HashFile(f.Name(), string(algo))
		if single.Error != "" {
			t.Fatalf("HashFile error for %s: %s", algo, single.Error)
		}
		multiHash, exists := multiMap[string(algo)]
		if !exists {
			t.Errorf("HashAll missing algorithm %s", algo)
		}
		if multiHash != single.Hash {
			t.Errorf("mismatch for %s: single-pass %q != individual %q", algo, multiHash, single.Hash)
		}
	}
}

func TestVerifyChecksumText(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "text_verify_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	file1 := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(file1, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	content := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9  sample.txt\n"
	app := NewApp()
	results := app.VerifyChecksumText(content, tempDir)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Valid {
		t.Errorf("expected valid verification, got mismatch: %s vs %s (error: %s)", results[0].Expected, results[0].Actual, results[0].Error)
	}
}

func TestReadFileContent_SafetyLimit(t *testing.T) {
	app := NewApp()

	// 1. Missing file error
	_, err := app.ReadFileContent("nonexistent_test_path_123.txt")
	if err == nil {
		t.Errorf("expected error for nonexistent file")
	}

	// 2. Empty path error
	_, err = app.ReadFileContent("")
	if err == nil {
		t.Errorf("expected error for empty path")
	}

	// 3. Normal small file read
	f, err := os.CreateTemp("", "small_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("valid small file")
	f.Close()

	content, err := app.ReadFileContent(f.Name())
	if err != nil {
		t.Fatalf("unexpected error reading small file: %v", err)
	}
	if content != "valid small file" {
		t.Errorf("expected 'valid small file', got %q", content)
	}
}

func TestSecureCompareHashes(t *testing.T) {
	h1 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	h2 := "B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9"
	h3 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde0"

	if !secureCompareHashes(h1, h2) {
		t.Errorf("expected case-insensitive hash match")
	}
	if secureCompareHashes(h1, h3) {
		t.Errorf("expected mismatch for differing hashes")
	}
	if secureCompareHashes("", "") {
		t.Errorf("expected false for empty hashes")
	}
}

func TestResolveTargetFilePath_Sanitization(t *testing.T) {
	baseDir := filepath.Join(os.TempDir(), "gopher_test")
	resolved := resolveTargetFilePath(baseDir, "test.txt")
	expected := filepath.Clean(filepath.Join(baseDir, "test.txt"))
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}

	// Quoted filename
	resolvedQuoted := resolveTargetFilePath(baseDir, `"test with quotes.txt"`)
	expectedQuoted := filepath.Clean(filepath.Join(baseDir, "test with quotes.txt"))
	if resolvedQuoted != expectedQuoted {
		t.Errorf("expected %q, got %q", expectedQuoted, resolvedQuoted)
	}
}

func TestApp_TempFileCleanup(t *testing.T) {
	app := NewApp()
	tempFile, err := app.WriteTempFile("test temp content")
	if err != nil {
		t.Fatalf("unexpected error creating temp file: %v", err)
	}

	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatalf("temp file was not created: %s", tempFile)
	}

	// Call cleanup
	app.cleanupTempFiles()

	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Errorf("temp file still exists after cleanup: %s", tempFile)
	}
}

func BenchmarkHashFileSHA256(b *testing.B) {
	f, err := os.CreateTemp("", "bench_sha256_*.bin")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())

	data := make([]byte, 1024*1024) // 1MB payload
	f.Write(data)
	f.Close()

	app := NewApp()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := app.HashFile(f.Name(), "sha256")
		if res.Error != "" {
			b.Fatal(res.Error)
		}
	}
}

func BenchmarkHashAll(b *testing.B) {
	f, err := os.CreateTemp("", "bench_hashall_*.bin")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())

	data := make([]byte, 1024*1024) // 1MB payload
	f.Write(data)
	f.Close()

	app := NewApp()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := app.HashAll(f.Name())
		if len(results) == 0 {
			b.Fatal("no results")
		}
	}
}

func BenchmarkVerifyChecksumText(b *testing.B) {
	f, err := os.CreateTemp("", "bench_verify_text_*.bin")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("benchmark payload data")
	f.Close()

	app := NewApp()
	hRes := app.HashFile(f.Name(), "sha256")
	manifest := fmt.Sprintf("%s  %s\n", hRes.Hash, filepath.Base(f.Name()))
	baseDir := filepath.Dir(f.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := app.VerifyChecksumText(manifest, baseDir)
		if len(results) == 0 || !results[0].Valid {
			b.Fatal("verification failed in benchmark")
		}
	}
}

