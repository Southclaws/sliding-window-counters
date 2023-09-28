package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/Southclaws/sliding-window-counters/ratelimit"
)

type Redis struct {
	client        *redis.Client
	limit         int
	limitPeriod   time.Duration // 1 hour for limitPeriodle
	counterWindow time.Duration // 1 minute for example, 1/60 of the period
}

func New(client *redis.Client, limit int, period, expiry time.Duration) *Redis {
	return &Redis{client, limit, period, expiry}
}

func (r *Redis) Increment(ctx context.Context, key string, incr int) (*ratelimit.LimitStatus, error) {
	now := time.Now()
	timestamp := fmt.Sprint(now.Truncate(r.counterWindow).Unix())

	val, err := r.client.HIncrBy(ctx, key, timestamp, int64(incr)).Result()
	if err != nil {
		return nil, err
	}

	if val == 1 {
		// If this hash was only just created, set its expiry.
		r.client.Expire(ctx, key, r.limitPeriod)
	} else if val >= int64(r.limit) {
		// Otherwise, check if just this fixed window counter period is over
		return nil, ratelimit.ErrRateLimitExceeded{
			Remaining: 0, Limit: r.limit, Period: r.limitPeriod, Reset: now.Add(r.limitPeriod)}
	}

	// Get all the bucket values and sum them.
	vals, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// The time to start summing from, any buckets before this are ignored.
	threshold := fmt.Sprint(now.Add(-r.limitPeriod).Unix())

	// NOTE: This sums ALL the values in the hash, for more information, see the
	// "Practical Considerations" section of the associated Figma blog post.
	total := 0
	for k, v := range vals {
		if k > threshold {
			i, _ := strconv.Atoi(v)
			total += i
		} else {
			// Clear the old hash keys
			r.client.HDel(ctx, key, k)
		}
	}

	if total >= r.limit {
		return nil, ratelimit.ErrRateLimitExceeded{
			Remaining: 0, Limit: r.limit, Period: r.limitPeriod, Reset: now.Add(r.limitPeriod)}
	}

	return &ratelimit.LimitStatus{
		Remaining: r.limit - total,
		Limit:     r.limit,
		Period:    r.limitPeriod,
		Reset:     now.Add(r.limitPeriod),
	}, nil
}
