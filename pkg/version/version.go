package version

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// GetVersionInfo retrieves version information from the Git repository at the given path
// defaultBranch specifies the main branch (e.g., "main" or "master"). If empty, attempts auto-detection.
func GetVersionInfo(repoPath string, defaultBranch string) (*Info, error) {
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

	// Auto-detect default branch if not specified
	if defaultBranch == "" {
		defaultBranch = detectDefaultBranch(repo)
	}

	// Store the default branch in info
	info.DefaultBranch = defaultBranch

	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit hash
	info.GitCommit = head.Hash().String()
	info.GitCommitShort = head.Hash().String()[:7]

	// Get branch name
	if head.Name().IsBranch() {
		info.GitBranch = head.Name().Short()
	} else {
		// Detached HEAD state
		info.GitBranch = "HEAD"
	}

	// Create branch slug
	info.GitBranchSlug = createBranchSlug(info.GitBranch)

	// Get git describe (tags)
	info.GitDescribe, info.LatestTag = getGitDescribe(repo, head.Hash())

	// Check for uncommitted changes
	info.IsDirty = hasUncommittedChanges(repo)

	// Determine version based on branch and tags
	if info.GitBranch == defaultBranch {
		// On default branch: use git describe if tags exist, otherwise branch-slug-ghash
		if info.GitDescribe != "" {
			info.Version = info.GitDescribe
		} else {
			info.Version = fmt.Sprintf("%s-g%s", info.GitBranchSlug, info.GitCommitShort)
		}
	} else {
		// On other branches: always use branch-slug-ghash format
		info.Version = fmt.Sprintf("%s-g%s", info.GitBranchSlug, info.GitCommitShort)
	}

	// Append timestamp suffix if there are uncommitted changes
	if info.IsDirty {
		timestamp := time.Now().UTC().Format("20060102150405")
		info.Version = fmt.Sprintf("%s-%s", info.Version, timestamp)
	}

	return info, nil
}

// parentDir returns the parent directory of the given path
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

// detectDefaultBranch attempts to detect the default branch from the repository
// It checks the symbolic ref of origin/HEAD, falling back to common defaults
func detectDefaultBranch(repo *git.Repository) string {
	// Try to get the default branch from origin/HEAD
	ref, err := repo.Reference(plumbing.NewRemoteHEADReferenceName("origin"), true)
	if err == nil && ref != nil {
		// Extract branch name from refs/remotes/origin/HEAD -> origin/main
		refName := ref.Name().Short()
		// Remove "origin/" prefix if present
		if strings.HasPrefix(refName, "origin/") {
			return strings.TrimPrefix(refName, "origin/")
		}
		return refName
	}

	// Fallback: check if main or master branch exists
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

	// Ultimate fallback
	return "main"
}

// hasUncommittedChanges checks if the repository has uncommitted changes
// Only checks for staged and unstaged modifications, not untracked files
func hasUncommittedChanges(repo *git.Repository) bool {
	worktree, err := repo.Worktree()
	if err != nil {
		return false
	}

	status, err := worktree.Status()
	if err != nil {
		return false
	}

	// Check only for modified, added, deleted, renamed, or copied files
	// Ignore untracked files (Untracked status)
	for _, fileStatus := range status {
		// Check staging area
		if fileStatus.Staging != git.Untracked && fileStatus.Staging != git.Unmodified {
			return true
		}
		// Check worktree (but not untracked files)
		if fileStatus.Worktree != git.Untracked && fileStatus.Worktree != git.Unmodified {
			return true
		}
	}

	return false
}

// createBranchSlug creates a slug from branch name
// Replaces / and _ with -, keeps only alphanumeric and -
func createBranchSlug(branch string) string {
	// Replace / and _ with -
	slug := strings.ReplaceAll(branch, "/", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Keep only alphanumeric and -
	reg := regexp.MustCompile("[^a-zA-Z0-9-]+")
	slug = reg.ReplaceAllString(slug, "")

	return slug
}

// getGitDescribe attempts to get the output similar to 'git describe --tags HEAD'
// Returns (describe, tagName) where describe is the full git describe output and tagName is just the tag
func getGitDescribe(repo *git.Repository, hash plumbing.Hash) (string, string) {
	// Get all tags and build a map of commit hash -> tag name
	tagRefs, err := repo.Tags()
	if err != nil {
		return "", ""
	}

	tagMap := make(map[plumbing.Hash]string)
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		tagMap[ref.Hash()] = ref.Name().Short()
		return nil
	})
	if err != nil {
		return "", ""
	}

	// Check if current commit is exactly at a tag
	if tagName, exists := tagMap[hash]; exists {
		return tagName, tagName
	}

	// Walk commit history to find the most recent tag
	commitIter, err := repo.Log(&git.LogOptions{
		From: hash,
	})
	if err != nil {
		return "", ""
	}
	defer commitIter.Close()

	distance := 0
	var foundTag string

	err = commitIter.ForEach(func(commit *object.Commit) error {
		if tagName, exists := tagMap[commit.Hash]; exists {
			foundTag = tagName
			return fmt.Errorf("found") // Stop iteration
		}
		distance++
		return nil
	})

	if foundTag != "" {
		// Format as tag-distance-ghash (e.g., v1.0.0-5-g1234567)
		shortHash := hash.String()[:7]
		describe := fmt.Sprintf("%s-%d-g%s", foundTag, distance, shortHash)
		return describe, foundTag
	}

	return "", ""
}

// String returns a formatted string representation of the version info
func (i *Info) String() string {
	return i.Version
}

// DetailedString returns a detailed multi-line string with all version information
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
Default Branch: %s
Latest Tag:     %s
Build Time:     %s
Dirty:          %s`,
		i.Version,
		i.GitCommit,
		i.GitBranch,
		i.DefaultBranch,
		tagStr,
		i.BuildTime,
		dirtyStr,
	)
}
