package logger

type Config struct {
	// NoColoredOutput forces logger to output things without
	// shell colorcodes.
	NoColoredOutput bool `envconfig:"optional"`
	// Show trace information (file name, line number, function name)?
	WithTrace bool `envconfig:"optional"`
	// Level is a logger's loglevel. Possible values: "DEBUG",
	// "INFO", "WARN", "ERROR", "FATAL", "TRACE". Setting this variable
	// to any other value will force INFO level.
	// Case-insensitive value.
	Level string `envconfig:"optional"`
}

func DefaultConfig() *Config {
	return &Config{
		Level:           LoggerLevelDisabled,
		NoColoredOutput: true,
		WithTrace:       false,
	}
}

func (c *Config) Validate() {
	if c.Level == "" {
		c.Level = "INFO"
	}
}
