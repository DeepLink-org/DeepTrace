// Copyright (c) OpenMMLab. All rights reserved.

package httpserver

import (
	"deeptrace/pkg/agent/util/storage"
)

type SendEventRequest struct {
	MsgType string  `json:"msg_type"`
	Content Content `json:"content"`
}

type Content struct {
	Text string `json:"text"`
}

type DefaultHandler struct {
	storage *storage.EventStorage
}
