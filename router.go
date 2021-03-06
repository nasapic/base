package base

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/csrf"
	"golang.org/x/time/rate"
)

type (
	Router struct {
		name string
		chi.Router

		// NOTE: Move responsability from router
		// to the limiters themselves.

		// Hourly rate
		maxReqPerHour uint64
		hourlyRate    rate.Limit
		hourlyLimiter *baseLimiter

		// Daily
		maxReqPerDay int
		dailyRate    rate.Limit
		dailyLimiter *baseLimiter
	}
)

type (
	baseLimiter struct {
		sync.Mutex
		*rate.Limiter
		last time.Time
	}
)

const (
	hourInSecs = 3600
	dayInSecs  = hourInSecs * 24
	zeroInt    = 0
	maxInt     = math.MaxInt64
)

func NewRouter(name string) *Router {
	name = genName(name, "router")

	rt := Router{
		name:   name,
		Router: chi.NewRouter(),

		// Hourly
		maxReqPerHour: maxInt,
		hourlyRate:    0,
		hourlyLimiter: &baseLimiter{
			Mutex:   sync.Mutex{},
			Limiter: rate.NewLimiter(maxInt, 1), // i.e.: 120 = 30 reqs / hour
			last:    time.Now(),
		},

		// Daily
		maxReqPerDay: maxInt,
		dailyRate:    0,
		dailyLimiter: &baseLimiter{
			Mutex:   sync.Mutex{},
			Limiter: rate.NewLimiter(maxInt, 1), // i.e.: 1728 = 50 reqs / day
			last:    time.Now(),
		},
	}

	rt.Use(middleware.RequestID)
	rt.Use(middleware.RealIP)
	rt.Use(middleware.Recoverer)
	rt.Use(middleware.Timeout(60 * time.Second))
	rt.Use(rt.MethodOverride)
	rt.Use(rt.CSRFProtection)
	rt.Use(rt.ThrottleLimit)

	return &rt
}

func (r *Router) Name() string {
	return r.name
}

func (r *Router) SetHourlyRate(maxReqsPerHour int) {
	if maxReqsPerHour <= 0 {
		r.hourlyRate = 0
	}

	hr := float64(hourInSecs / maxReqsPerHour)

	r.hourlyRate = rate.Limit(hr)

	r.dailyLimiter = &baseLimiter{
		Mutex:   sync.Mutex{},
		Limiter: rate.NewLimiter(r.hourlyRate, 1), // i.e.: 1728 = 50 reqs / day
		last:    time.Now(),
	}
}

func (r *Router) SetDailyRate(maxReqsPerDay int) {
	if maxReqsPerDay <= 0 {
		r.dailyRate = 0
	}

	dr := float64(dayInSecs / maxReqsPerDay)

	r.dailyRate = rate.Limit(dr)

	r.dailyLimiter = &baseLimiter{
		Mutex:   sync.Mutex{},
		Limiter: rate.NewLimiter(r.dailyRate, 1), // i.e.: 1728 = 50 reqs / day
		last:    time.Now(),
	}
}

// Middlewares
// MethodOverride to emulate PUT and PATCH HTTP method.
func (rt *Router) MethodOverride(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			method := r.PostFormValue("_method")
			if method == "" {
				method = r.Header.Get("X-HTTP-Method-Override")
			}
			if method == "PUT" || method == "PATCH" || method == "DELETE" {
				r.Method = method
			}
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// CSRFProtection add cross-site request forgery protecction to the handler.
func (rt *Router) CSRFProtection(next http.Handler) http.Handler {
	return csrf.Protect([]byte("32-byte-long-auth-key"), csrf.Secure(false))(next)
}

// ThrottleLimit add rate limit protection
func (rt *Router) ThrottleLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		updateLimiter(rt.hourlyLimiter)
		updateLimiter(rt.dailyLimiter)

		if !(rt.hourlyLimiter.Allow() && rt.dailyLimiter.Allow()) {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func updateLimiter(l *baseLimiter) {
	l.Lock()
	defer l.Unlock()

	l.last = time.Now()
}
