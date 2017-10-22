package httpmux

import (
	"fmt"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
)

func defaultNotFound(ctx *Context) {
	ctx.NotFound("no such route")
	return
}

func defaultCatchPanic(ctx *Context) {
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

		log.Errorln("[HTTPMUX]", msg)
		ctx.InternalServerError(msg)
	}
}

func defaultLog(ctx *Context) {
	var (
		method = ctx.Req.Method
		remote = ctx.Req.RemoteAddr
		path   = ctx.Req.URL.Path
		cost   = fmt.Sprintf("%0.4fs", time.Now().Sub(ctx.StartAt()).Seconds())
		code   = ctx.Res.(*Response).StatusCode() // note: 0 if Hijack-ed
		size   = ctx.Res.(*Response).Size()       // note: 0 if Hijack-ed
	)
	log.Println("[HTTPMUX]", code, method, remote, path, size, cost)
}
