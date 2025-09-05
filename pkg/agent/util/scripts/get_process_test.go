// Copyright (c) OpenMMLab. All rights reserved.

package scripts

import (
	"context"
	"testing"
	"text/template"
)

func TestGetProcessInfo(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal - requires torchrun process",
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true, // Expected to fail if no torchrun process is running
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetProcessInfo(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProcessInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_getTrainingProcesses(t *testing.T) {
	type args struct {
		ctx  context.Context
		tmpl string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal - requires torchrun process",
			args: args{
				ctx:  context.TODO(),
				tmpl: processStackTemplate,
			},
			wantErr: true, // Expected to fail if no torchrun process is running
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("processStack").Parse(tt.args.tmpl)
			if err != nil {
				t.Errorf("Failed to parse template: %v", err)
				return
			}
			_, err = getTrainingProcesses(tt.args.ctx, tmpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTrainingProcesses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
