package rabbitmq

// CosumeHandler is handler for consume messages
type CosumeHandler func([]byte) error

// ShutdownHandler is handler for shutdown event
type ShutdownHandler func()

// SubscribeOptions describes struct with options for subscriber
type SubscribeOptions struct {
	ConsumeHndl  CosumeHandler
	Shutdownhndl ShutdownHandler
}
