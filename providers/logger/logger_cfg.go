package logger

type Config struct {
	// NoColoredOutput forces logger to output things without
	// shell colorcodes.
	NoColoredOutput bool `envconfig:"default=true"`
	// Show trace information (file name, line number, function name)?
	WithTrace bool `envconfig:"default=false"`
	// Level is a logger's loglevel. Possible values: "DEBUG",
	// "INFO", "WARN", "ERROR", "FATAL", "TRACE". Setting this variable
	// to any other value will force INFO level.
	// Case-insensitive value.
	Level string `envconfig:"default=INFO"`
}

func DefaultConfig() *Config {
	return &Config{
		Level:           LoggerLevelDisabled,
		NoColoredOutput: true,
		WithTrace:       false,
	}
}
