package echo

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

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

func HydrationRequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get("x-request-id")
			if requestID != "" {
				return next(c)
			}

			id, _ := uuid.NewRandom()
			c.Request().Header.Set("x-request-id", id.String())
			return next(c)
		}
	}
}

const zerologWithReqID = "garageZerologWithReqID"

func HydrationLogger(log *zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get("x-request-id")
			if requestID == "" {
				c.Set(zerologWithReqID, log)
			} else {
				l := log.With().Str("x-request-id", requestID).Logger()
				c.Set(zerologWithReqID, &l)
			}

			return next(c)
		}
	}
}
