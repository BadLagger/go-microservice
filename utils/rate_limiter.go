package utils

import (
	"net/http"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	Limiter *rate.Limiter
}

func NewRateLimiter(reqsInSec, burst int) *RateLimiter {
	return &RateLimiter{
		Limiter: rate.NewLimiter(rate.Limit(reqsInSec), burst),
	}
}

func (rl *RateLimiter) LimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		if !rl.Limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
