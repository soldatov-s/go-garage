package base

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/x/stringsx"
)

type Enity struct {
	name           string
	providerName   string
	shuttingDown   bool
	watcherStopped bool
}

type EnityDeps struct {
	Name         string
	ProviderName string
}

func NewEnity(deps *EnityDeps) *Enity {
	return &Enity{
		name:           deps.Name,
		providerName:   deps.ProviderName,
		watcherStopped: true,
	}
}

func (e *Enity) GetName() string {
	return e.name
}

func (e *Enity) GetFullName() string {
	return stringsx.JoinStrings("_", e.providerName, e.name)
}

func (e *Enity) GetLogger(ctx context.Context) *zerolog.Logger {
	logger := zerolog.Ctx(ctx).With().Logger()
	logger = logger.With().Str("provider_type", e.providerName).Str("enity_name", e.name).Logger()
	return &logger
}

func (e *Enity) IsShuttingDown() bool {
	return e.shuttingDown
}

func (e *Enity) SetShuttingDown(v bool) {
	e.shuttingDown = v
}

func (e *Enity) IsWatcherStopped() bool {
	return e.watcherStopped
}

func (e *Enity) SetWatcher(v bool) {
	e.watcherStopped = v
}
