package sentry

import (
	"context"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stephenafamo/janus/monitor"
)

type Sentry struct {
	Hub *sentry.Hub
}

func (s Sentry) Recover(ctx context.Context, cause interface{}) {
	s.Hub.RecoverWithContext(ctx, cause)
}

func (s Sentry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), sentry.HubContextKey, s.Hub))
		r = r.WithContext(context.WithValue(r.Context(), monitor.CtxRequest, r))

		s.Hub.WithScope(func(scope *sentry.Scope) {
			scope.SetTransaction(r.URL.Path)

			span := sentry.StartSpan(
				r.Context(),
				"request",
				sentry.TransactionName(r.URL.Path),
				sentry.ContinueFromRequest(r),
			)
			defer span.Finish()

			r = r.WithContext(span.Context())

			scope.SetRequest(r)

			defer func() {
				if cause := recover(); cause != nil {
					s.Hub.RecoverWithContext(r.Context(), cause)
				}
			}()

			r = r.WithContext(context.WithValue(r.Context(),
				sentry.RequestContextKey, r))

			r = r.WithContext(context.WithValue(r.Context(),
				monitor.CtxScope, SentryScope{scope: scope}))

			next.ServeHTTP(w, r)
		})
	})

}

func (s Sentry) StartSpan(ctx context.Context, name string) (context.Context, monitor.Span) {
	span := sentry.StartSpan(ctx, name)
	return span.Context(), span
}

func (s Sentry) CaptureMessage(msg string, tags map[string]string) {
	s.Hub.WithScope(func(scope *sentry.Scope) {
		scope.SetTags(tags)
		s.Hub.CaptureMessage(msg)
	})
}

func (s Sentry) CaptureException(err error, tags map[string]string) {
	s.Hub.WithScope(func(scope *sentry.Scope) {
		scope.SetTags(tags)
		s.Hub.CaptureException(err)
	})
}

func (s Sentry) Flush(timeout time.Duration) {
	s.Hub.Flush(timeout)
}

type SentryScope struct {
	scope *sentry.Scope
}

func (s SentryScope) SetTransactionName(name string) {
	s.scope.SetTransaction(name)
}

func (s SentryScope) SetTag(key, value string) {
	s.scope.SetTag(key, value)
}

func (s SentryScope) SetUser(id, username, email string) {
	s.scope.SetUser(sentry.User{
		ID:       id,
		Username: username,
		Email:    email,
	})
}

type LoggingIntegration struct {
	SupressErrors bool
	Logger        interface {
		Printf(format string, a ...interface{}) (n int, err error)
	}
}

func (sli LoggingIntegration) Name() string {
	return "Logging"
}

func (sli LoggingIntegration) SetupOnce(client *sentry.Client) {
	client.AddEventProcessor(sli.processor)
}

func (sli LoggingIntegration) processor(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	sli.Logger.Printf("\n%s", event.Message)

	// print only the last exception
	if len(event.Exception) > 0 {
		sli.print(event)
	}

	if sli.SupressErrors {
		return nil
	}

	return event
}

func (sli LoggingIntegration) print(event *sentry.Event) {
	exception := event.Exception[len(event.Exception)-1]
	// Print the error message
	sli.Logger.Printf("\n%s", exception.Value)

	// Print the user details
	if event.User != (sentry.User{}) {
		sli.Logger.Printf("\nUser: Email %q, ID %q, IPAddress %q, Username %q",
			event.User.Email, event.User.ID, event.User.IPAddress, event.User.Username)
	}

	// Print the tags
	if len(event.Tags) > 0 {
		sli.Logger.Printf("\nTags:")
		for key, val := range event.Tags {
			sli.Logger.Printf("%s=%s\n", key, val)
		}
	}

	// Print some extra lines for readability
	sli.Logger.Printf("\n\n")

	if exception.Stacktrace != nil {
		for i := len(exception.Stacktrace.Frames) - 1; i >= 0; i-- {
			frame := exception.Stacktrace.Frames[i]
			// Print general info about the exception
			sli.Logger.Printf("%s:%d:%d %s\n",
				frame.AbsPath, frame.Lineno, frame.Colno, frame.Function)

			// Only print the first five frames
			if frame.ContextLine != "" && len(exception.Stacktrace.Frames)-i <= 5 {

				// Print the lines before the exception line
				for j := 0; j < len(frame.PreContext); j++ {
					line := frame.PreContext[j]
					sli.Logger.Printf("%04d | "+line+"\n", frame.Lineno-len(frame.PreContext)+j)
				}

				// Print the exception line
				sli.Logger.Printf("%04d > "+frame.ContextLine+"\n", frame.Lineno)

				// Print the lines after the exception
				for j := 0; j < len(frame.PostContext); j++ {
					line := frame.PostContext[j]
					sli.Logger.Printf("%04d | "+line+"\n", frame.Lineno+j)
				}
			}
			sli.Logger.Printf("\n")
		}
	}
}
