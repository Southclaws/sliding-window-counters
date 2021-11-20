package ratelimit

import (
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	RateLimitLimit     = "X-RateLimit-Limit"
	RateLimitRemaining = "X-RateLimit-Remaining"
	RateLimitReset     = "X-RateLimit-Reset"
	RetryAfter         = "Retry-After"
)

//
// NOTE:
//
// Prior art: https://github.com/sethvargo/go-limiter/tree/main/httplimit
// The KeyFunc interface and IPKeyFunc implementation is from Seth Vargo's
// go-limiter project. It's a nice minimal approach so I decided to use it here
// too. It also allows the user to implement their own key function based on
// their own user accounts system.
//

// KeyFunc is a function that generates a unique key from a request.
type KeyFunc func(r *http.Request) (string, error)

// IPKeyFunc is the default implementation for KeyFunc. It can read headers like
// X-Forwarded-For and CF-Connecting-IP etc.
func IPKeyFunc(headers ...string) KeyFunc {
	return func(r *http.Request) (string, error) {
		for _, h := range headers {
			if v := r.Header.Get(h); v != "" {
				return v, nil
			}
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return "", err
		}
		return ip, nil
	}
}

func Middleware(l Limiter, kf KeyFunc, weight int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			key, err := kf(r)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			exceeded, ok := l.Increment(ctx, key, weight).(RateLimitExceeded)
			if !ok {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			limit := exceeded.Limit
			remaining := exceeded.Remaining
			resetTime := exceeded.Reset.UTC().Format(time.RFC1123)

			w.Header().Set(RateLimitLimit, strconv.FormatUint(uint64(limit), 10))
			w.Header().Set(RateLimitRemaining, strconv.FormatUint(uint64(remaining), 10))
			w.Header().Set(RateLimitReset, resetTime)

			if !ok {
				w.Header().Set(RetryAfter, resetTime)
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
