// Copyright (c) OpenMMLab. All rights reserved.

package textparser

import (
	"context"
	"reflect"
	"testing"
	"time"

	pb "deeptrace/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLogParser_Parse(t *testing.T) {
	logtime, _ := time.Parse("2006-01-02 15:04:05", "2025-07-11 02:32:52")
	type args struct {
		ctx    context.Context
		inputs []string
	}
	tests := []struct {
		name    string
		p       *LogParser
		args    args
		want    []*pb.LogEntry
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "normal",
			p:    &LogParser{},
			args: args{
				ctx: context.TODO(),
				inputs: []string{
					"dsaljflkdjafljadslfj",
				},
			},
			want: []*pb.LogEntry{
				{
					Message: "dsaljflkdjafljadslfj",
				},
			},
			wantErr: false,
		},
		{
			name: "time",
			p:    &LogParser{},
			args: args{
				ctx: context.TODO(),
				inputs: []string{
					"[XTuner][RANK 15][2025-07-11 02:32:52][INFO] Gradient Accumulation: 4, Compile: True, CPU Offload: False",
				},
			},
			want: []*pb.LogEntry{
				{
					Message:   "[XTuner][RANK 15][2025-07-11 02:32:52][INFO] Gradient Accumulation: 4, Compile: True, CPU Offload: False",
					Timestamp: timestamppb.New(logtime),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &LogParser{}
			got, err := p.Parse(tt.args.ctx, tt.args.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogParser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LogParser.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
