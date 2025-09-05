// Copyright (c) OpenMMLab. All rights reserved.

package textparser

import (
	"context"
	"reflect"
	"testing"

	"deeptrace/logger"
	pb "deeptrace/v1"
)

func TestStackParser_Parse(t *testing.T) {
	stack_case_1 := `The frame stack for thread 2846 is empty
Traceback for thread 2843 (pt_autograd_0) [] (most recent call last):
    (Python) File "/usr/local/lib/python3.12/dist-packages/torch/autograd/function.py", line 307, in apply
        return user_fn(self, *args)
    (Python) File "/tmp/torchinductor_root/yy/asdhjfkjahdkfhakjhfdkj.py", line 101, in call
        buf0.copy_(primals_2, False)

Traceback for thread 1654 (python) [] (most recent call last):
    (Python) File "/usr/lib/python3.12/threading.py", line 1030, in _bootstrap
        self._bootstrap_inner()
    (Python) File "/usr/lib/python3.12/threading.py", line 1010, in run
        self._target(*self._args, **self._kwargs)
    (Python) File "/usr/lib/python3.12/threading.py", line 355, in wait
        waiter.acquire()`
	type args struct {
		ctx    context.Context
		inputs []string
	}
	tests := []struct {
		name    string
		p       *StackParser
		args    args
		want    []*pb.ThreadStack
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "normal",
			args: args{
				inputs: []string{stack_case_1},
			},
			want: []*pb.ThreadStack{
				{
					ThreadId:   2843,
					ThreadName: "pt_autograd_0",
					StackFrames: []string{
						"File \"/usr/local/lib/python3.12/dist-packages/torch/autograd/function.py\", line 307, in apply\nreturn user_fn(self, *args)",
						"File \"/tmp/torchinductor_root/yy/asdhjfkjahdkfhakjhfdkj.py\", line 101, in call\nbuf0.copy_(primals_2, False)"},
				},
				{
					ThreadId:   1654,
					ThreadName: "python",
					StackFrames: []string{
						"File \"/usr/lib/python3.12/threading.py\", line 1030, in _bootstrap\nself._bootstrap_inner()",
						"File \"/usr/lib/python3.12/threading.py\", line 1010, in run\nself._target(*self._args, **self._kwargs)",
						"File \"/usr/lib/python3.12/threading.py\", line 355, in wait\nwaiter.acquire()"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.p.Parse(tt.args.ctx, tt.args.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("StackParser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StackParser.Parse() = %v, want %v", logger.ToPrettyJSON(got), logger.ToPrettyJSON(tt.want))
			}
		})
	}
}
