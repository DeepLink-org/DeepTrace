// Copyright (c) OpenMMLab. All rights reserved.

package scripts

import (
	"context"
	"testing"
)

func TestGetCurrentNodeRankRange(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name       string
		args       args
		wantMinNum int
		wantMaxNum int
		wantErr    bool
	}{
		{
			name: "normal - requires torchrun process",
			args: args{
				ctx: context.TODO(),
			},
			wantMinNum: 0,
			wantMaxNum: 0,
			wantErr:    true, // Expected to fail if no torchrun process is running
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMinNum, gotMaxNum, err := GetCurrentNodeRankRange(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentNodeRankRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMinNum != tt.wantMinNum {
				t.Errorf("GetCurrentNodeRankRange() gotMinNum = %v, want %v", gotMinNum, tt.wantMinNum)
			}
			if gotMaxNum != tt.wantMaxNum {
				t.Errorf("GetCurrentNodeRankRange() gotMaxNum = %v, want %v", gotMaxNum, tt.wantMaxNum)
			}
		})
	}
}

func Test_parseRankRangeOutput(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name       string
		args       args
		wantMinNum int
		wantMaxNum int
		wantErr    bool
	}{
		{
			name: "valid range",
			args: args{
				output: "0:7",
			},
			wantMinNum: 0,
			wantMaxNum: 7,
			wantErr:    false,
		},
		{
			name: "invalid format",
			args: args{
				output: "invalid",
			},
			wantMinNum: 0,
			wantMaxNum: 0,
			wantErr:    true,
		},
		{
			name: "error output",
			args: args{
				output: "error: No launcher process found",
			},
			wantMinNum: 0,
			wantMaxNum: 0,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMinNum, gotMaxNum, err := parseRankRangeOutput(tt.args.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRankRangeOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMinNum != tt.wantMinNum {
				t.Errorf("parseRankRangeOutput() gotMinNum = %v, want %v", gotMinNum, tt.wantMinNum)
			}
			if gotMaxNum != tt.wantMaxNum {
				t.Errorf("parseRankRangeOutput() gotMaxNum = %v, want %v", gotMaxNum, tt.wantMaxNum)
			}
		})
	}
}
