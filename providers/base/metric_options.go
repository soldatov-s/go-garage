package base

// MetricOptions descrbes struct with options for metrics
type MetricOptions struct {
	// Metric is a metric
	Metric interface{}
	// MetricFunc is a func for update metric
	MetricFunc func(metric interface{})
}

type MapMetricsOptions map[string]*MetricOptions

func (mmo MapMetricsOptions) Add(src MapMetricsOptions) {
	for k, m := range src {
		mmo[k] = m
	}
}

// Checking function should accept no parameters and return
// boolean value which will be interpreted as dependency service readiness
// and a string which should provide error text if dependency service
// isn't ready and something else if dependency service is ready (for
// example, dependency service's version).
type CheckFunc func() (bool, string)

type MapCheckFunc map[string]CheckFunc

func (mcf MapCheckFunc) Add(src MapCheckFunc) {
	for k, m := range src {
		mcf[k] = m
	}
}
