package async

import "errors"

// ErrLoading is returned when Get() is called while a fetch is in progress.
var ErrLoading = errors.New("value is currently loading")
