package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ConvertConfig converts OpenClaw config to PicoClaw config format
func ConvertConfig(openclawConfig map[string]interface{}) map[string]interface{} {
	picoConfig := make(map[string]interface{})

	// Convert providers → model_list (new format) + providers (legacy compat)
	convertProviders(openclawConfig, picoConfig)

	// Convert agent defaults
	convertAgentDefaults(openclawConfig, picoConfig)

	// Convert channels
	convertChannels(openclawConfig, picoConfig)

	// Convert tools
	convertTools(openclawConfig, picoConfig)

	// Convert heartbeat
	convertHeartbeat(openclawConfig, picoConfig)

	// Convert MCP servers
	convertMCPServers(openclawConfig, picoConfig)

	return picoConfig
}

// MergeConfig merges converted config into existing PicoClaw config
func MergeConfig(existing, incoming map[string]interface{}) map[string]interface{} {
	if existing == nil {
		return incoming
	}
	merged := make(map[string]interface{})

	// Copy existing
	for k, v := range existing {
		merged[k] = v
	}

	// Merge incoming (incoming wins for new keys, deep merge for objects)
	for k, v := range incoming {
		if existVal, ok := merged[k]; ok {
			// Deep merge maps
			if existMap, isMap := existVal.(map[string]interface{}); isMap {
				if inMap, isMap2 := v.(map[string]interface{}); isMap2 {
					merged[k] = deepMerge(existMap, inMap)
					continue
				}
			}
		}
		merged[k] = v
	}

	return merged
}

// WriteConfig writes config to a file
func WriteConfig(config map[string]interface{}, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ReadConfig reads and parses a JSON config file
func ReadConfig(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// --- Internal conversion functions ---

func convertProviders(src, dst map[string]interface{}) {
	providers, ok := src["providers"].(map[string]interface{})
	if !ok {
		return
	}

	// Build model_list for new format
	var modelList []map[string]interface{}

	// Also preserve legacy providers format
	picoProviders := make(map[string]interface{})

	// Provider mapping: OpenClaw name → PicoClaw vendor prefix
	vendorMap := map[string]string{
		"openrouter": "openrouter",
		"anthropic":  "anthropic",
		"openai":     "openai",
		"gemini":     "gemini",
		"zhipu":      "zhipu",
		"groq":       "groq",
		"deepseek":   "deepseek",
		"ollama":     "ollama",
	}

	// Default model for each vendor
	defaultModels := map[string]string{
		"openrouter": "openrouter/anthropic/claude-sonnet-4.6",
		"anthropic":  "anthropic/claude-sonnet-4.6",
		"openai":     "openai/gpt-5.2",
		"gemini":     "gemini/gemini-2.0-flash",
		"zhipu":      "zhipu/glm-4.7",
		"groq":       "groq/llama-3.3-70b-versatile",
		"deepseek":   "deepseek/deepseek-chat",
		"ollama":     "ollama/llama3",
	}

	for name, v := range providers {
		provConf, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		apiKey, _ := provConf["api_key"].(string)
		if apiKey == "" {
			apiKey, _ = provConf["apiKey"].(string) // camelCase variant
		}
		apiBase, _ := provConf["api_base"].(string)
		if apiBase == "" {
			apiBase, _ = provConf["apiBase"].(string)
		}

		// Legacy providers format
		picoProvider := make(map[string]interface{})
		if apiKey != "" {
			picoProvider["api_key"] = apiKey
		}
		if apiBase != "" {
			picoProvider["api_base"] = apiBase
		}
		picoProviders[name] = picoProvider

		// New model_list format
		if vendorPrefix, ok := vendorMap[name]; ok {
			modelEntry := map[string]interface{}{
				"model_name": name,
				"model":      defaultModels[vendorPrefix],
			}
			if apiKey != "" {
				modelEntry["api_key"] = apiKey
			}
			if apiBase != "" {
				modelEntry["api_base"] = apiBase
			}
			modelList = append(modelList, modelEntry)
		}
	}

	if len(modelList) > 0 {
		dst["model_list"] = modelList
	}
	if len(picoProviders) > 0 {
		dst["providers"] = picoProviders
	}
}

func convertAgentDefaults(src, dst map[string]interface{}) {
	agent, ok := src["agent"].(map[string]interface{})
	if !ok {
		// Try agents.defaults
		if agents, ok := src["agents"].(map[string]interface{}); ok {
			agent, ok = agents["defaults"].(map[string]interface{})
			if !ok {
				return
			}
		} else {
			return
		}
	}

	picoAgent := map[string]interface{}{
		"defaults": map[string]interface{}{
			"workspace": "~/.picoclaw/workspace",
		},
	}
	defaults := picoAgent["defaults"].(map[string]interface{})

	// Handle model field specially — it can be a string OR an object
	if model, ok := agent["model"]; ok {
		switch m := model.(type) {
		case string:
			// Already a string — use as-is
			if m != "" {
				defaults["model"] = m
			}
		case map[string]interface{}:
			// Object like {"primary": "anthropic/claude-sonnet-4-5"}
			// Extract the string value from known keys
			for _, key := range []string{"primary", "name", "model", "default"} {
				if v, ok := m[key].(string); ok && v != "" {
					defaults["model"] = v
					break
				}
			}
		}
	}

	// Map other known fields (camelCase → snake_case), skip model (handled above)
	fieldMap := map[string]string{
		"max_tokens":          "max_tokens",
		"maxTokens":           "max_tokens",
		"temperature":         "temperature",
		"max_tool_iterations": "max_tool_iterations",
		"maxToolIterations":   "max_tool_iterations",
	}

	for srcKey, dstKey := range fieldMap {
		if v, ok := agent[srcKey]; ok {
			// Only set numeric values that are non-zero
			switch val := v.(type) {
			case float64:
				if val > 0 {
					defaults[dstKey] = v
				}
			case string:
				if val != "" {
					defaults[dstKey] = v
				}
			default:
				defaults[dstKey] = v
			}
		}
	}

	dst["agents"] = picoAgent
}

func convertChannels(src, dst map[string]interface{}) {
	channels, ok := src["channels"].(map[string]interface{})
	if !ok {
		return
	}

	picoChannels := make(map[string]interface{})

	// Supported PicoClaw channels
	supported := map[string]bool{
		"telegram": true, "discord": true, "qq": true,
		"dingtalk": true, "line": true, "slack": true,
		"feishu": true, "onebot": true,
	}

	for name, v := range channels {
		if !supported[name] {
			continue // skip unsupported channels (whatsapp, signal, etc.)
		}
		chConf, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		picoChannel := make(map[string]interface{})
		// Copy all fields, converting camelCase to snake_case
		for k, val := range chConf {
			picoChannel[camelToSnake(k)] = val
		}
		picoChannels[name] = picoChannel
	}

	if len(picoChannels) > 0 {
		dst["channels"] = picoChannels
	}
}

func convertTools(src, dst map[string]interface{}) {
	tools, ok := src["tools"].(map[string]interface{})
	if !ok {
		return
	}

	picoTools := make(map[string]interface{})

	// Web search tools
	if web, ok := tools["web"].(map[string]interface{}); ok {
		picoWeb := make(map[string]interface{})
		if brave, ok := web["brave"].(map[string]interface{}); ok {
			picoWeb["brave"] = brave
		}
		// DuckDuckGo enabled by default in PicoClaw
		picoWeb["duckduckgo"] = map[string]interface{}{
			"enabled":     true,
			"max_results": 5,
		}
		picoTools["web"] = picoWeb
	}

	// Cron tools
	if cron, ok := tools["cron"].(map[string]interface{}); ok {
		picoTools["cron"] = cron
	}

	if len(picoTools) > 0 {
		dst["tools"] = picoTools
	}
}

func convertHeartbeat(src, dst map[string]interface{}) {
	heartbeat, ok := src["heartbeat"].(map[string]interface{})
	if !ok {
		// Default heartbeat
		dst["heartbeat"] = map[string]interface{}{
			"enabled":  true,
			"interval": 30,
		}
		return
	}

	picoHeartbeat := map[string]interface{}{
		"enabled":  true,
		"interval": 30,
	}

	if enabled, ok := heartbeat["enabled"].(bool); ok {
		picoHeartbeat["enabled"] = enabled
	}
	if interval, ok := heartbeat["interval"].(float64); ok {
		picoHeartbeat["interval"] = interval
	}

	dst["heartbeat"] = picoHeartbeat
}

func convertMCPServers(src, dst map[string]interface{}) {
	// Try both camelCase and snake_case
	var mcpServers []interface{}
	if s, ok := src["mcp_servers"].([]interface{}); ok {
		mcpServers = s
	} else if s, ok := src["mcpServers"].([]interface{}); ok {
		mcpServers = s
	}

	if len(mcpServers) > 0 {
		dst["mcp_servers"] = mcpServers
	}
}

// --- Helpers ---

func camelToSnake(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

func deepMerge(base, overlay map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		if baseVal, ok := merged[k]; ok {
			if baseMap, isMap := baseVal.(map[string]interface{}); isMap {
				if overlayMap, isMap2 := v.(map[string]interface{}); isMap2 {
					merged[k] = deepMerge(baseMap, overlayMap)
					continue
				}
			}
		}
		merged[k] = v
	}
	return merged
}