package log

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const (
	LoggerLevelDebug    = "DEBUG"
	LoggerLevelInfo     = "INFO"
	LoggerLevelWarn     = "WARN"
	LoggerLevelError    = "ERROR"
	LoggerLevelFatal    = "FATAL"
	LoggerLevelPanic    = "PANIC"
	LoggerLevelTrace    = "TRACE"
	LoggerLevelDisabled = "DISABLED"
)

// Logger is a controlling structure for application's logger.
type Logger struct {
	// Main logger
	logger zerolog.Logger
	// Other created loggers.
	loggers     sync.Map
	initialized bool
}

// nolint
func loggerCtxByField(loggerCtx *zerolog.Context, field *Field) zerolog.Context {
	ctx := *loggerCtx
	switch field.Value.(type) {
	case bool:
		ctx = ctx.Bool(field.Name, field.Value.(bool))
	case float32:
		ctx = ctx.Float32(field.Name, field.Value.(float32))
	case float64:
		ctx = loggerCtx.Float64(field.Name, field.Value.(float64))
	case int:
		ctx = ctx.Int(field.Name, field.Value.(int))
	case int8:
		ctx = ctx.Int8(field.Name, field.Value.(int8))
	case int16:
		ctx = loggerCtx.Int16(field.Name, field.Value.(int16))
	case int32:
		ctx = loggerCtx.Int32(field.Name, field.Value.(int32))
	case int64:
		ctx = ctx.Int64(field.Name, field.Value.(int64))
	case interface{}:
		ctx = ctx.Interface(field.Name, field.Value)
	case string:
		ctx = ctx.Str(field.Name, field.Value.(string))
	case uint:
		ctx = ctx.Uint(field.Name, field.Value.(uint))
	case uint8:
		ctx = ctx.Uint8(field.Name, field.Value.(uint8))
	case uint16:
		ctx = ctx.Uint16(field.Name, field.Value.(uint16))
	case uint32:
		ctx = ctx.Uint32(field.Name, field.Value.(uint32))
	case uint64:
		ctx = ctx.Uint64(field.Name, field.Value.(uint64))
	}
	return ctx
}

// GetLogger creates new logger if not exists and fills it with defined fields.
// If requested logger already exists - it'll be returned.
func (l *Logger) GetLogger(name string, fields ...*Field) *zerolog.Logger {
	logger, found := l.loggers.Load(name)
	if found {
		return logger.(*zerolog.Logger)
	}

	loggerCtx := l.logger.With()

	for _, field := range fields {
		loggerCtx = loggerCtxByField(&loggerCtx, field)
	}

	log := loggerCtx.Logger()
	l.loggers.Store(name, &log)
	return &log
}

func NewLogger(cfg *Config) *Logger {
	l := &Logger{}

	cfg.SetDefault()
	switch strings.ToUpper(cfg.Level) {
	case LoggerLevelDebug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case LoggerLevelInfo:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case LoggerLevelWarn:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case LoggerLevelError:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case LoggerLevelFatal:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case LoggerLevelPanic:
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case LoggerLevelTrace:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case LoggerLevelDisabled:
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	var output io.Writer = os.Stdout

	if cfg.HumanFriendly {
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			NoColor:    cfg.NoColoredOutput,
			TimeFormat: time.RFC3339,
		}

		output.FormatLevel = func(i interface{}) string {
			var v string

			if ii, ok := i.(string); ok {
				ii = strings.ToUpper(ii)
				switch ii {
				case LoggerLevelDebug, LoggerLevelError, LoggerLevelFatal,
					LoggerLevelInfo, LoggerLevelWarn, LoggerLevelPanic, LoggerLevelTrace:
					v = fmt.Sprintf("%-5s", ii)
				default:
					v = ii
				}
			}

			return fmt.Sprintf("| %s |", v)
		}
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	l.logger = zerolog.New(output).With().Timestamp().Logger().Hook(TracingHook{WithTrace: cfg.WithTrace})

	l.initialized = true

	return l
}

// IsInitialized returns true if logger was initialized and configured.
func (l *Logger) IsInitialized() bool {
	return l.initialized
}

// BuildPackageLogger initilizes fields for domains and packages
func BuildPackageLogger(parent *zerolog.Logger, emptyStruct interface{}) *zerolog.Logger {
	log := parent.With().Logger()
	res := false
	packageTypes := []string{"domains", "internal"}
	packageType := ""
	pkgPath := reflect.TypeOf(emptyStruct).PkgPath()
	pathElements := strings.Split(pkgPath, "/")

	i := 0
	for _, pathElement := range pathElements {
		for _, packageType = range packageTypes {
			if pathElement == packageType {
				log = log.With().Str("type", pathElements[i]).Logger()
				log = log.With().Str("package", pathElements[i+1]).Logger()
				res = true

				break
			}
		}

		if res {
			break
		}
		i++
	}

	if packageType == "domains" {
		ver, _ := strconv.ParseInt(strings.TrimPrefix(pathElements[i+2], "v"), 10, 64)
		log = log.With().Int64("version", ver).Logger()
		i++
	}

	if i+3 <= len(pathElements) {
		log = log.With().Str("subsystem", pathElements[i+2]).Logger()
	}

	log.Info().Msg("initializing loggers...")

	return &log
}
