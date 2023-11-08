package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Southclaws/sliding-window-counters/ratelimit"
	"github.com/go-redis/redis/v8"
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

func (r *Redis) Increment(ctx context.Context, key string, incr int) error {
	now := time.Now()
	timestamp := fmt.Sprint(now.Truncate(r.counterWindow).Unix())

	val, err := r.client.HIncrBy(ctx, key, timestamp, int64(incr)).Result()
	if err != nil {
		return err
	}

	// check if current window has exceeded the limit
	if val >= int64(r.limit) {
		// Otherwise, check if just this fixed window counter period is over
		return ratelimit.ErrRateLimitExceeded(0, r.limit, r.limitPeriod, now.Add(r.limitPeriod))
	}

	// create or move whole limit period window expiry
	r.client.Expire(ctx, key, r.limitPeriod)

	// Get all the bucket values and sum them.
	vals, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return err
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

	if total >= int(r.limit) {
		return ratelimit.ErrRateLimitExceeded(0, r.limit, r.limitPeriod, now.Add(r.limitPeriod))
	}

	return nil
}
