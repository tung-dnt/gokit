package usermodel

import "errors"

// ErrNotFound indicates that the requested user does not exist.
var ErrNotFound = errors.New("user: not found")
