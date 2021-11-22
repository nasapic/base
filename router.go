package base

import (
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
	}
)

type (
	baseLimiter struct {
		sync.Mutex
		*rate.Limiter
		last time.Time
	}
)

var (
	// FIX: Make configurable
	hourRate = rate.Every(1 * time.Hour / 30)
	dayRate  = rate.Every(24 * time.Hour / 50)

	hourLimiter = baseLimiter{
		Mutex:   sync.Mutex{},
		Limiter: rate.NewLimiter(hourRate, 1),
		last:    time.Now(),
	}

	dayLimiter = baseLimiter{
		Mutex:   sync.Mutex{},
		Limiter: rate.NewLimiter(dayRate, 1),
		last:    time.Now(),
	}
)

func NewRouter(name string) *Router {
	name = genName(name, "router")

	rt := Router{
		name:   name,
		Router: chi.NewRouter(),
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
		updateLimiter(&hourLimiter)
		updateLimiter(&dayLimiter)

		if !hourLimiter.Allow() || !dayLimiter.Allow() {
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
