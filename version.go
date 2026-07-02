package version

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Info contains version information
type Info struct {
	Version        string
	GitCommit      string
	GitCommitShort string
	GitBranch      string
	GitBranchSlug  string
	GitDescribe    string
	LatestTag      string
	BuildTime      string
	IsDirty        bool
	DefaultBranch  string
}

// Options holds configuration for GetVersionInfo.
type Options struct {
	// DefaultBranch is the main/default branch name. Auto-detected if empty.
	DefaultBranch string
	// TagPrefix scopes tag lookup to tags beginning with this prefix.
	// By default the prefix is kept in the version output; set StripPrefix
	// to remove it.
	TagPrefix string
	// StripPrefix removes TagPrefix from the version output.
	// When false (default) the prefix is preserved in Version and GitDescribe.
	// LatestTag always holds the stripped semver portion regardless of this flag.
	StripPrefix bool
	// FilterPath restricts the commit distance counter to commits that
	// touched files under this path. Commits that exclusively modified
	// other directories are invisible to the version counter.
	FilterPath string
	// Target, when set, causes the output to be the next semver bumped from
	// the latest matching tag. Valid values: "patch", "minor", "major".
	// "dev" (or empty) is a no-op and produces the normal dev version string.
	// The dirty-tree timestamp suffix is suppressed for patch/minor/major.
	Target string
	// Latest, when true, outputs the nearest tag reachable from HEAD without
	// any generated suffix. Useful for querying the current release version.
	// Mutually exclusive with a real Target (patch/minor/major).
	Latest bool
}

// GetVersionInfo retrieves version information from the Git repository at the given path.
func GetVersionInfo(repoPath string, opts Options) (*Info, error) {
	// Find the git root by walking up until .git is found
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	origPath := absPath
	gitRoot := ""
	for {
		gitDir := absPath + "/.git"
		if fi, err := os.Stat(gitDir); err == nil && (fi.IsDir() || fi.Mode().IsRegular()) {
			gitRoot = absPath
			break
		}
		parent := parentDir(absPath)
		if parent == absPath {
			// Reached filesystem root
			return nil, fmt.Errorf("failed to open repository: no .git found from %s upwards", origPath)
		}
		absPath = parent
	}

	repo, err := git.PlainOpen(gitRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	info := &Info{
		BuildTime: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}

	defaultBranch := opts.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = detectDefaultBranch(repo)
	}
	info.DefaultBranch = defaultBranch

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	info.GitCommit = head.Hash().String()
	info.GitCommitShort = head.Hash().String()[:7]

	if head.Name().IsBranch() {
		info.GitBranch = head.Name().Short()
	} else {
		info.GitBranch = "HEAD"
	}

	info.GitBranchSlug = createBranchSlug(info.GitBranch)

	info.GitDescribe, info.LatestTag = getGitDescribe(repo, head.Hash(), opts.TagPrefix, opts.FilterPath, opts.StripPrefix)

	info.IsDirty = hasUncommittedChanges(repo)

	realTarget := opts.Target != "" && opts.Target != "dev"
	if opts.Latest && realTarget {
		return nil, fmt.Errorf("-latest and -target are mutually exclusive")
	}

	// -latest: return the nearest reachable tag without any generated suffix.
	if opts.Latest {
		if info.LatestTag == "" {
			return nil, fmt.Errorf("no tags found")
		}
		// Reconstruct the output tag name (prefix already applied by getGitDescribe
		// when StripPrefix is false, but LatestTag is always stripped — re-add here).
		latestOutput := info.LatestTag
		if !opts.StripPrefix && opts.TagPrefix != "" {
			latestOutput = opts.TagPrefix + info.LatestTag
		}
		info.Version = latestOutput
		return info, nil
	}

	// When HEAD is exactly on a tagged commit (or path-filtered distance is zero),
	// GitDescribe contains no "+dev." suffix — use the tag regardless of branch.
	atExactTag := info.LatestTag != "" && !strings.Contains(info.GitDescribe, "+dev.")
	if atExactTag {
		info.Version = info.GitDescribe
	} else if info.GitBranch == defaultBranch && info.GitDescribe != "" {
		info.Version = info.GitDescribe
	} else {
		info.Version = fmt.Sprintf("%s-g%s", info.GitBranchSlug, info.GitCommitShort)
	}

	if realTarget {
		bumped, err := bumpSemver(info.LatestTag, opts.Target)
		if err != nil {
			return nil, err
		}
		if !opts.StripPrefix && opts.TagPrefix != "" {
			bumped = opts.TagPrefix + bumped
		}
		info.Version = bumped
		return info, nil
	}

	if info.IsDirty {
		timestamp := time.Now().UTC().Format("20060102150405")
		info.Version = fmt.Sprintf("%s-%s", info.Version, timestamp)
	}

	return info, nil
}

// parentDir returns the parent directory of the given path.
func parentDir(path string) string {
	if path == "/" {
		return "/"
	}
	path = strings.TrimRight(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx <= 0 {
		return "/"
	}
	return path[:idx]
}

// detectDefaultBranch attempts to detect the default branch from the repository.
// It checks the symbolic ref of origin/HEAD, falling back to common defaults.
func detectDefaultBranch(repo *git.Repository) string {
	ref, err := repo.Reference(plumbing.NewRemoteHEADReferenceName("origin"), true)
	if err == nil && ref != nil {
		refName := ref.Name().Short()
		if strings.HasPrefix(refName, "origin/") {
			return strings.TrimPrefix(refName, "origin/")
		}
		return refName
	}

	branches := []string{"main", "master"}
	refs, err := repo.References()
	if err == nil {
		existingBranches := make(map[string]bool)
		refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Name().IsBranch() {
				existingBranches[ref.Name().Short()] = true
			}
			return nil
		})

		for _, branch := range branches {
			if existingBranches[branch] {
				return branch
			}
		}
	}

	return "main"
}

// hasUncommittedChanges checks if the repository has uncommitted changes.
// Only checks for staged and unstaged modifications, not untracked files.
func hasUncommittedChanges(repo *git.Repository) bool {
	worktree, err := repo.Worktree()
	if err != nil {
		return false
	}

	status, err := worktree.Status()
	if err != nil {
		return false
	}

	for _, fileStatus := range status {
		if fileStatus.Staging != git.Untracked && fileStatus.Staging != git.Unmodified {
			return true
		}
		if fileStatus.Worktree != git.Untracked && fileStatus.Worktree != git.Unmodified {
			return true
		}
	}

	return false
}

// createBranchSlug creates a slug from branch name.
// Replaces / and _ with -, keeps only alphanumeric and -.
func createBranchSlug(branch string) string {
	slug := strings.ReplaceAll(branch, "/", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	reg := regexp.MustCompile("[^a-zA-Z0-9-]+")
	slug = reg.ReplaceAllString(slug, "")

	return slug
}

// getGitDescribe implements git-describe-like logic with optional tag prefix scoping
// and path-based commit filtering.
//
// tagPrefix, if non-empty, restricts tag lookup to tags beginning with that prefix.
// The internal tagMap always stores stripped names so that LatestTag and bumpSemver
// always receive clean semver. When stripPrefix is false the prefix is re-added to
// the describe output and second return value.
//
// filterPath, if non-empty, causes the commit distance counter to skip commits
// that did not touch any file under that path.
//
// Returns (describe, strippedTagName). When the distance is 0 (either exactly at the
// tag or all commits since the tag are outside filterPath), describe equals the tag
// (with or without prefix per stripPrefix). When ahead, describe is
// "{tag}+dev.{N}.g{hash}" — valid semver build metadata.
func getGitDescribe(repo *git.Repository, hash plumbing.Hash, tagPrefix string, filterPath string, stripPrefix bool) (string, string) {
	tagRefs, err := repo.Tags()
	if err != nil {
		return "", ""
	}

	// outTag returns the tag name for version output: stripped when stripPrefix is
	// true, prefixed otherwise.
	outTag := func(stripped string) string {
		if stripPrefix || tagPrefix == "" {
			return stripped
		}
		return tagPrefix + stripped
	}

	// Build a map of commit-hash → stripped tag name.
	// Annotated tags are resolved to their target commit hash.
	tagMap := make(map[plumbing.Hash]string)
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		if tagPrefix != "" && !strings.HasPrefix(name, tagPrefix) {
			return nil
		}
		stripped := strings.TrimPrefix(name, tagPrefix)

		// Resolve annotated tags to their target commit hash.
		if tagObj, err := repo.TagObject(ref.Hash()); err == nil {
			tagMap[tagObj.Target] = stripped
		} else {
			tagMap[ref.Hash()] = stripped
		}
		return nil
	})
	if err != nil {
		return "", ""
	}

	// Exactly at a tag.
	if stripped, exists := tagMap[hash]; exists {
		return outTag(stripped), stripped
	}

	// Walk commit history to find the nearest ancestor that carries a tag.
	commitIter, err := repo.Log(&git.LogOptions{From: hash})
	if err != nil {
		return "", ""
	}
	defer commitIter.Close()

	distance := 0
	var foundTag string

	commitIter.ForEach(func(commit *object.Commit) error {
		if tagName, exists := tagMap[commit.Hash]; exists {
			foundTag = tagName
			return fmt.Errorf("stop") // non-nil stops iteration
		}

		if filterPath != "" {
			if commitTouchesPath(commit, filterPath) {
				distance++
			}
		} else {
			distance++
		}
		return nil
	})

	if foundTag == "" {
		return "", ""
	}

	// If path filtering reduced the relevant distance to zero, report as clean.
	if distance == 0 {
		return outTag(foundTag), foundTag
	}

	shortHash := hash.String()[:7]
	describe := fmt.Sprintf("%s+dev.%d.g%s", outTag(foundTag), distance, shortHash)
	return describe, foundTag
}

// commitTouchesPath reports whether commit changed at least one file whose path
// starts with the given prefix.
func commitTouchesPath(commit *object.Commit, path string) bool {
	stats, err := commit.Stats()
	if err != nil {
		return false
	}
	for _, stat := range stats {
		if strings.HasPrefix(stat.Name, path) {
			return true
		}
	}
	return false
}

// bumpSemver increments the given semver tag according to target ("patch",
// "minor", or "major"). An optional leading "v" is preserved. If tag is empty
// the base is treated as "0.0.0" with no "v" prefix.
func bumpSemver(tag string, target string) (string, error) {
	switch target {
	case "patch", "minor", "major":
	default:
		return "", fmt.Errorf("-target %q is not valid: must be patch, minor, or major", target)
	}

	raw := tag
	vPrefix := ""
	if strings.HasPrefix(raw, "v") || strings.HasPrefix(raw, "V") {
		vPrefix = string(raw[0])
		raw = raw[1:]
	}
	if raw == "" {
		raw = "0.0.0"
	}

	parts := strings.SplitN(raw, ".", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("tag %q is not a valid semver (expected major.minor.patch)", tag)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("tag %q: cannot parse major: %v", tag, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("tag %q: cannot parse minor: %v", tag, err)
	}
	// Strip any pre-release/build suffix from the patch component (e.g. "3-rc.1" → "3").
	patchStr := parts[2]
	if i := strings.IndexAny(patchStr, "-+"); i >= 0 {
		patchStr = patchStr[:i]
	}
	patch, err := strconv.Atoi(patchStr)
	if err != nil {
		return "", fmt.Errorf("tag %q: cannot parse patch: %v", tag, err)
	}

	switch target {
	case "patch":
		patch++
	case "minor":
		minor++
		patch = 0
	case "major":
		major++
		minor = 0
		patch = 0
	}

	return fmt.Sprintf("%s%d.%d.%d", vPrefix, major, minor, patch), nil
}

// String returns the version string.
func (i *Info) String() string {
	return i.Version
}

// DetailedString returns a detailed multi-line string with all version information.
func (i *Info) DetailedString() string {
	dirtyStr := "clean"
	if i.IsDirty {
		dirtyStr = "dirty"
	}
	tagStr := i.LatestTag
	if tagStr == "" {
		tagStr = "(none)"
	}
	return fmt.Sprintf(`Version:        %s
Commit:         %s
Branch:         %s
Branch Slug:    %s
Default Branch: %s
Latest Tag:     %s
Build Time:     %s
Dirty:          %s`,
		i.Version,
		i.GitCommit,
		i.GitBranch,
		i.GitBranchSlug,
		i.DefaultBranch,
		tagStr,
		i.BuildTime,
		dirtyStr,
	)
}
