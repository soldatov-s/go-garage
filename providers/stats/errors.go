package stats

import (
	"errors"

	goworkerr "github.com/soldatov-s/go-garage/providers/errors"
)

var (
	ErrEmptyStatistics     = errors.New("empty statistics")
	ErrEmptyDependencyName = errors.New("empty dependency name")
	ErrCheckFuncIsNil      = errors.New("pointer to checkFunc is nil")
	ErrEmptyMetricName     = errors.New("empty metric name")
)

func ErrInvalidProviderOptions(iface interface{}) error {
	return errors.New("passed provider options is not *" +
		goworkerr.ObjName(iface))
}

func ErrInvalidMetricOptions(iface interface{}) error {
	return errors.New("passed metric options is not *" +
		goworkerr.ObjName(iface))
}
