package echo

import (
	"net/http"

	"github.com/soldatov-s/go-garage/providers/httpsrv"
)

// BadRequest return err 400
func (ec Context) BadRequest(err error) error {
	return ec.JSON(
		http.StatusBadRequest,
		httpsrv.BadRequest(err),
	)
}

// Unauthorized return err 401
func (ec Context) Unauthorized(err error) error {
	return ec.JSON(
		http.StatusUnauthorized,
		httpsrv.Unauthorized(err),
	)
}

// Forbidden return err 403
func (ec Context) Forbidden(err error) error {
	return ec.JSON(
		http.StatusForbidden,
		httpsrv.Forbidden(err),
	)
}

// NotFound return err 404
func (ec Context) NotFound(err error) error {
	return ec.JSON(
		http.StatusNotFound,
		httpsrv.NotFound(err),
	)
}

// NotDeleted return err 406
func (ec Context) NotDeleted(err error) error {
	return ec.JSON(
		http.StatusNotAcceptable,
		httpsrv.NotDeleted(err),
	)
}

// HasExpired return err 408
func (ec Context) HasExpired(err error) error {
	return ec.JSON(
		http.StatusRequestTimeout,
		httpsrv.HasExpired(err),
	)
}

// CreateFailed return err 409
func (ec Context) CreateFailed(err error) error {
	return ec.JSON(
		http.StatusConflict,
		httpsrv.CreateFailed(err),
	)
}

// NotUpdated return err 409
func (ec Context) NotUpdated(err error) error {
	return ec.JSON(
		http.StatusConflict,
		httpsrv.NotUpdated(err),
	)
}

// InternalServerError return err 500
func (ec Context) InternalServerError(err error) error {
	return ec.JSON(
		http.StatusInternalServerError,
		httpsrv.InternalServerError(err),
	)
}

func (ec Context) OK(data interface{}) error {
	return ec.JSON(
		http.StatusOK,
		data,
	)
}

func (ec Context) OkResult() error {
	return ec.JSON(
		http.StatusOK,
		httpsrv.OkResult(),
	)
}
