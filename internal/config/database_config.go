package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DatabaseConfig 数据库配置结构
type DatabaseConfig struct {
	Tools struct {
		Database struct {
			Connections struct {
				Default struct {
					Alias       string `yaml:"alias"`
					Driver      string `yaml:"driver"`
					DSN         string `yaml:"dsn"`
					Description string `yaml:"description"`
				} `yaml:"default"`
			} `yaml:"connections"`
		} `yaml:"database"`
	} `yaml:"tools"`
}

// DatabaseConfigManager 数据库配置管理器
type DatabaseConfigManager struct {
	config *DatabaseConfig
}

// NewDatabaseConfigManager 创建数据库配置管理器
func NewDatabaseConfigManager(configPath string) (*DatabaseConfigManager, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config DatabaseConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &DatabaseConfigManager{
		config: &config,
	}, nil
}

// GetDefaultConnection 获取默认数据库连接配置
func (dcm *DatabaseConfigManager) GetDefaultConnection() (alias, driver, dsn string, err error) {
	if dcm.config == nil {
		return "", "", "", fmt.Errorf("配置未初始化")
	}

	defaultConn := dcm.config.Tools.Database.Connections.Default
	if defaultConn.Alias == "" || defaultConn.Driver == "" || defaultConn.DSN == "" {
		return "", "", "", fmt.Errorf("默认数据库连接配置不完整")
	}

	return defaultConn.Alias, defaultConn.Driver, defaultConn.DSN, nil
}

// GetDefaultAlias 获取默认连接别名
func (dcm *DatabaseConfigManager) GetDefaultAlias() string {
	if dcm.config == nil {
		return "mysql_test" // 默认值
	}
	return dcm.config.Tools.Database.Connections.Default.Alias
}
