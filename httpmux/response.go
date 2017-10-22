package httpmux

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// Response re-implement the http.ResponseWriter by rewrap an exists http.ResponseWriter
// thus to obtain the code / body size replied to the client
// See: https://golang.org/pkg/net/http/#ResponseWriter
type Response struct {
	w           http.ResponseWriter
	wroteHeader bool
	statusCode  int   // code responsed to client
	size        int64 // body size responsed to client

	errmsg string // optional: detected body error message
}

// NewResponse is exported
func NewResponse(w http.ResponseWriter) http.ResponseWriter {
	return &Response{w: w, statusCode: 0, size: 0}
}

// StatusCode is exported
func (r *Response) StatusCode() int { return r.statusCode }

// Size is exported
func (r *Response) Size() int64 { return r.size }

// ErrMsg is exported
func (r *Response) ErrMsg() string { return r.errmsg }

//
// re-implement http.ResponseWriter
//

// Header re-implement http.ResponseWriter
func (r *Response) Header() http.Header {
	return r.w.Header()
}

// WriteHeader re-implement http.ResponseWriter
func (r *Response) WriteHeader(code int) {
	if r.wroteHeader {
		log.Warnln("[HTTPMUX]", "multi WriteHeader called")
		return
	}
	r.w.WriteHeader(code)
	r.statusCode = code
	r.wroteHeader = true
}

// Write re-implement http.ResponseWriter
func (r *Response) Write(bs []byte) (n int, err error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err = r.w.Write(bs)
	r.size += int64(n)

	// detect possibility body error message
	var internalerr HTTPError
	if err := json.Unmarshal(bs, &internalerr); err == nil {
		if errmsg, ok := internalerr.Error.(string); ok && errmsg != "" {
			r.errmsg = errmsg
		}
	}
	return
}

//
// the other interfaces that default http response writer implemented
//

// Hijack is exported
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.w.(http.Hijacker).Hijack()
}

// CloseNotify is exported
func (r *Response) CloseNotify() <-chan bool {
	return r.w.(http.CloseNotifier).CloseNotify()
}

// Flush is exported
func (r *Response) Flush() {
	r.w.(http.Flusher).Flush()
}
