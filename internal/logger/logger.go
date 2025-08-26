package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	PERFORMANCE
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case PERFORMANCE:
		return "PERF"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志记录器
type Logger struct {
	mu       sync.RWMutex
	logDir   string
	logFiles map[string]*os.File
	enabled  bool
}

// NewLogger 创建新的日志记录器
func NewLogger(logDir string) *Logger {
	return &Logger{
		logDir:   logDir,
		logFiles: make(map[string]*os.File),
		enabled:  true,
	}
}

// getLogFileName 获取日志文件名
func (l *Logger) getLogFileName(level LogLevel) string {
	now := time.Now()
	dateStr := now.Format("20060102")
	
	switch level {
	case PERFORMANCE:
		return fmt.Sprintf("performance_%s.log", dateStr)
	case ERROR:
		return fmt.Sprintf("error_%s.log", dateStr)
	default:
		return fmt.Sprintf("general_%s.log", dateStr)
	}
}

// getLogFile 获取或创建日志文件
func (l *Logger) getLogFile(level LogLevel) (*os.File, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	fileName := l.getLogFileName(level)
	filePath := filepath.Join(l.logDir, fileName)
	
	// 检查是否已经打开了该文件
	if file, exists := l.logFiles[fileName]; exists {
		return file, nil
	}
	
	// 确保日志目录存在
	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}
	
	// 打开或创建日志文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %v", err)
	}
	
	// 缓存文件句柄
	l.logFiles[fileName] = file
	return file, nil
}

// writeToFile 写入日志到文件
func (l *Logger) writeToFile(level LogLevel, message string) {
	if !l.enabled {
		return
	}
	
	file, err := l.getLogFile(level)
	if err != nil {
		log.Printf("获取日志文件失败: %v", err)
		return
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level.String(), message)
	
	if _, err := file.WriteString(logEntry); err != nil {
		log.Printf("写入日志文件失败: %v", err)
	}
	
	// 立即刷新到磁盘
	file.Sync()
}

// Log 记录日志（同时输出到控制台和文件）
func (l *Logger) Log(level LogLevel, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	
	// 输出到控制台
	log.Printf("[%s] %s", level.String(), message)
	
	// 写入到文件
	l.writeToFile(level, message)
}

// Debug 记录调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log(DEBUG, format, args...)
}

// Info 记录信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(INFO, format, args...)
}

// Warn 记录警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.Log(WARN, format, args...)
}

// Error 记录错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(ERROR, format, args...)
}

// Performance 记录性能日志
func (l *Logger) Performance(format string, args ...interface{}) {
	l.Log(PERFORMANCE, format, args...)
}

// Close 关闭所有日志文件
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for _, file := range l.logFiles {
		file.Close()
	}
	l.logFiles = make(map[string]*os.File)
}

// SetEnabled 设置日志是否启用
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// 全局日志实例
var globalLogger *Logger
var once sync.Once

// InitLogger 初始化全局日志记录器
func InitLogger(logDir string) {
	once.Do(func() {
		globalLogger = NewLogger(logDir)
	})
}

// GetLogger 获取全局日志记录器
func GetLogger() *Logger {
	if globalLogger == nil {
		InitLogger("logs")
	}
	return globalLogger
}

// 便捷函数
func Debug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	GetLogger().Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

func Performance(format string, args ...interface{}) {
	GetLogger().Performance(format, args...)
}