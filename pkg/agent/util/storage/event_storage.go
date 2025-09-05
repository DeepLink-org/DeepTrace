// Copyright (c) OpenMMLab. All rights reserved.

package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"deeptrace/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func NewEventStorage(baseDir string, maxFileSize int64) (*EventStorage, error) {
	if baseDir == "" {
		baseDir = getBaseDir()
	}
	if maxFileSize <= 0 {
		maxFileSize = defaultMaxSize
	}

	if err := os.MkdirAll(baseDir, defaultDirPerm); err != nil {
		return nil, err
	}

	storage := &EventStorage{
		baseDir:        baseDir,
		maxFileSize:    maxFileSize,
		fileIndexes:    make(map[string]*FileIndex),
		pendingUpdates: NewPendingUpdateManager(),
		lockManager:    NewFileLockManager(),
		filePrefix:     "rank" + os.Getenv("NODE_RANK") + "_events_",
	}

	if err := storage.initializeStorage(); err != nil {
		return nil, err
	}

	return storage, nil
}

// Convenience method to get file lock
func (s *EventStorage) getFileLock(filePath string) *sync.Mutex {
	return s.lockManager.GetLock(filePath)
}

func getBaseDir() string {
	workDir := os.Getenv("WORK_DIR")
	if workDir == "" {
		return defaultBaseDir
	}
	return workDir
}

func (s *EventStorage) initializeStorage() error {
	// Rebuild index
	files, err := os.ReadDir(s.baseDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" || !strings.Contains(file.Name(), s.filePrefix) {
			continue
		}
		path := filepath.Join(s.baseDir, file.Name())
		if err := s.indexFile(path); err != nil {
			logger.Logger.Info("Indexing failed for", zap.String("filePath", path), zap.Error(err))
		}
	}

	// Set current file
	return s.rotateIfNeeded()
}

func (s *EventStorage) rotateIfNeeded() error {
	s.currentMutex.Lock()
	defer s.currentMutex.Unlock()

	// Create a new file
	newFileName := s.filePrefix + time.Now().Format("20060102") + "_" + uuid.New().String()[:8] + ".json"
	s.currentFile = filepath.Join(s.baseDir, newFileName)

	// Initialize file
	return s.initFile(s.currentFile)
}

func (s *EventStorage) initFile(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, defaultFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(`{"events":[]}`)
	return err
}

func (s *EventStorage) StoreEvent(event EventEntry) (string, error) {
	// Ensure necessary fields
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}
	if event.Type == "" {
		event.Type = "alert" // Default type
	}

	s.currentMutex.Lock()
	defer s.currentMutex.Unlock()

	// Read file content
	data, err := os.ReadFile(s.currentFile)
	if err != nil {
		return "", err
	}

	// Parse JSON
	var content struct {
		Events []EventEntry `json:"events"`
	}
	if err := json.Unmarshal(data, &content); err != nil {
		return "", err
	}

	// Check file size
	if len(data) > int(s.maxFileSize) {
		if err := s.rotateIfNeeded(); err != nil {
			return "", err
		}
		return s.StoreEvent(event)
	}

	// Add new record
	content.Events = append(content.Events, event)

	// Write back to file
	newData, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(s.currentFile, newData, defaultFilePerm); err != nil {
		return "", err
	}

	// Update index
	s.updateFileIndex(s.currentFile, event)

	return s.currentFile, nil
}

func (s *EventStorage) updateFileIndex(path string, event EventEntry) {
	s.indexMutex.Lock()
	defer s.indexMutex.Unlock()

	if idx, ok := s.fileIndexes[path]; ok {
		if event.Timestamp < idx.MinTime {
			idx.MinTime = event.Timestamp
		}
		if event.Timestamp > idx.MaxTime {
			idx.MaxTime = event.Timestamp
		}
		if event.Severity > idx.MaxSeverity {
			idx.MaxSeverity = event.Severity
		}
		idx.AllProcessed = false
		idx.EventTypes[event.Type] = true
	} else {
		s.fileIndexes[path] = &FileIndex{
			Path:         path,
			MinTime:      event.Timestamp,
			MaxTime:      event.Timestamp,
			MaxSeverity:  event.Severity,
			EventTypes:   map[string]bool{event.Type: true},
			AllProcessed: false,
		}
	}
}

func (s *EventStorage) indexFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var content struct {
		Events []EventEntry `json:"events"`
	}
	if err := json.Unmarshal(data, &content); err != nil {
		return err
	}

	if len(content.Events) == 0 {
		return nil
	}

	idx := &FileIndex{
		Path:         path,
		MinTime:      content.Events[0].Timestamp,
		MaxTime:      content.Events[0].Timestamp,
		EventTypes:   make(map[string]bool),
		AllProcessed: false,
	}

	hasUnprocessed := false
	for _, event := range content.Events {
		if event.Timestamp < idx.MinTime {
			idx.MinTime = event.Timestamp
		}
		if event.Timestamp > idx.MaxTime {
			idx.MaxTime = event.Timestamp
		}
		if event.Severity > idx.MaxSeverity {
			idx.MaxSeverity = event.Severity
		}
		if !event.Processed {
			hasUnprocessed = true
		}
		idx.EventTypes[event.Type] = true
	}
	idx.AllProcessed = !hasUnprocessed

	s.indexMutex.Lock()
	s.fileIndexes[path] = idx
	s.indexMutex.Unlock()

	return nil
}

func (s *EventStorage) LoadEvents(filter EventFilter) ([]EventEntry, error) {
	var allEvents []EventEntry

	if filter.EndTime == 0 {
		filter.EndTime = time.Now().UnixMilli()
	}

	// Pre-filter files using index
	candidateFiles := s.getCandidateFiles(filter)

	for _, path := range candidateFiles {
		events, err := s.loadFile(path, filter)
		if err != nil {
			logger.Logger.Info("Error loading", zap.String("filePath", path), zap.Error(err))
			continue
		}
		allEvents = append(allEvents, events...)
	}

	// Sort by timestamp in descending order
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp > allEvents[j].Timestamp
	})

	// After all events are processed, update storage uniformly
	// Generally, if there is additional processing logic in the upper layer, it should be executed by the upper layer..
	if err := s.ApplyPendingUpdates(); err != nil {
		logger.Logger.Error("error applying updates", zap.Error(err))
		return nil, fmt.Errorf("error applying updates: %w", err)
	}

	return allEvents, nil
}

// Apply all pending updates
func (s *EventStorage) ApplyPendingUpdates() error {
	s.pendingUpdates.mutex.Lock()
	defer s.pendingUpdates.mutex.Unlock()

	for filePath, fileUpdates := range s.pendingUpdates.updates {
		if len(fileUpdates) == 0 {
			continue
		}

		if err := s.applySingleFileUpdates(filePath, fileUpdates); err != nil {
			logger.Logger.Error("failed to apply file updates", zap.String("filePath", filePath), zap.Error(err))
		}

		// Clear processed updates
		delete(s.pendingUpdates.updates, filePath)
	}

	return nil
}

func (s *EventStorage) applySingleFileUpdates(filePath string, fileUpdates map[string]EventUpdate) error {
	// Get file lock
	fileLock := s.getFileLock(filePath)
	fileLock.Lock()
	defer fileLock.Unlock()

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Parse JSON
	var content struct {
		Events []EventEntry `json:"events"`
	}
	if err := json.Unmarshal(data, &content); err != nil {
		return fmt.Errorf("error unmarshaling file %s: %w", filePath, err)
	}

	// Apply updates
	updated := false
	for i := range content.Events {
		if update, exists := fileUpdates[content.Events[i].ID]; exists {
			content.Events[i].Processed = update.Processed
			content.Events[i].ProcessedAt = update.ProcessedAt
			updated = true
		}
	}

	if !updated {
		return nil
	}

	// Write back to file
	newData, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling updated content: %w", err)
	}

	if err := os.WriteFile(filePath, newData, defaultFilePerm); err != nil {
		return fmt.Errorf("error writing updated file: %w", err)
	}

	// Update index
	if err := s.indexFile(filePath); err != nil {
		logger.Logger.Info("Error updating index for", zap.String("filePath", filePath), zap.Error(err))
	}

	return nil
}

func (s *EventStorage) getCandidateFiles(filter EventFilter) []string {
	s.indexMutex.RLock()
	defer s.indexMutex.RUnlock()

	var candidates []string
	for path, idx := range s.fileIndexes {
		// Time range check
		if idx.MaxTime < filter.StartTime || idx.MinTime > filter.EndTime {
			continue
		}

		// Severity level check
		if idx.MaxSeverity < filter.MinSeverity {
			continue
		}

		// Event type check
		if filter.Type != "" && !idx.EventTypes[filter.Type] {
			continue
		}

		if filter.Unprocessed && idx.AllProcessed {
			continue
		}

		candidates = append(candidates, path)
	}
	return candidates
}

func (s *EventStorage) loadFile(path string, filter EventFilter) ([]EventEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var content struct {
		Events []EventEntry `json:"events"`
	}
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, err
	}

	var filtered []EventEntry
	for _, event := range content.Events {
		if event.Timestamp < filter.StartTime ||
			event.Timestamp > filter.EndTime {
			continue
		}

		if event.Severity < filter.MinSeverity {
			continue
		}

		if filter.Type != "" && event.Type != filter.Type {
			continue
		}

		if filter.Source != "" && event.Source != filter.Source {
			continue
		}

		if filter.JobID != "" && event.JobID != filter.JobID {
			continue
		}

		if filter.Unprocessed && event.Processed {
			continue
		}

		// Add to pending update list (do not update file immediately)
		s.MarkPendingProcessed(event.ID, path)
		filtered = append(filtered, event)
	}

	return filtered, nil
}

// Mark event as pending update
func (s *EventStorage) MarkPendingProcessed(eventID, filePath string) {
	s.pendingUpdates.AddUpdate(filePath, eventID, EventUpdate{
		Processed:   true,
		ProcessedAt: time.Now().UnixMilli(),
	})
}
