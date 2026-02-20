package uninstall

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ════════════════════════════════════════════════════════════
// OpenClaw
// ════════════════════════════════════════════════════════════

// StopOpenClaw kills any running OpenClaw processes
func StopOpenClaw() error {
	exec.Command("openclaw", "daemon", "stop").Run()
	exec.Command("pkill", "-f", "openclaw gateway").Run()
	exec.Command("pkill", "-f", "openclaw").Run()
	return nil
}

// RemoveBinary uninstalls the OpenClaw npm package
func RemoveBinary() error {
	cmd := exec.Command("npm", "uninstall", "-g", "openclaw")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("pnpm", "remove", "-g", "openclaw")
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// RemoveData removes a directory (e.g. ~/.openclaw or ~/.picoclaw)
func RemoveData(dir string) error {
	return os.RemoveAll(dir)
}

// RemoveLaunchAgents removes macOS launch agents matching a keyword
func RemoveLaunchAgents() []string {
	return removeLaunchAgentsMatching("openclaw", "clawdbot")
}

// VerifyRemoved checks that OpenClaw is fully removed
func VerifyRemoved() (binaryGone, dataGone, agentsGone bool) {
	_, err := exec.LookPath("openclaw")
	binaryGone = err != nil

	home, _ := os.UserHomeDir()
	_, err = os.Stat(filepath.Join(home, ".openclaw"))
	dataGone = os.IsNotExist(err)

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

// ════════════════════════════════════════════════════════════
// PicoClaw
// ════════════════════════════════════════════════════════════

// StopPicoClaw kills any running PicoClaw processes
func StopPicoClaw() error {
	exec.Command("picoclaw", "daemon", "stop").Run()
	exec.Command("pkill", "-f", "picoclaw gateway").Run()
	exec.Command("pkill", "-f", "picoclaw").Run()
	return nil
}

// RemovePicoClawBinary removes the picoclaw binary
func RemovePicoClawBinary() error {
	path, err := exec.LookPath("picoclaw")
	if err != nil {
		return nil // not installed, nothing to do
	}

	// Try direct removal
	if err := os.Remove(path); err == nil {
		return nil
	}

	// Fall back to sudo
	cmd := exec.Command("sudo", "rm", "-f", path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemovePicoClawLaunchAgents removes macOS launch agents for PicoClaw
func RemovePicoClawLaunchAgents() []string {
	return removeLaunchAgentsMatching("picoclaw")
}

// VerifyPicoClawRemoved checks that PicoClaw is fully removed
func VerifyPicoClawRemoved() (binaryGone, dataGone, agentsGone bool) {
	_, err := exec.LookPath("picoclaw")
	binaryGone = err != nil

	home, _ := os.UserHomeDir()
	_, err = os.Stat(filepath.Join(home, ".picoclaw"))
	dataGone = os.IsNotExist(err)

	launchDir := filepath.Join(home, "Library", "LaunchAgents")
	entries, _ := os.ReadDir(launchDir)
	agentsGone = true
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Name()), "picoclaw") {
			agentsGone = false
			break
		}
	}

	return
}

// ════════════════════════════════════════════════════════════
// Shared helpers
// ════════════════════════════════════════════════════════════

func removeLaunchAgentsMatching(keywords ...string) []string {
	home, _ := os.UserHomeDir()
	launchDir := filepath.Join(home, "Library", "LaunchAgents")

	var removed []string
	entries, err := os.ReadDir(launchDir)
	if err != nil {
		return removed
	}

	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		matched := false
		for _, kw := range keywords {
			if strings.Contains(name, kw) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		fullPath := filepath.Join(launchDir, entry.Name())
		exec.Command("launchctl", "unload", fullPath).Run()
		if err := os.Remove(fullPath); err == nil {
			removed = append(removed, entry.Name())
		}
	}

	return removed
}