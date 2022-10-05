package monitor

import (
	"net/http"
	"time"
)

type ctxKey string

// CtxUserID is the context key for the scope
var CtxScope ctxKey = "scope"

type Monitor interface {
	// Any implementation must set the scope to the request context in the middleware
	Middleware(http.Handler) http.Handler
	CaptureMessage(string)
	CaptureException(error)
	Flush(timeout time.Duration)
}

type Scope interface {
	SetTag(key, value string)
	SetUser(id, username, email string)
}
