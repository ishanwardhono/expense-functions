package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
)

// Router is a minimal method+path router supporting `{name}` path params, e.g.
// "/expenses/{id}". It is dependency-free (stdlib only).
type Router struct {
	routes   []route
	notFound HandlerFunc
}

type route struct {
	method  string
	segs    []segment
	handler HandlerFunc
}

type segment struct {
	literal string
	param   string // non-empty when this segment is a {param}
}

// NewRouter returns a Router whose default not-found handler returns a 404
// typed error.
func NewRouter() *Router {
	return &Router{
		notFound: func(w http.ResponseWriter, r *http.Request) error {
			return apierr.NotFound("no route for %s %s", r.Method, r.URL.Path)
		},
	}
}

// Handle registers a handler for a method + path pattern.
func (rt *Router) Handle(method, pattern string, h HandlerFunc) {
	rt.routes = append(rt.routes, route{
		method:  method,
		segs:    parsePattern(pattern),
		handler: h,
	})
}

// ServeHTTP makes the Router a HandlerFunc-compatible entry point so it can be
// wrapped by Middleware.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	reqSegs := splitPath(r.URL.Path)
	for _, rte := range rt.routes {
		if rte.method != r.Method {
			continue
		}
		params, ok := match(rte.segs, reqSegs)
		if !ok {
			continue
		}
		if len(params) > 0 {
			r = r.WithContext(context.WithValue(r.Context(), paramsKey{}, params))
		}
		return rte.handler(w, r)
	}
	return rt.notFound(w, r)
}

type paramsKey struct{}

// Param returns the named path parameter, or "" if absent.
func Param(r *http.Request, name string) string {
	if m, ok := r.Context().Value(paramsKey{}).(map[string]string); ok {
		return m[name]
	}
	return ""
}

func parsePattern(pattern string) []segment {
	parts := splitPath(pattern)
	segs := make([]segment, len(parts))
	for i, p := range parts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			segs[i] = segment{param: p[1 : len(p)-1]}
		} else {
			segs[i] = segment{literal: p}
		}
	}
	return segs
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func match(segs []segment, parts []string) (map[string]string, bool) {
	if len(segs) != len(parts) {
		return nil, false
	}
	var params map[string]string
	for i, s := range segs {
		if s.param != "" {
			if params == nil {
				params = make(map[string]string)
			}
			params[s.param] = parts[i]
			continue
		}
		if s.literal != parts[i] {
			return nil, false
		}
	}
	return params, true
}
