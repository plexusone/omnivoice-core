package pipeline

import "errors"

var (
	// ErrPipelineActive is returned when trying to start a pipeline that is already active.
	ErrPipelineActive = errors.New("pipeline is already active")
)
