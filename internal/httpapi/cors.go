package httpapi

import (
	"net/http"
	"net/url"
	"strings"
)

const (
	corsAllowMethods  = "GET, POST, PATCH, DELETE, OPTIONS"
	corsAllowHeaders  = "Authorization, Content-Type, Idempotency-Key, X-Request-ID"
	corsExposeHeaders = "X-Request-ID, Content-Disposition"
)

var allowedCORSRequestHeaders = map[string]bool{
	"authorization":   true,
	"content-type":    true,
	"idempotency-key": true,
	"x-request-id":    true,
}

func corsMiddleware(allowedOrigin string, routes []routeMetadata, next http.Handler) http.Handler {
	if allowedOrigin == "" {
		return next
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origins := request.Header.Values("Origin")
		if len(origins) == 0 {
			next.ServeHTTP(response, request)
			return
		}
		origin := origins[0]
		if request.Method == http.MethodOptions {
			handleCORSPreflight(response, request, allowedOrigin, routes)
			return
		}
		if !matchRoute(routes, request.URL.Path, request.Method) {
			next.ServeHTTP(response, request)
			return
		}
		appendVary(response.Header(), "Origin")
		if len(origins) != 1 || !validRequestOrigin(origin) || origin != allowedOrigin {
			writeError(response, request, http.StatusForbidden, "CORS_ORIGIN_DENIED", "origin denied")
			return
		}
		response.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		response.Header().Set("Access-Control-Expose-Headers", corsExposeHeaders)
		next.ServeHTTP(response, request)
	})
}

func handleCORSPreflight(response http.ResponseWriter, request *http.Request, allowedOrigin string, routes []routeMetadata) {
	if !matchKnownPath(routes, request.URL.Path) {
		http.NotFound(response, request)
		return
	}
	appendVary(response.Header(), "Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers")
	origins := request.Header.Values("Origin")
	requestedMethods := request.Header.Values("Access-Control-Request-Method")
	origin, requestedMethod := "", ""
	if len(origins) == 1 {
		origin = origins[0]
	}
	if len(requestedMethods) == 1 {
		requestedMethod = requestedMethods[0]
	}
	methodAllowed := requestedMethod == http.MethodOptions || matchRoute(routes, request.URL.Path, requestedMethod)
	if len(origins) != 1 || len(requestedMethods) != 1 || !validRequestOrigin(origin) || origin != allowedOrigin || !methodAllowed || !validCORSRequestHeaders(request.Header.Values("Access-Control-Request-Headers")) {
		writeError(response, request, http.StatusForbidden, "CORS_PREFLIGHT_DENIED", "preflight denied")
		return
	}
	response.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	response.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
	response.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
	response.Header().Set("Access-Control-Max-Age", "600")
	response.Header().Set("Access-Control-Expose-Headers", corsExposeHeaders)
	response.WriteHeader(http.StatusNoContent)
}

func validRequestOrigin(origin string) bool {
	if origin == "" || strings.TrimSpace(origin) != origin || origin == "*" || strings.Contains(origin, ",") {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Opaque != "" || parsed.User != nil || parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return origin == parsed.Scheme+"://"+parsed.Host
}

func validCORSRequestHeaders(values []string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		for _, header := range strings.Split(value, ",") {
			header = strings.ToLower(strings.TrimSpace(header))
			if header == "" || !allowedCORSRequestHeaders[header] {
				return false
			}
		}
	}
	return true
}

func appendVary(header http.Header, values ...string) {
	existing := map[string]bool{}
	for _, line := range header.Values("Vary") {
		for _, value := range strings.Split(line, ",") {
			existing[strings.ToLower(strings.TrimSpace(value))] = true
		}
	}
	for _, value := range values {
		if !existing[strings.ToLower(value)] {
			header.Add("Vary", value)
			existing[strings.ToLower(value)] = true
		}
	}
}
