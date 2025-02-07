package errors

type ErrorCode string

const (
	ErrNotFound ErrorCode = "NotFound"
	ErrInternal ErrorCode = "Internal"
)
