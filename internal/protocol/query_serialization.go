package protocol

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type queryEntry struct {
	Key   string
	Value any
}

func serializeQuery(baseURL string, sources [][]queryEntry) (string, string) {
	requestPart, fragment, _ := strings.Cut(baseURL, "#")
	if fragment != "" {
		fragment = "#" + fragment
	}
	path, baseQuery, _ := strings.Cut(requestPart, "?")
	merged := make(map[string]string)

	for _, segment := range strings.Split(baseQuery, "&") {
		if segment == "" {
			continue
		}
		encodedKey, encodedValue, hasEquals := strings.Cut(segment, "=")
		if !hasEquals {
			encodedValue = ""
		}
		key, err := url.PathUnescape(encodedKey)
		if err != nil {
			return "", "INVALID_BASE_URL_QUERY"
		}
		value, err := url.PathUnescape(encodedValue)
		if err != nil {
			return "", "INVALID_BASE_URL_QUERY"
		}
		if key == "" {
			return "", "INVALID_QUERY_KEY"
		}
		merged[key] = value
	}

	for _, source := range sources {
		for _, entry := range source {
			if entry.Key == "" {
				return "", "INVALID_QUERY_KEY"
			}
			text, tombstone, ok := scalarText(entry.Value)
			if !ok {
				return "", "INVALID_QUERY_VALUE"
			}
			if tombstone {
				delete(merged, entry.Key)
			} else {
				merged[entry.Key] = text
			}
		}
	}

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, percentEncode(key)+"="+percentEncode(merged[key]))
	}
	if len(parts) == 0 {
		return path + fragment, ""
	}
	return path + "?" + strings.Join(parts, "&") + fragment, ""
}

func scalarText(value any) (string, bool, bool) {
	switch typed := value.(type) {
	case nil:
		return "", true, true
	case string:
		return typed, false, true
	case bool:
		return strconv.FormatBool(typed), false, true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return "", false, false
		}
		if typed == 0 {
			return "0", false, true
		}
		return strconv.FormatFloat(typed, 'g', -1, 64), false, true
	case int:
		return strconv.Itoa(typed), false, true
	default:
		return "", false, false
	}
}

func percentEncode(value string) string {
	var builder strings.Builder
	for _, current := range []byte(value) {
		if current >= 'A' && current <= 'Z' || current >= 'a' && current <= 'z' ||
			current >= '0' && current <= '9' || strings.ContainsRune("-._~", rune(current)) {
			builder.WriteByte(current)
		} else {
			fmt.Fprintf(&builder, "%%%02X", current)
		}
	}
	return builder.String()
}
