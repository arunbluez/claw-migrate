package detect

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Installation holds detected installation info
type Installation struct {
	Found         bool
	HomeDir       string // e.g. ~/.openclaw or ~/.picoclaw
	ConfigPath    string
	WorkspaceDir  string
	BinaryPath    string
	Version       string
	WorkspaceFiles map[string]bool // which workspace files exist
	HasMemory     bool
	HasSkills     bool
	HasCron       bool
	HasSessions   bool
	Config        map[string]interface{} // parsed JSON config
}

// SystemInfo holds system details
type SystemInfo struct {
	OS   string // darwin, linux, windows
	Arch string // arm64, amd64
	Home string // user home directory
}

// GetSystemInfo returns current system info
func GetSystemInfo() SystemInfo {
	home, _ := os.UserHomeDir()
	return SystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		Home: home,
	}
}

// DetectOpenClaw checks for an OpenClaw installation
func DetectOpenClaw() Installation {
	home, _ := os.UserHomeDir()
	inst := Installation{
		HomeDir:        filepath.Join(home, ".openclaw"),
		WorkspaceFiles: make(map[string]bool),
	}

	// Check if directory exists
	if _, err := os.Stat(inst.HomeDir); os.IsNotExist(err) {
		return inst
	}
	inst.Found = true

	// Config
	inst.ConfigPath = filepath.Join(inst.HomeDir, "openclaw.json")
	if _, err := os.Stat(inst.ConfigPath); err == nil {
		inst.Config = parseJSONFile(inst.ConfigPath)
	}

	// Workspace
	inst.WorkspaceDir = filepath.Join(inst.HomeDir, "workspace")

	// Check workspace files
	wsFiles := []string{"SOUL.md", "IDENTITY.md", "AGENTS.md", "USER.md", "TOOLS.md", "HEARTBEAT.md"}
	for _, f := range wsFiles {
		path := filepath.Join(inst.WorkspaceDir, f)
		if _, err := os.Stat(path); err == nil {
			inst.WorkspaceFiles[f] = true
		}
	}

	// Check subdirectories
	inst.HasMemory = dirHasFiles(filepath.Join(inst.WorkspaceDir, "memory"))
	inst.HasSkills = dirHasFiles(filepath.Join(inst.WorkspaceDir, "skills"))
	inst.HasCron = dirHasFiles(filepath.Join(inst.WorkspaceDir, "cron"))
	inst.HasSessions = dirHasFiles(filepath.Join(inst.WorkspaceDir, "sessions"))

	// Check binary
	if path, err := exec.LookPath("openclaw"); err == nil {
		inst.BinaryPath = path
		if out, err := exec.Command("openclaw", "--version").Output(); err == nil {
			inst.Version = strings.TrimSpace(string(out))
		}
	}

	return inst
}

// DetectPicoClaw checks for a PicoClaw installation
func DetectPicoClaw() Installation {
	home, _ := os.UserHomeDir()
	inst := Installation{
		HomeDir:        filepath.Join(home, ".picoclaw"),
		WorkspaceFiles: make(map[string]bool),
	}

	// Check if directory exists
	if _, err := os.Stat(inst.HomeDir); err == nil {
		inst.Found = true
	}

	// Config
	inst.ConfigPath = filepath.Join(inst.HomeDir, "config.json")
	if _, err := os.Stat(inst.ConfigPath); err == nil {
		inst.Config = parseJSONFile(inst.ConfigPath)
	}

	// Workspace
	inst.WorkspaceDir = filepath.Join(inst.HomeDir, "workspace")

	// Check workspace files
	wsFiles := []string{"SOUL.md", "IDENTITY.md", "AGENTS.md", "USER.md", "TOOLS.md", "HEARTBEAT.md"}
	for _, f := range wsFiles {
		path := filepath.Join(inst.WorkspaceDir, f)
		if _, err := os.Stat(path); err == nil {
			inst.WorkspaceFiles[f] = true
		}
	}

	// Check binary
	if path, err := exec.LookPath("picoclaw"); err == nil {
		inst.BinaryPath = path
		if out, err := exec.Command("picoclaw", "--version").Output(); err == nil {
			inst.Version = strings.TrimSpace(string(out))
		}
	}

	return inst
}

// GetProviderKeys extracts provider API key names from OpenClaw config
func GetProviderKeys(config map[string]interface{}) []string {
	var keys []string
	if providers, ok := config["providers"].(map[string]interface{}); ok {
		for name := range providers {
			keys = append(keys, name)
		}
	}
	return keys
}

// GetConfiguredChannels returns channel names that are enabled
func GetConfiguredChannels(config map[string]interface{}) []string {
	var channels []string
	if ch, ok := config["channels"].(map[string]interface{}); ok {
		for name, v := range ch {
			if chConf, ok := v.(map[string]interface{}); ok {
				if enabled, ok := chConf["enabled"].(bool); ok && enabled {
					channels = append(channels, name)
				}
			}
		}
	}
	return channels
}

// GetMCPServers returns MCP server names from config
func GetMCPServers(config map[string]interface{}) []string {
	var servers []string

	// Check mcp_servers array
	if mcpArr, ok := config["mcp_servers"].([]interface{}); ok {
		for _, s := range mcpArr {
			if srv, ok := s.(map[string]interface{}); ok {
				if name, ok := srv["name"].(string); ok {
					servers = append(servers, name)
				}
			}
		}
	}

	// Check mcpServers (camelCase variant)
	if mcpArr, ok := config["mcpServers"].([]interface{}); ok {
		for _, s := range mcpArr {
			if srv, ok := s.(map[string]interface{}); ok {
				if name, ok := srv["name"].(string); ok {
					servers = append(servers, name)
				}
			}
		}
	}

	return servers
}

// helpers

func parseJSONFile(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

func dirHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// CountFileLines counts lines in a file
func CountFileLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return len(strings.Split(string(data), "\n"))
}
