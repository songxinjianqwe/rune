package util

import "time"

// ErrorCode is the API util code type.
type ErrorCode int

// API util codes.
const (
	// Factory errors
	IdInUse ErrorCode = iota
	InvalidIdFormat

	// Container errors
	ContainerNotExists
	ContainerPaused
	ContainerNotStopped
	ContainerNotRunning
	ContainerNotPaused

	// Process errors
	NoProcessOps

	// Common errors
	ConfigInvalid
	ConsoleExists
	SystemError
)

func (c ErrorCode) String() string {
	switch c {
	case IdInUse:
		return "Id already in use"
	case InvalidIdFormat:
		return "Invalid format"
	case ContainerPaused:
		return "Container paused"
	case ConfigInvalid:
		return "Invalid configuration"
	case SystemError:
		return "System util"
	case ContainerNotExists:
		return "Container does not exist"
	case ContainerNotStopped:
		return "Container is not stopped"
	case ContainerNotRunning:
		return "Container is not running"
	case ConsoleExists:
		return "Console exists for process"
	case ContainerNotPaused:
		return "Container is not paused"
	case NoProcessOps:
		return "No process operations"
	default:
		return "Unknown util"
	}
}

// Error is the API util type.
type Error interface {
	error
	// Returns the util code for this util.
	Code() ErrorCode
}

func NewGenericError(err error, c ErrorCode) Error {
	if le, ok := err.(Error); ok {
		return le
	}
	genericError := &GenericError{
		Timestamp: time.Now(),
		Cause:     err,
		ErrorCode: c,
	}
	if err != nil {
		genericError.Message = err.Error()
	}
	return genericError
}

type GenericError struct {
	Timestamp time.Time
	ErrorCode ErrorCode
	Cause     error `json:"-"`
	Message   string
}

func (e *GenericError) Error() string {
	return e.Message
}

func (e *GenericError) Code() ErrorCode {
	return e.ErrorCode
}