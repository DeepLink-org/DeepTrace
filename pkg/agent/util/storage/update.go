// Copyright (c) OpenMMLab. All rights reserved.

package storage

import (
	"sync"
)

type PendingUpdateManager struct {
	updates map[string]map[string]EventUpdate // filePath -> eventID -> update
	mutex   sync.Mutex
}

type EventUpdate struct {
	Processed   bool
	ProcessedAt int64
}

func NewPendingUpdateManager() *PendingUpdateManager {
	return &PendingUpdateManager{
		updates: make(map[string]map[string]EventUpdate),
	}
}

// Add update request
func (m *PendingUpdateManager) AddUpdate(filePath, eventID string, update EventUpdate) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.updates[filePath]; !exists {
		m.updates[filePath] = make(map[string]EventUpdate)
	}

	m.updates[filePath][eventID] = update
}
