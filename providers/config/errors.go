package config

import (
	"errors"
)

var (
	ErrFailedParse = errors.New("failed to parse config")
	// ErrConfigNotABoolean appears when trying to enable trace mode
	// for logger in runtime, but not a boolean was passed.
	ErrConfigNotABoolean = errors.New("passed value is not boolean")

	// ErrConfigNotAString appears when trying to change logging level
	// for logger in runtime, but not a string was passed.
	ErrConfigNotAString = errors.New("passed value is not a string")

	// ErrConfigUnknownLoggingConfigurationKey appears when trying to
	// change logger configuration in runtime, but unknown configuration
	// key was passed. See Configuration.SetLoggerConfiguration() function
	// switch for a list of supported keys.
	ErrConfigUnknownLoggingConfigurationKey = errors.New("unknown logger configuration key")

	// ErrConfigUnknownLoggingLevel appears when trying to change logging
	// level and passed string isn't a supported logging level. See
	// logger.go for a list of supported logging levels, or use your IDE's
	// autocompletion with "gowork.LoggerLevel".
	ErrConfigUnknownLoggingLevel = errors.New("unknown logging level, disable logging")

	ErrConfigNoProviderRegistred = errors.New("no configuration providers registered")
)
