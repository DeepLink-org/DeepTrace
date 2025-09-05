// Copyright (c) OpenMMLab. All rights reserved.

package stacktrace

import (
	"testing"

	pb "deeptrace/v1"

	"github.com/stretchr/testify/assert"
)

func Test_processType(t *testing.T) {
	type args struct {
		typ string
	}
	tests := []struct {
		name string
		args args
		want pb.ProcessType
	}{
		{
			name: "trainer process",
			args: args{typ: "trainer"},
			want: pb.ProcessType_PROCESS_TRAINER,
		},
		{
			name: "dataloader process",
			args: args{typ: "dataloader"},
			want: pb.ProcessType_PROCESS_DATA_LOADER,
		},
		{
			name: "unspecified process",
			args: args{typ: "unknown"},
			want: pb.ProcessType_PROCESS_UNSPECIFIED,
		},
		{
			name: "empty process type",
			args: args{typ: ""},
			want: pb.ProcessType_PROCESS_UNSPECIFIED,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processType(tt.args.typ)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPythonStack_Fetch(t *testing.T) {
	type args struct {
		pid int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "normal - requires pystack",
			args:    args{pid: 1234},
			want:    "",   // Would depend on pystack output
			wantErr: true, // Expected to fail if pystack is not available
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ps := &PythonStack{
				sem: make(chan struct{}, 1),
			}

			got, err := ps.Fetch(tt.args.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("PythonStack.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_pyStack(t *testing.T) {
	type args struct {
		pid int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "normal - requires pystack command",
			args:    args{pid: 1234},
			want:    "",   // Would depend on pystack output
			wantErr: true, // Expected to fail if pystack command is not available
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pyStack(tt.args.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("pyStack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
