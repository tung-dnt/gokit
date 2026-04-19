package recipe

import "errors"

// ErrIndexingFailed indicates that document indexing failed.
var ErrIndexingFailed = errors.New("recipe: indexing failed")

// ErrRetrievalFailed indicates that document retrieval failed.
var ErrRetrievalFailed = errors.New("recipe: retrieval failed")
