package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// AIConfig AI配置结构
type AIConfig struct {
	Tools ToolsConfig `yaml:"tools"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	AI AIToolsConfig `yaml:"ai"`
}

// AIToolsConfig AI工具配置
type AIToolsConfig struct {
	Enabled         bool                      `yaml:"enabled"`
	Description     string                    `yaml:"description"`
	DefaultProvider string                    `yaml:"default_provider"`
	DefaultModel    string                    `yaml:"default_model"`
	Common          CommonConfig              `yaml:"common"`
	FunctionModels  map[string]FunctionModel  `yaml:"function_models"`
	Ollama          ProviderConfig            `yaml:"ollama"`
	OpenAI          ProviderConfig            `yaml:"openai"`
	Anthropic       ProviderConfig            `yaml:"anthropic"`
}

// FunctionModel 功能特定模型配置
type FunctionModel struct {
	Provider    string `yaml:"provider"`
	Model       string `yaml:"model"`
	Description string `yaml:"description"`
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	Enabled bool     `yaml:"enabled"`
	BaseURL string   `yaml:"base_url"`
	APIKey  string   `yaml:"api_key"`
	Models  []string `yaml:"models"`
}

// CommonConfig 通用配置
type CommonConfig struct {
	Timeout     int     `yaml:"timeout"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// AIConfigManager AI配置管理器
type AIConfigManager struct {
	config *AIToolsConfig
}

// NewAIConfigManager 创建新的AI配置管理器
func NewAIConfigManager(configPath string) (*AIConfigManager, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config AIConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 处理环境变量替换（仅对启用的提供商）
	if err := config.Tools.AI.processEnvironmentVariables(); err != nil {
		return nil, fmt.Errorf("处理环境变量失败: %v", err)
	}

	return &AIConfigManager{
		config: &config.Tools.AI,
	}, nil
}

// processEnvironmentVariables 处理环境变量替换（仅对启用的提供商）
func (ai *AIToolsConfig) processEnvironmentVariables() error {
	// OpenAI 仅在启用时解析 API Key
	if ai.OpenAI.Enabled && ai.OpenAI.APIKey != "" && strings.HasPrefix(ai.OpenAI.APIKey, "${") && strings.HasSuffix(ai.OpenAI.APIKey, "}") {
		envVar := strings.Trim(ai.OpenAI.APIKey, "${}")
		if apiKey := os.Getenv(envVar); apiKey != "" {
			ai.OpenAI.APIKey = apiKey
		} else {
			return fmt.Errorf("环境变量 %s 未设置", envVar)
		}
	}

	// Anthropic 仅在启用时解析 API Key
	if ai.Anthropic.Enabled && ai.Anthropic.APIKey != "" && strings.HasPrefix(ai.Anthropic.APIKey, "${") && strings.HasSuffix(ai.Anthropic.APIKey, "}") {
		envVar := strings.Trim(ai.Anthropic.APIKey, "${}")
		if apiKey := os.Getenv(envVar); apiKey != "" {
			ai.Anthropic.APIKey = apiKey
		} else {
			return fmt.Errorf("环境变量 %s 未设置", envVar)
		}
	}

	return nil
}

// GetProvider 获取指定提供商配置
func (m *AIConfigManager) GetProvider(name string) (*ProviderConfig, bool) {
	switch name {
	case "ollama":
		return &m.config.Ollama, m.config.Ollama.Enabled
	case "openai":
		return &m.config.OpenAI, m.config.OpenAI.Enabled
	case "anthropic":
		return &m.config.Anthropic, m.config.Anthropic.Enabled
	default:
		return nil, false
	}
}

// GetDefaultProvider 获取默认提供商
func (m *AIConfigManager) GetDefaultProvider() string {
	return m.config.DefaultProvider
}

// GetDefaultModel 获取默认模型
func (m *AIConfigManager) GetDefaultModel() string {
	return m.config.DefaultModel
}

// GetCommonConfig 获取通用配置
func (m *AIConfigManager) GetCommonConfig() *CommonConfig {
	return &m.config.Common
}

// IsProviderEnabled 检查提供商是否启用
func (m *AIConfigManager) IsProviderEnabled(name string) bool {
	provider, exists := m.GetProvider(name)
	return exists && provider.Enabled
}

// GetAvailableProviders 获取可用的提供商列表
func (m *AIConfigManager) GetAvailableProviders() []string {
	var providers []string

	if m.config.Ollama.Enabled {
		providers = append(providers, "ollama")
	}
	if m.config.OpenAI.Enabled {
		providers = append(providers, "openai")
	}
	if m.config.Anthropic.Enabled {
		providers = append(providers, "anthropic")
	}

	return providers
}

// GetProviderModels 获取指定提供商的可用模型
func (m *AIConfigManager) GetProviderModels(providerName string) []string {
	if provider, exists := m.GetProvider(providerName); exists && provider.Enabled {
		return provider.Models
	}
	return nil
}

// GetFunctionModel 获取指定功能的模型配置
func (m *AIConfigManager) GetFunctionModel(function string) (string, string, bool) {
	if m.config.FunctionModels == nil {
		// 如果没有配置功能特定模型，使用默认配置
		return m.config.DefaultProvider, m.config.DefaultModel, true
	}
	
	if functionModel, exists := m.config.FunctionModels[function]; exists {
		return functionModel.Provider, functionModel.Model, true
	}
	
	// 如果没有找到特定功能的配置，使用默认配置
	return m.config.DefaultProvider, m.config.DefaultModel, true
}

// GetAvailableFunctions 获取已配置的功能列表
func (m *AIConfigManager) GetAvailableFunctions() []string {
	if m.config.FunctionModels == nil {
		return []string{}
	}
	
	var functions []string
	for function := range m.config.FunctionModels {
		functions = append(functions, function)
	}
	return functions
}
