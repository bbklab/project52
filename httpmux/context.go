package mux

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	contentText   = "text/plain"
	contentJSON   = "application/json"
	contentBinary = "application/octet-stream"
)

// Context for request scope
type Context struct {
	Req *http.Request
	Res http.ResponseWriter
	Qs  Params // query params
	Ps  Params // path params

	Mux *Mux
}

// Params is a map of name/value pairs for path or query params.
type Params map[string]string

func newContext(r *http.Request, w http.ResponseWriter, ps Params, m *Mux) *Context {
	qs := make(Params) // query params
	r.ParseForm()
	for k, v := range r.Form {
		qs[k] = v[0]
	}

	return &Context{
		Req: r,
		Res: w,
		Qs:  qs,
		Ps:  ps,
		Mux: m,
	}
}

// JSON ...
func (ctx *Context) JSON(code int, data interface{}) {
	bs, err := json.Marshal(data)
	if err != nil {
		ctx.Error(500, err.Error())
		return
	}

	ctx.Res.Header().Set("Content-Type", contentJSON+"; charset=UTF-8")
	ctx.Res.WriteHeader(code)
	ctx.Res.Write(bs)
}

// Data ...
func (ctx *Context) Data(code int, data []byte) {
	ctx.Res.Header().Set("Content-Type", contentBinary)
	ctx.Res.WriteHeader(code)
	ctx.Res.Write(data)
}

// Text ...
func (ctx *Context) Text(code int, data string) {
	ctx.Res.Header().Set("Content-Type", contentText)
	ctx.Res.WriteHeader(code)
	ctx.Res.Write([]byte(data))
}

// Redirect ...
func (ctx *Context) Redirect(url string, code int) {
	if code == 0 {
		code = http.StatusFound
	}
	http.Redirect(ctx.Res, ctx.Req, url, code)
}

// Status ...
func (ctx *Context) Status(code int) {
	ctx.Res.WriteHeader(code)
}

// NotFound ...
func (ctx *Context) NotFound(data interface{}) {
	ctx.Error(http.StatusNotFound, data)
}

// Conflict ...
func (ctx *Context) Conflict(data interface{}) {
	ctx.Error(http.StatusConflict, data)
}

// BadRequest ...
func (ctx *Context) BadRequest(data interface{}) {
	ctx.Error(http.StatusBadRequest, data)
}

// Error ...
func (ctx *Context) Error(code int, data interface{}) {
	var msg string

	switch v := data.(type) {
	case error:
		msg = v.Error()
	case string:
		msg = v
	default:
		bs, err := json.Marshal(data)
		if err == nil {
			msg = string(bs)
			break
		}
		msg = fmt.Sprintf("%v", data)
	}

	ctx.Res.WriteHeader(code)
	json.NewEncoder(ctx.Res).Encode(map[string]string{
		"error": msg,
	})
}
