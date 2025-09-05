// Copyright (c) OpenMMLab. All rights reserved.

package alerts

import (
	"fmt"
	"testing"

	pb "deeptrace/v1"
)

// TestSendEachNodeAlertToFeishu tests the SendEachNodeAlertToFeishu function
func TestSendEachNodeAlertToFeishu(t *testing.T) {
	// Define test cases
	tests := []struct {
		name       string
		webhookURL string
		results    []NodeAlerts
		prefix     string
		noAlert    []bool
		wantErr    bool
	}{
		{
			name:       "Normal case - valid webhook URL and alert results",
			webhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
			results: []NodeAlerts{
				{
					NodeAddr: "192.168.1.1",
					Alerts: []*pb.AlertRecord{
						{
							Severity:  pb.Severity_CRITICAL,
							Message:   "Test alert message",
							Timestamp: nil,
						},
					},
					Error: nil,
				},
			},
			prefix:  "Test Alert",
			noAlert: []bool{false},
			wantErr: false,
		},
		{
			name:       "Error case - webhook URL is empty",
			webhookURL: "",
			results: []NodeAlerts{
				{
					NodeAddr: "192.168.1.1",
					Alerts: []*pb.AlertRecord{
						{
							Severity:  pb.Severity_CRITICAL,
							Message:   "Test alert message",
							Timestamp: nil,
						},
					},
					Error: nil,
				},
			},
			prefix:  "Test Alert",
			noAlert: []bool{false},
			wantErr: true,
		},
		{
			name:       "Error case - results are empty",
			webhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
			results:    []NodeAlerts{},
			prefix:     "Test Alert",
			noAlert:    []bool{},
			wantErr:    true,
		},
		{
			name:       "No alerts need to be sent",
			webhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
			results: []NodeAlerts{
				{
					NodeAddr: "192.168.1.1",
					Alerts:   []*pb.AlertRecord{},
					Error:    nil,
				},
			},
			prefix:  "Test Alert",
			noAlert: []bool{true},
			wantErr: false,
		},
		{
			name:       "Node processing error",
			webhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
			results: []NodeAlerts{
				{
					NodeAddr: "192.168.1.1",
					Alerts:   nil,
					Error:    fmt.Errorf("Failed to connect to node"),
				},
			},
			prefix:  "Test Alert",
			noAlert: []bool{false},
			wantErr: false,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only test the error handling logic of the function.
			err := SendAlertsToFeishu(tt.webhookURL, tt.results, tt.prefix, tt.noAlert)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendAlertsToFeishu() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
