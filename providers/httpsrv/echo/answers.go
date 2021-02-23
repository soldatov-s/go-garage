package echo

import (
	"net/http"

	"github.com/soldatov-s/go-garage/providers/httpsrv"
)

// BadRequest return err 400
func BadRequest(ec Context, err error) error {
	return ec.JSON(
		http.StatusBadRequest,
		httpsrv.BadRequest(err),
	)
}

// Unauthorized return err 401
func Unauthorized(ec Context, err error) error {
	return ec.JSON(
		http.StatusUnauthorized,
		httpsrv.Unauthorized(err),
	)
}

// Forbidden return err 403
func Forbidden(ec Context, err error) error {
	return ec.JSON(
		http.StatusForbidden,
		httpsrv.Forbidden(err),
	)
}

// NotFound return err 404
func NotFound(ec Context, err error) error {
	return ec.JSON(
		http.StatusNotFound,
		httpsrv.NotFound(err),
	)
}

// NotDeleted return err 406
func NotDeleted(ec Context, err error) error {
	return ec.JSON(
		http.StatusNotAcceptable,
		httpsrv.NotDeleted(err),
	)
}

// HasExpired return err 408
func HasExpired(ec Context, err error) error {
	return ec.JSON(
		http.StatusRequestTimeout,
		httpsrv.HasExpired(err),
	)
}

// CreateFailed return err 409
func CreateFailed(ec Context, err error) error {
	return ec.JSON(
		http.StatusConflict,
		httpsrv.CreateFailed(err),
	)
}

// NotUpdated return err 409
func NotUpdated(ec Context, err error) error {
	return ec.JSON(
		http.StatusConflict,
		httpsrv.NotUpdated(err),
	)
}

// InternalServerError return err 500
func InternalServerError(ec Context, err error) error {
	return ec.JSON(
		http.StatusInternalServerError,
		httpsrv.InternalServerError(err),
	)
}

func OK(ec Context, data interface{}) error {
	return ec.JSON(
		http.StatusOK,
		data,
	)
}

func OkResult(ec Context) error {
	return ec.JSON(
		http.StatusOK,
		httpsrv.OkResult(),
	)
}
