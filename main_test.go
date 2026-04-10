package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseUmask(t *testing.T) {
	mask, err := parseUmask("022")
	if err != nil {
		t.Fatalf("parseUmask returned error: %v", err)
	}
	if mask != 0o022 {
		t.Fatalf("parseUmask returned %04o, want 0022", mask)
	}
}

func TestParseUmaskRejectsInvalidValue(t *testing.T) {
	if _, err := parseUmask("888"); err == nil {
		t.Fatal("parseUmask accepted invalid octal input")
	}
}

func TestRemaskModeForRegularFile(t *testing.T) {
	got := remaskMode(0o600, 0o022)
	if got.Perm() != 0o644 {
		t.Fatalf("remaskMode returned %04o, want 0644", got.Perm())
	}
}

func TestRemaskModeForExecutableFile(t *testing.T) {
	got := remaskMode(0o700, 0o022)
	if got.Perm() != 0o755 {
		t.Fatalf("remaskMode returned %04o, want 0755", got.Perm())
	}
}

func TestRemaskModeForDirectory(t *testing.T) {
	got := remaskMode(os.ModeDir|0o700, 0o022)
	if got.Perm() != 0o755 {
		t.Fatalf("remaskMode returned %04o, want 0755", got.Perm())
	}
}

func TestRunRecursivelyRemasksDirectoryTree(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dir")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	execFile := filepath.Join(dir, "tool")
	if err := os.WriteFile(execFile, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	link := filepath.Join(root, "dir-link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	if err := run([]string{"022", root}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	assertPerms(t, root, 0o755)
	assertPerms(t, dir, 0o755)
	assertPerms(t, file, 0o644)
	assertPerms(t, execFile, 0o755)

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat on symlink failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to remain a symlink", link)
	}
}

func assertPerms(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed for %s: %v", path, err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("%s has mode %04o, want %04o", path, info.Mode().Perm(), want)
	}
}
