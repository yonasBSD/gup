package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFile(t *testing.T) {
	t.Parallel()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		f, err := os.CreateTemp(t.TempDir(), "testfile")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
		if !IsFile(f.Name()) {
			t.Errorf("IsFile(%q) = false, want true", f.Name())
		}
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if IsFile(dir) {
			t.Errorf("IsFile(%q) = true, want false for directory", dir)
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		t.Parallel()
		if IsFile("/non/existent/path") {
			t.Error("IsFile should return false for non-existent path")
		}
	})
}

func TestIsDir(t *testing.T) {
	t.Parallel()

	t.Run("existing directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if !IsDir(dir) {
			t.Errorf("IsDir(%q) = false, want true", dir)
		}
	})

	t.Run("file", func(t *testing.T) {
		t.Parallel()
		f, err := os.CreateTemp(t.TempDir(), "testfile")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
		if IsDir(f.Name()) {
			t.Errorf("IsDir(%q) = true, want false for file", f.Name())
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		t.Parallel()
		if IsDir("/non/existent/path") {
			t.Error("IsDir should return false for non-existent path")
		}
	})
}

func TestIsHiddenFile(t *testing.T) {
	t.Parallel()

	t.Run("hidden file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		hidden := filepath.Join(dir, ".hidden")
		if err := os.WriteFile(hidden, []byte("test"), FileModeCreatingFile); err != nil {
			t.Fatal(err)
		}
		if !IsHiddenFile(hidden) {
			t.Errorf("IsHiddenFile(%q) = false, want true", hidden)
		}
	})

	t.Run("non-hidden file", func(t *testing.T) {
		t.Parallel()
		f, err := os.CreateTemp(t.TempDir(), "visible")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
		if IsHiddenFile(f.Name()) {
			t.Errorf("IsHiddenFile(%q) = true, want false", f.Name())
		}
	})

	t.Run("non-existent hidden path", func(t *testing.T) {
		t.Parallel()
		if IsHiddenFile("/non/existent/.hidden") {
			t.Error("IsHiddenFile should return false for non-existent path")
		}
	})

	t.Run("directory with dot prefix", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dotDir := filepath.Join(dir, ".hiddendir")
		if err := os.Mkdir(dotDir, FileModeCreatingDir); err != nil {
			t.Fatal(err)
		}
		if IsHiddenFile(dotDir) {
			t.Errorf("IsHiddenFile(%q) = true, want false for directory", dotDir)
		}
	})
}
