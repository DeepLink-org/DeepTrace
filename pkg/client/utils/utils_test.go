// Copyright (c) OpenMMLab. All rights reserved.

package utils

import (
	"fmt"
	"testing"
)

// TestExtractNodes tests various scenarios of the extractNodes function
func TestExtractNodes(t *testing.T) {
	tests := []struct {
		name    string   // Test case name
		input   string   // Input output string
		want    []string // Expected node list
		wantErr bool     // Whether an error is expected
	}{
		{
			name:  "Normal case: nodes wrapped in single quotes",
			input: `some prefix nodes=['node1:host1', 'node2:host2', 'node3'] some suffix`,
			want:  []string{"host1", "host2", "node3"},
		},
		{
			name:  "Empty node list",
			input: `result nodes=[] completed`,
			want:  []string{}, // Empty list but no error
		},
		{
			name: "Multi-line node list",
			input: `nodes=[
				'nodeA:hostA', 
				'nodeB:hostB',
				'nodeC'
			] more content`,
			want: []string{"hostA", "hostB", "nodeC"},
		},
		{
			name:  "Nodes with spaces and consecutive commas",
			input: `nodes=['a:host a',  'b:hostb',  'c:host\nc'  ]`,
			want:  []string{"hosta", "hostb", "hostc"},
		},
		{
			name:  "Pure node names without colons",
			input: `nodes=[node1, node2, node3]`,
			want:  []string{"node1", "node2", "node3"},
		},
		{
			name:    "No node information",
			input:   `this is a test with no nodes`,
			wantErr: true, // Expect error (nodes=[] pattern not found)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractNodes(tt.input)

			// Verify that errors match expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("Error verification failed: actual error=%v, expected error=%v", err, tt.wantErr)
				return
			}

			// Verify that node list matches expectations
			if len(got) != len(tt.want) {
				t.Errorf("Length mismatch: actual=%d, expected=%d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Index %d mismatch: actual=%s, expected=%s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		err      bool
	}{
		{"rank123", 123, false},
		{"abc456def", 456, false},
		{"123abc", 123, false},
		{"abc123", 123, false},
		{"123", 123, false},
		{"rank-123", 123, false},       // Note: current implementation matches all numbers
		{"rank123test456", 123, false}, // Only match the first number
		{"rank", 0, true},
		{"", 0, true},
		{"abc", 0, true},
		{"rank123.45", 123, false}, // Note: decimal part will be ignored
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ExtractNumber(tt.input)
			if (err != nil) != tt.err {
				t.Fatalf("Expected error: %v, actual error: %v", tt.err, err)
			}
			if result != tt.expected {
				t.Fatalf("Expected result: %d, actual result: %d", tt.expected, result)
			}
		})
	}
}

func TestCompareRank(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"rank1", "rank2", -1},
		{"rank2", "rank1", 1},
		{"rank10", "rank2", 1}, // Numeric part 10 > 2
		{"rank1", "rank1", 0},
		{"abc123", "def123", 0},
		{"abc123", "def456", -1},
		{"123abc", "456def", -1},
		{"rank-1", "rank-2", -1},  // Note: current implementation treats -1 as 1, -2 as 2
		{"rank1.5", "rank1", 0},   // Note: current implementation treats 1.5 as 1
		{"rank1a", "rank1b", 0},   // Numeric parts are the same
		{"rank1a", "rank10b", -1}, // Numeric part 1 < 10
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.a, tt.b), func(t *testing.T) {
			result := CompareRank(tt.a, tt.b)
			if result != tt.expected {
				t.Fatalf("Expected result: %d, actual result: %d", tt.expected, result)
			}
		})
	}
}

// Test error handling scenarios
func TestCompareRank_ErrorHandling(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"rank", "rank1", 0}, // a has no numbers
		{"rank1", "rank", 0}, // b has no numbers
		{"rank", "rank", 0},  // both have no numbers
		{"", "rank1", 0},     // a is empty
		{"rank1", "", 0},     // b is empty
		{"", "", 0},          // both are empty
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("error_%s_%s", tt.a, tt.b), func(t *testing.T) {
			result := CompareRank(tt.a, tt.b)
			if result != tt.expected {
				t.Fatalf("Expected result: %d, actual result: %d", tt.expected, result)
			}
		})
	}
}
