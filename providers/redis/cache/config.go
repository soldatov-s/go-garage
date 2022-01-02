package rediscache

import (
	"time"
)

const (
	defaulKeyPrefix = "garage_"
	defaultScanSize = 10
)

type Config struct {
	// KeyPrefix is a prefix for eache key in redis
	KeyPrefix string `envconfig:"optional"`
	// ClearTime is a time of live item
	ClearTime       time.Duration `envconfig:"optional"`
	ScanSize        int64         `envconfig:"optional"`
	GlobalKeyPrefix string        `envconfig:"-"`
}

// SetDefault checks connection options. If required field is empty - it will
// be filled with some default value.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if cfgCopy.KeyPrefix == "" {
		cfgCopy.KeyPrefix = defaulKeyPrefix
	}

	if cfgCopy.ScanSize == 0 {
		cfgCopy.ScanSize = defaultScanSize
	}

	return &cfgCopy
}
