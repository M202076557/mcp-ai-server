package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SecurityConfig 安全配置结构
type SecurityConfig struct {
	Security SecuritySettings `yaml:"security"`
}

// SecuritySettings 安全设置
type SecuritySettings struct {
	AllowedPaths    []string        `yaml:"allowed_paths"`
	BlockedPaths    []string        `yaml:"blocked_paths"`
	PathRules       PathRules       `yaml:"path_rules"`
	CommandSecurity CommandSecurity `yaml:"command_security"`
	ResourceLimits  ResourceLimits  `yaml:"resource_limits"`
}

// PathRules 路径规则
type PathRules struct {
	AllowRelativePaths bool     `yaml:"allow_relative_paths"`
	AllowAbsolutePaths bool     `yaml:"allow_absolute_paths"`
	MaxPathDepth       int      `yaml:"max_path_depth"`
	AllowedExtensions  []string `yaml:"allowed_extensions"`
	BlockedExtensions  []string `yaml:"blocked_extensions"`
}

// CommandSecurity 命令安全设置
type CommandSecurity struct {
	AllowedCommands []string `yaml:"allowed_commands"`
	BlockedCommands []string `yaml:"blocked_commands"`
}

// ResourceLimits 资源限制
type ResourceLimits struct {
	MaxFileSize       int64 `yaml:"max_file_size"`
	MaxDirectoryItems int   `yaml:"max_directory_items"`
	MaxCommandOutput  int64 `yaml:"max_command_output"`
}

// SecurityManager 安全管理器
type SecurityManager struct {
	config *SecurityConfig
}

// NewSecurityManager 创建新的安全管理器
func NewSecurityManager(configPath string) (*SecurityManager, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config SecurityConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &SecurityManager{
		config: &config,
	}, nil
}

// IsPathAllowed 检查路径是否允许访问
func (sm *SecurityManager) IsPathAllowed(path string) error {
	// 简化实现：允许当前目录和子目录
	if strings.Contains(path, "..") {
		return fmt.Errorf("不安全的文件路径")
	}
	return nil
}

// CheckFileSize 检查文件大小
func (sm *SecurityManager) CheckFileSize(size int64) error {
	// 简化实现：允许最大100MB
	if size > 100*1024*1024 {
		return fmt.Errorf("文件大小超过限制")
	}
	return nil
}

// IsCommandAllowed 检查命令是否允许执行
func (sm *SecurityManager) IsCommandAllowed(command string) error {
	// 简化实现：允许基本命令
	allowedCommands := []string{"ls", "cat", "echo", "pwd", "whoami", "date", "ps", "top", "mkdir", "rm", "cp", "mv"}
	for _, allowed := range allowedCommands {
		if command == allowed {
			return nil
		}
	}
	return fmt.Errorf("命令 %s 不被允许执行", command)
}

// CheckCommandOutput 检查命令输出大小
func (sm *SecurityManager) CheckCommandOutput(size int64) error {
	// 简化实现：允许最大10MB输出
	if size > 10*1024*1024 {
		return fmt.Errorf("命令输出大小超过限制")
	}
	return nil
}

// CheckDirectoryItems 检查目录项数
func (sm *SecurityManager) CheckDirectoryItems(count int) error {
	// 简化实现：允许最大1000个目录项
	if count > 1000 {
		return fmt.Errorf("目录项数超过限制")
	}
	return nil
}
