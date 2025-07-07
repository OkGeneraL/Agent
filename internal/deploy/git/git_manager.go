package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
)

// GitManager handles Git repository operations
type GitManager struct {
	auditLogger *logging.AuditLogger
	workDir     string
	mu          sync.RWMutex
}

// CloneOptions contains options for cloning a repository
type CloneOptions struct {
	URL        string
	Branch     string
	Tag        string
	Commit     string
	Depth      int
	Recursive  bool
	Auth       map[string]string
	SSHKeyPath string
	Username   string
	Password   string
}

// RepositoryInfo contains information about a cloned repository
type RepositoryInfo struct {
	Path      string
	URL       string
	Branch    string
	Commit    string
	Tag       string
	ClonedAt  time.Time
	Size      int64
}

// NewGitManager creates a new Git manager
func NewGitManager(auditLogger *logging.AuditLogger) (*GitManager, error) {
	workDir := "/tmp/superagent-repos"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	gm := &GitManager{
		auditLogger: auditLogger,
		workDir:     workDir,
	}

	return gm, nil
}

// CloneRepository clones a Git repository
func (gm *GitManager) CloneRepository(ctx context.Context, url, branch string, auth map[string]string) (string, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Create unique directory for this clone
	repoName := extractRepoName(url)
	timestamp := time.Now().Format("20060102-150405")
	clonePath := filepath.Join(gm.workDir, fmt.Sprintf("%s-%s", repoName, timestamp))

	logrus.Infof("Cloning repository %s to %s", url, clonePath)

	// Prepare clone command
	cmd := exec.CommandContext(ctx, "git", "clone")
	
	// Add branch if specified
	if branch != "" {
		cmd.Args = append(cmd.Args, "--branch", branch)
	}
	
	// Add depth for shallow clone
	cmd.Args = append(cmd.Args, "--depth", "1")
	
	// Add recursive for submodules
	cmd.Args = append(cmd.Args, "--recursive")
	
	// Add URL and destination
	cmd.Args = append(cmd.Args, url, clonePath)

	// Set up authentication
	if err := gm.setupAuth(cmd, auth); err != nil {
		return "", fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Execute clone command
	output, err := cmd.CombinedOutput()
	if err != nil {
		gm.auditLogger.LogEvent("GIT_CLONE_FAILED", map[string]interface{}{
			"url":    url,
			"branch": branch,
			"error":  err.Error(),
			"output": string(output),
		})
		return "", fmt.Errorf("failed to clone repository: %w, output: %s", err, output)
	}

	// Verify clone was successful
	if _, err := os.Stat(filepath.Join(clonePath, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("clone failed: .git directory not found")
	}

	// Get repository info
	info, err := gm.getRepositoryInfo(clonePath)
	if err != nil {
		logrus.Warnf("Failed to get repository info: %v", err)
	}

	gm.auditLogger.LogEvent("GIT_CLONE_SUCCESS", map[string]interface{}{
		"url":       url,
		"branch":    branch,
		"path":      clonePath,
		"commit":    info.Commit,
		"size":      info.Size,
	})

	return clonePath, nil
}

// CloneRepositoryWithOptions clones a repository with detailed options
func (gm *GitManager) CloneRepositoryWithOptions(ctx context.Context, options CloneOptions) (string, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Create unique directory for this clone
	repoName := extractRepoName(options.URL)
	timestamp := time.Now().Format("20060102-150405")
	clonePath := filepath.Join(gm.workDir, fmt.Sprintf("%s-%s", repoName, timestamp))

	logrus.Infof("Cloning repository %s to %s with options", options.URL, clonePath)

	// Prepare clone command
	cmd := exec.CommandContext(ctx, "git", "clone")
	
	// Add branch or tag
	if options.Branch != "" {
		cmd.Args = append(cmd.Args, "--branch", options.Branch)
	} else if options.Tag != "" {
		cmd.Args = append(cmd.Args, "--branch", options.Tag)
	}
	
	// Add depth for shallow clone
	if options.Depth > 0 {
		cmd.Args = append(cmd.Args, "--depth", fmt.Sprintf("%d", options.Depth))
	}
	
	// Add recursive for submodules
	if options.Recursive {
		cmd.Args = append(cmd.Args, "--recursive")
	}
	
	// Add URL and destination
	cmd.Args = append(cmd.Args, options.URL, clonePath)

	// Set up authentication
	if err := gm.setupAuthWithOptions(cmd, options); err != nil {
		return "", fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Execute clone command
	output, err := cmd.CombinedOutput()
	if err != nil {
		gm.auditLogger.LogEvent("GIT_CLONE_FAILED", map[string]interface{}{
			"url":    options.URL,
			"branch": options.Branch,
			"tag":    options.Tag,
			"error":  err.Error(),
			"output": string(output),
		})
		return "", fmt.Errorf("failed to clone repository: %w, output: %s", err, output)
	}

	// Checkout specific commit if specified
	if options.Commit != "" {
		if err := gm.checkoutCommit(ctx, clonePath, options.Commit); err != nil {
			return "", fmt.Errorf("failed to checkout commit: %w", err)
		}
	}

	// Verify clone was successful
	if _, err := os.Stat(filepath.Join(clonePath, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("clone failed: .git directory not found")
	}

	// Get repository info
	info, err := gm.getRepositoryInfo(clonePath)
	if err != nil {
		logrus.Warnf("Failed to get repository info: %v", err)
	}

	gm.auditLogger.LogEvent("GIT_CLONE_SUCCESS", map[string]interface{}{
		"url":       options.URL,
		"branch":    options.Branch,
		"tag":       options.Tag,
		"commit":    options.Commit,
		"path":      clonePath,
		"final_commit": info.Commit,
		"size":      info.Size,
	})

	return clonePath, nil
}

// GetRepositoryInfo returns information about a cloned repository
func (gm *GitManager) GetRepositoryInfo(repoPath string) (*RepositoryInfo, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return gm.getRepositoryInfo(repoPath)
}

// getRepositoryInfo gets repository information (internal method)
func (gm *GitManager) getRepositoryInfo(repoPath string) (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		Path:     repoPath,
		ClonedAt: time.Now(),
	}

	// Get remote URL
	cmd := exec.Command("git", "-C", repoPath, "config", "--get", "remote.origin.url")
	if output, err := cmd.Output(); err == nil {
		info.URL = strings.TrimSpace(string(output))
	}

	// Get current branch
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if output, err := cmd.Output(); err == nil {
		info.Branch = strings.TrimSpace(string(output))
	}

	// Get current commit
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	if output, err := cmd.Output(); err == nil {
		info.Commit = strings.TrimSpace(string(output))
	}

	// Get current tag (if any)
	cmd = exec.Command("git", "-C", repoPath, "describe", "--tags", "--exact-match", "HEAD")
	if output, err := cmd.Output(); err == nil {
		info.Tag = strings.TrimSpace(string(output))
	}

	// Get repository size
	if size, err := getDirSize(repoPath); err == nil {
		info.Size = size
	}

	return info, nil
}

// PullRepository pulls latest changes from remote
func (gm *GitManager) PullRepository(ctx context.Context, repoPath string, auth map[string]string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	logrus.Infof("Pulling latest changes for repository at %s", repoPath)

	// Prepare pull command
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "origin")

	// Set up authentication
	if err := gm.setupAuth(cmd, auth); err != nil {
		return fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Execute pull command
	output, err := cmd.CombinedOutput()
	if err != nil {
		gm.auditLogger.LogEvent("GIT_PULL_FAILED", map[string]interface{}{
			"path":   repoPath,
			"error":  err.Error(),
			"output": string(output),
		})
		return fmt.Errorf("failed to pull repository: %w, output: %s", err, output)
	}

	gm.auditLogger.LogEvent("GIT_PULL_SUCCESS", map[string]interface{}{
		"path":   repoPath,
		"output": string(output),
	})

	return nil
}

// CheckoutCommit checks out a specific commit
func (gm *GitManager) CheckoutCommit(ctx context.Context, repoPath, commit string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	return gm.checkoutCommit(ctx, repoPath, commit)
}

// checkoutCommit checks out a specific commit (internal method)
func (gm *GitManager) checkoutCommit(ctx context.Context, repoPath, commit string) error {
	logrus.Infof("Checking out commit %s in repository at %s", commit, repoPath)

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", commit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		gm.auditLogger.LogEvent("GIT_CHECKOUT_FAILED", map[string]interface{}{
			"path":   repoPath,
			"commit": commit,
			"error":  err.Error(),
			"output": string(output),
		})
		return fmt.Errorf("failed to checkout commit: %w, output: %s", err, output)
	}

	gm.auditLogger.LogEvent("GIT_CHECKOUT_SUCCESS", map[string]interface{}{
		"path":   repoPath,
		"commit": commit,
	})

	return nil
}

// CheckoutBranch checks out a specific branch
func (gm *GitManager) CheckoutBranch(ctx context.Context, repoPath, branch string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	logrus.Infof("Checking out branch %s in repository at %s", branch, repoPath)

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		gm.auditLogger.LogEvent("GIT_CHECKOUT_FAILED", map[string]interface{}{
			"path":   repoPath,
			"branch": branch,
			"error":  err.Error(),
			"output": string(output),
		})
		return fmt.Errorf("failed to checkout branch: %w, output: %s", err, output)
	}

	gm.auditLogger.LogEvent("GIT_CHECKOUT_SUCCESS", map[string]interface{}{
		"path":   repoPath,
		"branch": branch,
	})

	return nil
}

// GetCommitInfo returns information about a specific commit
func (gm *GitManager) GetCommitInfo(ctx context.Context, repoPath, commit string) (map[string]string, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	info := make(map[string]string)

	// Get commit hash
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", commit)
	if output, err := cmd.Output(); err == nil {
		info["hash"] = strings.TrimSpace(string(output))
	}

	// Get commit message
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-1", "--pretty=format:%s", commit)
	if output, err := cmd.Output(); err == nil {
		info["message"] = strings.TrimSpace(string(output))
	}

	// Get commit author
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-1", "--pretty=format:%an", commit)
	if output, err := cmd.Output(); err == nil {
		info["author"] = strings.TrimSpace(string(output))
	}

	// Get commit date
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-1", "--pretty=format:%cd", commit)
	if output, err := cmd.Output(); err == nil {
		info["date"] = strings.TrimSpace(string(output))
	}

	return info, nil
}

// ListBranches lists all branches in the repository
func (gm *GitManager) ListBranches(ctx context.Context, repoPath string) ([]string, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var branches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "*") {
			// Remove "remotes/origin/" prefix if present
			if strings.HasPrefix(line, "remotes/origin/") {
				line = strings.TrimPrefix(line, "remotes/origin/")
			}
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// ListTags lists all tags in the repository
func (gm *GitManager) ListTags(ctx context.Context, repoPath string) ([]string, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "tag", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var tags []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}

	return tags, nil
}

// CleanupRepository removes a cloned repository
func (gm *GitManager) CleanupRepository(repoPath string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	logrus.Infof("Cleaning up repository at %s", repoPath)

	if err := os.RemoveAll(repoPath); err != nil {
		gm.auditLogger.LogEvent("GIT_CLEANUP_FAILED", map[string]interface{}{
			"path":  repoPath,
			"error": err.Error(),
		})
		return fmt.Errorf("failed to cleanup repository: %w", err)
	}

	gm.auditLogger.LogEvent("GIT_CLEANUP_SUCCESS", map[string]interface{}{
		"path": repoPath,
	})

	return nil
}

// CleanupAll removes all cloned repositories
func (gm *GitManager) CleanupAll() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	logrus.Info("Cleaning up all repositories")

	entries, err := os.ReadDir(gm.workDir)
	if err != nil {
		return fmt.Errorf("failed to read work directory: %w", err)
	}

	var errors []string
	for _, entry := range entries {
		if entry.IsDir() {
			path := filepath.Join(gm.workDir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				errors = append(errors, fmt.Sprintf("failed to remove %s: %v", path, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}

	gm.auditLogger.LogEvent("GIT_CLEANUP_ALL_SUCCESS", map[string]interface{}{
		"work_dir": gm.workDir,
	})

	return nil
}

// setupAuth sets up authentication for git commands
func (gm *GitManager) setupAuth(cmd *exec.Cmd, auth map[string]string) error {
	if auth == nil {
		return nil
	}

	// Set up environment variables for authentication
	env := os.Environ()

	// Handle SSH key authentication
	if sshKey, exists := auth["ssh_key"]; exists {
		// Write SSH key to temporary file
		keyFile := filepath.Join(os.TempDir(), fmt.Sprintf("superagent-ssh-key-%d", time.Now().Unix()))
		if err := os.WriteFile(keyFile, []byte(sshKey), 0600); err != nil {
			return fmt.Errorf("failed to write SSH key: %w", err)
		}

		// Set up SSH command to use the key
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", keyFile)
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	}

	// Handle username/password authentication
	if username, exists := auth["username"]; exists {
		if password, exists := auth["password"]; exists {
			// For HTTPS URLs, we can use credential helper
			env = append(env, fmt.Sprintf("GIT_ASKPASS=echo"))
			env = append(env, fmt.Sprintf("GIT_USERNAME=%s", username))
			env = append(env, fmt.Sprintf("GIT_PASSWORD=%s", password))
		}
	}

	// Handle personal access token
	if token, exists := auth["token"]; exists {
		env = append(env, fmt.Sprintf("GIT_ASKPASS=echo"))
		env = append(env, fmt.Sprintf("GIT_USERNAME=token"))
		env = append(env, fmt.Sprintf("GIT_PASSWORD=%s", token))
	}

	cmd.Env = env
	return nil
}

// setupAuthWithOptions sets up authentication with detailed options
func (gm *GitManager) setupAuthWithOptions(cmd *exec.Cmd, options CloneOptions) error {
	env := os.Environ()

	// Handle SSH key authentication
	if options.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", options.SSHKeyPath)
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	}

	// Handle username/password authentication
	if options.Username != "" && options.Password != "" {
		env = append(env, fmt.Sprintf("GIT_ASKPASS=echo"))
		env = append(env, fmt.Sprintf("GIT_USERNAME=%s", options.Username))
		env = append(env, fmt.Sprintf("GIT_PASSWORD=%s", options.Password))
	}

	// Handle auth map
	if options.Auth != nil {
		if err := gm.setupAuth(cmd, options.Auth); err != nil {
			return err
		}
	}

	cmd.Env = env
	return nil
}

// Helper functions

// extractRepoName extracts repository name from URL
func extractRepoName(url string) string {
	// Remove .git suffix if present
	if strings.HasSuffix(url, ".git") {
		url = strings.TrimSuffix(url, ".git")
	}

	// Extract last part of URL
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown"
}

// getDirSize calculates directory size recursively
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// ValidateRepository checks if a path contains a valid Git repository
func (gm *GitManager) ValidateRepository(repoPath string) error {
	// Check if .git directory exists
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Try to run git status to verify it's a valid repository
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("invalid git repository: %w", err)
	}

	return nil
}

// GetWorkDir returns the working directory for repositories
func (gm *GitManager) GetWorkDir() string {
	return gm.workDir
}

// SetWorkDir sets the working directory for repositories
func (gm *GitManager) SetWorkDir(workDir string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	gm.workDir = workDir
	return nil
}