package httpmux

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httputil"
	stdpath "path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/bbklab/tracker/pkg/utils"
)

// HandleFunc is exported ...
type HandleFunc func(*Context)

// Mux is a minimal http router implement
type Mux struct {
	sync.RWMutex              // protect Routes & FileRoutes
	prefix       string       // prefix for all routes
	debug        bool         // debug to log each request (and response?)
	Routes       []*Route     // all http routes
	FileRoutes   []*FileRoute // all httpdir file routes
	PreMidwares  []HandleFunc // pre global midwares
	PostMidwares []HandleFunc // post global midwares triggered afterwards
	NotFound     HandleFunc   // not found handler
	CatchPanic   HandleFunc   // panic handler
	AuditLog     HandleFunc   // audit handler
}

// New create an instance of Mux
func New(prefix string) *Mux {
	return &Mux{
		prefix:       prefix,
		debug:        false,
		Routes:       make([]*Route, 0),
		PreMidwares:  make([]HandleFunc, 0),
		PostMidwares: make([]HandleFunc, 0),
		NotFound:     defaultNotFound,
		CatchPanic:   defaultCatchPanic,
		AuditLog:     defaultLog,
	}
}

// ServeHTTP implement http.Handler
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		method = r.Method
		path   = r.URL.Path
	)

	// reset w as our http.ResponseWriter implemention thus
	// we could caught the response code & response body size
	// unless the http handler Hijack-ed the in-comming connection
	w = NewResponse(w)

	// new http request scope context
	// note: reuse this contex for every handlers & midwares
	var (
		ctx = newContext(r, w, m)
	)

	if m.debug {
		reqbs, _ := httputil.DumpRequest(r, true)
		log.Println(string(reqbs))
	}

	// global midwares: panic recovery
	defer m.CatchPanic(ctx)

	// global midware: log
	defer m.AuditLog(ctx)

	// pre midware handlers
	if mws := m.PreMidwares; len(mws) > 0 {
		for _, mw := range mws {
			mw(ctx)
			if ctx.isAbort() { // if ctx marked as abort, quit immediately
				return
			}
		}
	}

	// route the request to the right way

	// hit httpdir file route, seving static files
	// TODO replaced this by http.FileServer
	bs, ctype, matched, err := m.bestMatchFile(path)
	if matched {
		if err != nil {
			ctx.AutoError(err)
			return
		}

		if ctype != "" {
			ctx.Res.Header().Set("Content-Type", ctype)
		}
		ctx.Res.WriteHeader(200)
		ctx.Res.Write(bs)
		return
	}

	// hit general http route, serving by matched handlers
	route, params := m.bestMatch(method, path)

	// not found
	if route == nil {
		m.NotFound(ctx)
		return
	}

	// found
	ctx.withPathParams(params) // set contex path parameters
	for _, h := range route.handlers {
		h(ctx)             // protect panic
		if ctx.isAbort() { //  if ctx marked as abort, quit immediately
			return
		}
	}

	// post midware handlers
	if mws := m.PostMidwares; len(mws) > 0 {
		for _, mw := range mws {
			mw(ctx)
			if ctx.isAbort() { // if ctx marked as abort, quit immediately
				return
			}
		}
	}
}

// AllRoutes ...
func (m *Mux) AllRoutes() []*Route {
	m.RLock()
	defer m.RUnlock()
	return m.Routes[:]
}

// AllFileRoutes is exported
func (m *Mux) AllFileRoutes() []*FileRoute {
	m.RLock()
	defer m.RUnlock()
	return m.FileRoutes[:]
}

// AddRoute ...
func (m *Mux) AddRoute(method, pattern string, handlers []HandleFunc) {
	route := newRoute(method, m.prefix+pattern, handlers)
	m.Lock()
	m.Routes = append(m.Routes, route)
	m.Unlock()
}

// AddFileRoute ...
func (m *Mux) AddFileRoute(uri, httpdir string) {
	fullpath := strings.TrimSuffix(m.prefix+uri, "/")
	m.Lock()
	m.FileRoutes = append(m.FileRoutes, &FileRoute{fullpath, httpdir})
	m.Unlock()
}

// SetNotFound ...
func (m *Mux) SetNotFound(h HandleFunc) {
	m.NotFound = h
}

// SetCatchPanic ...
func (m *Mux) SetCatchPanic(h HandleFunc) {
	m.CatchPanic = h
}

// SetAuditLog ...
func (m *Mux) SetAuditLog(h HandleFunc) {
	m.AuditLog = h
}

// SetDebug is exported
func (m *Mux) SetDebug(flag bool) {
	m.debug = flag
}

// SetGlobalPreMidware ...
func (m *Mux) SetGlobalPreMidware(h HandleFunc) {
	m.Lock()
	defer m.Unlock()
	for _, mw := range m.PreMidwares {
		if utils.FuncName(mw) == utils.FuncName(h) {
			return
		}
	}
	m.PreMidwares = append(m.PreMidwares, h)
}

// DelGlobalPreMidware ...
func (m *Mux) DelGlobalPreMidware(h HandleFunc) {
	m.Lock()
	defer m.Unlock()
	for idx, mw := range m.PreMidwares {
		if utils.FuncName(mw) == utils.FuncName(h) {
			m.PreMidwares = append(m.PreMidwares[:idx], m.PreMidwares[idx+1:]...)
			return
		}
	}
}

// SetGlobalPostMidware ...
func (m *Mux) SetGlobalPostMidware(h HandleFunc) {
	m.Lock()
	defer m.Unlock()
	for _, mw := range m.PostMidwares {
		if utils.FuncName(mw) == utils.FuncName(h) {
			return
		}
	}
	m.PostMidwares = append(m.PostMidwares, h)
}

// DelGlobalPostMidware ...
func (m *Mux) DelGlobalPostMidware(h HandleFunc) {
	m.Lock()
	defer m.Unlock()
	for idx, mw := range m.PostMidwares {
		if utils.FuncName(mw) == utils.FuncName(h) {
			m.PostMidwares = append(m.PostMidwares[:idx], m.PostMidwares[idx+1:]...)
			return
		}
	}
}

// GET is exported ...
func (m *Mux) GET(pattern string, hs ...HandleFunc) {
	m.AddRoute("GET", pattern, hs)
}

// POST is exported ...
func (m *Mux) POST(pattern string, hs ...HandleFunc) {
	m.AddRoute("POST", pattern, hs)
}

// PATCH is exported ...
func (m *Mux) PATCH(pattern string, hs ...HandleFunc) {
	m.AddRoute("PATCH", pattern, hs)
}

// PUT is exported ...
func (m *Mux) PUT(pattern string, hs ...HandleFunc) {
	m.AddRoute("PUT", pattern, hs)
}

// DELETE is exported ...
func (m *Mux) DELETE(pattern string, hs ...HandleFunc) {
	m.AddRoute("DELETE", pattern, hs)
}

// HEAD is exported ...
func (m *Mux) HEAD(pattern string, hs ...HandleFunc) {
	m.AddRoute("HEAD", pattern, hs)
}

// OPTIONS is exported ...
func (m *Mux) OPTIONS(pattern string, hs ...HandleFunc) {
	m.AddRoute("OPTIONS", pattern, hs)
}

// ANY is exported ...
func (m *Mux) ANY(pattern string, hs ...HandleFunc) {
	m.AddRoute("*", pattern, hs)
}

// FILE is exported ...
func (m *Mux) FILE(uri, httpdir string) {
	m.AddFileRoute(uri, httpdir)
}

// bestMatch try to find the best matched route and it's path params kv
/*
eg1:
  request: GET /user/all
  will match routes like:
	/user/all   - this is what we expect
	/user/:id

eg2:
  request: GET /user/bbk/repoxxx
  will match routes like:
    /user/:name/repoxxx   - this is what we expect
    /user/:name/repo
*/
func (m *Mux) bestMatch(method, path string) (*Route, Params) {
	path = strings.TrimSuffix(path, "/")

	// itera all routes to find all of matched routes
	matched := make([]*Route, 0, 0)
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

func (m *Mux) bestMatchFile(path string) ([]byte, string, bool, error) {
	path = strings.TrimSuffix(path, "/")

	// itera all file routes to find the first matched httpdir route
	for _, route := range m.AllFileRoutes() {
		if strings.HasPrefix(path, route.FullPath) {
			relative := strings.TrimPrefix(path, route.FullPath)
			localpath := stdpath.Join(route.HTTPDir, relative)
			ctype := mime.TypeByExtension(filepath.Ext(localpath)) // See More: net/http.serveContent()
			bs, err := ioutil.ReadFile(localpath)
			return bs, ctype, true, err
		}
	}

	return nil, "", false, nil
}

// FileRoute represents single user defined httpdir files route to serving static files
type FileRoute struct {
	FullPath string // request full path (with prefix)
	HTTPDir  string // local http dir path
}

// Route represents single user defined http route
type Route struct {
	Method  string
	Pattern string

	handlers  []HandleFunc
	reg       *regexp.Regexp // generated from Pattern, for matching to capture path params
	paramKeys []string       // captured path param key names
	numField  int            // nb of splited fields
	lastField string         // last field
	wildchars bool           // contains wildchars or not
}

func newRoute(method, pattern string, handlers []HandleFunc) *Route {
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
		reg    = regexp.MustCompile(`/:([a-zA-Z0-9._@-]+)`)
		keys   = make([]string, 0, 0)
		fields = strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	)

	newRegStr := reg.ReplaceAllStringFunc(pattern, func(s string) string {
		keys = append(keys, s[2:]) // trim heading 2 chars /:
		return fmt.Sprintf("/(?P<%s>[a-zA-Z0-9._@-]+)", s[2:])
	})

	if strings.Contains(newRegStr, "***") {
		r.wildchars = true
		newRegStr = strings.Replace(newRegStr, "***", ".*", -1)
	}

	r.reg = regexp.MustCompile(newRegStr)
	r.paramKeys = keys
	r.numField = len(fields)
	if r.numField > 0 {
		r.lastField = fields[r.numField-1]
	}
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
	var (
		matchedStrs  = make([]string, 0)
		matchedNames = make([]string, 0)
	)

	// match fields nb
	fields := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if !r.wildchars { // if not contains wildchars, the nb of fields must be equal, otherwise not hit
		if len(fields) != r.numField {
			return nil, false
		}
	} else { // if contains wildchars, the nb of fields must >= pattern, otherwise not hit
		if len(fields) < r.numField {
			return nil, false
		}
	}

	// try exact match the whole path
	if !strings.Contains(r.Pattern, "/:") {
		matched = path == r.Pattern
		return
	}

	// exact check on the last field if last field not path parameter and not wildchars
	if len(fields) > 0 {
		lastField := fields[len(fields)-1]
		if !strings.HasPrefix(r.lastField, ":") && !strings.Contains(r.lastField, "***") {
			// lastField not path parameter and not wildchars
			if lastField != r.lastField {
				return
			}
		}
	}

	// try regexp match the whole path
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
