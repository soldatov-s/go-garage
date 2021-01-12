package msgs

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	providerswithmetrics "github.com/soldatov-s/go-garage/providers/base/providers/with_metrics"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/stats"
)

const (
	ProvidersName = "messages"
)

// Provider is an interface every provider should follow.
type Provider interface {
	provider.IProvider
	provider.IEniter
	stats.IProviderMetrics

	// SendMessage should send message. Passed message structure depends
	// on used provider, you should consult provider's documentation about
	// it.
	SendMessage(connectionName string, message interface{}) error
	// Subscribe should subscribe to something. Passed channel will be
	// used for message reciving, passed options used for provider's
	// subscription configuration. Messages that will pass thru channel
	// should be casted to appropriate type.
	// Provider should launch separate goroutine to work with incoming
	// messages. Caller should also do that to work with messages.
	Subscribe(connectionName string, options interface{}) error
}

// Messaging structure controls all inter-service messaging in gowork.
type MessagingServers struct {
	*providerswithmetrics.BaseProvidersWithMetrics
}

func NewMessagingServers(ctx context.Context) *MessagingServers {
	return &MessagingServers{
		BaseProvidersWithMetrics: providerswithmetrics.NewBaseProvidersWithMetrics(ctx, ProvidersName),
	}
}

// GetProvider returns provider interface to caller.
// Returns error if provider isn't registered.
func (ms *MessagingServers) GetProvider(providerName string) (Provider, error) {
	if v, err := ms.BaseProviders.GetProvider(providerName); err != nil {
		return nil, err
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, errors.ErrBadTypeOfProvider
}

// SendMessage sends message via specified provider using specified
// connection.
func (ms *MessagingServers) SendMessage(providerName, connectionName string, message interface{}) error {
	// Check if provider and connection names isn't breaking rules.
	if err := ms.CheckProviderAndEnityNames(providerName, connectionName); err != nil {
		return err
	}

	// Get provider.
	prov, err := ms.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.SendMessage(connectionName, message)
}

// Subscribe subscribes to something via specified provider using specified
// connection.
func (ms *MessagingServers) Subscribe(providerName, connectionName string, options interface{}) error {
	// Check if provider and connection names isn't breaking rules.
	if err := ms.CheckProviderAndEnityNames(providerName, connectionName); err != nil {
		return err
	}

	// Get provider.
	prov, err := ms.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.Subscribe(connectionName, options)
}

// GetAllMetrics collect all metrics for Databases
func (ms *MessagingServers) GetAllMetrics(out stats.MapMetricsOptions) (stats.MapMetricsOptions, error) {
	return ms.BaseProvidersWithMetrics.GetAllMetrics(ProvidersName, out)
}

// GetAllAliveHandlers collect all aliveHandlers for Databases
func (ms *MessagingServers) GetAllAliveHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return ms.BaseProvidersWithMetrics.GetAllAliveHandlers(ProvidersName, out)
}

// GetAllReadyHandlers collect all readyHandlers for Databases
func (ms *MessagingServers) GetAllReadyHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return ms.BaseProvidersWithMetrics.GetAllReadyHandlers(ProvidersName, out)
}
