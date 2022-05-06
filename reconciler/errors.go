package reconciler

import (
	"time"
)

type ReconcileError struct {
	Message    string
	RetryAfter time.Duration
	ReRaise    bool
}

func (e *ReconcileError) Error() string {
	return e.Message
}

func (e *ReconcileError) Err() error {
	if e.ReRaise {
		return e
	}
	return nil
}

type ErrorOptionFunc func(e *ReconcileError)

func WithRetryAfter(d time.Duration) ErrorOptionFunc {
	return func(e *ReconcileError) {
		e.RetryAfter = d
	}
}

func DisableReRaise() ErrorOptionFunc {
	return func(e *ReconcileError) {
		e.ReRaise = false
	}
}

func NewReconcileError(msg string, opts ...ErrorOptionFunc) *ReconcileError {
	e := &ReconcileError{
		Message: msg,
		ReRaise: true,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func IsReconcileError(e error) (*ReconcileError, bool) {
	err, ok := e.(*ReconcileError)
	return err, ok
}
