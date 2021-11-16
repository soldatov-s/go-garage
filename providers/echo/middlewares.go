package echo

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var ErrNotFoundZerolog = errors.New("not found zerolog")

// CORSDefault allows requests from any origin wth GET, HEAD, PUT, POST or DELETE method.
func CORSDefault() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
		},
		AllowHeaders: []string{
			"Accept",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"X-Request-Id",
		},
	})
}

func generator(ctx context.Context) string {
	logger := zerolog.Ctx(ctx)
	rid := ""
	id, err := uuid.NewRandom()
	if err != nil {
		logger.Err(err).Msg("generate request id")
	} else {
		rid = id.String()
	}
	return rid
}

func RequestID(ctx context.Context) echo.MiddlewareFunc {
	requestIDConfig := middleware.RequestIDConfig{
		Skipper: middleware.DefaultSkipper,
		Generator: func() string {
			return generator(ctx)
		},
	}

	return middleware.RequestIDWithConfig(requestIDConfig)
}

func GetReqID(ec echo.Context) string {
	return ec.Request().Header.Get("x-request-id")
}

const zerologCtxKey = "zerolog"

func HydrationZerolog(ctx context.Context) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get("x-request-id")
			l := zerolog.Ctx(ctx).With().Str("request_id", requestID).Logger()
			c.Set(zerologCtxKey, &l)
			return next(c)
		}
	}
}

func GetZerologger(ec echo.Context) (*zerolog.Logger, error) {
	l := ec.Get(zerologCtxKey)
	logger, ok := l.(*zerolog.Logger)
	if !ok {
		return nil, ErrNotFoundZerolog
	}
	return logger, nil
}
