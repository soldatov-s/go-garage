package log

import (
	"runtime"
	"strings"

	"github.com/rs/zerolog"
)

// This adds tracing ability to zerolog. By default it will add package
// name to every log line. If package version specified in package name
// (e.g. "domains/name/v1", note "v1" ending) - it will parse version and
// add it as separate field. Currently only one-digit-version is supported.
// Note that fully enabled tracing (context.Configuration.Gowork.Logger.WithTrace
// set to true) might eat your performance!
type TracingHook struct {
	WithTrace bool
}

func NewTracingHook(withTrace bool) *TracingHook {
	return &TracingHook{
		WithTrace: withTrace,
	}
}

func (h *TracingHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Append package name. Also, if version included - package name
	// should be cleared from version and version field should be added
	// separately.
	pc, _, _, ok := runtime.Caller(3)
	if !ok {
		e.Msg("Caller information could not be retrieved! No auto-pkgname-version appending will be done and no tracing is possible!")
	} else {
		frame := runtime.FuncForPC(pc)
		callerName := frame.Name()

		// "httpserver-provider-echo") with caller's signature.
		callerPackageAndSignatureRaw := strings.Split(callerName, "/")
		callerPackageAndSignature := callerPackageAndSignatureRaw[len(callerPackageAndSignatureRaw)-1]

		if h.WithTrace || (zerolog.GlobalLevel() == zerolog.TraceLevel && level == zerolog.TraceLevel) {
			// Get caller's function signature.
			functionName := strings.Join(strings.Split(callerPackageAndSignature, ".")[1:], ".")
			fileName, lineNo := frame.FileLine(pc)
			e = e.Str("function", functionName)
			e = e.Str("file", fileName)
			// nolint:staticcheck,wastedassign // skip SA4006
			e = e.Int("line", lineNo)
		}
	}
}
