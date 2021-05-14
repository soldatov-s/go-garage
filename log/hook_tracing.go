package log

import (
	"fmt"
	"runtime"
	"strconv"
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

func (h TracingHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Append package name. Also, if version included - package name
	// should be cleared from version and version field should be added
	// separately.
	pc, _, _, ok := runtime.Caller(3)
	if !ok {
		fmt.Println(
			"Caller information could not be retrieved!",
			"No auto-pkgname-version appending will be done and no tracing is possible!",
		)
	} else {
		frame := runtime.FuncForPC(pc)
		callerName := frame.Name()

		// Get caller's package name.
		callerNameSplitted := strings.Split(callerName, ".")
		var packageName string
		if len(callerNameSplitted) > 2 {
			if strings.HasPrefix(callerNameSplitted[len(callerNameSplitted)-2], "(") {
				packageName = strings.Join(callerNameSplitted[:len(callerNameSplitted)-2], ".")
			} else {
				packageName = strings.Join(callerNameSplitted[:len(callerNameSplitted)-1], ".")
			}
		} else {
			// The "main.main" situation.
			packageName = callerNameSplitted[0]
		}

		// We should get last package name (e.g. "v1" o
		// "httpserver-provider-echo") with caller's signature.
		callerPackageAndSignatureRaw := strings.Split(callerName, "/")
		callerPackageAndSignature := callerPackageAndSignatureRaw[len(callerPackageAndSignatureRaw)-1]

		// Get caller's domain version, if specified.
		if strings.HasPrefix(callerPackageAndSignature, "v") {
			versionRaw := string(strings.Split(callerPackageAndSignature, ".")[0][1])
			version, err := strconv.ParseInt(versionRaw, 10, 64)
			if err == nil {
				e = e.Int64("version", version)
			}

			// If we have versioned domain - we should omit last "v1"
			// part for ease of reading.
			packageNameRaw := strings.Split(packageName, "/")
			packageName = strings.Join(packageNameRaw[:len(packageNameRaw)-1], "/")
		}

		e = e.Str("package", packageName)

		if h.WithTrace || (zerolog.GlobalLevel() == zerolog.TraceLevel && level == zerolog.TraceLevel) {
			// Get caller's function signature.
			functionName := strings.Join(strings.Split(callerPackageAndSignature, ".")[1:], ".")

			fileName, lineNo := frame.FileLine(pc)
			e = e.Str("function", functionName)
			e = e.Str("file", fileName)
			// This triggers SA4006 "never used" from staticcheck, but
			// it will be used, therefore:
			// nolint
			e = e.Int("line", lineNo)
		}
	}
}
