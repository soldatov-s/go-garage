package envconfig

import (
	"github.com/vrischmann/envconfig"
)

const DefaultProviderName = "envconfig"

// Provider is a control structure that provides envconfig functionality
// to gowork.
type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

// Parse executes parsing sequence.
func (ec *Provider) Parse(structure interface{}) error {
	// Error here will never rise as we always provide pointer to Init().
	return envconfig.Init(structure)
}
