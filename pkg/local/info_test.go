package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetInfoForFile(t *testing.T) {
	tmp := t.TempDir()

	testFile := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	fileInfo, dirInfo, err := GetInfo(testFile, false)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if fileInfo.Type != "file" {
		t.Fatalf("expected type=file, got %s", fileInfo.Type)
	}
	if fileInfo.SizeBytes != 11 {
		t.Fatalf("expected size=11, got %d", fileInfo.SizeBytes)
	}
	if fileInfo.Mode != "0644" {
		t.Fatalf("expected mode=0644, got %s", fileInfo.Mode)
	}
	if fileInfo.Name != "test.txt" {
		t.Fatalf("expected name=test.txt, got %s", fileInfo.Name)
	}
	if fileInfo.IsDir {
		t.Fatalf("expected IsDir=false")
	}
	if dirInfo.Path != "" {
		t.Fatalf("expected empty dir info for file, got %+v", dirInfo)
	}
}

func TestGetInfoForDirectory(t *testing.T) {
	tmp := t.TempDir()

	testDir := filepath.Join(tmp, "testdata")
	if err := os.Mkdir(testDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Create some test files
	file1 := filepath.Join(testDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0o644); err != nil {
		t.Fatalf("write file1 failed: %v", err)
	}
	file2 := filepath.Join(testDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("content2"), 0o644); err != nil {
		t.Fatalf("write file2 failed: %v", err)
	}

	// Test without directory details
	fileInfo, _, err := GetInfo(testDir, false)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if fileInfo.Type != "directory" {
		t.Fatalf("expected type=directory, got %s", fileInfo.Type)
	}
	if !fileInfo.IsDir {
		t.Fatalf("expected IsDir=true")
	}

	// Test with directory details
	_, dirInfoWithDetails, err := GetInfo(testDir, true)
	if err != nil {
		t.Fatalf("GetInfo with details failed: %v", err)
	}

	if dirInfoWithDetails.FileCount != 2 {
		t.Fatalf("expected file_count=2, got %d", dirInfoWithDetails.FileCount)
	}
	if dirInfoWithDetails.TotalBytes != 16 { // "content1" (8) + "content2" (8)
		t.Fatalf("expected total_bytes=16, got %d", dirInfoWithDetails.TotalBytes)
	}
}

func TestGetInfoForNonExistentPath(t *testing.T) {
	tmp := t.TempDir()
	nonExistent := filepath.Join(tmp, "does_not_exist")

	_, _, err := GetInfo(nonExistent, false)
	if err == nil {
		t.Fatalf("expected error for non-existent path")
	}
}

func TestGetInfoForEmptyDirectory(t *testing.T) {
	tmp := t.TempDir()

	emptyDir := filepath.Join(tmp, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	_, dirInfo, err := GetInfo(emptyDir, true)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if dirInfo.FileCount != 0 {
		t.Fatalf("expected file_count=0 for empty directory, got %d", dirInfo.FileCount)
	}
	if dirInfo.TotalBytes != 0 {
		t.Fatalf("expected total_bytes=0 for empty directory, got %d", dirInfo.TotalBytes)
	}
}

func TestGetInfoForNestedDirectory(t *testing.T) {
	tmp := t.TempDir()

	parentDir := filepath.Join(tmp, "parent")
	if err := os.Mkdir(parentDir, 0o755); err != nil {
		t.Fatalf("mkdir parent failed: %v", err)
	}

	childDir := filepath.Join(parentDir, "child")
	if err := os.Mkdir(childDir, 0o755); err != nil {
		t.Fatalf("mkdir child failed: %v", err)
	}

	childFile := filepath.Join(childDir, "nested.txt")
	if err := os.WriteFile(childFile, []byte("nested content"), 0o644); err != nil {
		t.Fatalf("write nested file failed: %v", err)
	}

	_, dirInfo, err := GetInfo(parentDir, true)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if dirInfo.FileCount != 1 {
		t.Fatalf("expected file_count=1, got %d", dirInfo.FileCount)
	}
	if dirInfo.TotalBytes != 14 { // "nested content"
		t.Fatalf("expected total_bytes=14, got %d", dirInfo.TotalBytes)
	}
}

func TestGetInfoPermissionFormat(t *testing.T) {
	tmp := t.TempDir()

	// Test that mode is correctly formatted as 4-digit octal
	testCases := []struct {
		path string
		perm os.FileMode
	}{
		{"file_644", 0o644},
		{"file_600", 0o600},
		{"file_755", 0o755},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			testFile := filepath.Join(tmp, tc.path)
			if err := os.WriteFile(testFile, []byte("test"), tc.perm); err != nil {
				t.Fatalf("write test file failed: %v", err)
			}

			fileInfo, _, err := GetInfo(testFile, false)
			if err != nil {
				t.Fatalf("GetInfo failed: %v", err)
			}

			// Check that mode is a 4-character octal string
			if len(fileInfo.Mode) != 4 {
				t.Fatalf("expected mode to be 4 characters, got %d: %s", len(fileInfo.Mode), fileInfo.Mode)
			}
			// Check that mode only contains digits
			for _, c := range fileInfo.Mode {
				if c < '0' || c > '7' {
					t.Fatalf("expected mode to be octal digits, got: %s", fileInfo.Mode)
				}
			}
		})
	}
}
