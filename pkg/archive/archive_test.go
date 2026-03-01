package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestZipAndUnzip(t *testing.T) {
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, `a.txt`)
	if err := os.WriteFile(srcFile, []byte(`hello`), 0o600); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	zipPath := filepath.Join(t.TempDir(), `out.zip`)
	if _, err := Zip(srcDir, zipPath); err != nil {
		t.Fatalf("zip failed: %v", err)
	}

	destDir := filepath.Join(t.TempDir(), `unzipped`)
	files, _, err := Unzip(zipPath, destDir)
	if err != nil {
		t.Fatalf("unzip failed: %v", err)
	}
	if files == 0 {
		t.Fatalf("expected extracted files > 0")
	}
}

func TestUnzipRejectsZipSlip(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), `slip.zip`)
	createZipWithUnsafeEntry(t, zipPath, `../evil.txt`, []byte(`x`))

	destDir := filepath.Join(t.TempDir(), `dest`)
	if _, _, err := Unzip(zipPath, destDir); err == nil {
		t.Fatalf("expected zip slip to be rejected")
	}
}

func createZipWithUnsafeEntry(t *testing.T, zipPath, name string, content []byte) {
	t.Helper()

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip failed: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry failed: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("write zip entry failed: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer failed: %v", err)
	}
}
