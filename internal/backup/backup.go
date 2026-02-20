package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Result holds backup operation result
type Result struct {
	Path    string
	Size    int64
	Success bool
	Error   error
}

// BackupInfo describes a found backup file
type BackupInfo struct {
	Path      string
	Filename  string
	Size      int64
	Timestamp string // extracted from filename
}

// CreateBackup creates a tar.gz backup of the OpenClaw directory
func CreateBackup(openclawDir string) Result {
	home, _ := os.UserHomeDir()
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("openclaw-backup-%s.tar.gz", timestamp)
	backupPath := filepath.Join(home, filename)

	// Use tar to create backup
	cmd := exec.Command("tar", "-czf", backupPath, "-C", filepath.Dir(openclawDir), filepath.Base(openclawDir))
	if err := cmd.Run(); err != nil {
		return Result{Error: fmt.Errorf("tar failed: %w", err)}
	}

	// Get file size
	info, err := os.Stat(backupPath)
	if err != nil {
		return Result{Path: backupPath, Error: fmt.Errorf("could not stat backup: %w", err)}
	}

	return Result{
		Path:    backupPath,
		Size:    info.Size(),
		Success: true,
	}
}

// VerifyBackup checks that the backup file is valid
func VerifyBackup(backupPath string) error {
	cmd := exec.Command("tar", "-tzf", backupPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("backup verification failed: %w", err)
	}
	return nil
}

// ListBackups finds all openclaw backup files in the home directory
func ListBackups() []BackupInfo {
	home, _ := os.UserHomeDir()
	pattern := filepath.Join(home, "openclaw-backup-*.tar.gz")
	matches, _ := filepath.Glob(pattern)

	var backups []BackupInfo
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		filename := filepath.Base(path)
		// Extract timestamp from filename: openclaw-backup-20260220-140013.tar.gz
		ts := strings.TrimPrefix(filename, "openclaw-backup-")
		ts = strings.TrimSuffix(ts, ".tar.gz")

		backups = append(backups, BackupInfo{
			Path:      path,
			Filename:  filename,
			Size:      info.Size(),
			Timestamp: ts,
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp > backups[j].Timestamp
	})

	return backups
}

// RestoreBackup extracts a backup archive to restore ~/.openclaw
func RestoreBackup(backupPath string) error {
	home, _ := os.UserHomeDir()
	openclawDir := filepath.Join(home, ".openclaw")

	// Remove existing .openclaw if present
	if _, err := os.Stat(openclawDir); err == nil {
		if err := os.RemoveAll(openclawDir); err != nil {
			return fmt.Errorf("could not remove existing ~/.openclaw: %w", err)
		}
	}

	// Extract backup
	cmd := exec.Command("tar", "-xzf", backupPath, "-C", home)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}

// FormatSize formats bytes into human-readable size
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}