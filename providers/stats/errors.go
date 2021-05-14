package stats

import (
	"errors"
)

var (
	ErrEmptyStatistics      = errors.New("empty statistics")
	ErrEmptyDependencyName  = errors.New("empty dependency name")
	ErrCheckFuncIsNil       = errors.New("pointer to checkFunc is nil")
	ErrEmptyMetricName      = errors.New("empty metric name")
	ErrInvalidMetricOptions = errors.New("passed metric options is not valid")
)
