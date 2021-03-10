package gopcua

// CosumeHandler is handler for consume messages
type CosumeHandler func(interface{}) error

// SubscribeOptions describes struct with options for subscriber
type SubscribeOptions struct {
	ConsumeHndl CosumeHandler
}
