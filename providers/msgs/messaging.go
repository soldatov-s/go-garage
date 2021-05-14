package msgs

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
)

const (
	CollectorName = "messages"
)

// Provider is an interface every provider should follow.
type ProviderGateway interface {
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

// Collector is a controlling structure for all caches.
type Collector struct {
	*base.CollectorWithMetrics
}

func NewCollector(ctx context.Context) (*Collector, error) {
	c, err := base.NewCollectorWithMetrics(ctx, CollectorName)
	if err != nil {
		return nil, errors.Wrap(err, "new collector with metrics")
	}
	return &Collector{c}, nil
}

// GetProvider returns requested caches provider. It'll return error if
// providers wasn't registered.
func (c *Collector) GetProvider(providerName string) (ProviderGateway, error) {
	p, err := c.CollectorWithMetrics.GetProvider(providerName)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	pg, ok := p.(ProviderGateway)
	if !ok {
		return nil, errors.Wrap(base.ErrBadTypeOfProvider, "expected ProviderGateway")
	}

	return pg, nil
}

// SendMessage sends message via specified provider using specified
// connection.
func (c *Collector) SendMessage(providerName, connectionName string, message interface{}) error {
	// Get provider.
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.SendMessage(connectionName, message)
}

// Subscribe subscribes to something via specified provider using specified
// connection.
func (c *Collector) Subscribe(providerName, connectionName string, options interface{}) error {
	// Get provider.
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.Subscribe(connectionName, options)
}

// GetMetrics collect all metrics for Caches
func (c *Collector) GetMetrics(ctx context.Context) (base.MapMetricsOptions, error) {
	return c.CollectorWithMetrics.GetMetrics(ctx)
}

// GetAliveHandlers collect all aliveHandlers for Caches
func (c *Collector) GetAliveHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	return c.CollectorWithMetrics.GetAliveHandlers(ctx)
}

// GetReadyHandlers collect all readyHandlers for Caches
func (c *Collector) GetReadyHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	return c.CollectorWithMetrics.GetReadyHandlers(ctx)
}

func NewContext(ctx context.Context) (context.Context, error) {
	c, err := NewCollector(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "new collector %q", CollectorName)
	}
	return base.NewContextByName(ctx, CollectorName, c)
}

func FromContext(ctx context.Context) (*Collector, error) {
	v, err := base.FromContextByName(ctx, CollectorName)
	if err != nil {
		return nil, errors.Wrap(err, "get from context by name")
	}
	c, ok := v.(*Collector)
	if !ok {
		return nil, base.ErrFailedTypeCast
	}
	return c, nil
}
