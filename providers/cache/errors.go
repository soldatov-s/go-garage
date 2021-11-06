package cache

import "errors"

var (
	ErrEmptyCachees    = errors.New("empty caches")
	ErrNotFoundInCache = errors.New("not found in cache")
)
