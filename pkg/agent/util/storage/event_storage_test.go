// Copyright (c) OpenMMLab. All rights reserved.

package storage

import (
	"os"
	"testing"
	"time"
)

func TestEventStorage_StoreEvent(t *testing.T) {
	type args struct {
		event EventEntry
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Store a valid event",
			args: args{
				event: EventEntry{
					ID:        "test-event-id",
					Source:    "test-source",
					Type:      "test-type",
					JobID:     "test-job-id",
					Message:   "Test message",
					Timestamp: time.Now().UnixMilli(),
					Severity:  1,
					Metadata:  make(Metadata),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewEventStorage("", 0)
			if err != nil {
				t.Fatalf("Failed to create EventStorage: %v", err)
			}
			defer os.RemoveAll(storage.baseDir)

			_, err = storage.StoreEvent(tt.args.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("EventStorage.StoreEvent() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify the event was stored by loading it back
			filter := EventFilter{
				StartTime: 0,
				EndTime:   time.Now().UnixMilli(),
			}

			events, err := storage.LoadEvents(filter)
			if err != nil {
				t.Errorf("EventStorage.LoadEvents() error = %v, wantErr %v", err, false)
			}

			if len(events) != 1 {
				t.Errorf("EventStorage.LoadEvents() = %d events, want 1", len(events))
			}

			if events[0].ID != tt.args.event.ID {
				t.Errorf("EventStorage.LoadEvents() = event ID %s, want %s", events[0].ID, tt.args.event.ID)
			}
		})
	}
}
