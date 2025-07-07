package git

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"superagent/internal/config"
	"superagent/internal/logging"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/sirupsen/logrus"
	gossh "golang.org/x/crypto/ssh"
)

// GitManager manages Git repository operations
type GitManager struct {
	config      *config.Config
	auditLogger *logging.AuditLogger
	cache       map[string]*RepositoryInfo
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// RepositoryInfo holds information about a repository
type RepositoryInfo struct {
	URL         string                 `json:"url"`
	Path        string                 `json:"path"`
	Branch      string                 `json:"branch"`
	CommitHash  string                 `json:"commit_hash"`
	LastPull    time.Time              `json:"last_pull"`
	Size        int64                  `json:"size"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CloneOptions represents options for cloning a repository
type CloneOptions struct {
	URL         string            `json:"url"`
	Branch      string            `json:"branch"`
	Tag         string            `json:"tag"`
	CommitHash  string            `json:"commit_hash"`
	Depth       int               `json:"depth"`
	Auth        AuthConfig        `json:"auth"`
	Environment map[string]string `json:"environment"`
	Timeout     time.Duration     `json:"timeout"`
	Force       bool              `json:"force"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type       string `json:"type"`        // ssh, https, token
	Username   string `json:"username"`
	Password   string `json:"password"`
	Token      string `json:"token"`
	SSHKeyPath string `json:"ssh_key_path"`
	SSHKeyData string `json:"ssh_key_data"`
}

// BuildSpec represents a build specification
type BuildSpec struct {
	Commands    []string          `json:"commands"`
	Environment map[string]string `json:"environment"`
	WorkingDir  string            `json:"working_dir"`
	Timeout     time.Duration     `json:"timeout"`
	BuildArgs   map[string]string `json:"build_args"`
}

// NewGitManager creates a new Git manager
func NewGitManager(cfg *config.Config, auditLogger *logging.AuditLogger) (*GitManager, error) {
	// Create cache directory
	if err := os.MkdirAll(cfg.Git.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	gm := &GitManager{
		config:      cfg,
		auditLogger: auditLogger,
		cache:       make(map[string]*RepositoryInfo),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start background tasks
	gm.wg.Add(1)
	go gm.cleanupCache()

	return gm, nil
}

// CloneRepository clones a repository
func (gm *GitManager) CloneRepository(ctx context.Context, opts *CloneOptions) (*RepositoryInfo, error) {
	logrus.Infof("Cloning repository: %s", opts.URL)

	// Generate cache key
	cacheKey := gm.generateCacheKey(opts)
	repoPath := filepath.Join(gm.config.Git.CacheDir, cacheKey)

	// Check if repository already exists
	if gm.repositoryExists(repoPath) {
		if !opts.Force {
			// Try to pull latest changes
			if err := gm.pullRepository(ctx, repoPath, opts); err != nil {
				logrus.Warnf("Failed to pull repository, will re-clone: %v", err)
				// Remove and re-clone
				os.RemoveAll(repoPath)
			} else {
				return gm.getRepositoryInfo(repoPath, opts)
			}
		} else {
			// Force re-clone
			os.RemoveAll(repoPath)
		}
	}

	// Set up authentication
	auth, err := gm.setupAuth(opts.Auth)
	if err != nil {
		gm.auditLogger.LogError("GIT_AUTH_SETUP_FAILED", err, map[string]interface{}{
			"url": opts.URL,
		})
		return nil, fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Create clone options
	cloneOpts := &git.CloneOptions{
		URL:           opts.URL,
		Auth:          auth,
		SingleBranch:  true,
		Depth:         opts.Depth,
		ReferenceName: plumbing.HEAD,
		Progress:      os.Stdout,
	}

	// Set branch or tag
	if opts.Branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
	} else if opts.Tag != "" {
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(opts.Tag)
	}

	// Apply timeout
	cloneCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		cloneCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Clone repository
	repo, err := git.PlainCloneContext(cloneCtx, repoPath, false, cloneOpts)
	if err != nil {
		gm.auditLogger.LogError("GIT_CLONE_FAILED", err, map[string]interface{}{
			"url":    opts.URL,
			"branch": opts.Branch,
			"tag":    opts.Tag,
		})
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Checkout specific commit if specified
	if opts.CommitHash != "" {
		if err := gm.checkoutCommit(repo, opts.CommitHash); err != nil {
			gm.auditLogger.LogError("GIT_CHECKOUT_FAILED", err, map[string]interface{}{
				"url":    opts.URL,
				"commit": opts.CommitHash,
			})
			return nil, fmt.Errorf("failed to checkout commit: %w", err)
		}
	}

	// Get repository info
	repoInfo, err := gm.getRepositoryInfo(repoPath, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Cache repository info
	gm.mu.Lock()
	gm.cache[cacheKey] = repoInfo
	gm.mu.Unlock()

	gm.auditLogger.LogEvent("GIT_CLONE_SUCCESS", map[string]interface{}{
		"url":    opts.URL,
		"branch": opts.Branch,
		"tag":    opts.Tag,
		"path":   repoPath,
	})

	logrus.Infof("Repository cloned successfully: %s", opts.URL)
	return repoInfo, nil
}

// PullRepository pulls latest changes from a repository
func (gm *GitManager) PullRepository(ctx context.Context, repoPath string, opts *CloneOptions) error {
	return gm.pullRepository(ctx, repoPath, opts)
}

// BuildRepository builds a repository using the specified build spec
func (gm *GitManager) BuildRepository(ctx context.Context, repoPath string, buildSpec *BuildSpec) error {
	logrus.Infof("Building repository: %s", repoPath)

	// Validate build spec
	if len(buildSpec.Commands) == 0 {
		return fmt.Errorf("no build commands specified")
	}

	// Set working directory
	workDir := repoPath
	if buildSpec.WorkingDir != "" {
		workDir = filepath.Join(repoPath, buildSpec.WorkingDir)
	}

	// Apply timeout
	buildCtx := ctx
	if buildSpec.Timeout > 0 {
		var cancel context.CancelFunc
		buildCtx, cancel = context.WithTimeout(ctx, buildSpec.Timeout)
		defer cancel()
	}

	// Execute build commands
	for i, command := range buildSpec.Commands {
		logrus.Infof("Executing build command %d: %s", i+1, command)

		if err := gm.executeCommand(buildCtx, command, workDir, buildSpec.Environment); err != nil {
			gm.auditLogger.LogError("GIT_BUILD_COMMAND_FAILED", err, map[string]interface{}{
				"command": command,
				"step":    i + 1,
				"repo":    repoPath,
			})
			return fmt.Errorf("build command failed: %w", err)
		}
	}

	gm.auditLogger.LogEvent("GIT_BUILD_SUCCESS", map[string]interface{}{
		"repo":     repoPath,
		"commands": len(buildSpec.Commands),
	})

	logrus.Infof("Repository built successfully: %s", repoPath)
	return nil
}

// GetRepositoryInfo returns information about a repository
func (gm *GitManager) GetRepositoryInfo(repoPath string) (*RepositoryInfo, error) {
	return gm.getRepositoryInfo(repoPath, nil)
}

// ListRepositories lists all cached repositories
func (gm *GitManager) ListRepositories() []*RepositoryInfo {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	repos := make([]*RepositoryInfo, 0, len(gm.cache))
	for _, info := range gm.cache {
		repos = append(repos, info)
	}

	return repos
}

// DeleteRepository deletes a repository from cache
func (gm *GitManager) DeleteRepository(repoPath string) error {
	// Remove from filesystem
	if err := os.RemoveAll(repoPath); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	// Remove from cache
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for key, info := range gm.cache {
		if info.Path == repoPath {
			delete(gm.cache, key)
			break
		}
	}

	gm.auditLogger.LogEvent("GIT_REPOSITORY_DELETED", map[string]interface{}{
		"path": repoPath,
	})

	return nil
}

// generateCacheKey generates a cache key for a repository
func (gm *GitManager) generateCacheKey(opts *CloneOptions) string {
	key := fmt.Sprintf("%s_%s_%s_%s", 
		sanitizeForPath(opts.URL), 
		sanitizeForPath(opts.Branch), 
		sanitizeForPath(opts.Tag), 
		sanitizeForPath(opts.CommitHash))
	
	if len(key) > 200 {
		// Use hash for very long keys
		return fmt.Sprintf("repo_%x", sha256.Sum256([]byte(key)))
	}
	
	return key
}

// repositoryExists checks if a repository exists
func (gm *GitManager) repositoryExists(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// pullRepository pulls latest changes from a repository
func (gm *GitManager) pullRepository(ctx context.Context, repoPath string, opts *CloneOptions) error {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get working tree
	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get working tree: %w", err)
	}

	// Set up authentication
	auth, err := gm.setupAuth(opts.Auth)
	if err != nil {
		return fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Pull options
	pullOpts := &git.PullOptions{
		Auth:     auth,
		Progress: os.Stdout,
	}

	// Apply timeout
	pullCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		pullCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Pull changes
	err = workTree.PullContext(pullCtx, pullOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	return nil
}

// checkoutCommit checks out a specific commit
func (gm *GitManager) checkoutCommit(repo *git.Repository, commitHash string) error {
	// Get working tree
	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get working tree: %w", err)
	}

	// Parse commit hash
	hash := plumbing.NewHash(commitHash)

	// Checkout commit
	err = workTree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout commit: %w", err)
	}

	return nil
}

// setupAuth sets up authentication for Git operations
func (gm *GitManager) setupAuth(authConfig AuthConfig) (transport.AuthMethod, error) {
	switch authConfig.Type {
	case "ssh":
		return gm.setupSSHAuth(authConfig)
	case "https", "token":
		return gm.setupHTTPAuth(authConfig)
	default:
		// Try default authentication methods
		if authConfig.SSHKeyPath != "" || authConfig.SSHKeyData != "" {
			return gm.setupSSHAuth(authConfig)
		}
		if authConfig.Username != "" || authConfig.Token != "" {
			return gm.setupHTTPAuth(authConfig)
		}
		return nil, nil
	}
}

// setupSSHAuth sets up SSH authentication
func (gm *GitManager) setupSSHAuth(authConfig AuthConfig) (transport.AuthMethod, error) {
	var sshKey []byte
	var err error

	if authConfig.SSHKeyData != "" {
		sshKey = []byte(authConfig.SSHKeyData)
	} else if authConfig.SSHKeyPath != "" {
		sshKey, err = os.ReadFile(authConfig.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %w", err)
		}
	} else if gm.config.Git.SSHKeyPath != "" {
		sshKey, err = os.ReadFile(gm.config.Git.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no SSH key provided")
	}

	// Parse SSH key
	var signer gossh.Signer
	if gm.config.Git.SSHKeyPassphrase != "" {
		signer, err = gossh.ParsePrivateKeyWithPassphrase(sshKey, []byte(gm.config.Git.SSHKeyPassphrase))
	} else {
		signer, err = gossh.ParsePrivateKey(sshKey)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// Create SSH auth method
	return &ssh.PublicKeys{
		User:   "git",
		Signer: signer,
		HostKeyCallbackHelper: ssh.HostKeyCallbackHelper{
			HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		},
	}, nil
}

// setupHTTPAuth sets up HTTP authentication
func (gm *GitManager) setupHTTPAuth(authConfig AuthConfig) (transport.AuthMethod, error) {
	if authConfig.Token != "" {
		return &http.BasicAuth{
			Username: "token",
			Password: authConfig.Token,
		}, nil
	}

	if authConfig.Username != "" {
		password := authConfig.Password
		if password == "" {
			password = gm.config.Git.Password
		}
		return &http.BasicAuth{
			Username: authConfig.Username,
			Password: password,
		}, nil
	}

	if gm.config.Git.Token != "" {
		return &http.BasicAuth{
			Username: "token",
			Password: gm.config.Git.Token,
		}, nil
	}

	if gm.config.Git.Username != "" {
		return &http.BasicAuth{
			Username: gm.config.Git.Username,
			Password: gm.config.Git.Password,
		}, nil
	}

	return nil, nil
}

// getRepositoryInfo gets information about a repository
func (gm *GitManager) getRepositoryInfo(repoPath string, opts *CloneOptions) (*RepositoryInfo, error) {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get HEAD commit
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

// Get commit (not used but needed for validation)
	_, err = repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get repository size
	size, err := gm.getRepositorySize(repoPath)
	if err != nil {
		logrus.Warnf("Failed to get repository size: %v", err)
	}

	info := &RepositoryInfo{
		Path:       repoPath,
		CommitHash: head.Hash().String(),
		LastPull:   time.Now(),
		Size:       size,
		Status:     "ready",
	}

	if opts != nil {
		info.URL = opts.URL
		info.Branch = opts.Branch
	}

	return info, nil
}

// getRepositorySize calculates the size of a repository
func (gm *GitManager) getRepositorySize(repoPath string) (int64, error) {
	var size int64
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
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

// executeCommand executes a shell command
func (gm *GitManager) executeCommand(ctx context.Context, command, workDir string, env map[string]string) error {
	// Create command
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	for key, value := range gm.config.Agent.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, string(output))
	}

	logrus.Debugf("Command output: %s", string(output))
	return nil
}

// cleanupCache cleans up old repository cache entries
func (gm *GitManager) cleanupCache() {
	defer gm.wg.Done()

	ticker := time.NewTicker(gm.config.Git.CacheRetention)
	defer ticker.Stop()

	for {
		select {
		case <-gm.ctx.Done():
			return
		case <-ticker.C:
			gm.performCacheCleanup()
		}
	}
}

// performCacheCleanup performs cache cleanup
func (gm *GitManager) performCacheCleanup() {
	cutoff := time.Now().Add(-gm.config.Git.CacheRetention)

	gm.mu.Lock()
	defer gm.mu.Unlock()

	for key, info := range gm.cache {
		if info.LastPull.Before(cutoff) {
			// Remove from filesystem
			if err := os.RemoveAll(info.Path); err != nil {
				logrus.Errorf("Failed to remove cached repository: %v", err)
				continue
			}

			// Remove from cache
			delete(gm.cache, key)
			logrus.Infof("Cleaned up cached repository: %s", info.Path)
		}
	}
}

// Close closes the Git manager
func (gm *GitManager) Close() error {
	gm.cancel()
	gm.wg.Wait()
	return nil
}

// sanitizeForPath sanitizes a string for use in file paths
func sanitizeForPath(input string) string {
	// Replace invalid characters with underscores
	result := strings.ReplaceAll(input, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	result = strings.ReplaceAll(result, ":", "_")
	result = strings.ReplaceAll(result, "*", "_")
	result = strings.ReplaceAll(result, "?", "_")
	result = strings.ReplaceAll(result, "\"", "_")
	result = strings.ReplaceAll(result, "<", "_")
	result = strings.ReplaceAll(result, ">", "_")
	result = strings.ReplaceAll(result, "|", "_")
	return result
}