package httpapi

import "strings"

type routeMetadata struct {
	pattern string
	methods map[string]bool
	allow   string
}

func activeRouteMetadata(dependencies Dependencies) []routeMetadata {
	routes := make([]routeMetadata, 0, 10)
	if dependencies.Auth != nil {
		routes = append(routes,
			newRouteMetadata("/api/v1/auth/login", "POST"),
			newRouteMetadata("/api/v1/auth/me", "GET"),
			newRouteMetadata("/api/v1/auth/logout", "POST"),
		)
	}
	if dependencies.Auth != nil && dependencies.Orders != nil {
		routes = append(routes,
			orderCollectionMetadata(),
			orderDetailMetadata(),
		)
		if dependencies.OrderActions {
			for _, action := range []string{"confirm", "fulfill", "ship", "complete", "cancel"} {
				routes = append(routes, orderActionMetadata(action))
			}
		}
	}
	if dependencies.Auth != nil && dependencies.Attachments != nil && !dependencies.DisableAttachmentRoutes {
		routes = append(routes, attachmentCollectionMetadata(), attachmentDetailMetadata())
	}
	if dependencies.Auth != nil && dependencies.Refunds != nil && !dependencies.DisableRefundRoutes {
		routes = append(routes, refundCollectionMetadata(), refundCreateMetadata(), refundDecisionMetadata("approve"), refundDecisionMetadata("reject"))
	}
	if dependencies.Auth != nil && dependencies.Dashboard != nil && !dependencies.DisableDashboardRoutes {
		routes = append(routes, dashboardSummaryMetadata(), dashboardOrderStatusMetadata(), dashboardTrendMetadata())
	}
	return routes
}

func newRouteMetadata(pattern string, methods ...string) routeMetadata {
	return routeMetadata{pattern: pattern, methods: methodSet(methods...), allow: strings.Join(methods, ", ")}
}

func orderCollectionMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/orders", "GET", "POST")
}

func orderDetailMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/orders/{orderId}", "GET", "PATCH")
}

func orderActionMetadata(action string) routeMetadata {
	return newRouteMetadata("/api/v1/orders/{orderId}/"+action, "POST")
}

func attachmentCollectionMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/attachments", "POST")
}

func attachmentDetailMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/attachments/{attachmentId}", "GET", "DELETE")
}

func refundCollectionMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/refunds", "GET")
}

func refundCreateMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/orders/{orderId}/refunds", "POST")
}

func refundDecisionMetadata(action string) routeMetadata {
	return newRouteMetadata("/api/v1/refunds/{refundId}/"+action, "POST")
}

func dashboardSummaryMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/dashboard/summary", "GET")
}

func dashboardOrderStatusMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/dashboard/order-status", "GET")
}

func dashboardTrendMetadata() routeMetadata {
	return newRouteMetadata("/api/v1/dashboard/trend", "GET")
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
