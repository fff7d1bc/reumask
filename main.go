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

const (
	modeReadWrite = 0o666
	modeAll       = 0o777
	usageText     = "usage: reumask [--dry-run] <umask> <path> [<path> ...]"
)

type config struct {
	dryRun bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "reumask: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, positional, err := parseArgs(args)
	if err != nil {
		return err
	}

	umask, err := parseUmask(positional[0])
	if err != nil {
		return err
	}

	for _, path := range positional[1:] {
		if err := runPath(path, umask, cfg); err != nil {
			return err
		}
	}

	return nil
}

func parseArgs(args []string) (config, []string, error) {
	var cfg config

	fs := flag.NewFlagSet("reumask", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "print changes without applying them")

	if err := fs.Parse(args); err != nil {
		return config{}, nil, err
	}
	if fs.NArg() < 2 {
		return config{}, nil, errors.New(usageText)
	}

	return cfg, fs.Args(), nil
}

func runPath(path string, umask fs.FileMode, cfg config) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing to chmod symlink")
	}

	if info.IsDir() {
		return filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			return applyUmask(walkPath, d.Type(), umask, cfg)
		})
	}

	return applyUmask(path, info.Mode(), umask, cfg)
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

func applyUmask(path string, mode fs.FileMode, umask fs.FileMode, cfg config) error {
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
	if cfg.dryRun {
		return nil
	}
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
