package server

import "os"

var defaultTraceRoute = "/init"

const (
	EnvPORT              = "PORT"
	EnvHOST              = "HOST"
	EnvTLS               = "ENABLE_TLS"
	EnvTargetHost        = "TARGET_HOST"  // The host where the request will be sent
	EnvTraceHeaderPrefix = "TRACE_PREFIX" // The prefix of headers to be cleared
	EnvTraceRoute        = "TRACE_ROUTE"  // The route where the trace request will be initiated
)

func GetEnv(name, fallback string) string {
	res := os.Getenv(name)
	if res == "" {
		res = fallback
	}
	return res
}
