// Copyright (c) OpenMMLab. All rights reserved.

package storage

import (
	"testing"
)

func TestNewPendingUpdateManager(t *testing.T) {
	manager := NewPendingUpdateManager()

	if manager == nil {
		t.Error("NewPendingUpdateManager should not return nil")
	}

	if manager.updates == nil {
		t.Error("NewPendingUpdateManager should initialize updates map")
	}
}

func TestPendingUpdateManager_AddUpdate(t *testing.T) {
	type args struct {
		filePath string
		eventID  string
		update   EventUpdate
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Add a new update",
			args: args{
				filePath: "/test/file.txt",
				eventID:  "event1",
				update: EventUpdate{
					Processed:   true,
					ProcessedAt: 1234567890,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewPendingUpdateManager()

			manager.AddUpdate(tt.args.filePath, tt.args.eventID, tt.args.update)

			// Verify the update was added
			if _, exists := manager.updates[tt.args.filePath]; !exists {
				t.Errorf("PendingUpdateManager.AddUpdate() should create an entry for the file path %s", tt.args.filePath)
			}

			if _, exists := manager.updates[tt.args.filePath][tt.args.eventID]; !exists {
				t.Errorf("PendingUpdateManager.AddUpdate() should create an entry for the event ID %s", tt.args.eventID)
			}

			if manager.updates[tt.args.filePath][tt.args.eventID] != tt.args.update {
				t.Error("PendingUpdateManager.AddUpdate() should store the correct update")
			}

			// Test adding another update for the same file path
			eventID2 := "event2"
			update2 := EventUpdate{
				Processed:   false,
				ProcessedAt: 9876543210,
			}

			manager.AddUpdate(tt.args.filePath, eventID2, update2)

			// Verify both updates exist
			if _, exists := manager.updates[tt.args.filePath][tt.args.eventID]; !exists {
				t.Errorf("PendingUpdateManager.AddUpdate() should preserve existing entries for the file path %s", tt.args.filePath)
			}

			if manager.updates[tt.args.filePath][eventID2] != update2 {
				t.Error("PendingUpdateManager.AddUpdate() should store the correct second update")
			}
		})
	}
}
