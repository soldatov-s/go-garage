package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/soldatov-s/go-garage/domains"
	"github.com/soldatov-s/go-garage/meta"
	"github.com/soldatov-s/go-garage/providers"
	baseproviders "github.com/soldatov-s/go-garage/providers/base/providers"
	"github.com/soldatov-s/go-garage/providers/cache"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
)

// Loop is application loop, exit on SIGTERM
func Loop(ctx context.Context) {
	var closeSignal chan os.Signal
	m := meta.Get(ctx)
	log := logger.Get(ctx).GetLogger(m.Name, nil)

	exit := make(chan struct{})
	closeSignal = make(chan os.Signal, 1)
	signal.Notify(closeSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-closeSignal
		_ = Shutdown(ctx)
		log.Info().Msg("Exit program")
		close(exit)
	}()

	// Exit app if chan is closed
	<-exit
}

// getAllMetrics return all metrics from databases and caches
func getAllMetrics(ctx context.Context) (stats.MapMetricsOptions, error) {
	metrics := make(stats.MapMetricsOptions)
	p := providers.Get(ctx)
	var err error
	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(stats.IProvidersMetrics); ok {
			if _, err = m.GetAllMetrics(metrics); err != nil {
				return false
			}
		}

		return true
	})

	return metrics, err
}

// getAllAliveHandlers return all aliveHandlers from databases and caches
// nolint : duplicate
func getAllAliveHandlers(ctx context.Context) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)
	p := providers.Get(ctx)
	var err error
	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(stats.IProvidersMetrics); ok {
			if _, err = m.GetAllAliveHandlers(handlers); err != nil {
				return false
			}
		}

		return true
	})
	return handlers, err
}

// getAllReadyHandlers return all readyHandlers from databases and caches
// nolint : duplicate
func getAllReadyHandlers(ctx context.Context) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)
	p := providers.Get(ctx)
	var err error
	p.Range(func(k, v interface{}) bool {
		if m, ok := v.(stats.IProvidersMetrics); ok {
			if _, err = m.GetAllReadyHandlers(handlers); err != nil {
				return false
			}
		}

		return true
	})
	return handlers, err
}

func StartStatistics(ctx context.Context) error {
	s := stats.Get(ctx)
	if s == nil {
		return nil
	}

	// Collecting all metrics from context
	metrics, err := getAllMetrics(ctx)
	if err != nil {
		return err
	}

	// Collecting all aliveHandlers from context
	aliveHandlers, err := getAllAliveHandlers(ctx)
	if err != nil {
		return err
	}

	// Collecting all readyHandlers from context
	readyHandlers, err := getAllReadyHandlers(ctx)
	if err != nil {
		return err
	}

	return s.Start(metrics, aliveHandlers, readyHandlers)
}

func providersOrder() []string {
	return []string{db.ProvidersName, cache.ProvidersName, httpsrv.ProvidersName}
}

func CreateAppContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, _ = providers.Create(ctx)
	ctx, _ = domains.Create(ctx)
	return ctx
}

// Start all providers
func Start(ctx context.Context) error {
	provs := providers.Get(ctx)
	for _, v := range providersOrder() {
		if p, ok := provs.Load(v); ok {
			if err := p.(baseproviders.IBaseProviders).Start(); err != nil {
				return err
			}
		}
	}

	return StartStatistics(ctx)
}

// Shutdown all providers
func Shutdown(ctx context.Context) error {
	provs := providers.Get(ctx)
	for _, v := range utils.ReverseStringSlice(providersOrder()) {
		if p, ok := provs.Load(v); ok {
			if err := p.(baseproviders.IBaseProviders).Shutdown(); err != nil {
				return err
			}
		}
	}

	return nil
}
