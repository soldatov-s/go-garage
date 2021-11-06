package echo

// import (
// 	"net/http"
// 	"os"
// 	"testing"

// 	"github.com/labstack/echo/v4"
// 	"github.com/rs/zerolog"
// 	"github.com/soldatov-s/go-garage/providers/logger"
// 	"github.com/stretchr/testify/require"
// )

// type EmptyWriter struct{}

// func (e *EmptyWriter) Header() http.Header {
// 	return make(map[string][]string)
// }

// func (e *EmptyWriter) Write(data []byte) (int, error) {
// 	return 0, nil
// }

// func (e *EmptyWriter) WriteHeader(statusCode int) {
// }

// func TestHandlerLogger(t *testing.T) {
// 	log := &logger.Logger{}
// 	log.Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	eh := echo.New()
// 	req, err := http.NewRequest("GET", "http://localhost", nil)

// 	require.Nil(t, err)

// 	ec := eh.NewContext(req, &EmptyWriter{})

// 	l := log.GetLogger("httpsrv", nil)
// 	zeroLog, requestID, err := HandlerLogger(l, ec)

// 	// Output message "test" with requestID
// 	zeroLog.Info().Msg("test")

// 	require.Nil(t, err)
// 	require.NotEqual(t, "", requestID)
// }

// func TestHandlerLoggerWithEmptyParent(t *testing.T) {
// 	eh := echo.New()
// 	req, err := http.NewRequest("GET", "http://localhost", nil)

// 	require.Nil(t, err)

// 	ec := eh.NewContext(req, &EmptyWriter{})

// 	zeroLog, requestID, err := HandlerLogger(nil, ec)

// 	// Output message "test" with requestID
// 	log := zeroLog.Level(zerolog.InfoLevel).Output(os.Stdout)
// 	log.Info().Msg("test")

// 	require.NotNil(t, err)
// 	require.Equal(t, "", requestID)
// }
