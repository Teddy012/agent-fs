package local

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testReadResult is a helper to assert common result properties
func testReadResult(t *testing.T, result ReadResult, content string, lineCount, byteCount int, truncated bool, sliceType string) {
	t.Helper()
	if result.Content != content {
		t.Fatalf("expected content=%q, got %q", content, result.Content)
	}
	if result.LineCount != lineCount {
		t.Fatalf("expected line_count=%d, got %d", lineCount, result.LineCount)
	}
	if result.ByteCount != byteCount {
		t.Fatalf("expected byte_count=%d, got %d", byteCount, result.ByteCount)
	}
	if result.Truncated != truncated {
		t.Fatalf("expected truncated=%v, got %v", truncated, result.Truncated)
	}
	if result.SliceType != sliceType {
		t.Fatalf("expected slice_type=%s, got %s", sliceType, result.SliceType)
	}
}

func TestReadFullFile(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	result, err := ReadFile(testFile, ReadOptions{})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	testReadResult(t, result, content, 3, len(content), false, "full")
}

func TestReadHeadLines(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	result, err := ReadFile(testFile, ReadOptions{HeadLines: 3})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	testReadResult(t, result, "line1\nline2\nline3\n", 3, 18, false, "head")
}

func TestReadTailLines(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	result, err := ReadFile(testFile, ReadOptions{TailLines: 2})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	testReadResult(t, result, "line4\nline5\n", 2, 12, true, "tail")
}

func TestReadBytes(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.txt")
	content := "hello world, this is a test file"
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	result, err := ReadFile(testFile, ReadOptions{Bytes: 5})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	testReadResult(t, result, "hello", 0, 5, true, "bytes")
}

// TestReadExceedsFile tests that when the read amount exceeds file size,
// the full content is returned and truncated=false
func TestReadExceedsFile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		opts      ReadOptions
		wantTrunc bool
	}{
		{
			name:      "head lines exceeds file",
			content:   "line1\nline2\n",
			opts:      ReadOptions{HeadLines: 10},
			wantTrunc: false,
		},
		{
			name:      "bytes exceeds file",
			content:   "small",
			opts:      ReadOptions{Bytes: 1000},
			wantTrunc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testFile := filepath.Join(tmp, "test.txt")
			if err := os.WriteFile(testFile, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("write test file failed: %v", err)
			}

			result, err := ReadFile(testFile, tt.opts)
			if err != nil {
				t.Fatalf("ReadFile failed: %v", err)
			}

			if result.Content != tt.content {
				t.Fatalf("expected content=%q, got %q", tt.content, result.Content)
			}
			if result.Truncated != tt.wantTrunc {
				t.Fatalf("expected truncated=%v, got %v", tt.wantTrunc, result.Truncated)
			}
		})
	}
}

func TestReadEmptyFile(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "empty.txt")
	if err := os.WriteFile(testFile, []byte{}, 0o600); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	result, err := ReadFile(testFile, ReadOptions{})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if result.Content != "" {
		t.Fatalf("expected empty content, got %q", result.Content)
	}
	if result.LineCount != 0 {
		t.Fatalf("expected line_count=0, got %d", result.LineCount)
	}
	if result.ByteCount != 0 {
		t.Fatalf("expected byte_count=0, got %d", result.ByteCount)
	}
}

func TestReadFileNotFound(t *testing.T) {
	tmp := t.TempDir()
	nonExistent := filepath.Join(tmp, "does_not_exist.txt")

	_, err := ReadFile(nonExistent, ReadOptions{})
	if err == nil {
		t.Fatalf("expected error for non-existent file")
	}
}

func TestReadDirectory(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "dir")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	_, err := ReadFile(dir, ReadOptions{})
	if err == nil {
		t.Fatalf("expected error for directory path")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Fatalf("expected directory error, got: %v", err)
	}
}

func TestReadMutuallyExclusiveFlags(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	// Test head + tail
	_, err := ReadFile(testFile, ReadOptions{HeadLines: 5, TailLines: 5})
	if err == nil {
		t.Fatalf("expected error for both head and tail")
	}

	// Test head + bytes
	_, err = ReadFile(testFile, ReadOptions{HeadLines: 5, Bytes: 10})
	if err == nil {
		t.Fatalf("expected error for both head and bytes")
	}

	// Test tail + bytes
	_, err = ReadFile(testFile, ReadOptions{TailLines: 5, Bytes: 10})
	if err == nil {
		t.Fatalf("expected error for both tail and bytes")
	}
}

func TestReadSingleLine(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.txt")
	content := "single line"
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	result, err := ReadFile(testFile, ReadOptions{})
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if result.Content != content {
		t.Fatalf("expected content=%q, got %q", content, result.Content)
	}
	// Single line without newline should still count as 1 line
	if result.LineCount != 1 {
		t.Fatalf("expected line_count=1, got %d", result.LineCount)
	}
}
