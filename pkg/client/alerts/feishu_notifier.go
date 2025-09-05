// Copyright (c) OpenMMLab. All rights reserved.

package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type FeishuTextMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

func SendAlertsToFeishu(webhookURL string, results []NodeAlerts, prefix string, noAlert []bool) error {
	if webhookURL == "" {
		return fmt.Errorf("Feishu Webhook address cannot be empty")
	}
	if len(results) == 0 {
		return fmt.Errorf("No alert information to send")
	}

	// Iterate through each node and send messages separately
	for idx, nodeResult := range results {
		// Build single message content
		var msgBuffer bytes.Buffer
		msgBuffer.WriteString(fmt.Sprintf("【%s】\n", prefix))
		msgBuffer.WriteString(fmt.Sprintf("Node: %s\n", nodeResult.NodeAddr))
		msgBuffer.WriteString(fmt.Sprintf("Processing time: %s\n", time.Now().Format("2006-01-02 15:04:05")))
		msgBuffer.WriteString("--------------------\n")

		if nodeResult.Error != nil {
			// Node processing error case
			msgBuffer.WriteString("Status: Failed\n")
			msgBuffer.WriteString(fmt.Sprintf("Error message: %v\n", nodeResult.Error))
		} else {
			// Node has alert results case
			if len(nodeResult.Alerts) == 0 {
				msgBuffer.WriteString("Status: No alerts\n")
				msgBuffer.WriteString("No matching alert information found\n")
			} else {
				msgBuffer.WriteString(fmt.Sprintf("Status: Found %d alerts\n", len(nodeResult.Alerts)))
				for i, alert := range nodeResult.Alerts {
					// Parse alert time
					t := time.Time{}
					if alert.Timestamp != nil {
						t = alert.Timestamp.AsTime()
					}
					// Display all alerts one by one (no limit on quantity)
					msgBuffer.WriteString(fmt.Sprintf(
						"%d. Time: %s\n   Level: %s\n   Message: %s\n",
						i+1,
						t.Local().Format("2006-01-02 15:04:05.000"),
						alert.Severity.String(),
						alert.Message,
					))
				}
			}
		}
		if !noAlert[idx] {
			// Construct Feishu message
			feishuMsg := FeishuTextMessage{
				MsgType: "text",
			}
			feishuMsg.Content.Text = msgBuffer.String()

			// Serialize message to JSON
			msgJSON, err := json.Marshal(feishuMsg)
			if err != nil {
				fmt.Printf("Node %s message serialization failed: %v, skipped\n", nodeResult.NodeAddr, err)
				continue
			}

			// Send HTTP request to Feishu Webhook
			resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(msgJSON))
			if err != nil {
				fmt.Printf("Node %s message sending failed: %v, skipped\n", nodeResult.NodeAddr, err)
				continue
			}
			resp.Body.Close() // Close immediately, don't wait for subsequent processing

			// Check response status
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Node %s Feishu interface returned non-success status: %s, skipped\n", nodeResult.NodeAddr, resp.Status)
				continue
			}

			fmt.Printf("Alert information from node %s has been sent to Feishu\n", nodeResult.NodeAddr)
		}

	}

	return nil
}
