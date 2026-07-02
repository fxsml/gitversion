package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// helper: initialise a bare git repo and return repo + worktree
func initRepo(t *testing.T) (string, *git.Repository, *git.Worktree) {
	t.Helper()
	dir, err := os.MkdirTemp("", "gitversion-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	return dir, repo, w
}

// helper: write a file, stage it, and commit; returns the commit hash
func writeAndCommit(t *testing.T, dir string, w *git.Worktree, filePath, content, message string) plumbing.Hash {
	t.Helper()
	full := filepath.Join(dir, filePath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := w.Add(filePath); err != nil {
		t.Fatalf("Add: %v", err)
	}
	hash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com"},
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return hash
}

// helper: create a lightweight tag
func lightweightTag(t *testing.T, repo *git.Repository, name string, hash plumbing.Hash) {
	t.Helper()
	ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/tags/"+name), hash)
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("SetReference (tag %s): %v", name, err)
	}
}

func TestGetVersionInfoFromSubdirectory(t *testing.T) {
	dir, repo, w := initRepo(t)

	subDir := filepath.Join(dir, "subdir1", "subdir2")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	commit := writeAndCommit(t, dir, w, "test.txt", "test", "Initial commit")

	info, err := GetVersionInfo(subDir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}

	if info.GitCommit != commit.String() {
		t.Errorf("GitCommit = %q, want %q", info.GitCommit, commit.String())
	}
	if info.GitBranch != "master" && info.GitBranch != "main" {
		t.Errorf("GitBranch = %q, want 'master' or 'main'", info.GitBranch)
	}
	expectedVersion := info.GitBranchSlug + "-g" + info.GitCommitShort
	if info.Version != expectedVersion {
		t.Errorf("Version = %q, want %q", info.Version, expectedVersion)
	}

	_ = repo // used only for setup
}

func TestCreateBranchSlug(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{"simple branch", "main", "main"},
		{"feature branch with slash", "feature/new-feature", "feature-new-feature"},
		{"branch with underscore", "feature_branch", "feature-branch"},
		{"complex branch name", "feature/JIRA-123_update", "feature-JIRA-123-update"},
		{"branch with special characters", "feature/test@123", "feature-test123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createBranchSlug(tt.branch)
			if result != tt.expected {
				t.Errorf("createBranchSlug(%q) = %q, want %q", tt.branch, result, tt.expected)
			}
		})
	}
}

func TestGetVersionInfo(t *testing.T) {
	dir, _, w := initRepo(t)

	commit := writeAndCommit(t, dir, w, "test.txt", "test", "Initial commit")

	info, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}

	if info.GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
	if info.GitCommit != commit.String() {
		t.Errorf("GitCommit = %q, want %q", info.GitCommit, commit.String())
	}
	if len(info.GitCommitShort) != 7 {
		t.Errorf("GitCommitShort length = %d, want 7", len(info.GitCommitShort))
	}
	if info.GitBranch != "master" && info.GitBranch != "main" {
		t.Errorf("GitBranch = %q, want 'master' or 'main'", info.GitBranch)
	}
	if info.BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
	expectedVersion := info.GitBranchSlug + "-g" + info.GitCommitShort
	if info.Version != expectedVersion {
		t.Errorf("Version = %q, want %q", info.Version, expectedVersion)
	}
}

func TestGetVersionInfoWithBranch(t *testing.T) {
	dir, repo, w := initRepo(t)

	writeAndCommit(t, dir, w, "test.txt", "test", "Initial commit")

	branchName := "feature/test-branch"
	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}

	ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+branchName), headRef.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("SetReference: %v", err)
	}
	if err := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branchName),
	}); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	info, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}

	if info.GitBranch != branchName {
		t.Errorf("GitBranch = %q, want %q", info.GitBranch, branchName)
	}

	expectedSlug := "feature-test-branch"
	if info.GitBranchSlug != expectedSlug {
		t.Errorf("GitBranchSlug = %q, want %q", info.GitBranchSlug, expectedSlug)
	}
	if !strings.HasPrefix(info.Version, expectedSlug+"-") {
		t.Errorf("Version = %q, want prefix %q", info.Version, expectedSlug+"-")
	}
}

func TestInfoString(t *testing.T) {
	info := &Info{
		Version:        "v1.0.0",
		GitCommit:      "abc123def456",
		GitCommitShort: "abc123d",
		GitBranch:      "main",
		BuildTime:      "2025-01-01T00:00:00Z",
	}

	result := info.String()
	if result != "v1.0.0" {
		t.Errorf("String() = %q, want %q", result, "v1.0.0")
	}
}

func TestInfoDetailedString(t *testing.T) {
	info := &Info{
		Version:        "v1.0.0",
		GitCommit:      "abc123def456",
		GitCommitShort: "abc123d",
		GitBranch:      "main",
		DefaultBranch:  "main",
		GitDescribe:    "v1.0.0",
		LatestTag:      "v1.0.0",
		BuildTime:      "2025-01-01T00:00:00Z",
	}

	result := info.DetailedString()

	checks := []struct {
		desc string
		want string
	}{
		{"version", "v1.0.0"},
		{"full commit", "abc123def456"},
		{"branch", "main"},
		{"default branch label", "Default Branch: main"},
		{"latest tag label", "Latest Tag:     v1.0.0"},
		{"build time", "2025-01-01T00:00:00Z"},
		{"dirty status", "clean"},
	}
	for _, c := range checks {
		if !strings.Contains(result, c.want) {
			t.Errorf("DetailedString() missing %s (%q)", c.desc, c.want)
		}
	}
}

func TestGetVersionInfoWithUncommittedChanges(t *testing.T) {
	dir, _, w := initRepo(t)

	testFile := filepath.Join(dir, "test.txt")
	writeAndCommit(t, dir, w, "test.txt", "test", "Initial commit")

	info, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}
	if info.IsDirty {
		t.Error("IsDirty should be false for clean working tree")
	}
	if strings.Contains(info.Version, "-202") {
		t.Errorf("Version should not have timestamp suffix for clean tree: %s", info.Version)
	}

	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	infoDirty, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}
	if !infoDirty.IsDirty {
		t.Error("IsDirty should be true for dirty working tree")
	}
	if !strings.Contains(infoDirty.Version, "-202") {
		t.Errorf("Version should have timestamp suffix for dirty tree: %s", infoDirty.Version)
	}

	parts := strings.Split(infoDirty.Version, "-")
	lastPart := parts[len(parts)-1]
	if len(lastPart) != 14 {
		t.Errorf("Timestamp suffix should be 14 digits, got %d: %s", len(lastPart), lastPart)
	}
}

// TestGetVersionInfoAtTag verifies that a commit exactly at a tag returns the bare tag.
func TestGetVersionInfoAtTag(t *testing.T) {
	dir, repo, w := initRepo(t)

	hash := writeAndCommit(t, dir, w, "file.txt", "v1", "Initial commit")
	lightweightTag(t, repo, "v1.0.0", hash)

	info, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}

	if info.Version != "v1.0.0" {
		t.Errorf("Version = %q, want %q", info.Version, "v1.0.0")
	}
	if info.LatestTag != "v1.0.0" {
		t.Errorf("LatestTag = %q, want %q", info.LatestTag, "v1.0.0")
	}
}

// TestGetVersionInfoAheadOfTag verifies the {tag}+dev.{N}.g{hash} format.
func TestGetVersionInfoAheadOfTag(t *testing.T) {
	dir, repo, w := initRepo(t)

	tagHash := writeAndCommit(t, dir, w, "file.txt", "v1", "Initial commit")
	lightweightTag(t, repo, "v1.0.0", tagHash)

	writeAndCommit(t, dir, w, "file.txt", "v2", "Second commit")
	writeAndCommit(t, dir, w, "file.txt", "v3", "Third commit")

	info, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}

	// Expect v1.0.0+dev.2.g<hash>
	if !strings.HasPrefix(info.Version, "v1.0.0+dev.2.g") {
		t.Errorf("Version = %q, want prefix %q", info.Version, "v1.0.0+dev.2.g")
	}
	if info.LatestTag != "v1.0.0" {
		t.Errorf("LatestTag = %q, want %q", info.LatestTag, "v1.0.0")
	}
}

// TestGetVersionInfoWithTagPrefix verifies that -prefix scopes tag lookup.
func TestGetVersionInfoWithTagPrefix(t *testing.T) {
	dir, repo, w := initRepo(t)

	// Tag for service-a and service-b
	tagHash := writeAndCommit(t, dir, w, "file.txt", "v1", "Initial commit")
	lightweightTag(t, repo, "service-a/v1.2.3", tagHash)
	lightweightTag(t, repo, "service-b/v9.0.0", tagHash)

	// One more commit ahead
	writeAndCommit(t, dir, w, "file.txt", "v2", "Another commit")

	t.Run("prefix preserved by default", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{TagPrefix: "service-a/v"})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		// prefix kept in output: service-a/v1.2.3+dev.1.g…
		if !strings.HasPrefix(info.Version, "service-a/v1.2.3+dev.1.g") {
			t.Errorf("Version = %q, want prefix %q", info.Version, "service-a/v1.2.3+dev.1.g")
		}
		// LatestTag is always the stripped semver portion
		if info.LatestTag != "1.2.3" {
			t.Errorf("LatestTag = %q, want %q", info.LatestTag, "1.2.3")
		}
	})

	t.Run("strip-prefix removes prefix from output", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{TagPrefix: "service-a/v", StripPrefix: true})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if !strings.HasPrefix(info.Version, "1.2.3+dev.1.g") {
			t.Errorf("Version = %q, want prefix %q", info.Version, "1.2.3+dev.1.g")
		}
		if info.LatestTag != "1.2.3" {
			t.Errorf("LatestTag = %q, want %q", info.LatestTag, "1.2.3")
		}
	})

	t.Run("service-b prefix scopes independently", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{TagPrefix: "service-b/v", StripPrefix: true})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if !strings.HasPrefix(info.Version, "9.0.0+dev.1.g") {
			t.Errorf("Version = %q, want prefix %q", info.Version, "9.0.0+dev.1.g")
		}
	})

	t.Run("non-matching prefix falls back to branch-slug format", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{TagPrefix: "no-such-service/v"})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		// No matching tags → branch-slug-ghash
		if strings.Contains(info.Version, "+dev.") {
			t.Errorf("Version = %q should not contain '+dev.' when no tags match", info.Version)
		}
	})
}

// TestGetVersionInfoWithPathFilter verifies that -path only counts commits
// that touched files under the specified directory.
func TestGetVersionInfoWithPathFilter(t *testing.T) {
	dir, repo, w := initRepo(t)

	// Initial tagged commit (touches both services)
	tagHash := writeAndCommit(t, dir, w, "service-a/main.go", "v1", "Initial service-a")
	lightweightTag(t, repo, "v1.0.0", tagHash)

	// Commit 1: only service-a
	writeAndCommit(t, dir, w, "service-a/main.go", "v2", "Update service-a")
	// Commit 2: only service-b
	writeAndCommit(t, dir, w, "service-b/main.go", "v1", "Add service-b")
	// Commit 3: only service-a
	writeAndCommit(t, dir, w, "service-a/main.go", "v3", "Update service-a again")

	// Without path filter: 3 commits ahead
	t.Run("no filter counts all commits", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if !strings.HasPrefix(info.Version, "v1.0.0+dev.3.g") {
			t.Errorf("Version = %q, want prefix v1.0.0+dev.3.g", info.Version)
		}
	})

	// With -path service-a/: 2 commits touch service-a/ (commits 1 and 3)
	t.Run("filter service-a counts 2", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{FilterPath: "service-a/"})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if !strings.HasPrefix(info.Version, "v1.0.0+dev.2.g") {
			t.Errorf("Version = %q, want prefix v1.0.0+dev.2.g", info.Version)
		}
	})

	// With -path service-b/: 1 commit touches service-b/ (commit 2)
	t.Run("filter service-b counts 1", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{FilterPath: "service-b/"})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if !strings.HasPrefix(info.Version, "v1.0.0+dev.1.g") {
			t.Errorf("Version = %q, want prefix v1.0.0+dev.1.g", info.Version)
		}
	})

	// With -path service-c/: 0 commits touch service-c/ → treated as clean (bare tag)
	t.Run("filter service-c reports clean (distance 0)", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{FilterPath: "service-c/"})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if info.Version != "v1.0.0" {
			t.Errorf("Version = %q, want %q", info.Version, "v1.0.0")
		}
	})
}

// TestGetVersionInfoWithPrefixAndPath combines both flags — the canonical monorepo scenario.
func TestGetVersionInfoWithPrefixAndPath(t *testing.T) {
	dir, repo, w := initRepo(t)

	tagHash := writeAndCommit(t, dir, w, "service-a/main.go", "v1", "Initial commit")
	lightweightTag(t, repo, "service-a/v1.2.3", tagHash)

	// 2 commits touch service-a/, 1 touches service-b/ only
	writeAndCommit(t, dir, w, "service-a/main.go", "v2", "Update service-a")
	writeAndCommit(t, dir, w, "service-b/main.go", "sb", "Add service-b")
	writeAndCommit(t, dir, w, "service-a/main.go", "v3", "Update service-a again")

	t.Run("prefix preserved by default", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{
			TagPrefix:  "service-a/v",
			FilterPath: "service-a/",
		})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if !strings.HasPrefix(info.Version, "service-a/v1.2.3+dev.2.g") {
			t.Errorf("Version = %q, want prefix %q", info.Version, "service-a/v1.2.3+dev.2.g")
		}
		if info.LatestTag != "1.2.3" {
			t.Errorf("LatestTag = %q, want %q", info.LatestTag, "1.2.3")
		}
	})

	t.Run("strip-prefix", func(t *testing.T) {
		info, err := GetVersionInfo(dir, Options{
			TagPrefix:   "service-a/v",
			FilterPath:  "service-a/",
			StripPrefix: true,
		})
		if err != nil {
			t.Fatalf("GetVersionInfo: %v", err)
		}
		if !strings.HasPrefix(info.Version, "1.2.3+dev.2.g") {
			t.Errorf("Version = %q, want prefix %q", info.Version, "1.2.3+dev.2.g")
		}
		shortHash := fmt.Sprintf("g%s", info.GitCommitShort)
		if !strings.Contains(info.Version, shortHash) {
			t.Errorf("Version %q should contain short hash %q", info.Version, shortHash)
		}
	})
}

// TestBumpSemver covers the pure bump logic in isolation.
func TestBumpSemver(t *testing.T) {
	tests := []struct {
		tag    string
		target string
		want   string
		errMsg string
	}{
		// patch bumps
		{"v1.2.3", "patch", "v1.2.4", ""},
		{"v1.2.0", "patch", "v1.2.1", ""},
		{"1.2.3", "patch", "1.2.4", ""},   // no v prefix preserved
		{"v0.0.0", "patch", "v0.0.1", ""},
		// minor bumps — patch resets to 0
		{"v1.2.3", "minor", "v1.3.0", ""},
		{"1.0.9", "minor", "1.1.0", ""},
		// major bumps — minor and patch reset to 0
		{"v1.2.3", "major", "v2.0.0", ""},
		{"v0.9.9", "major", "v1.0.0", ""},
		// no tags → base is 0.0.0, no v prefix
		{"", "patch", "0.0.1", ""},
		{"", "minor", "0.1.0", ""},
		{"", "major", "1.0.0", ""},
		// pre-release suffix in patch component is stripped before bumping
		{"v1.0.0-rc.1", "patch", "v1.0.1", ""},
		// invalid target
		{"v1.0.0", "bogus", "", "-target"},
		// non-semver tag
		{"not-semver", "patch", "", "not a valid semver"},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s/%s", tt.tag, tt.target)
		t.Run(name, func(t *testing.T) {
			got, err := bumpSemver(tt.tag, tt.target)
			if tt.errMsg != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errMsg)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bumpSemver(%q, %q) = %q, want %q", tt.tag, tt.target, got, tt.want)
			}
		})
	}
}

// TestGetVersionInfoWithTarget verifies the -target flag end-to-end.
func TestGetVersionInfoWithTarget(t *testing.T) {
	t.Run("patch at tag", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.2.3", hash)

		info, err := GetVersionInfo(dir, Options{Target: "patch"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "v1.2.4" {
			t.Errorf("Version = %q, want v1.2.4", info.Version)
		}
	})

	t.Run("minor at tag", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.2.3", hash)

		info, err := GetVersionInfo(dir, Options{Target: "minor"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "v1.3.0" {
			t.Errorf("Version = %q, want v1.3.0", info.Version)
		}
	})

	t.Run("major at tag", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.2.3", hash)

		info, err := GetVersionInfo(dir, Options{Target: "major"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "v2.0.0" {
			t.Errorf("Version = %q, want v2.0.0", info.Version)
		}
	})

	t.Run("patch ahead of tag — same result as at tag", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.2.3", hash)
		writeAndCommit(t, dir, w, "f.txt", "2", "more work")

		info, err := GetVersionInfo(dir, Options{Target: "patch"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// target always bumps the last tag, regardless of distance
		if info.Version != "v1.2.4" {
			t.Errorf("Version = %q, want v1.2.4", info.Version)
		}
	})

	t.Run("no tags — base is 0.0.0", func(t *testing.T) {
		dir, _, w := initRepo(t)
		writeAndCommit(t, dir, w, "f.txt", "1", "init")

		info, err := GetVersionInfo(dir, Options{Target: "minor"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "0.1.0" {
			t.Errorf("Version = %q, want 0.1.0", info.Version)
		}
	})

	t.Run("dirty tree — no timestamp suffix when target is set", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.0.0", hash)
		// make the tree dirty
		if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("dirty"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		info, err := GetVersionInfo(dir, Options{Target: "patch"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "v1.0.1" {
			t.Errorf("Version = %q, want v1.0.1 (no dirty suffix)", info.Version)
		}
	})

	t.Run("with prefix strip-prefix", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "svc/f.go", "1", "init")
		lightweightTag(t, repo, "svc/v2.3.4", hash)
		writeAndCommit(t, dir, w, "svc/f.go", "2", "more work")

		info, err := GetVersionInfo(dir, Options{TagPrefix: "svc/v", StripPrefix: true, Target: "minor"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "2.4.0" {
			t.Errorf("Version = %q, want 2.4.0", info.Version)
		}
	})

	t.Run("with prefix no strip — prefix preserved in bumped version", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "svc/f.go", "1", "init")
		lightweightTag(t, repo, "svc/v2.3.4", hash)
		writeAndCommit(t, dir, w, "svc/f.go", "2", "more work")

		info, err := GetVersionInfo(dir, Options{TagPrefix: "svc/v", Target: "minor"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "svc/v2.4.0" {
			t.Errorf("Version = %q, want svc/v2.4.0", info.Version)
		}
	})

	t.Run("dev target is a no-op identical to omitting target", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.0.0", hash)
		writeAndCommit(t, dir, w, "f.txt", "2", "more work")

		infoNoTarget, err := GetVersionInfo(dir, Options{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		infoDev, err := GetVersionInfo(dir, Options{Target: "dev"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if infoDev.Version != infoNoTarget.Version {
			t.Errorf("dev Version = %q, want %q (same as no target)", infoDev.Version, infoNoTarget.Version)
		}
	})

	t.Run("invalid target returns error", func(t *testing.T) {
		dir, _, w := initRepo(t)
		writeAndCommit(t, dir, w, "f.txt", "1", "init")

		_, err := GetVersionInfo(dir, Options{Target: "huge"})
		if err == nil {
			t.Fatal("expected error for invalid target, got nil")
		}
	})
}

// TestGetVersionInfoExactTagOnNonDefaultBranch verifies that a commit that is
// exactly at a tag produces the tag version regardless of which branch is active.
func TestGetVersionInfoExactTagOnNonDefaultBranch(t *testing.T) {
	dir, repo, w := initRepo(t)

	hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
	lightweightTag(t, repo, "v2.5.0", hash)

	// Create and switch to a feature branch at the same commit.
	branchName := "feature/release-prep"
	ref := plumbing.NewHashReference(
		plumbing.ReferenceName("refs/heads/"+branchName), hash)
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("SetReference: %v", err)
	}
	if err := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branchName),
	}); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	info, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}

	// Even on a non-default branch, exact tag → use the tag directly.
	if info.Version != "v2.5.0" {
		t.Errorf("Version = %q, want v2.5.0", info.Version)
	}
}

// TestGetVersionInfoExactTagNonDefaultBranchAhead verifies that commits AHEAD of
// a tag on a non-default branch still use the branch-slug format (not the tag).
func TestGetVersionInfoExactTagNonDefaultBranchAhead(t *testing.T) {
	dir, repo, w := initRepo(t)

	tagHash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
	lightweightTag(t, repo, "v1.0.0", tagHash)

	// Create feature branch, add a commit ahead of the tag.
	branchName := "feature/wip"
	ref := plumbing.NewHashReference(
		plumbing.ReferenceName("refs/heads/"+branchName), tagHash)
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("SetReference: %v", err)
	}
	if err := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branchName),
	}); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	writeAndCommit(t, dir, w, "f.txt", "2", "wip commit")

	info, err := GetVersionInfo(dir, Options{})
	if err != nil {
		t.Fatalf("GetVersionInfo: %v", err)
	}

	// Ahead of tag on non-default branch → branch-slug format.
	expectedSlug := "feature-wip"
	if !strings.HasPrefix(info.Version, expectedSlug+"-g") {
		t.Errorf("Version = %q, want prefix %q", info.Version, expectedSlug+"-g")
	}
}

// TestGetVersionInfoWithLatest covers the -latest flag.
func TestGetVersionInfoWithLatest(t *testing.T) {
	t.Run("at tag returns bare tag", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v3.1.0", hash)

		info, err := GetVersionInfo(dir, Options{Latest: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "v3.1.0" {
			t.Errorf("Version = %q, want v3.1.0", info.Version)
		}
	})

	t.Run("ahead of tag returns nearest tag not the dev version", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.0.0", hash)
		writeAndCommit(t, dir, w, "f.txt", "2", "more work")
		writeAndCommit(t, dir, w, "f.txt", "3", "even more")

		info, err := GetVersionInfo(dir, Options{Latest: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "v1.0.0" {
			t.Errorf("Version = %q, want v1.0.0 (tag, not dev version)", info.Version)
		}
	})

	t.Run("no tags returns error", func(t *testing.T) {
		dir, _, w := initRepo(t)
		writeAndCommit(t, dir, w, "f.txt", "1", "init")

		_, err := GetVersionInfo(dir, Options{Latest: true})
		if err == nil {
			t.Fatal("expected error when no tags, got nil")
		}
	})

	t.Run("with prefix — prefix preserved by default", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "svc/f.go", "1", "init")
		lightweightTag(t, repo, "svc/v1.5.0", hash)
		writeAndCommit(t, dir, w, "svc/f.go", "2", "more")

		info, err := GetVersionInfo(dir, Options{TagPrefix: "svc/v", Latest: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "svc/v1.5.0" {
			t.Errorf("Version = %q, want svc/v1.5.0", info.Version)
		}
	})

	t.Run("with prefix and strip-prefix", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "svc/f.go", "1", "init")
		lightweightTag(t, repo, "svc/v1.5.0", hash)
		writeAndCommit(t, dir, w, "svc/f.go", "2", "more")

		info, err := GetVersionInfo(dir, Options{TagPrefix: "svc/v", StripPrefix: true, Latest: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "1.5.0" {
			t.Errorf("Version = %q, want 1.5.0", info.Version)
		}
	})

	t.Run("latest and target are mutually exclusive", func(t *testing.T) {
		dir, _, w := initRepo(t)
		writeAndCommit(t, dir, w, "f.txt", "1", "init")

		_, err := GetVersionInfo(dir, Options{Latest: true, Target: "patch"})
		if err == nil {
			t.Fatal("expected error for -latest + -target, got nil")
		}
	})

	t.Run("latest with dev target is allowed (dev is a no-op)", func(t *testing.T) {
		dir, repo, w := initRepo(t)
		hash := writeAndCommit(t, dir, w, "f.txt", "1", "init")
		lightweightTag(t, repo, "v1.0.0", hash)

		info, err := GetVersionInfo(dir, Options{Latest: true, Target: "dev"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "v1.0.0" {
			t.Errorf("Version = %q, want v1.0.0", info.Version)
		}
	})
}
