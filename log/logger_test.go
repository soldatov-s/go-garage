package log

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestLoggerInitialization(t *testing.T) {
	logger := NewLogger(&Config{Level: LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})
	// zerolog.Logger's thing (Logger) isn't a pointer, so we set another
	// boolean variable to determine that logger was initialized.
	require.True(t, logger.IsInitialized())
}

func testLoggerInitializationWithSettedLoggingLevel(t *testing.T, level string) *zerolog.Logger {
	logger := NewLogger(&Config{Level: level, NoColoredOutput: true, WithTrace: false})
	require.True(t, logger.IsInitialized())
	return logger.GetLogger("test", nil)
}

func TestLoggerInitializationWithDebugLoggingLevel(t *testing.T) {
	log := testLoggerInitializationWithSettedLoggingLevel(t, "DeBuG")
	log.Debug().Msg("Debug level test. Message should be visible.")
}

func TestLoggerInitializationWithInfoLoggingLevel(t *testing.T) {
	log := testLoggerInitializationWithSettedLoggingLevel(t, "iNFo")
	log.Info().Msg("Info level test. Message should be visible.")
}

func TestLoggerInitializationWithWarnLoggingLevel(t *testing.T) {
	log := testLoggerInitializationWithSettedLoggingLevel(t, "WarN")
	log.Warn().Msg("Warn level test. Message should be visible.")
}

func TestLoggerInitializationWithErrorLoggingLevel(t *testing.T) {
	log := testLoggerInitializationWithSettedLoggingLevel(t, "eRRoR")
	log.Error().Msg("Error level test. Message should be visible.")
}

func TestLoggerInitializationWithTraceLoggingLevel(t *testing.T) {
	log := testLoggerInitializationWithSettedLoggingLevel(t, "TraCe")
	log.Trace().Msg("Trace level test. Message should be visible.")
}

// Calling Logger.Fatal() will also call os.Exit() after message
// printing, won't test here.
func TestLoggerInitializationWithFatalLoggingLevel(t *testing.T) {
	testLoggerInitializationWithSettedLoggingLevel(t, "FaTAl")
}

func TestLoggerInitializationWithInvalidLoggerLevel(t *testing.T) {
	testLoggerInitializationWithSettedLoggingLevel(t, "BaD")
}

func TestLoggerInitializationWithFields(t *testing.T) {
	logger := NewLogger(&Config{Level: LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

	fields := []*Field{
		{Name: "bool", Value: true},
		{Name: "float32", Value: float32(32.00)},
		{Name: "float64", Value: float64(64.00)},
		{Name: "int", Value: int(0)},
		{Name: "int8", Value: int8(8)},
		{Name: "int16", Value: int16(16)},
		{Name: "int32", Value: int32(32)},
		{Name: "int64", Value: int64(64)},
		{Name: "interface", Value: "interface value"},
		{Name: "string", Value: "test string"},
		{Name: "uint", Value: uint(0)},
		{Name: "uint8", Value: uint8(8)},
		{Name: "uint16", Value: uint16(16)},
		{Name: "uint32", Value: uint32(32)},
		{Name: "uint64", Value: uint64(64)},
	}

	log := logger.GetLogger("test", fields...)
	log.Info().Msg("Test")
}
