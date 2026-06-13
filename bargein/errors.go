package bargein

import "errors"

// ErrNoSTTEvents is returned when Start is called without attaching STT events.
var ErrNoSTTEvents = errors.New("no STT events attached")
