# reumask

`reumask` reapplies a target umask to an existing file or directory tree.

It is meant for the case where files were created under one umask, and later need
to look as if they had been created under a different umask from the start.

Example:

- a regular file created under `077` starts as `0600`
- after `chmod +x`, it becomes `0700`
- `reumask 022 file` changes it to `0755`

Rules:

- directories are treated as if they were created from `0777`
- non-executable files are treated as if they were created from `0666`
- files with any execute bit set are treated as if they were created from `0777`
- directory input is processed recursively
- symlinks are ignored
- output is printed only for paths whose permissions actually changed
- `--dry-run` prints planned changes without applying them

Usage:

```sh
reumask [--dry-run] <umask> <path>
```

Examples:

```sh
reumask 022 .
reumask 027 some/file
reumask --dry-run 022 some/tree
```

Build:

```sh
make build
make static
make test
make release
```
