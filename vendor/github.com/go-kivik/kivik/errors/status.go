package errors

var statusTextStrings = map[int]string{
	400: "bad_request",
	401: "unauthorized",
	403: "forbidden",
	404: "not_found",
	405: "method_not_allowed",
	408: "request_timeout",
	409: "conflict",
	412: "precondition_failed",
	415: "unsupported_media_type",
	416: "requested_range_not_satisfiable",
	417: "expectation_failed",

	500: "internal_server_error",
	501: "not_implemented",

	600: "unknown",
	601: "network_error",
	602: "bad_response",
}

// statusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown to Kivik.
func statusText(code int) string {
	if text, ok := statusTextStrings[code]; ok {
		return text
	}
	return "unknown"
}
