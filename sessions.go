package main

import "sync"

var (
	userSessions map[string]*UserSession
	sessionsMu   sync.RWMutex
)

func initSessions() {
	userSessions = make(map[string]*UserSession)
}

func getOrCreateUserSession(userID string) *UserSession {
	sessionsMu.RLock()
	session, exists := userSessions[userID]
	sessionsMu.RUnlock()
	if exists {
		return session
	}

	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	session, exists = userSessions[userID]
	if exists {
		return session
	}

	newSession := &UserSession{UserID: userID}
	userSessions[userID] = newSession
	return newSession
}

func clearUserSession(userID string) {
	sessionsMu.Lock()
	delete(userSessions, userID)
	sessionsMu.Unlock()
}
