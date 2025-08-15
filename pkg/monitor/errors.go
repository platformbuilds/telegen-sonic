package monitor

import "errors"

var (
	ErrConcurrencyLimit = errors.New("At most 2 concurrent jobs are allowed. Try again later.")
	ErrJobNotFound = errors.New("job not found")
)
