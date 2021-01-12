package garage

const (
	defaultHTTPServer = "private"
)

// Config describes struct with options for Statistics provider
type Config struct {
	// HTTPProviderName is a name http server provider for server, where will be
	// handled statistics and metrics requests
	HTTPProviderName string `envconfig:"optional"`
	// HTTPEnityName is a name of http server
	HTTPEnityName string `envconfig:"optional"`
}

func (c *Config) Validate() {
	if c.HTTPEnityName == "" {
		c.HTTPEnityName = defaultHTTPServer
	}
}
