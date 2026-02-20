package detect

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Installation holds detected installation info
type Installation struct {
	Found          bool
	HomeDir        string // e.g. ~/.openclaw or ~/.picoclaw
	ConfigPath     string
	WorkspaceDir   string
	BinaryPath     string
	Version        string
	WorkspaceFiles map[string]bool // which standard workspace files exist
	ExtraFiles     []string        // non-standard .md files in workspace root
	ExtraDirs      []string        // non-standard directories in workspace root
	HasMemory      bool
	HasSkills      bool
	HasCron        bool
	HasSessions    bool
	Config         map[string]interface{} // parsed JSON config
	ConfigSummary  ConfigSummary          // human-readable config overview
}

// WorkspaceItem describes a file or directory in the workspace
type WorkspaceItem struct {
	Name    string
	IsDir   bool
	Lines   int   // for files
	Files   int   // for directories (recursive count)
	Size    int64 // total size in bytes
}

// ConfigSummary holds extracted config details for display
type ConfigSummary struct {
	DefaultModel      string
	MaxTokens         int
	Temperature       float64
	HeartbeatEnabled  bool
	HeartbeatInterval int
	WorkspacePath     string
	ConfigFileSize    int64
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

// StandardFiles are the well-known agent workspace files
var StandardFiles = map[string]bool{
	"SOUL.md": true, "IDENTITY.md": true, "AGENTS.md": true,
	"USER.md": true, "TOOLS.md": true, "HEARTBEAT.md": true,
}

// StandardDirs are the well-known workspace subdirectories
var StandardDirs = map[string]bool{
	"memory": true, "skills": true, "cron": true, "sessions": true,
	"state": true, "config": true, ".git": true, ".openclaw": true,
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
		inst.ConfigSummary = extractConfigSummary(inst.Config, inst.ConfigPath)
	}

	// Workspace
	inst.WorkspaceDir = filepath.Join(inst.HomeDir, "workspace")

	// Scan ALL workspace contents
	entries, err := os.ReadDir(inst.WorkspaceDir)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() {
				if !StandardDirs[name] {
					inst.ExtraDirs = append(inst.ExtraDirs, name)
				}
			} else {
				if StandardFiles[name] {
					inst.WorkspaceFiles[name] = true
				} else if name != ".DS_Store" && name != ".gitignore" {
					inst.ExtraFiles = append(inst.ExtraFiles, name)
				}
			}
		}
	}

	// Check standard subdirectories
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

// CountDirFiles recursively counts files in a directory
func CountDirFiles(path string) int {
	count := 0
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// DirSize returns total size of a directory in bytes
func DirSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
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

func extractConfigSummary(config map[string]interface{}, configPath string) ConfigSummary {
	cs := ConfigSummary{}

	// File size
	if info, err := os.Stat(configPath); err == nil {
		cs.ConfigFileSize = info.Size()
	}

	// Agent defaults
	if agent, ok := config["agent"].(map[string]interface{}); ok {
		if m, ok := agent["model"].(string); ok {
			cs.DefaultModel = m
		}
		if mt, ok := agent["maxTokens"].(float64); ok {
			cs.MaxTokens = int(mt)
		}
		if mt, ok := agent["max_tokens"].(float64); ok {
			cs.MaxTokens = int(mt)
		}
		if t, ok := agent["temperature"].(float64); ok {
			cs.Temperature = t
		}
	}
	// Try agents.defaults too
	if agents, ok := config["agents"].(map[string]interface{}); ok {
		if defaults, ok := agents["defaults"].(map[string]interface{}); ok {
			if m, ok := defaults["model"].(string); ok && cs.DefaultModel == "" {
				cs.DefaultModel = m
			}
			if mt, ok := defaults["maxTokens"].(float64); ok && cs.MaxTokens == 0 {
				cs.MaxTokens = int(mt)
			}
			if mt, ok := defaults["max_tokens"].(float64); ok && cs.MaxTokens == 0 {
				cs.MaxTokens = int(mt)
			}
			if w, ok := defaults["workspace"].(string); ok {
				cs.WorkspacePath = w
			}
		}
	}

	// Heartbeat
	if hb, ok := config["heartbeat"].(map[string]interface{}); ok {
		if enabled, ok := hb["enabled"].(bool); ok {
			cs.HeartbeatEnabled = enabled
		}
		if interval, ok := hb["interval"].(float64); ok {
			cs.HeartbeatInterval = int(interval)
		}
	}

	return cs
}
