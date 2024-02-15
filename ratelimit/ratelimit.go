package ratelimit

import (
	"context"
	"fmt"
	"time"
)

type Limiter interface {
	Increment(context.Context, string, int) (*LimitStatus, error)
}

type LimitStatus struct {
	Remaining int
	Limit     int
	Period    time.Duration
	Reset     time.Time
}

type ErrRateLimitExceeded LimitStatus

func (e ErrRateLimitExceeded) Error() string {
	return fmt.Sprintf(
		"rate limit of %d per %v has been exceeded and resets at %v",
		e.Limit, e.Period, e.Reset)
}
