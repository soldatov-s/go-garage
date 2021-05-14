package echo

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/base"
)

// HandlerLogger handler that adds requesID into logger fields
func HandlerLogger(parent *zerolog.Logger, ec echo.Context) (log zerolog.Logger, requestID string, err error) {
	if parent == nil {
		return zerolog.Logger{}, "", base.ErrLoggerPointerIsNil
	}

	requestID = ec.Request().Header.Get("x-request-id")
	if requestID == "" {
		newUUID, err := uuid.NewUUID()
		if err != nil {
			parent.Error().Err(err).Msg("Failed to generate new requestID")
			return *parent, "", nil
		}
		requestID = strings.ReplaceAll(newUUID.String(), "-", "")
	}
	log = parent.With().Str("requestID", requestID).Logger()

	return log, requestID, nil
}
