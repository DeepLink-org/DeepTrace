// Copyright (c) OpenMMLab. All rights reserved.

package storage

import (
	"testing"
)

func TestNewFileLockManager(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Initialize manager and locks map",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFileLockManager()

			if manager == nil {
				t.Error("NewFileLockManager should not return nil")
			}

			if manager.locks == nil {
				t.Error("NewFileLockManager should initialize locks map")
			}
		})
	}
}

func TestFileLockManager_GetLock(t *testing.T) {
	type args struct {
		filePath  string
		filePath2 string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Get lock for a file path",
			args: args{
				filePath:  "/test/file.txt",
				filePath2: "/test/file2.txt",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFileLockManager()

			// Test getting a lock for a file path
			lock1 := manager.GetLock(tt.args.filePath)

			if lock1 == nil {
				t.Error("GetLock should not return nil")
			}

			// Test getting the same lock for the same file path
			lock2 := manager.GetLock(tt.args.filePath)

			if lock1 != lock2 {
				t.Error("GetLock should return the same lock for the same file path")
			}

			// Test getting different locks for different file paths
			lock3 := manager.GetLock(tt.args.filePath2)

			if lock1 == lock3 {
				t.Error("GetLock should return different locks for different file paths")
			}
		})
	}
}
