package gopcua

import "github.com/gopcua/opcua"

// CosumeHandler is handler for consume messages
type CosumeHandler func(*opcua.PublishNotificationData) error

// SubscribeOptions describes struct with options for subscriber
type SubscribeOptions struct {
	ConsumeHndl CosumeHandler
}
