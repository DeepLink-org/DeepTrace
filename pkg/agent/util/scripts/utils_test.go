// Copyright (c) OpenMMLab. All rights reserved.

package scripts

import (
	"context"
	"testing"
	"text/template"
)

func Test_executeScript(t *testing.T) {
	type args struct {
		ctx  context.Context
		tmpl *template.Template
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				ctx:  context.TODO(),
				tmpl: template.Must(template.New("test").Parse("echo 'hello world'")),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeScript(tt.args.ctx, tt.args.tmpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_executeShellScript(t *testing.T) {
	type args struct {
		scriptContent string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				scriptContent: "echo 'hello world'",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeShellScript(tt.args.scriptContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeShellScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
