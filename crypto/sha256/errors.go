package sha256

import "errors"

var (
	// ErrMismatchedHashAndPassword the error returned from CompareHashAndPassword when a password and hash do
	// not match.
	ErrMismatchedHashAndPassword = errors.New("hashedPassword is not the hash of the given password")
	// ErrBadAlgorithm the error returned from newFromHash when a hash not include necessary prefix
	ErrBadAlgorithm = errors.New("bad algorithm")
)
