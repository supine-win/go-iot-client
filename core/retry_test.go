package core

import (
	"testing"
	"time"
)

func TestRetryPolicyNormalize(t *testing.T) {
	p := RetryPolicy{MaxRetries: -1, RetryDelay: 0}.Normalize()
	if p.MaxRetries != 0 {
		t.Fatalf("expect maxRetries=0 got %d", p.MaxRetries)
	}
	if p.RetryDelay != 100*time.Millisecond {
		t.Fatalf("expect default retryDelay 100ms got %v", p.RetryDelay)
	}
}

