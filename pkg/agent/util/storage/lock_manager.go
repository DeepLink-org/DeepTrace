// Copyright (c) OpenMMLab. All rights reserved.

package storage

import (
	"sync"
)

// File lock manager
type FileLockManager struct {
	locks map[string]*sync.Mutex
	mutex sync.Mutex // Mutex to protect the locks map
}

func NewFileLockManager() *FileLockManager {
	return &FileLockManager{
		locks: make(map[string]*sync.Mutex),
	}
}

// Get file lock (thread-safe)
func (m *FileLockManager) GetLock(filePath string) *sync.Mutex {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Create lock if it doesn't exist
	if _, exists := m.locks[filePath]; !exists {
		m.locks[filePath] = &sync.Mutex{}
	}

	return m.locks[filePath]
}
