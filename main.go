package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

const (
	modeReadWrite = 0o666
	modeAll       = 0o777
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "reumask: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: reumask <umask> <path>")
	}

	umask, err := parseUmask(args[0])
	if err != nil {
		return err
	}

	info, err := os.Lstat(args[1])
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing to chmod symlink")
	}

	if info.IsDir() {
		return filepath.WalkDir(args[1], func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			return applyUmask(path, d.Type(), umask)
		})
	}

	return applyUmask(args[1], info.Mode(), umask)
}

func parseUmask(raw string) (fs.FileMode, error) {
	value, err := strconv.ParseUint(raw, 8, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid umask %q", raw)
	}
	if value > 0o777 {
		return 0, fmt.Errorf("umask %q is out of range", raw)
	}

	return fs.FileMode(value), nil
}

func applyUmask(path string, mode fs.FileMode, umask fs.FileMode) error {
	if mode&os.ModeSymlink != 0 {
		return nil
	}

	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	targetMode := remaskMode(info.Mode(), umask)
	if info.Mode().Perm() == targetMode.Perm() &&
		info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) ==
			targetMode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) {
		return nil
	}

	fmt.Printf("[%s -> %s] %s\n", formatMode(info.Mode()), formatMode(targetMode), path)
	return os.Chmod(path, targetMode)
}

func remaskMode(current fs.FileMode, umask fs.FileMode) fs.FileMode {
	base := fs.FileMode(modeReadWrite)
	if current.IsDir() || current.Perm()&0o111 != 0 {
		base = modeAll
	}

	special := current & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	return special | (base &^ umask)
}

func formatMode(mode fs.FileMode) string {
	return fmt.Sprintf("%#04o", mode.Perm())
}
