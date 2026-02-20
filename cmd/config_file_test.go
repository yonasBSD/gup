package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRenameWithBackupSwap_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "gup.json.tmp")
	dst := filepath.Join(dir, "gup.json")

	if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := renameWithBackupSwap(src, dst); err != nil {
		t.Fatalf("renameWithBackupSwap() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Clean(dst))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("updated content = %q, want %q", string(got), "new")
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("src file should be moved, stat err = %v", err)
	}
}

func TestRenameWithBackupSwap_RestoreOnFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "missing.tmp")
	dst := filepath.Join(dir, "gup.json")

	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := renameWithBackupSwap(src, dst); err == nil {
		t.Fatal("renameWithBackupSwap() should return error")
	}

	got, err := os.ReadFile(filepath.Clean(dst))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("restored content = %q, want %q", string(got), "old")
	}
}

func Test_shouldRetryRenameWithReplace(t *testing.T) {
	t.Parallel()

	dst := filepath.Join(t.TempDir(), "gup.json")
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if !shouldRetryRenameWithReplace(os.ErrExist, dst) {
		t.Fatal("shouldRetryRenameWithReplace() should return true for os.ErrExist")
	}

	got := shouldRetryRenameWithReplace(os.ErrNotExist, dst)
	if runtime.GOOS == goosWindows {
		if !got {
			t.Fatal("shouldRetryRenameWithReplace() should return true on Windows when dst exists")
		}
		return
	}
	if got {
		t.Fatal("shouldRetryRenameWithReplace() should return false on non-Windows for non-exist error")
	}
}

func Test_renameWithReplace_errorWhenSrcMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "missing.tmp")
	dst := filepath.Join(dir, "gup.json")
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := renameWithReplace(src, dst); err == nil {
		t.Fatal("renameWithReplace() should return error when source file does not exist")
	}
}
