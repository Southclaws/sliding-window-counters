# Golang Sliding Window Counters Rate Limiter

> _(I tried to come up with a nicer name...)_

This is a simple rate limiter built based on [this blog post](https://www.figma.com/blog/an-alternative-approach-to-rate-limiting) from Figma's engineering team.

See the post for information about the requirements and design of the actual algorithm.

## Usage

The rate limiter satisfies this interface:

```go
type Limiter interface {
    Increment(context.Context, string, int) error
}
```

The implementation is backed by Redis and uses the go-redis library.

```go
client := goredis.NewClient(&goredis.Options{
    Addr: "localhost:6379",
})

ratelimiter := redis.New(client, 10, time.Minute, time.Hour)

ratelimiter.Increment(ctx, "user_id", 1)
```

## Middleware

There's a http middleware too, for convenience. Inspired by Seth Vargo's [rate limit](https://github.com/sethvargo/go-limiter) library:

```go
ratelimiter := redis.New(client, 10, time.Minute, time.Hour)
mw := ratelimit.Middleware(ratelimiter, ratelimit.IPKeyFunc, 1)
// use mw in your favourite HTTP library
```

The `Middleware` function has a `weight` parameter, which allows you to give a higher increment weight to certain routes. So for example, your base rate limit can be 1000 requests per hour and each request has a weight of 1, but a particularly computationally intensive endpoint may want to have a weight of 10, so each request increments the internal counter by 10 instead of 1.

If you use multiple middleware instances, make sure you don't add one to the global mux/router otherwise your requests will be triggering two rate limit calculations (and thus, Redis network I/O operations). This could be solved in future by a centralised middleware controller that provides a tree of limiters with order of precedence rules etc.
