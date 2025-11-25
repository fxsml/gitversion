# gitversion

A simple Git versioning utility that generates version strings based on Git repository state.

## Installation

### From source

```bash
go install github.com/fxsml/gitversion@latest
```

### Build locally

```bash
make build
```

## Usage

### Basic usage

```bash
gitversion
```

Output: `main-ge7e38cf` (when on main branch without tags)
Or: `v1.0.0-5-ge7e38cf` (when 5 commits ahead of v1.0.0 tag)

### Show help

```bash
gitversion help
```

### Detailed output

```bash
gitversion -detailed
```

Output:
```
Version:        main-ge7e38cf
Commit:         e7e38cf7d71fe815c4f3bde53ad3bb23e57f5a5f
Branch:         main
Default Branch: main
Latest Tag:     (none)
Build Time:     2025-11-25T11:11:47Z
Dirty:          clean
```

### Specify repository path

```bash
gitversion -path /path/to/repo
```

### Show only version

```bash
gitversion -short
```

### Specify default branch

```bash
gitversion -default-branch master
```

## Version Logic

The tool uses different strategies based on whether you're on the default branch:

### Default Branch (main/master - auto-detected or specified)
- **At tagged commit:** Uses tag name (e.g., `v1.0.0`)
- **Ahead of tag:** Uses `git describe` format (e.g., `v1.0.0-5-g1234567`)
- **No tags in history:** Uses `{branch-slug}-g{short-commit-hash}`

### Other Branches
- **Always:** Uses `{branch-slug}-g{short-commit-hash}` (regardless of tags)

### Uncommitted Changes
- **Dirty working tree:** Appends timestamp suffix `-YYYYMMDDHHMMSS`
- **Note:** Only tracks modifications to tracked files, ignores untracked files

### Branch Slug
Sanitizes the branch name: replaces `/` and `_` with `-`, keeps only alphanumeric and `-`

### Default Branch Detection
- Auto-detected from `origin/HEAD` or falls back to `main`/`master`
- Can be overridden with `-default-branch` flag
- **Important:** Determines whether to use git describe format or simple branch-commit format

## Examples

| Branch | Position | Clean/Dirty | Output |
|--------|----------|-------------|--------|
| main (default) | at tag v1.0.0 | clean | v1.0.0 |
| main (default) | at tag v1.0.0 | dirty | v1.0.0-20251125115903 |
| main (default) | 5 commits after v1.0.0 | clean | v1.0.0-5-gabc123d |
| main (default) | no tags | clean | main-gabc123d |
| main (default) | no tags | dirty | main-gabc123d-20251125115903 |
| feature/new-feature | any | clean | feature-new-feature-gabc123d |
| feature/new-feature | any | dirty | feature-new-feature-gabc123d-20251125115903 |

## Development

### Run tests

```bash
make test
```

### Run tests with coverage

```bash
make test-cover
```

### Generate coverage report

```bash
make cover
```

### Run all checks

```bash
make all
```

### Available make targets

Run `make help` to see all available targets.

## License

MIT License - Copyright 2025 Steve
