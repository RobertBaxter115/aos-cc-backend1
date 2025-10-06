package user

import (
	"sync"
	"time"
)

// UserActivity represents a user activity event
type UserActivity struct {
	UserID    string
	Timestamp int64
}

// UserState tracks the activity state of a user
type UserState struct {
	UserID           string
	LastActivityTime time.Time
	ActivityCount    int // Count of activities in the prediction window
	IsConnected      bool
	AllocatedNodeID  string
}

// UserTracker tracks user activities and states
type UserTracker struct {
	mu     sync.RWMutex
	users  map[string]*UserState
	window time.Duration // Time window for tracking activity
}

// NewUserTracker creates a new user tracker
func NewUserTracker(activityWindow time.Duration) *UserTracker {
	return &UserTracker{
		users:  make(map[string]*UserState),
		window: activityWindow,
	}
}

// RecordActivity records a user activity
func (t *UserTracker) RecordActivity(userID string, timestamp time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.users[userID]
	if !exists {
		state = &UserState{
			UserID:           userID,
			LastActivityTime: timestamp,
			ActivityCount:    1,
		}
		t.users[userID] = state
	} else {
		state.LastActivityTime = timestamp
		state.ActivityCount++
	}
}

// GetUserState retrieves the current state of a user
func (t *UserTracker) GetUserState(userID string) (*UserState, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state, ok := t.users[userID]
	return state, ok
}

// MarkConnected marks a user as connected
func (t *UserTracker) MarkConnected(userID, nodeID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.users[userID]
	if !exists {
		state = &UserState{
			UserID: userID,
		}
		t.users[userID] = state
	}

	state.IsConnected = true
	state.AllocatedNodeID = nodeID
}

// MarkDisconnected marks a user as disconnected
func (t *UserTracker) MarkDisconnected(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if state, exists := t.users[userID]; exists {
		state.IsConnected = false
		state.AllocatedNodeID = ""
	}
}

// GetActiveUsers returns users who have been active recently
func (t *UserTracker) GetActiveUsers(since time.Time) []*UserState {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var active []*UserState
	for _, state := range t.users {
		if state.LastActivityTime.After(since) {
			active = append(active, state)
		}
	}
	return active
}

// GetLikelyToConnect returns users who are likely to connect based on activity
func (t *UserTracker) GetLikelyToConnect(threshold int, within time.Duration) []*UserState {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := time.Now().Add(-within)
	var likely []*UserState

	for _, state := range t.users {
		if !state.IsConnected &&
			state.LastActivityTime.After(cutoff) &&
			state.ActivityCount >= threshold {
			likely = append(likely, state)
		}
	}
	return likely
}

// CleanupOldActivity removes old activity records
func (t *UserTracker) CleanupOldActivity(before time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for userID, state := range t.users {
		if !state.IsConnected && state.LastActivityTime.Before(before) {
			delete(t.users, userID)
		}
	}
}

// GetConnectedUsers returns all currently connected users
func (t *UserTracker) GetConnectedUsers() []*UserState {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var connected []*UserState
	for _, state := range t.users {
		if state.IsConnected {
			connected = append(connected, state)
		}
	}
	return connected
}

// ResetActivityCount resets the activity count for a user
func (t *UserTracker) ResetActivityCount(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if state, exists := t.users[userID]; exists {
		state.ActivityCount = 0
	}
}
