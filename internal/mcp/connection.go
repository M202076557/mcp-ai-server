package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

// stdioConnection stdio连接管理
type stdioConnection struct {
	reader     io.Reader
	writer     io.Writer
	scanner    *bufio.Scanner
	mu         sync.RWMutex
	running    bool
	responses  map[interface{}]*Message
	responseCh chan *Message
	stopCh     chan struct{}
}

// newStdioConnection 创建新的stdio连接
func newStdioConnection(reader io.Reader, writer io.Writer) *stdioConnection {
	return &stdioConnection{
		reader:     reader,
		writer:     writer,
		scanner:    bufio.NewScanner(reader),
		responses:  make(map[interface{}]*Message),
		responseCh: make(chan *Message, 100),
		stopCh:     make(chan struct{}),
	}
}

// Start 启动连接
func (c *stdioConnection) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("连接已在运行")
	}

	c.running = true

	// 启动消息读取循环
	go c.readLoop()

	// 启动响应处理循环
	go c.responseLoop()

	return nil
}

// Stop 停止连接
func (c *stdioConnection) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.running = false
	close(c.stopCh)

	return nil
}

// IsRunning 检查是否正在运行
func (c *stdioConnection) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.running
}

// SendMessage 发送消息
func (c *stdioConnection) SendMessage(msg *Message) error {
	if !c.IsRunning() {
		return fmt.Errorf("连接未运行")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	data = append(data, '\n')

	_, err = c.writer.Write(data)
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	// 如果是请求消息，记录等待响应
	if msg.IsRequest() {
		c.mu.Lock()
		c.responses[msg.ID] = nil // 标记为等待响应
		c.mu.Unlock()
	}

	return nil
}

// readLoop 消息读取循环
func (c *stdioConnection) readLoop() {
	for c.IsRunning() {
		if !c.scanner.Scan() {
			if err := c.scanner.Err(); err != nil {
				log.Printf("读取消息错误: %v", err)
			}
			break
		}

		line := c.scanner.Text()
		if line == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 发送到响应通道
		select {
		case c.responseCh <- &msg:
		case <-c.stopCh:
			return
		default:
			log.Printf("响应通道已满，丢弃消息")
		}
	}
}

// responseLoop 响应处理循环
func (c *stdioConnection) responseLoop() {
	for {
		select {
		case msg := <-c.responseCh:
			c.handleResponse(msg)
		case <-c.stopCh:
			return
		}
	}
}

// handleResponse 处理响应消息
func (c *stdioConnection) handleResponse(msg *Message) {
	if msg.IsResponse() {
		c.mu.Lock()
		if _, exists := c.responses[msg.ID]; exists {
			c.responses[msg.ID] = msg
		}
		c.mu.Unlock()
	}
}

// WaitForResponse 等待指定ID的响应
func (c *stdioConnection) WaitForResponse(id interface{}, timeout time.Duration) *Message {
	start := time.Now()

	for time.Since(start) < timeout {
		c.mu.RLock()
		if response, exists := c.responses[id]; exists && response != nil {
			c.mu.RUnlock()

			// 清理响应
			c.mu.Lock()
			delete(c.responses, id)
			c.mu.Unlock()

			return response
		}
		c.mu.RUnlock()

		time.Sleep(10 * time.Millisecond)
	}

	return nil
}
