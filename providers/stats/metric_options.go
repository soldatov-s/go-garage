package stats

// MetricOptions descrbes struct with options for metrics
type MetricOptions struct {
	// Metric is a metric
	Metric interface{}
	// MetricFunc is a func for update metric
	MetricFunc func(metric interface{})
}

type MapMetricsOptions map[string]*MetricOptions

func (mmo MapMetricsOptions) Fill(src MapMetricsOptions) {
	for k, m := range src {
		mmo[k] = m
	}
}

func (mmo MapMetricsOptions) RegistrateMetric(prov ProviderGateway) error {
	for k, m := range mmo {
		err := prov.RegisterMetric(k, m)
		if err != nil {
			return err
		}
	}

	return nil
}
