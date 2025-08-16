package monitor

import "errors"

var (
	ErrConcurrencyLimit = errors.New("only 2 concurrent jobs are allowed, try again later")
	ErrJobNotFound      = errors.New("job not found")
)
