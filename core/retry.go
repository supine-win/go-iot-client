package core

import "time"

type RetryPolicy struct {
	MaxRetries int
	RetryDelay time.Duration
}

func (p RetryPolicy) Normalize() RetryPolicy {
	if p.MaxRetries < 0 {
		p.MaxRetries = 0
	}
	if p.RetryDelay <= 0 {
		p.RetryDelay = 100 * time.Millisecond
	}
	return p
}

