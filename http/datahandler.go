//go:build ignore

// WIP experimentation

package http

import "net/http"

type HTTPDataRequest[T Contextual] struct {
	*http.Request
	Data T
}

type Contextual interface {
	ContextKey() interface{}
}

type HTTPDataHandler[T Contextual] interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, data T)
}

type HTTPDataHandlerFunc[T Contextual] func(w http.ResponseWriter, r *http.Request, data T)

func (f HTTPDataHandlerFunc[T]) ServeHTTP(w http.ResponseWriter, r *http.Request, data T) {
	f(w, r, data)
}

func ToHandler[T Contextual](h HTTPDataHandler[T]) http.Handler {
	key := (*new(T)).ContextKey()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := r.Context.Value(key).(T)
		if !ok {
			http.Error(http.StatusBadRequest)
			return
		}
		h.ServeHTTP(w, r, data)
	})
}

func FromRequest(r *http.Request) (T, error)

type Middleware[T Contextual] func(w http.ResponseWriter, r *http.Request) (T, bool)

func WithMiddleware[T Contextual](middle Middleware[T]) *Router[T]

// What's a good way of combining a bunch of middleware?

type allContext struct {
	auth    myAuth
	session session
	f       funky
}

func combine(m1 Middleware[myAuth], m2 Middleware[session], m3 Middleware[funky]) Middleware[allContext] {
	return func(w http.ResponseWriter, r *http.Request) (allContext, bool) {
		d1, ok1 := m1(w, r)
		d2, ok2 := m2(w, r)
		d3, ok3 := m3(w, r)
		if ok1 && ok2 && ok3 {
			return allContext{
				auth: d1,
				session: d2,
				f: d3
			}, true
		}
		return false
	}
}

type Router[T Contextual] struct {
}

func (r *Router[T]) Get(f HTTPDataHandler[T])

func (r *Router[T]) Post(f HTTPDataHandler[T])

func (r *Router[T]) ServeHTTP(w http.ResponseWriter, r *http.Request)
