package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "reumask: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	dryRun, positional, err := parseArgs(args)
	if err != nil {
		return err
	}

	umask, err := parseUmask(positional[0])
	if err != nil {
		return err
	}

	for _, path := range positional[1:] {
		if err := runPath(path, umask, dryRun); err != nil {
			return err
		}
	}

	return nil
}

func parseArgs(args []string) (bool, []string, error) {
	var dryRun bool
	fs := flag.NewFlagSet("reumask", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.BoolVar(&dryRun, "dry-run", false, "print changes without applying them")

	if err := fs.Parse(args); err != nil {
		return false, nil, err
	}
	if fs.NArg() < 2 {
		return false, nil, errors.New("usage: reumask [--dry-run] <umask> <path> [<path> ...]")
	}

	return dryRun, fs.Args(), nil
}

func runPath(path string, umask fs.FileMode, dryRun bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		fmt.Fprintf(os.Stderr, "skipping %s: path is a symlink\n", path)
		return nil
	}

	if info.IsDir() {
		return filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			return applyUmask(walkPath, info.Mode(), umask, dryRun)
		})
	}

	return applyUmask(path, info.Mode(), umask, dryRun)
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

func applyUmask(path string, mode fs.FileMode, umask fs.FileMode, dryRun bool) error {
	if mode&os.ModeSymlink != 0 {
		return nil
	}

	targetMode := remaskMode(mode, umask)
	if mode.Perm() == targetMode.Perm() &&
		mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) ==
			targetMode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) {
		return nil
	}

	fmt.Printf("[%s -> %s] %s\n", formatMode(mode), formatMode(targetMode), path)
	if dryRun {
		return nil
	}
	return os.Chmod(path, targetMode)
}

func remaskMode(current fs.FileMode, umask fs.FileMode) fs.FileMode {
	base := fs.FileMode(0o666)
	if current.IsDir() || current.Perm()&0o111 != 0 {
		base = 0o777
	}

	special := current & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	return special | (base &^ umask)
}

func formatMode(mode fs.FileMode) string {
	return fmt.Sprintf("%#04o", mode.Perm())
}
