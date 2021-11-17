package base

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/csrf"
)

type (
	Router struct {
		name string
		chi.Router
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
