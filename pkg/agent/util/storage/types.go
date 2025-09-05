// Copyright (c) OpenMMLab. All rights reserved.

package storage

import "sync"

const (
	defaultBaseDir  = "/tmp"
	defaultMaxSize  = 10 * 1024 * 1024 // 10MB
	defaultFilePerm = 0644
	defaultDirPerm  = 0755
)

// EventEntry represents a generic event entry
type EventEntry struct {
	ID          string   `json:"id"`           // Unique event ID
	Source      string   `json:"source"`       // Event source ("training", "system")
	Type        string   `json:"type"`         // Event type ("alert", "audit", "metric")
	JobID       string   `json:"job_id"`       // Associated job ID
	Message     string   `json:"message"`      // Event message
	Timestamp   int64    `json:"timestamp"`    // Timestamp (milliseconds)
	Severity    int32    `json:"severity"`     // Severity level
	Metadata    Metadata `json:"metadata"`     // Extended metadata
	Processed   bool     `json:"processed"`    
	ProcessedAt int64    `json:"processed_at"` // Processing timestamp
}

// Metadata stores extended properties of events
type Metadata map[string]interface{}

// EventStorage manages event storage
type EventStorage struct {
	baseDir        string
	filePrefix     string
	maxFileSize    int64
	currentFile    string
	currentMutex   sync.Mutex
	indexMutex     sync.RWMutex
	fileIndexes    map[string]*FileIndex // File index
	pendingUpdates *PendingUpdateManager // Pending updates manager
	lockManager    *FileLockManager      // File lock manager
}

// FileIndex file index for accelerating queries
type FileIndex struct {
	Path         string
	MinTime      int64
	MaxTime      int64
	MaxSeverity  int32
	EventTypes   map[string]bool
	AllProcessed bool
}

// EventFilter defines event query filters
type EventFilter struct {
	StartTime   int64
	EndTime     int64
	MinSeverity int32
	Type        string
	Source      string
	JobID       string
	Unprocessed bool
}
