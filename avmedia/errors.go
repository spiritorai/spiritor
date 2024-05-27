package avmedia

import "fmt"

// Style Guide Error Rules
// * Public functions should only return these errors
// * Private function should not return these errors
// * If calling a public function from inside this package then do
//   not double wrap the error, use %w to embed it instead

// File Operation
// All downstream filesystem operations are wrapped in this. This should
// be considered a critical error by the caller.

type ErrFileOp struct {
	Err error
}

func (e ErrFileOp) Error() string {
	return fmt.Sprintf("file operation error: %v", e.Err)
}

func (e ErrFileOp) Unwrap() error {
	return e.Err
}

// Validation
// All validation related errors are wrapped in this. Callers should
// avoid these errors through prevalidation.

type ErrValidation struct {
	Err error
}

func (e ErrValidation) Error() string {
	return fmt.Sprintf("validation error: %v", e.Err)
}

func (e ErrValidation) Unwrap() error {
	return e.Err
}

// Size Cap Exceeded
// Used specifically for when a transform is unable to shrink the
// file below the size cap. Callers may consider this error recoverable.

type ErrSizeCapExceeded struct {
	SizeCap  int64
	FileSize int64
}

func (e ErrSizeCapExceeded) Error() string {
	return fmt.Sprintf("file size [%v] exceeded the maximum size cap: %v", e.SizeCap, e.FileSize)
}
