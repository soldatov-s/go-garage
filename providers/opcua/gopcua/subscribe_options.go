package gopcua

import (
	"github.com/gopcua/opcua/monitor"
)

// CosumeHandler is handler for consume messages
type CosumeHandler func(*monitor.DataChangeMessage) error

// SubscribeOptions describes struct with options for subscriber
type SubscribeOptions struct {
	ConsumeHndl CosumeHandler
}
