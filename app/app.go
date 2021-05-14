package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/domains"
	"github.com/soldatov-s/go-garage/log"
	"github.com/soldatov-s/go-garage/meta"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/cache"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
	"github.com/soldatov-s/go-garage/providers/opcua"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
)

// Loop is application loop, exit on SIGTERM
func Loop(ctx context.Context) error {
	var closeSignal chan os.Signal
	m, err := meta.FromContext(ctx)
	if err != nil {
		return errors.Wrap(err, "get meta")
	}

	exit := make(chan struct{})
	closeSignal = make(chan os.Signal, 1)
	signal.Notify(closeSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-closeSignal
		if err := Shutdown(ctx); err != nil {
			log.FromContext(ctx).GetLogger(m.Name, nil).Fatal().Err(err).Msg("shutdown service")
		}
		log.FromContext(ctx).GetLogger(m.Name, nil).Info().Msg("exit service")
		close(exit)
	}()

	// Exit app if chan is closed
	<-exit

	return nil
}

// getAllMetrics return all metrics from databases and caches
func getAllMetrics(ctx context.Context) (stats.MapMetricsOptions, error) {
	metrics := make(stats.MapMetricsOptions)
	p, err := base.FromContext(ctx)
	if err != nil {
		return nil, base.ErrNotFoundCollectors
	}

	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(stats.IProvidersMetrics); ok {
			if _, err = m.GetAllMetrics(metrics); err != nil {
				return false
			}
		}

		return true
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to get metrics")
	}

	return metrics, nil
}

// getAllAliveHandlers return all aliveHandlers from databases and caches
// nolint : duplicate
func getAllAliveHandlers(ctx context.Context) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)
	p, err := base.FromContext(ctx)
	if err != nil {
		return nil, base.ErrNotFoundCollectors
	}

	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(stats.IProvidersMetrics); ok {
			if _, err = m.GetAllAliveHandlers(handlers); err != nil {
				return false
			}
		}

		return true
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to get alive handlers")
	}

	return handlers, nil
}

// getAllReadyHandlers return all readyHandlers from databases and caches
// nolint : duplicate
func getAllReadyHandlers(ctx context.Context) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)
	p, err := base.FromContext(ctx)
	if err != nil {
		return nil, base.ErrNotFoundCollectors
	}

	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(stats.IProvidersMetrics); ok {
			if _, err = m.GetAllReadyHandlers(handlers); err != nil {
				return false
			}
		}

		return true
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to get ready handlers")
	}

	return handlers, nil
}

func StartStatistics(ctx context.Context) error {
	s, err := stats.FromContext(ctx)
	if err != nil {
		return err
	}

	if s == nil {
		return nil
	}

	// Collecting all metrics from context
	metrics, err := getAllMetrics(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get all metrics")
	}

	// Collecting all aliveHandlers from context
	aliveHandlers, err := getAllAliveHandlers(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get all alive handlers")
	}

	// Collecting all readyHandlers from context
	readyHandlers, err := getAllReadyHandlers(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get all ready handlers")
	}

	if err := s.Start(ctx, metrics, aliveHandlers, readyHandlers); err != nil {
		return errors.Wrap(err, "failed to start statistics providers")
	}

	return nil
}

func providersOrder() []string {
	return []string{db.CollectorName, cache.CollectorName, httpsrv.CollectorName, opcua.CollectorName}
}

func NewContext(ctx context.Context) (context.Context, error) {
	ctx, err := base.NewContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create providers context")
	}
	ctx, err = domains.NewContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create domain context")
	}
	return ctx, nil
}

// Start all providers and domains
func Start(ctx context.Context) error {
	provs, err := base.FromContext(ctx)
	if err != nil {
		return base.ErrNotFoundCollectors
	}

	for _, v := range providersOrder() {
		if p, ok := provs.Load(v); ok {
			if err := p.(base.EntityGateway).Start(ctx); err != nil {
				return errors.Wrap(err, "failed to start provider")
			}
		}
	}

	doms, err := domains.FromContext(ctx)
	if err != nil {
		return domains.ErrNotFoundDomain
	}

	doms.Range(func(k, v interface{}) bool {
		if domain, ok := v.(domains.IBaseDomainStarter); ok {
			if err = domain.Start(); err != nil {
				return false
			}
		}
		return true
	})

	if err != nil {
		return errors.Wrap(err, "failed to start domains")
	}

	if err := StartStatistics(ctx); err != nil {
		return errors.Wrap(err, "failed to start statistics")
	}

	return nil
}

// Shutdown all domains and providers
func Shutdown(ctx context.Context) error {
	var err error
	doms, err := domains.FromContext(ctx)
	if err != nil {
		return domains.ErrNotFoundDomain
	}

	doms.Range(func(k, v interface{}) bool {
		if domain, ok := v.(domains.IBaseDomainShutdowner); ok {
			if err = domain.Shutdown(); err != nil {
				return false
			}
		}
		return true
	})

	if err != nil {
		return errors.Wrap(err, "failed to shutdown domains")
	}

	provs, err := base.FromContext(ctx)
	if err != nil {
		return base.ErrNotFoundCollectors
	}

	for _, v := range utils.ReverseStringSlice(providersOrder()) {
		if p, ok := provs.Load(v); ok {
			if err := p.(base.EntityGateway).Shutdown(ctx); err != nil {
				return errors.Wrap(err, "failed to shutdown providers")
			}
		}
	}

	return nil
}
