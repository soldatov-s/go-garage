package stats

type Service struct {
	// Connection metrics
	Metrics MapMetricsOptions
	// Connection alive handlers
	AliveHandlers MapCheckFunc
	// Connection ready handlers
	ReadyHandlers MapCheckFunc
}

// GetAliveHandlers return array of the aliveHandlers from service
func (s *Service) GetAliveHandlers(prefix string) MapCheckFunc {
	if s.AliveHandlers == nil {
		s.AliveHandlers = make(MapCheckFunc)
	}

	return s.AliveHandlers
}

// GetMetrics return map of the metrics from cache connection
func (s *Service) GetMetrics(prefix string) MapMetricsOptions {
	if s.Metrics == nil {
		s.Metrics = make(MapMetricsOptions)
	}

	return s.Metrics
}

// GetReadyHandlers return array of the readyHandlers from cache connection
func (s *Service) GetReadyHandlers(prefix string) MapCheckFunc {
	if s.ReadyHandlers == nil {
		s.ReadyHandlers = make(MapCheckFunc)
	}

	return s.ReadyHandlers
}
