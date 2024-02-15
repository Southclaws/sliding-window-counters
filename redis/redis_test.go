package redis

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/sliding-window-counters/ratelimit"
)

func limiter() (Redis, func()) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ratelimiter := New(client, 10, time.Second*5, time.Second)

	return *ratelimiter, func() { client.Close() }
}

func TestRateLimit(t *testing.T) {
	rateLimiter, close := limiter()
	defer close()
	key := "user1"

	for i := 0; i < rateLimiter.limit-1; i++ {
		limitStatus, err := rateLimiter.Increment(context.Background(), key, 1)
		require.NoError(t, err)
		assert.Equal(t, limitStatus.Remaining, rateLimiter.limit-(i+1))
	}

	_, err := rateLimiter.Increment(context.Background(), key, 1)
	require.Error(t, err)
	var errExceeded ratelimit.ErrRateLimitExceeded
	require.ErrorAs(t, err, &errExceeded)
	assert.Equal(t, errExceeded.Remaining, 0)
}

func TestRateLimitReset(t *testing.T) {
	rateLimiter, close := limiter()
	defer close()
	key := "user2"

	for i := 0; i < rateLimiter.limit-1; i++ {
		limitStatus, err := rateLimiter.Increment(context.Background(), key, 1)
		require.NoError(t, err)
		assert.Equal(t, limitStatus.Remaining, rateLimiter.limit-(i+1))
	}

	//Wait 6 seconds
	time.Sleep(time.Second + rateLimiter.limitPeriod)

	limitStatus, err := rateLimiter.Increment(context.Background(), key, 1)
	require.NoError(t, err)
	assert.Equal(t, limitStatus.Remaining, rateLimiter.limit-1)
}

func TestTest(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ratelimiter := New(client, 10, time.Minute, time.Second)

	for {
		userid := fmt.Sprint("user", rand.Intn(10))
		_, err := ratelimiter.Increment(context.Background(), userid, 1)
		fmt.Println("call", userid, err)
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
	}
}
