package httpapi

import "strings"

type routeMetadata struct {
	pattern string
	methods map[string]bool
}

func activeRouteMetadata(dependencies Dependencies) []routeMetadata {
	routes := make([]routeMetadata, 0, 5)
	if dependencies.Auth != nil {
		routes = append(routes,
			routeMetadata{pattern: "/api/v1/auth/login", methods: methodSet("POST")},
			routeMetadata{pattern: "/api/v1/auth/me", methods: methodSet("GET")},
			routeMetadata{pattern: "/api/v1/auth/logout", methods: methodSet("POST")},
		)
	}
	if dependencies.Auth != nil && dependencies.Orders != nil {
		routes = append(routes,
			routeMetadata{pattern: "/api/v1/orders", methods: methodSet("GET", "POST")},
			routeMetadata{pattern: "/api/v1/orders/{orderId}", methods: methodSet("GET", "PATCH")},
		)
		if dependencies.OrderActions {
			for _, action := range []string{"confirm", "fulfill", "ship", "complete", "cancel"} {
				routes = append(routes, routeMetadata{pattern: "/api/v1/orders/{orderId}/" + action, methods: methodSet("POST")})
			}
		}
	}
	return routes
}

func methodSet(methods ...string) map[string]bool {
	result := make(map[string]bool, len(methods))
	for _, method := range methods {
		result[method] = true
	}
	return result
}

func matchRoute(routes []routeMetadata, path, method string) bool {
	for _, route := range routes {
		if matchRoutePath(route.pattern, path) && route.methods[method] {
			return true
		}
	}
	return false
}

func matchKnownPath(routes []routeMetadata, path string) bool {
	for _, route := range routes {
		if matchRoutePath(route.pattern, path) {
			return true
		}
	}
	return false
}

func matchRoutePath(pattern, path string) bool {
	patternParts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for index, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if pathParts[index] == "" {
				return false
			}
			continue
		}
		if part != pathParts[index] {
			return false
		}
	}
	return true
}
