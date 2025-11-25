package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGetVersionInfoFromSubdirectory(t *testing.T) {
	// Create a temporary directory for test repository
	tempDir, err := os.MkdirTemp("", "gitversion-test-subdir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir1", "subdir2")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a test file in the root
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add and commit the file
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := w.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commit, err := w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test GetVersionInfo from subdirectory
	info, err := GetVersionInfo(subDir, "")
	if err != nil {
		t.Fatalf("GetVersionInfo failed from subdirectory: %v", err)
	}

	// Verify basic fields
	if info.GitCommit != commit.String() {
		t.Errorf("GitCommit = %q, want %q", info.GitCommit, commit.String())
	}

	if info.GitBranch != "master" && info.GitBranch != "main" {
		t.Errorf("GitBranch = %q, want 'master' or 'main'", info.GitBranch)
	}

	// Version should be {branch-slug}-g{short-commit} since we have no tags
	expectedVersion := info.GitBranchSlug + "-g" + info.GitCommitShort
	if info.Version != expectedVersion {
		t.Errorf("Version = %q, want %q", info.Version, expectedVersion)
	}
}

func TestCreateBranchSlug(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "simple branch",
			branch:   "main",
			expected: "main",
		},
		{
			name:     "feature branch with slash",
			branch:   "feature/new-feature",
			expected: "feature-new-feature",
		},
		{
			name:     "branch with underscore",
			branch:   "feature_branch",
			expected: "feature-branch",
		},
		{
			name:     "complex branch name",
			branch:   "feature/JIRA-123_update",
			expected: "feature-JIRA-123-update",
		},
		{
			name:     "branch with special characters",
			branch:   "feature/test@123",
			expected: "feature-test123",
		},
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
	// Create a temporary directory for test repository
	tempDir, err := os.MkdirTemp("", "gitversion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add and commit the file
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := w.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commit, err := w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test GetVersionInfo (auto-detect default branch)
	info, err := GetVersionInfo(tempDir, "")
	if err != nil {
		t.Fatalf("GetVersionInfo failed: %v", err)
	}

	// Verify basic fields
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

	// Version should be {branch-slug}-g{short-commit} since we have no tags
	expectedVersion := info.GitBranchSlug + "-g" + info.GitCommitShort
	if info.Version != expectedVersion {
		t.Errorf("Version = %q, want %q", info.Version, expectedVersion)
	}
}

func TestGetVersionInfoWithBranch(t *testing.T) {
	// Create a temporary directory for test repository
	tempDir, err := os.MkdirTemp("", "gitversion-test-branch-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := w.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if _, err := w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create and checkout a feature branch
	branchName := "feature/test-branch"
	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+branchName), headRef.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	if err := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branchName),
	}); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Test GetVersionInfo on feature branch
	info, err := GetVersionInfo(tempDir, "")
	if err != nil {
		t.Fatalf("GetVersionInfo failed: %v", err)
	}

	if info.GitBranch != branchName {
		t.Errorf("GitBranch = %q, want %q", info.GitBranch, branchName)
	}

	// Version should be {branch-slug}-{short-commit}
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

	// Check that all important fields are present
	if !strings.Contains(result, "v1.0.0") {
		t.Error("DetailedString() should contain version")
	}
	if !strings.Contains(result, "abc123def456") {
		t.Error("DetailedString() should contain full commit")
	}
	if !strings.Contains(result, "main") {
		t.Error("DetailedString() should contain branch")
	}
	if !strings.Contains(result, "Default Branch: main") {
		t.Error("DetailedString() should contain default branch")
	}
	if !strings.Contains(result, "Latest Tag:     v1.0.0") {
		t.Error("DetailedString() should contain latest tag")
	}
	if !strings.Contains(result, "2025-01-01T00:00:00Z") {
		t.Error("DetailedString() should contain build time")
	}
	if !strings.Contains(result, "clean") {
		t.Error("DetailedString() should contain dirty status")
	}
}

func TestGetVersionInfoWithUncommittedChanges(t *testing.T) {
	// Create a temporary directory for test repository
	tempDir, err := os.MkdirTemp("", "gitversion-test-dirty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init repository: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := w.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if _, err := w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test with clean working tree first
	info, err := GetVersionInfo(tempDir, "")
	if err != nil {
		t.Fatalf("GetVersionInfo failed: %v", err)
	}

	if info.IsDirty {
		t.Error("IsDirty should be false for clean working tree")
	}

	// Version should NOT have timestamp suffix for clean tree
	if strings.Contains(info.Version, "-202") {
		t.Errorf("Version should not have timestamp suffix for clean tree: %s", info.Version)
	}

	// Now modify an existing tracked file (not create a new untracked one)
	testFile = filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Test with dirty working tree
	infoDirty, err := GetVersionInfo(tempDir, "")
	if err != nil {
		t.Fatalf("GetVersionInfo failed: %v", err)
	}

	if !infoDirty.IsDirty {
		t.Error("IsDirty should be true for dirty working tree")
	}

	// Version should have timestamp suffix in format YYYYMMDDHHMMSS
	if !strings.Contains(infoDirty.Version, "-202") {
		t.Errorf("Version should have timestamp suffix for dirty tree: %s", infoDirty.Version)
	}

	// Verify timestamp format (should be 14 digits)
	parts := strings.Split(infoDirty.Version, "-")
	lastPart := parts[len(parts)-1]
	if len(lastPart) != 14 {
		t.Errorf("Timestamp suffix should be 14 digits, got %d: %s", len(lastPart), lastPart)
	}
}
