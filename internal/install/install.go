package install

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	// LatestVersion is the current PicoClaw release
	LatestVersion = "v0.1.2"
	// BaseURL for GitHub releases
	BaseURL = "https://github.com/sipeed/picoclaw/releases/download"
)

// GetDownloadURL returns the appropriate download URL for the current platform
func GetDownloadURL() (string, string, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	var filename string
	switch {
	case os == "darwin" && arch == "arm64":
		filename = fmt.Sprintf("picoclaw-%s-macos-arm64.zip", LatestVersion)
	case os == "darwin" && arch == "amd64":
		filename = fmt.Sprintf("picoclaw-%s-macos-amd64.zip", LatestVersion)
	case os == "linux" && arch == "arm64":
		filename = fmt.Sprintf("picoclaw-%s-linux-arm64.tar.gz", LatestVersion)
	case os == "linux" && arch == "amd64":
		filename = fmt.Sprintf("picoclaw-%s-linux-amd64.tar.gz", LatestVersion)
	default:
		return "", "", fmt.Errorf("unsupported platform: %s/%s", os, arch)
	}

	url := fmt.Sprintf("%s/%s/%s", BaseURL, LatestVersion, filename)
	return url, filename, nil
}

// Download downloads a file from URL to the given path
func Download(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Extract extracts the downloaded archive
func Extract(archivePath, destDir string) (string, error) {
	ext := filepath.Ext(archivePath)
	switch ext {
	case ".zip":
		// macOS zip
		cmd := exec.Command("unzip", "-o", archivePath, "-d", destDir)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("unzip failed: %w", err)
		}
	case ".gz":
		// Linux tar.gz
		cmd := exec.Command("tar", "-xzf", archivePath, "-C", destDir)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("tar extract failed: %w", err)
		}
	default:
		return "", fmt.Errorf("unknown archive format: %s", ext)
	}

	// Find the picoclaw binary in the extracted files
	binaryPath := filepath.Join(destDir, "picoclaw")
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Try common patterns
	patterns := []string{
		filepath.Join(destDir, "picoclaw-*", "picoclaw"),
		filepath.Join(destDir, "picoclaw"),
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("picoclaw binary not found in extracted archive")
}

// InstallBinary copies the binary to /usr/local/bin (may require sudo)
func InstallBinary(binaryPath string) error {
	destPath := "/usr/local/bin/picoclaw"

	// Make executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}

	// Try direct copy first
	if err := copyFile(binaryPath, destPath); err == nil {
		return nil
	}

	// Fall back to sudo
	cmd := exec.Command("sudo", "cp", binaryPath, destPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunOnboard runs picoclaw onboard
func RunOnboard() error {
	cmd := exec.Command("picoclaw", "onboard")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// BuildFromSource clones and builds PicoClaw from source
func BuildFromSource(workDir string) error {
	repoDir := filepath.Join(workDir, "picoclaw")

	// Clone
	cmd := exec.Command("git", "clone", "https://github.com/sipeed/picoclaw.git", repoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Make deps
	cmd = exec.Command("make", "deps")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make deps failed: %w", err)
	}

	// Make install
	cmd = exec.Command("make", "install")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
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
	if err != nil {
		return err
	}

	return os.Chmod(dst, 0755)
}
