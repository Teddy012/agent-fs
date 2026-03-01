package sandbox

import (
	"path/filepath"
	"testing"
)

func TestResolvePathWithWorkspace(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(workspaceEnv, tmp)

	okPath := filepath.Join(tmp, `data`, `file.txt`)
	resolved, err := ResolveWritePath(okPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved != okPath {
		t.Fatalf("unexpected resolved path: %s", resolved)
	}
}

func TestResolvePathTraversalDenied(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(workspaceEnv, tmp)

	parent := filepath.Dir(tmp)
	outside := filepath.Join(parent, `outside.txt`)
	if _, err := ResolveReadPath(outside); err == nil {
		t.Fatalf("expected traversal error")
	}
}

func TestResolvePathWithoutWorkspace(t *testing.T) {
	t.Setenv(workspaceEnv, ``)

	target := filepath.Join(t.TempDir(), `x.txt`)
	if _, err := ResolveWritePath(target); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
