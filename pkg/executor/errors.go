package executor

import "errors"

var (
	// ErrExecutorNotFound is returned when the requested executor is not registered
	ErrExecutorNotFound = errors.New("executor not found")
	
	// ErrActionNotSupported is returned when the executor doesn't support the requested action
	ErrActionNotSupported = errors.New("action not supported by executor")
	
	// ErrNilExecutor is returned when trying to register a nil executor
	ErrNilExecutor = errors.New("executor cannot be nil")
	
	// ErrInvalidPayload is returned when the task payload is invalid
	ErrInvalidPayload = errors.New("invalid task payload")
	
	// ErrExecutionFailed is returned when task execution fails
	ErrExecutionFailed = errors.New("task execution failed")
	
	// ErrExecutorClosed is returned when trying to use a closed executor
	ErrExecutorClosed = errors.New("executor is closed")
)