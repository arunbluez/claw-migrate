package uninstall

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StopOpenClaw kills any running OpenClaw processes
func StopOpenClaw() error {
	// Try graceful stop first
	exec.Command("openclaw", "daemon", "stop").Run()

	// Kill remaining processes
	exec.Command("pkill", "-f", "openclaw gateway").Run()
	exec.Command("pkill", "-f", "openclaw").Run()

	return nil
}

// RemoveBinary uninstalls the OpenClaw npm package
func RemoveBinary() error {
	// Try npm first
	cmd := exec.Command("npm", "uninstall", "-g", "openclaw")
	if err := cmd.Run(); err != nil {
		// Try pnpm
		cmd = exec.Command("pnpm", "remove", "-g", "openclaw")
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// RemoveData removes the ~/.openclaw directory
func RemoveData(openclawDir string) error {
	return os.RemoveAll(openclawDir)
}

// RemoveLaunchAgents removes macOS launch agents for OpenClaw
func RemoveLaunchAgents() []string {
	home, _ := os.UserHomeDir()
	launchDir := filepath.Join(home, "Library", "LaunchAgents")

	var removed []string

	entries, err := os.ReadDir(launchDir)
	if err != nil {
		return removed
	}

	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, "openclaw") || strings.Contains(name, "clawdbot") {
			fullPath := filepath.Join(launchDir, entry.Name())

			// Unload first
			exec.Command("launchctl", "unload", fullPath).Run()

			// Remove file
			if err := os.Remove(fullPath); err == nil {
				removed = append(removed, entry.Name())
			}
		}
	}

	return removed
}

// VerifyRemoved checks that OpenClaw is fully removed
func VerifyRemoved() (binaryGone, dataGone, agentsGone bool) {
	// Check binary
	_, err := exec.LookPath("openclaw")
	binaryGone = err != nil

	// Check data
	home, _ := os.UserHomeDir()
	_, err = os.Stat(filepath.Join(home, ".openclaw"))
	dataGone = os.IsNotExist(err)

	// Check launch agents
	launchDir := filepath.Join(home, "Library", "LaunchAgents")
	entries, _ := os.ReadDir(launchDir)
	agentsGone = true
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, "openclaw") || strings.Contains(name, "clawdbot") {
			agentsGone = false
			break
		}
	}

	return
}
