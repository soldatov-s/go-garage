package rabbitmq

// ConsumeHandler is handler for consume messages
type ConsumeHandler interface {
	Consume([]byte) error
}

// ShutdownHandler is handler for shutdown event
type ShutdownHandler interface {
	Shutdown()
}

// SubscribeOptions describes interface with options for subscriber
type SubscribeOptions interface {
	ConsumeHandler
	ShutdownHandler
}
