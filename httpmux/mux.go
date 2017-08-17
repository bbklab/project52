package mux

import (
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
)

// HTTPHandler ...
type HTTPHandler func(*Context)

// Mux is a minimal http router implement
type Mux struct {
	sync.RWMutex               // protect Routes
	Routes       []*Route      // all http routes
	Midwares     []HTTPHandler // global midwares
	NotFound     HTTPHandler   // not found handler
}

// New create an instance of Mux
func New() *Mux {
	return &Mux{
		Routes:   make([]*Route, 0),
		Midwares: make([]HTTPHandler, 0),
		NotFound: func(ctx *Context) { // default not found handler
			ctx.Error(404, "no such route")
		},
	}
}

// ServeHTTP implement http.Handler
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// panic protection
	defer func() {
		if r := recover(); r != nil {
			var msg string
			switch v := r.(type) {
			case error:
				msg = v.Error()
			default:
				msg = fmt.Sprintf("%v", v)
			}

			stack := make([]byte, 4096)
			runtime.Stack(stack, true)
			msg = fmt.Sprintf("PANIC RECOVER: %s\n\n%s\n", msg, string(stack))

			log.Errorln(msg)
			http.Error(w, msg, 500)
		}
	}()

	// global midwares: log, auth ...

	// log midware
	method, remote, path := r.Method, r.RemoteAddr, r.URL.Path
	log.Println(method, remote, path)

	// route the request to the right way
	route, params := m.bestMatch(method, path)
	if route != nil {
		ctx := newContext(r, w, params, m)
		for _, h := range route.handlers {
			h(ctx) // protect panic
		}
		return
	}

	// not found
	ctx := newContext(r, w, nil, m)
	m.NotFound(ctx)
}

// AllRoutes ...
func (m *Mux) AllRoutes() []*Route {
	m.RLock()
	defer m.RUnlock()
	return m.Routes[:]
}

// AddRoute ...
func (m *Mux) AddRoute(method, pattern string, handlers []HTTPHandler) {
	route := newRoute(method, pattern, handlers)
	m.Lock()
	m.Routes = append(m.Routes, route)
	m.Unlock()
}

// SetNotFound ...
func (m *Mux) SetNotFound(h HTTPHandler) {
	m.NotFound = h
}

// SetMidware ...
func (m *Mux) SetMidware(hs ...HTTPHandler) {
	m.Lock()
	m.Midwares = append(m.Midwares, hs...)
	m.Unlock()
}

// Get ...
func (m *Mux) Get(pattern string, hs ...HTTPHandler) {
	m.AddRoute("GET", pattern, hs)
}

// Post ...
func (m *Mux) Post(pattern string, hs ...HTTPHandler) {
	m.AddRoute("POST", pattern, hs)
}

// Patch ...
func (m *Mux) Patch(pattern string, hs ...HTTPHandler) {
	m.AddRoute("PATCH", pattern, hs)
}

// Put ...
func (m *Mux) Put(pattern string, hs ...HTTPHandler) {
	m.AddRoute("PUT", pattern, hs)
}

// Delete ...
func (m *Mux) Delete(pattern string, hs ...HTTPHandler) {
	m.AddRoute("DELETE", pattern, hs)
}

// Head ...
func (m *Mux) Head(pattern string, hs ...HTTPHandler) {
	m.AddRoute("HEAD", pattern, hs)
}

// Options ...
func (m *Mux) Options(pattern string, hs ...HTTPHandler) {
	m.AddRoute("OPTIONS", pattern, hs)
}

// Any ...
func (m *Mux) Any(pattern string, hs ...HTTPHandler) {
	m.AddRoute("*", pattern, hs)
}

// bestMatch try to find the best matched route and it's path params kv
/*
eg:
  request: GET /user/all
  will match routes like:
	/user/all   - this is what we expect
	/user/:id
*/
func (m *Mux) bestMatch(method, path string) (*Route, Params) {
	matched := make([]*Route, 0, 0)

	// itera all routes to find all of matched routes
	for _, route := range m.AllRoutes() {
		if _, ok := route.match(method, path); !ok {
			continue
		}
		matched = append(matched, route)
	}

	if len(matched) == 0 {
		return nil, nil
	}

	// make sort to find the best one and it's path params
	sort.Sort(routeSourter(matched))
	best := matched[0]
	params, _ := best.match(method, path)
	return best, params
}

// Route represents single user defined http route
type Route struct {
	Method  string
	Pattern string

	handlers  []HTTPHandler
	reg       *regexp.Regexp // generated from Pattern, for matching to capture path params
	paramKeys []string       // captured path param key names
	numField  int            // nb of splited fields
}

func newRoute(method, pattern string, handlers []HTTPHandler) *Route {
	pattern = strings.TrimSuffix(pattern, "/")
	r := &Route{
		Method:   method,
		Pattern:  pattern,
		handlers: handlers,
	}
	r.init(pattern)
	return r
}

func (r *Route) init(pattern string) {
	var (
		reg    = regexp.MustCompile(`/:([a-zA-Z0-9_]+)`)
		keys   = make([]string, 0, 0)
		fields = strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	)

	newRegStr := reg.ReplaceAllStringFunc(pattern, func(s string) string {
		keys = append(keys, s[2:]) // trim heading 2 chars /:
		return fmt.Sprintf("/(?P<%s>[a-zA-Z0-9_]+)", s[2:])
	})

	r.reg = regexp.MustCompile(newRegStr)
	r.paramKeys = keys
	r.numField = len(fields)
}

// match check http request against it's method & url path
func (r *Route) match(method, path string) (params Params, matched bool) {
	// match the method
	switch r.Method {
	case "*":
	case method:
	default:
		return
	}

	// match the url path
	path = strings.TrimSuffix(path, "/")
	var (
		matchedStrs  = make([]string, 0)
		matchedNames = make([]string, 0)
	)

	// match fields nb
	fields := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(fields) != r.numField {
		return nil, false
	}

	// exact match
	if !strings.Contains(r.Pattern, "/:") {
		matched = path == r.Pattern
		return
	}

	// regexp match
	if r.reg == nil {
		return
	}

	matchedStrs = r.reg.FindStringSubmatch(path)
	if len(matchedStrs) == 0 {
		return
	}
	matched = true

	matchedNames = r.reg.SubexpNames()
	if len(matchedStrs) != len(matchedNames) {
		return
	}

	// obtain the matched path params
	params = make(map[string]string)
	for idx, name := range matchedNames {
		if name != "" {
			params[name] = matchedStrs[idx]
		}
	}

	return
}

type routeSourter []*Route

func (s routeSourter) Len() int           { return len(s) }
func (s routeSourter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s routeSourter) Less(i, j int) bool { return len(s[i].paramKeys) < len(s[j].paramKeys) }
