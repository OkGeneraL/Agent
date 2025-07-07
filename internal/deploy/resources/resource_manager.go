package resources

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
)

// ResourceManager handles resource management and monitoring
type ResourceManager struct {
	auditLogger *logging.AuditLogger
	mu          sync.RWMutex
}

// ResourceLimits defines resource constraints for containers
type ResourceLimits struct {
	CPULimit          float64 `json:"cpu_limit"`           // CPU cores (e.g., 1.5)
	MemoryLimit       int64   `json:"memory_limit"`        // Memory in bytes
	SwapLimit         int64   `json:"swap_limit"`          // Swap in bytes
	DiskLimit         int64   `json:"disk_limit"`          // Disk in bytes
	NetworkLimit      int64   `json:"network_limit"`       // Network bandwidth in bytes/sec
	ProcessLimit      int     `json:"process_limit"`       // Maximum number of processes
	FileLimit         int     `json:"file_limit"`          // Maximum number of open files
	CPUShares         int     `json:"cpu_shares"`          // CPU shares (relative weight)
	CPUPeriod         int     `json:"cpu_period"`          // CPU CFS period in microseconds
	CPUQuota          int     `json:"cpu_quota"`           // CPU CFS quota in microseconds
	CPUSetCPUs        string  `json:"cpu_set_cpus"`        // CPU affinity (e.g., "0-3")
	CPUSetMems        string  `json:"cpu_set_mems"`        // Memory node affinity
	BlkioWeight       int     `json:"blkio_weight"`        // Block IO weight (10-1000)
	MemorySwap        int64   `json:"memory_swap"`         // Memory + swap limit
	MemoryReservation int64   `json:"memory_reservation"`  // Memory soft limit
	KernelMemory      int64   `json:"kernel_memory"`       // Kernel memory limit
	OomKillDisable    bool    `json:"oom_kill_disable"`    // Disable OOM killer
	OomScoreAdj       int     `json:"oom_score_adj"`       // OOM score adjustment
	ShmSize           string  `json:"shm_size"`            // Shared memory size
	Ulimits           []string `json:"ulimits"`            // User limits
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	ContainerID    string    `json:"container_id"`
	CPUUsage       float64   `json:"cpu_usage"`        // Current CPU usage percentage
	MemoryUsage    int64     `json:"memory_usage"`     // Current memory usage in bytes
	MemoryLimit    int64     `json:"memory_limit"`     // Memory limit in bytes
	SwapUsage      int64     `json:"swap_usage"`       // Current swap usage in bytes
	NetworkRx      int64     `json:"network_rx"`       // Network bytes received
	NetworkTx      int64     `json:"network_tx"`       // Network bytes transmitted
	DiskRead       int64     `json:"disk_read"`        // Disk bytes read
	DiskWrite      int64     `json:"disk_write"`       // Disk bytes written
	ProcessCount   int       `json:"process_count"`    // Number of processes
	ThreadCount    int       `json:"thread_count"`     // Number of threads
	FileDescriptors int      `json:"file_descriptors"` // Open file descriptors
	Timestamp      time.Time `json:"timestamp"`
}

// ResourceQuota defines resource quotas for applications
type ResourceQuota struct {
	AppID           string         `json:"app_id"`
	MaxCPU          float64        `json:"max_cpu"`           // Maximum CPU cores
	MaxMemory       int64          `json:"max_memory"`        // Maximum memory
	MaxStorage      int64          `json:"max_storage"`       // Maximum storage
	MaxContainers   int            `json:"max_containers"`    // Maximum containers
	MaxBandwidth    int64          `json:"max_bandwidth"`     // Maximum network bandwidth
	ReserveCPU      float64        `json:"reserve_cpu"`       // Reserved CPU cores
	ReserveMemory   int64          `json:"reserve_memory"`    // Reserved memory
	PriorityClass   string         `json:"priority_class"`    // Resource priority
	Limits          ResourceLimits `json:"limits"`            // Default limits
}

// ResourceAllocation tracks allocated resources
type ResourceAllocation struct {
	AppID       string    `json:"app_id"`
	ContainerID string    `json:"container_id"`
	CPU         float64   `json:"cpu"`
	Memory      int64     `json:"memory"`
	Storage     int64     `json:"storage"`
	Bandwidth   int64     `json:"bandwidth"`
	AllocatedAt time.Time `json:"allocated_at"`
}

// NewResourceManager creates a new resource manager
func NewResourceManager(auditLogger *logging.AuditLogger) (*ResourceManager, error) {
	rm := &ResourceManager{
		auditLogger: auditLogger,
	}

	auditLogger.LogEvent("RESOURCE_MANAGER_INITIALIZED", map[string]interface{}{})

	return rm, nil
}

// ValidateResourceLimits validates resource limits against quotas
func (rm *ResourceManager) ValidateResourceLimits(appID string, limits ResourceLimits, quota ResourceQuota) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Validate CPU limits
	if limits.CPULimit > quota.MaxCPU {
		return fmt.Errorf("CPU limit %.2f exceeds quota %.2f for app %s", 
			limits.CPULimit, quota.MaxCPU, appID)
	}

	// Validate memory limits
	if limits.MemoryLimit > quota.MaxMemory {
		return fmt.Errorf("memory limit %d exceeds quota %d for app %s", 
			limits.MemoryLimit, quota.MaxMemory, appID)
	}

	// Validate storage limits
	if limits.DiskLimit > quota.MaxStorage {
		return fmt.Errorf("disk limit %d exceeds quota %d for app %s", 
			limits.DiskLimit, quota.MaxStorage, appID)
	}

	// Validate bandwidth limits
	if limits.NetworkLimit > quota.MaxBandwidth {
		return fmt.Errorf("network limit %d exceeds quota %d for app %s", 
			limits.NetworkLimit, quota.MaxBandwidth, appID)
	}

	// Validate CPU shares (should be between 2 and 262144)
	if limits.CPUShares != 0 && (limits.CPUShares < 2 || limits.CPUShares > 262144) {
		return fmt.Errorf("CPU shares %d must be between 2 and 262144", limits.CPUShares)
	}

	// Validate BlkIO weight (should be between 10 and 1000)
	if limits.BlkioWeight != 0 && (limits.BlkioWeight < 10 || limits.BlkioWeight > 1000) {
		return fmt.Errorf("BlkIO weight %d must be between 10 and 1000", limits.BlkioWeight)
	}

	// Validate OOM score adjustment (should be between -1000 and 1000)
	if limits.OomScoreAdj < -1000 || limits.OomScoreAdj > 1000 {
		return fmt.Errorf("OOM score adjustment %d must be between -1000 and 1000", limits.OomScoreAdj)
	}

	rm.auditLogger.LogEvent("RESOURCE_LIMITS_VALIDATED", map[string]interface{}{
		"app_id":       appID,
		"cpu_limit":    limits.CPULimit,
		"memory_limit": limits.MemoryLimit,
		"disk_limit":   limits.DiskLimit,
	})

	return nil
}

// AllocateResources allocates resources for a container
func (rm *ResourceManager) AllocateResources(appID, containerID string, limits ResourceLimits) (*ResourceAllocation, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	allocation := &ResourceAllocation{
		AppID:       appID,
		ContainerID: containerID,
		CPU:         limits.CPULimit,
		Memory:      limits.MemoryLimit,
		Storage:     limits.DiskLimit,
		Bandwidth:   limits.NetworkLimit,
		AllocatedAt: time.Now(),
	}

	rm.auditLogger.LogEvent("RESOURCES_ALLOCATED", map[string]interface{}{
		"app_id":       appID,
		"container_id": containerID,
		"cpu":          allocation.CPU,
		"memory":       allocation.Memory,
		"storage":      allocation.Storage,
		"bandwidth":    allocation.Bandwidth,
	})

	logrus.Infof("Allocated resources for container %s: CPU=%.2f, Memory=%d, Storage=%d", 
		containerID, allocation.CPU, allocation.Memory, allocation.Storage)

	return allocation, nil
}

// DeallocateResources deallocates resources for a container
func (rm *ResourceManager) DeallocateResources(containerID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.auditLogger.LogEvent("RESOURCES_DEALLOCATED", map[string]interface{}{
		"container_id": containerID,
	})

	logrus.Infof("Deallocated resources for container %s", containerID)

	return nil
}

// GetResourceUsage gets current resource usage for a container
func (rm *ResourceManager) GetResourceUsage(ctx context.Context, containerID string) (*ResourceUsage, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Get container stats using docker stats
	cmd := exec.CommandContext(ctx, "docker", "stats", containerID, "--no-stream", "--format", 
		"table {{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}")
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid stats output")
	}

	// Parse the stats line (skip header)
	statsLine := strings.Fields(lines[1])
	if len(statsLine) < 5 {
		return nil, fmt.Errorf("insufficient stats data")
	}

	usage := &ResourceUsage{
		ContainerID: containerID,
		Timestamp:   time.Now(),
	}

	// Parse CPU usage
	cpuStr := strings.TrimSuffix(statsLine[0], "%")
	if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
		usage.CPUUsage = cpu
	}

	// Parse memory usage
	memParts := strings.Split(statsLine[1], "/")
	if len(memParts) == 2 {
		if memUsage, err := parseSize(strings.TrimSpace(memParts[0])); err == nil {
			usage.MemoryUsage = memUsage
		}
		if memLimit, err := parseSize(strings.TrimSpace(memParts[1])); err == nil {
			usage.MemoryLimit = memLimit
		}
	}

	// Parse network I/O
	netParts := strings.Split(statsLine[2], "/")
	if len(netParts) == 2 {
		if netRx, err := parseSize(strings.TrimSpace(netParts[0])); err == nil {
			usage.NetworkRx = netRx
		}
		if netTx, err := parseSize(strings.TrimSpace(netParts[1])); err == nil {
			usage.NetworkTx = netTx
		}
	}

	// Parse block I/O
	blockParts := strings.Split(statsLine[3], "/")
	if len(blockParts) == 2 {
		if diskRead, err := parseSize(strings.TrimSpace(blockParts[0])); err == nil {
			usage.DiskRead = diskRead
		}
		if diskWrite, err := parseSize(strings.TrimSpace(blockParts[1])); err == nil {
			usage.DiskWrite = diskWrite
		}
	}

	// Parse process count
	if pids, err := strconv.Atoi(statsLine[4]); err == nil {
		usage.ProcessCount = pids
	}

	return usage, nil
}

// MonitorResourceUsage continuously monitors resource usage
func (rm *ResourceManager) MonitorResourceUsage(ctx context.Context, containerID string, interval time.Duration, callback func(*ResourceUsage)) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			usage, err := rm.GetResourceUsage(ctx, containerID)
			if err != nil {
				logrus.Warnf("Failed to get resource usage for container %s: %v", containerID, err)
				continue
			}

			if callback != nil {
				callback(usage)
			}
		}
	}
}

// EnforceResourceLimits enforces resource limits on a container
func (rm *ResourceManager) EnforceResourceLimits(ctx context.Context, containerID string, limits ResourceLimits) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	logrus.Infof("Enforcing resource limits for container %s", containerID)

	// Update container resource limits using docker update
	cmd := exec.CommandContext(ctx, "docker", "update")

	// Add CPU limits
	if limits.CPULimit > 0 {
		cmd.Args = append(cmd.Args, "--cpus", fmt.Sprintf("%.2f", limits.CPULimit))
	}

	// Add memory limits
	if limits.MemoryLimit > 0 {
		cmd.Args = append(cmd.Args, "--memory", fmt.Sprintf("%d", limits.MemoryLimit))
	}

	// Add swap limits
	if limits.SwapLimit > 0 {
		cmd.Args = append(cmd.Args, "--memory-swap", fmt.Sprintf("%d", limits.SwapLimit))
	}

	// Add CPU shares
	if limits.CPUShares > 0 {
		cmd.Args = append(cmd.Args, "--cpu-shares", fmt.Sprintf("%d", limits.CPUShares))
	}

	// Add CPU period and quota
	if limits.CPUPeriod > 0 {
		cmd.Args = append(cmd.Args, "--cpu-period", fmt.Sprintf("%d", limits.CPUPeriod))
	}
	if limits.CPUQuota > 0 {
		cmd.Args = append(cmd.Args, "--cpu-quota", fmt.Sprintf("%d", limits.CPUQuota))
	}

	// Add CPU affinity
	if limits.CPUSetCPUs != "" {
		cmd.Args = append(cmd.Args, "--cpuset-cpus", limits.CPUSetCPUs)
	}
	if limits.CPUSetMems != "" {
		cmd.Args = append(cmd.Args, "--cpuset-mems", limits.CPUSetMems)
	}

	// Add memory reservation
	if limits.MemoryReservation > 0 {
		cmd.Args = append(cmd.Args, "--memory-reservation", fmt.Sprintf("%d", limits.MemoryReservation))
	}

	// Add kernel memory
	if limits.KernelMemory > 0 {
		cmd.Args = append(cmd.Args, "--kernel-memory", fmt.Sprintf("%d", limits.KernelMemory))
	}

	// Add process limits
	if limits.ProcessLimit > 0 {
		cmd.Args = append(cmd.Args, "--pids-limit", fmt.Sprintf("%d", limits.ProcessLimit))
	}

	// Add container ID
	cmd.Args = append(cmd.Args, containerID)

	// Execute update command
	output, err := cmd.CombinedOutput()
	if err != nil {
		rm.auditLogger.LogEvent("RESOURCE_LIMITS_ENFORCEMENT_FAILED", map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
			"output":       string(output),
		})
		return fmt.Errorf("failed to enforce resource limits: %w, output: %s", err, output)
	}

	rm.auditLogger.LogEvent("RESOURCE_LIMITS_ENFORCED", map[string]interface{}{
		"container_id": containerID,
		"cpu_limit":    limits.CPULimit,
		"memory_limit": limits.MemoryLimit,
	})

	logrus.Infof("Resource limits enforced for container %s", containerID)
	return nil
}

// CheckResourceViolations checks if a container violates resource limits
func (rm *ResourceManager) CheckResourceViolations(usage *ResourceUsage, limits ResourceLimits) []string {
	var violations []string

	// Check CPU usage
	if limits.CPULimit > 0 && usage.CPUUsage > limits.CPULimit*100 {
		violations = append(violations, fmt.Sprintf("CPU usage %.2f%% exceeds limit %.2f%%", 
			usage.CPUUsage, limits.CPULimit*100))
	}

	// Check memory usage
	if limits.MemoryLimit > 0 && usage.MemoryUsage > limits.MemoryLimit {
		violations = append(violations, fmt.Sprintf("Memory usage %d bytes exceeds limit %d bytes", 
			usage.MemoryUsage, limits.MemoryLimit))
	}

	// Check process count
	if limits.ProcessLimit > 0 && usage.ProcessCount > limits.ProcessLimit {
		violations = append(violations, fmt.Sprintf("Process count %d exceeds limit %d", 
			usage.ProcessCount, limits.ProcessLimit))
	}

	return violations
}

// GetSystemResourceInfo gets system-wide resource information
func (rm *ResourceManager) GetSystemResourceInfo(ctx context.Context) (map[string]interface{}, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	info := make(map[string]interface{})

	// Get CPU information
	cpuCmd := exec.CommandContext(ctx, "nproc")
	if output, err := cpuCmd.Output(); err == nil {
		if cpuCount, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			info["cpu_cores"] = cpuCount
		}
	}

	// Get memory information
	memCmd := exec.CommandContext(ctx, "free", "-b")
	if output, err := memCmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 2 {
				if totalMem, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					info["total_memory"] = totalMem
				}
				if availMem, err := strconv.ParseInt(fields[6], 10, 64); err == nil {
					info["available_memory"] = availMem
				}
			}
		}
	}

	// Get disk information
	diskCmd := exec.CommandContext(ctx, "df", "-B1", "/")
	if output, err := diskCmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				if totalDisk, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					info["total_disk"] = totalDisk
				}
				if availDisk, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
					info["available_disk"] = availDisk
				}
			}
		}
	}

	info["timestamp"] = time.Now()
	return info, nil
}

// CalculateResourceEfficiency calculates resource efficiency metrics
func (rm *ResourceManager) CalculateResourceEfficiency(usage *ResourceUsage, limits ResourceLimits) map[string]float64 {
	efficiency := make(map[string]float64)

	// CPU efficiency
	if limits.CPULimit > 0 {
		efficiency["cpu"] = (usage.CPUUsage / 100) / limits.CPULimit
	}

	// Memory efficiency
	if limits.MemoryLimit > 0 {
		efficiency["memory"] = float64(usage.MemoryUsage) / float64(limits.MemoryLimit)
	}

	// Overall efficiency (average of CPU and memory)
	if cpuEff, hasCPU := efficiency["cpu"]; hasCPU {
		if memEff, hasMem := efficiency["memory"]; hasMem {
			efficiency["overall"] = (cpuEff + memEff) / 2
		}
	}

	return efficiency
}

// OptimizeResourceLimits suggests optimized resource limits based on usage patterns
func (rm *ResourceManager) OptimizeResourceLimits(usageHistory []*ResourceUsage, currentLimits ResourceLimits) ResourceLimits {
	if len(usageHistory) == 0 {
		return currentLimits
	}

	// Calculate usage statistics
	var totalCPU, maxCPU float64
	var totalMem, maxMem int64
	var totalProc, maxProc int

	for _, usage := range usageHistory {
		totalCPU += usage.CPUUsage
		if usage.CPUUsage > maxCPU {
			maxCPU = usage.CPUUsage
		}

		totalMem += usage.MemoryUsage
		if usage.MemoryUsage > maxMem {
			maxMem = usage.MemoryUsage
		}

		totalProc += usage.ProcessCount
		if usage.ProcessCount > maxProc {
			maxProc = usage.ProcessCount
		}
	}

count := float64(len(usageHistory))
	_ = totalCPU / count // avgCPU not used
	_ = float64(totalMem) / count // avgMem not used

	// Optimize limits with safety margins
	optimized := currentLimits

	// CPU: Use 95th percentile with 20% safety margin
	if maxCPU > 0 {
		optimized.CPULimit = (maxCPU/100) * 1.2
	}

	// Memory: Use maximum usage with 30% safety margin
	if maxMem > 0 {
		optimized.MemoryLimit = int64(float64(maxMem) * 1.3)
	}

	// Process limit: Use maximum with 50% safety margin
	if maxProc > 0 {
		optimized.ProcessLimit = int(float64(maxProc) * 1.5)
	}

	return optimized
}

// parseSize parses size strings like "1.5GB", "512MB"
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, nil
	}

	// Remove units and parse
	sizeStr = strings.ToUpper(sizeStr)
	
	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "TB") {
		multiplier = 1024 * 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "TB")
	} else if strings.HasSuffix(sizeStr, "B") {
		sizeStr = strings.TrimSuffix(sizeStr, "B")
	}

	size, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	}

	return int64(size * float64(multiplier)), nil
}