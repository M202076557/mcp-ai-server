package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DatabaseConnectionConfig 单个数据库连接配置
type DatabaseConnectionConfig struct {
	Alias       string `yaml:"alias"`
	Driver      string `yaml:"driver"`
	DSN         string `yaml:"dsn"`
	Description string `yaml:"description"`
}

// DatabaseConnectionsConfig 数据库连接配置结构
type DatabaseConnectionsConfig struct {
	MaxPoolSize     int                                    `yaml:"max_pool_size"`
	MaxIdleConns    int                                    `yaml:"max_idle_conns"`
	ConnMaxLifetime string                                 `yaml:"conn_max_lifetime"`
	Connections     map[string]DatabaseConnectionConfig   `yaml:",inline"`
}

// DatabaseConfig 数据库配置结构
type DatabaseConfig struct {
	Tools struct {
		Database struct {
			Connections DatabaseConnectionsConfig `yaml:"connections"`
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

	defaultConn, exists := dcm.config.Tools.Database.Connections.Connections["default"]
	if !exists {
		return "", "", "", fmt.Errorf("默认数据库连接配置不存在")
	}

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
	defaultConn, exists := dcm.config.Tools.Database.Connections.Connections["default"]
	if !exists {
		return "mysql_test" // 默认值
	}
	return defaultConn.Alias
}

// GetConnection 根据连接名获取数据库连接配置
func (dcm *DatabaseConfigManager) GetConnection(name string) (alias, driver, dsn string, err error) {
	if dcm.config == nil {
		return "", "", "", fmt.Errorf("配置未初始化")
	}

	conn, exists := dcm.config.Tools.Database.Connections.Connections[name]
	if !exists {
		return "", "", "", fmt.Errorf("数据库连接配置 '%s' 不存在", name)
	}

	if conn.Alias == "" || conn.Driver == "" || conn.DSN == "" {
		return "", "", "", fmt.Errorf("数据库连接配置 '%s' 不完整", name)
	}

	return conn.Alias, conn.Driver, conn.DSN, nil
}

// GetAvailableAliases 获取所有可用的数据库连接别名
func (dcm *DatabaseConfigManager) GetAvailableAliases() []string {
	if dcm.config == nil {
		return []string{}
	}

	var aliases []string
	for _, conn := range dcm.config.Tools.Database.Connections.Connections {
		if conn.Alias != "" {
			aliases = append(aliases, conn.Alias)
		}
	}
	return aliases
}
