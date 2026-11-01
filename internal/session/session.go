// Package session 管理内存中的用户会话。
package session

import (
	"sync"

	"quiz/internal/quiz"
)

// Session 是单个用户的内存会话状态。
type Session struct {
	UserID string

	mu                   sync.Mutex
	Mode                 string // "review" / "quiz" / "incorrect_review"
	Course               string
	CurrentQuestions     []quiz.OutputQuestion // 速刷模式逐步取题时使用
	CurrentQuestionIndex int
}

// Lock/Unlock 透传内部锁，供需要修改会话状态的调用方使用。
func (s *Session) Lock()   { s.mu.Lock() }
func (s *Session) Unlock() { s.mu.Unlock() }

// Manager 线程安全地持有所有用户会话。
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewManager 创建会话管理器。
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

// Get 获取或创建用户会话。
func (m *Manager) Get(userID string) *Session {
	m.mu.RLock()
	s, ok := m.sessions[userID]
	m.mu.RUnlock()
	if ok {
		return s
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok = m.sessions[userID]; ok {
		return s
	}
	s = &Session{UserID: userID}
	m.sessions[userID] = s
	return s
}

// Clear 删除用户会话。
func (m *Manager) Clear(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, userID)
}
