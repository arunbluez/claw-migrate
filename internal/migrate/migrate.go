package migrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/arunbluez/claw-migrate/internal/config"
)


// FileResult tracks the migration result for a single file
type FileResult struct {
	Source      string
	Dest       string
	Name       string
	Lines      int
	Migrated   bool
	Skipped    bool
	BackedUp   bool
	Error      error
}

// Result tracks the overall migration result
type Result struct {
	Files        []FileResult
	ConfigResult *FileResult
	TotalFiles   int
	Migrated     int
	Skipped      int
	Errors       int
}

// SkipEntries are items we never migrate
var SkipEntries = map[string]bool{
	".git":       true,
	".openclaw":  true,
	".DS_Store":  true,
	".gitignore": true,
	"sessions":   true, // incompatible format
}

// MigrateWorkspace copies the ENTIRE workspace from OpenClaw to PicoClaw
// including all files, custom directories, project folders, etc.
func MigrateWorkspace(srcWorkspace, dstWorkspace string, force bool) Result {
	result := Result{}

	// Ensure destination exists
	os.MkdirAll(dstWorkspace, 0755)

	// Scan source workspace and migrate everything
	entries, err := os.ReadDir(srcWorkspace)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip certain entries
		if SkipEntries[name] {
			continue
		}

		srcPath := filepath.Join(srcWorkspace, name)
		dstPath := filepath.Join(dstWorkspace, name)

		if entry.IsDir() {
			// Migrate entire directory recursively
			os.MkdirAll(dstPath, 0755)
			dirResults := migrateDirectory(srcPath, dstPath, force)
			for _, fr := range dirResults {
				result.Files = append(result.Files, fr)
				result.TotalFiles++
				if fr.Migrated {
					result.Migrated++
				} else if fr.Skipped {
					result.Skipped++
				} else if fr.Error != nil {
					result.Errors++
				}
			}
		} else {
			// Migrate file
			fr := migrateFile(srcPath, dstPath, name, force)
			result.Files = append(result.Files, fr)
			result.TotalFiles++
			if fr.Migrated {
				result.Migrated++
			} else if fr.Skipped {
				result.Skipped++
			} else if fr.Error != nil {
				result.Errors++
			}
		}
	}

	return result
}

// MigrateConfig converts and writes the PicoClaw config
func MigrateConfig(openclawConfigPath, picoConfigPath string, force bool) FileResult {
	fr := FileResult{
		Source: openclawConfigPath,
		Dest:   picoConfigPath,
		Name:   "config.json",
	}

	// Read OpenClaw config
	ocConfig, err := config.ReadConfig(openclawConfigPath)
	if err != nil {
		fr.Error = fmt.Errorf("read openclaw config: %w", err)
		return fr
	}

	// Convert to PicoClaw format
	picoConfig := config.ConvertConfig(ocConfig)

	// Read existing PicoClaw config if present
	existingConfig, _ := config.ReadConfig(picoConfigPath)

	// Merge (existing config takes precedence for manually configured values)
	if existingConfig != nil {
		picoConfig = config.MergeConfig(existingConfig, picoConfig)
	}

	// Backup existing config if present
	if _, err := os.Stat(picoConfigPath); err == nil {
		backupPath := picoConfigPath + ".bak"
		if err := copyFileSafe(picoConfigPath, backupPath); err == nil {
			fr.BackedUp = true
		}
	}

	// Write config
	if err := config.WriteConfig(picoConfig, picoConfigPath); err != nil {
		fr.Error = fmt.Errorf("write picoclaw config: %w", err)
		return fr
	}

	fr.Migrated = true
	return fr
}

// --- Internal helpers ---

func migrateFile(src, dst, name string, force bool) FileResult {
	fr := FileResult{
		Source: src,
		Dest:   dst,
		Name:   name,
	}

	// Check source exists
	srcInfo, err := os.Stat(src)
	if os.IsNotExist(err) {
		fr.Skipped = true
		return fr
	}

	// Count lines
	if data, err := os.ReadFile(src); err == nil {
		fr.Lines = len(strings.Split(string(data), "\n"))
	}

	// Check if destination already exists
	if _, err := os.Stat(dst); err == nil && !force {
		// File exists and not force — backup then overwrite
		backupPath := dst + ".bak"
		copyFileSafe(dst, backupPath)
		fr.BackedUp = true
	}

	// Copy file
	if err := copyFileSafe(src, dst); err != nil {
		fr.Error = fmt.Errorf("copy %s: %w", name, err)
		return fr
	}

	// Preserve permissions
	os.Chmod(dst, srcInfo.Mode())

	fr.Migrated = true
	return fr
}

func migrateDirectory(srcDir, dstDir string, force bool) []FileResult {
	var results []FileResult

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			os.MkdirAll(dstPath, 0755)
			subResults := migrateDirectory(srcPath, dstPath, force)
			results = append(results, subResults...)
		} else {
			name := filepath.Join(filepath.Base(srcDir), entry.Name())
			fr := migrateFile(srcPath, dstPath, name, force)
			results = append(results, fr)
		}
	}

	return results
}

func copyFileSafe(src, dst string) error {
	// Ensure parent directory exists
	os.MkdirAll(filepath.Dir(dst), 0755)

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
