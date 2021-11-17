package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/soldatov-s/go-garage/base"
)

type Logger struct {
	zerolog zerolog.Logger
	*base.MetricsStorage
}

func NewLogger(ctx context.Context, config *Config) (*Logger, error) {
	logger := &Logger{
		MetricsStorage: base.NewMetricsStorage(),
	}
	config.SetDefault()
	level, err := zerolog.ParseLevel(strings.ToLower(config.Level))
	if err != nil {
		return nil, errors.Wrap(err, "parse level")
	}

	zerolog.SetGlobalLevel(level)

	output := buildLoggerOutput(config.HumanFriendly, config.NoColoredOutput)

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	l := zerolog.New(output).With().Timestamp().Logger()
	l = l.Hook(NewTracingHook(config.WithTrace))

	logger.zerolog = l
	if err := logger.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	return logger, nil
}

func (l *Logger) Zerolog() *zerolog.Logger {
	return &l.zerolog
}

func buildLoggerOutput(isHumanFriendly, isNoColoredOutput bool) io.Writer {
	if !isHumanFriendly {
		return os.Stdout
	}

	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    isNoColoredOutput,
		TimeFormat: time.RFC3339,
	}

	output.FormatLevel = func(i interface{}) string {
		var v string

		if ii, ok := i.(string); ok {
			ii = strings.ToUpper(ii)
			switch ii {
			case zerolog.DebugLevel.String(), zerolog.ErrorLevel.String(), zerolog.FatalLevel.String(),
				zerolog.InfoLevel.String(), zerolog.WarnLevel.String(), zerolog.PanicLevel.String(),
				zerolog.TraceLevel.String():
				v = fmt.Sprintf("%-5s", ii)
			default:
				v = ii
			}
		}

		return fmt.Sprintf("| %s |", v)
	}

	return output
}

func (l *Logger) buildMetrics(_ context.Context) error {
	fullName := "logger"

	helpWarns := "How many warnings occurred."
	warnsMetric, err := l.MetricsStorage.GetMetrics().AddIncCounter(fullName, "warns total", helpWarns)
	if err != nil {
		return errors.Wrap(err, "add counter metric")
	}
	l.zerolog = l.zerolog.Hook(NewMetricWarnHook(warnsMetric))

	helpErrors := "How many errors occurred."
	errorsMetric, err := l.MetricsStorage.GetMetrics().AddIncCounter(fullName, "errors total", helpErrors)
	if err != nil {
		return errors.Wrap(err, "add counter metric")
	}
	l.zerolog = l.zerolog.Hook(NewMetricErrorHook(errorsMetric))

	return nil
}
