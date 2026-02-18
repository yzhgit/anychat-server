package websocket

import (
	"encoding/json"
	"sync"

	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
)

// Manager WebSocket连接管理器
type Manager struct {
	clients map[string]*Client // userID -> client
	mu      sync.RWMutex
}

// NewManager 创建WebSocket连接管理器
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// Register 注册新客户端，若同一用户已有连接则关闭旧连接
func (m *Manager) Register(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, exists := m.clients[client.UserID]; exists {
		close(old.Done)
		logger.Info("Replaced existing WebSocket connection", zap.String("userID", client.UserID))
	}

	m.clients[client.UserID] = client
	logger.Info("WebSocket client registered", zap.String("userID", client.UserID))
}

// Unregister 注销客户端（仅当传入的client是当前活跃client时才注销）
func (m *Manager) Unregister(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if current, exists := m.clients[client.UserID]; exists && current == client {
		delete(m.clients, client.UserID)
		logger.Info("WebSocket client unregistered", zap.String("userID", client.UserID))
	}
}

// SendToUser 向指定用户发送原始消息，返回是否成功
func (m *Manager) SendToUser(userID string, data []byte) bool {
	m.mu.RLock()
	client, exists := m.clients[userID]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	select {
	case client.Send <- data:
		return true
	default:
		logger.Warn("WebSocket send buffer full, dropping message", zap.String("userID", userID))
		return false
	}
}

// SendMessageToUser 向指定用户发送结构化消息
func (m *Manager) SendMessageToUser(userID string, msg *Message) bool {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("Failed to marshal WebSocket message", zap.Error(err))
		return false
	}
	return m.SendToUser(userID, data)
}

// IsOnline 检查用户是否在线
func (m *Manager) IsOnline(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.clients[userID]
	return exists
}

// OnlineCount 获取当前在线用户数
func (m *Manager) OnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}
