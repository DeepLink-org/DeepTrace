// Copyright (c) OpenMMLab. All rights reserved.

package rules

import (
	"context"
	pb "deeptrace/v1"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestThreadsStacksEqual(t *testing.T) {
	type args struct {
		ctx      context.Context
		time1    string
		time2    string
		processa *pb.ProcessInfo
		processb *pb.ProcessInfo
	}
	tests := []struct {
		name      string
		args      args
		wantEqual bool
		wantDiff  string
		wantErr   bool
	}{
		{
			name: "normal",
			args: args{
				ctx:   context.TODO(),
				time1: "time1",
				time2: "time2",
				processa: &pb.ProcessInfo{
					Pid: 112,
					Threads: []*pb.ThreadStack{
						{
							ThreadId: 345,
							StackFrames: []string{
								"bbbb",
								"aaaa",
							},
						},
						{
							ThreadId: 123,
							StackFrames: []string{
								"aaaa",
								"bbbb",
							},
						},
					},
				},
				processb: &pb.ProcessInfo{
					Pid: 112,
					Threads: []*pb.ThreadStack{
						{
							ThreadId: 123,
							StackFrames: []string{
								"aaaa",
								"bbbb",
							},
						},
						{
							ThreadId: 345,
							StackFrames: []string{
								"bbbb",
								"aaaa",
							},
						},
					},
				},
			},
			wantEqual: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEqual, gotDiff, err := ThreadsStacksEqual(tt.args.ctx, tt.args.time1, tt.args.time2, tt.args.processa, tt.args.processb)
			if (err != nil) != tt.wantErr {
				t.Errorf("ThreadsStacksEqual() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotEqual != tt.wantEqual {
				t.Errorf("ThreadsStacksEqual() gotEqual = %v, want %v", gotEqual, tt.wantEqual)
			}
			if gotDiff != tt.wantDiff {
				t.Errorf("ThreadsStacksEqual() gotDiff = %v, want %v", gotDiff, tt.wantDiff)
			}
		})
	}
}

func TestPstreeEqual(t *testing.T) {
	hangPstree, err := os.ReadFile("./proccessInfo.json")
	if err != nil {
		fmt.Printf("failed to read json file: %v", err)
		return
	}
	pstrees := [][]*pb.ProcessInfo{}
	json.Unmarshal(hangPstree, &pstrees)

	type args struct {
		ctx   context.Context
		time1 string
		time2 string
		psa   []*pb.ProcessInfo
		psb   []*pb.ProcessInfo
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   []ProccessInfoDiff
		wantErr bool
	}{
		{
			name: "hang1",
			args: args{
				ctx:   context.TODO(),
				time1: "time1",
				time2: "time2",
				psa:   pstrees[0],
				psb:   pstrees[1],
			},
			want: false,
			want1: []ProccessInfoDiff{
				{
					Rank:  "RANK7",
					Diff:  "Detected different process stacks. RANK7, Process type: PROCESS_DATA_LOADER, Process ID:40826, Thread ID: 40826, Stack frame level: 9\n[time1]Stack frame content: File \"/usr/local/lib/python3.12/dist-packages/xxx/tools/sft.py\", line 553, in __getitem__\ntime.sleep(2)\n[time2]Stack frame content: File \"/usr/local/lib/python3.12/dist-packages/xxx/tools/sft.py\", line 563, in __getitem__\ntime.sleep(2)\n",
					PType: pb.ProcessType_PROCESS_DATA_LOADER,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := PstreeEqual(tt.args.ctx, tt.args.time1, tt.args.time2, tt.args.psa, tt.args.psb)
			if (err != nil) != tt.wantErr {
				t.Errorf("PstreeEqual() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PstreeEqual() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("PstreeEqual() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
