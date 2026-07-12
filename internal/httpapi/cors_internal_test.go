package httpapi

import (
	"net/http"
	"testing"
)

func TestAppendVaryPreservesAndDeduplicatesValues(t *testing.T) {
	header := http.Header{"Vary": []string{"Accept-Encoding, Origin"}}
	appendVary(header, "Origin", "Access-Control-Request-Method")
	if values := header.Values("Vary"); len(values) != 2 || values[0] != "Accept-Encoding, Origin" || values[1] != "Access-Control-Request-Method" {
		t.Fatalf("Vary = %v", values)
	}
}
