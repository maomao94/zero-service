package antsx

import "errors"

var (
	ErrPendingExpired = errors.New("antsx: pending entry expired")
	ErrDuplicateID    = errors.New("antsx: duplicate promise id")
	ErrRegistryClosed = errors.New("antsx: registry closed")
)
