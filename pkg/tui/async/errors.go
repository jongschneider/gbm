package async

import "fmt"

// ErrLoading is returned when Get() is called while a fetch is in progress.
var ErrLoading = fmt.Errorf("value is currently loading")
