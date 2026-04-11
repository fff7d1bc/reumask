package main

import (
	"bytes"
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

func TestParseArgsDryRun(t *testing.T) {
	cfg, positional, err := parseArgs([]string{"--dry-run", "022", "path1", "path2"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if !cfg.dryRun {
		t.Fatal("parseArgs did not enable dry-run")
	}
	if len(positional) != 3 || positional[0] != "022" || positional[1] != "path1" || positional[2] != "path2" {
		t.Fatalf("parseArgs returned unexpected positional args: %#v", positional)
	}
}

func TestParseArgsUsageError(t *testing.T) {
	if _, _, err := parseArgs([]string{"022"}); err == nil || err.Error() != usageText {
		t.Fatalf("parseArgs returned %v, want %q", err, usageText)
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

func TestRunDryRunDoesNotChangePermissions(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	output := captureStdout(t, func() {
		if err := run([]string{"--dry-run", "022", file}); err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	})

	if output == "" {
		t.Fatal("expected dry-run to print planned change")
	}
	assertPerms(t, file, 0o600)
}

func TestRunAcceptsMultiplePaths(t *testing.T) {
	root := t.TempDir()
	file1 := filepath.Join(root, "file1")
	file2 := filepath.Join(root, "file2")

	if err := os.WriteFile(file1, []byte("one"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(file2, []byte("two"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := run([]string{"022", file1, file2}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	assertPerms(t, file1, 0o644)
	assertPerms(t, file2, 0o644)
}

func TestRunWarnsButDoesNotFailForTopLevelSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	stderr := captureStderr(t, func() {
		if err := run([]string{"022", link}); err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	})

	if stderr == "" {
		t.Fatal("expected warning for top-level symlink")
	}
	assertPerms(t, target, 0o600)
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}

	return captureOutput(t, reader, writer, func() (*os.File, func(*os.File)) {
		original := os.Stdout
		return original, func(file *os.File) {
			os.Stdout = file
		}
	}, fn)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}

	return captureOutput(t, reader, writer, func() (*os.File, func(*os.File)) {
		original := os.Stderr
		return original, func(file *os.File) {
			os.Stderr = file
		}
	}, fn)
}

func captureOutput(t *testing.T, reader *os.File, writer *os.File, getStream func() (*os.File, func(*os.File)), fn func()) string {
	t.Helper()

	original, setStream := getStream()
	setStream(writer)
	defer func() {
		setStream(original)
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	return buf.String()
}
