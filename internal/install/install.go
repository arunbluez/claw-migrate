package install

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// FallbackVersion is used if we can't reach GitHub API
	FallbackVersion = "0.1.2"
	// RepoAPI for fetching latest release
	RepoAPI = "https://api.github.com/repos/sipeed/picoclaw/releases/latest"
	// BaseURL for GitHub releases
	BaseURL = "https://github.com/sipeed/picoclaw/releases/download"
)

// LatestVersion holds the resolved version (fetched or fallback)
var LatestVersion string

// FetchLatestVersion queries GitHub API for the latest PicoClaw release tag
func FetchLatestVersion() string {
	if LatestVersion != "" {
		return LatestVersion
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", RepoAPI, nil)
	if err != nil {
		LatestVersion = FallbackVersion
		return LatestVersion
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		LatestVersion = FallbackVersion
		return LatestVersion
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		LatestVersion = FallbackVersion
		return LatestVersion
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		LatestVersion = FallbackVersion
		return LatestVersion
	}

	// Strip leading "v" if present (tag is "v0.1.2", we need "0.1.2")
	LatestVersion = strings.TrimPrefix(release.TagName, "v")
	if LatestVersion == "" {
		LatestVersion = FallbackVersion
	}
	return LatestVersion
}

// VersionTag returns the version with "v" prefix for display
func VersionTag() string {
	return "v" + FetchLatestVersion()
}

// GetDownloadURL returns the appropriate download URL for the current platform
// PicoClaw release naming: picoclaw_{OS}_{arch}.tar.gz
//   OS:   Darwin, Linux, Freebsd
//   arch: arm64, x86_64, armv6, mips64, riscv64
func GetDownloadURL() (string, string, error) {
	version := FetchLatestVersion()
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go OS names to PicoClaw release names
	osName := ""
	switch goos {
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	case "freebsd":
		osName = "Freebsd"
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", goos)
	}

	// Map Go arch names to PicoClaw release names
	archName := ""
	switch goarch {
	case "arm64":
		archName = "arm64"
	case "amd64":
		archName = "x86_64"
	case "arm":
		archName = "armv6"
	case "mips64":
		archName = "mips64"
	case "riscv64":
		archName = "riscv64"
	default:
		return "", "", fmt.Errorf("unsupported architecture: %s", goarch)
	}

	filename := fmt.Sprintf("picoclaw_%s_%s.tar.gz", osName, archName)
	url := fmt.Sprintf("%s/v%s/%s", BaseURL, version, filename)
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

// Extract extracts the downloaded tar.gz archive
func Extract(archivePath, destDir string) (string, error) {
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", destDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tar extract failed: %w", err)
	}

	// Find the picoclaw binary in the extracted files
	binaryPath := filepath.Join(destDir, "picoclaw")
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Try common patterns (some releases nest in a subdirectory)
	patterns := []string{
		filepath.Join(destDir, "picoclaw-*", "picoclaw"),
		filepath.Join(destDir, "picoclaw_*", "picoclaw"),
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

	// Ensure /usr/local/bin exists
	os.MkdirAll("/usr/local/bin", 0755)

	// Try direct copy first
	if err := copyFile(binaryPath, destPath); err == nil {
		return nil
	}

	// Fall back to sudo
	exec.Command("sudo", "mkdir", "-p", "/usr/local/bin").Run()
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