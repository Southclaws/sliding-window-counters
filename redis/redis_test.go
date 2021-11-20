package redis

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	ratelimiter, close := limiter()
	defer close()

	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user1", 1))
	assert.Error(t, ratelimiter.Increment(context.Background(), "user1", 1))
}

func TestRateLimitReset(t *testing.T) {
	ratelimiter, close := limiter()
	defer close()

	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
	assert.Error(t, ratelimiter.Increment(context.Background(), "user2", 1))

	time.Sleep(time.Second * 6)

	assert.NoError(t, ratelimiter.Increment(context.Background(), "user2", 1))
}

func TestTest(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ratelimiter := New(client, 10, time.Minute, time.Second)

	for {
		userid := fmt.Sprint("user", rand.Intn(10))
		err := ratelimiter.Increment(context.Background(), userid, 1)
		fmt.Println("call", userid, err)
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
	}
}

func limiter() (Redis, func()) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ratelimiter := New(client, 10, time.Second*5, time.Second)

	return *ratelimiter, func() { client.Close() }
}
