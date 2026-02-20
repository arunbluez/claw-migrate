package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Result holds backup operation result
type Result struct {
	Path    string
	Size    int64
	Success bool
	Error   error
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
