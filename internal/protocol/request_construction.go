package protocol

import (
	"regexp"
	"sort"
	"strings"
)

var rowReference = regexp.MustCompile(`^\$row\.([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)$`)

type DataRef struct {
	Method string         `json:"method"`
	URL    string         `json:"url"`
	Params map[string]any `json:"params"`
}

type RequestAction struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

type RequestMapping struct {
	Path  map[string]any `json:"path"`
	Query map[string]any `json:"query"`
	Body  map[string]any `json:"body"`
}

type RequestInput struct {
	Kind           string         `json:"kind"`
	DataRef        *DataRef       `json:"dataRef"`
	Action         *RequestAction `json:"action"`
	RequestMapping RequestMapping `json:"requestMapping"`
	Row            map[string]any `json:"row"`
}

type BuiltRequest struct {
	Method string         `json:"method"`
	URL    string         `json:"url"`
	Body   map[string]any `json:"body"`
}

type RequestResult struct {
	OK      bool          `json:"ok"`
	Request *BuiltRequest `json:"request,omitempty"`
	Code    string        `json:"code,omitempty"`
	Path    string        `json:"path,omitempty"`
}

func BuildRequest(input RequestInput) RequestResult {
	if input.Kind == "dataRef" {
		method := input.DataRef.Method
		if method == "" {
			method = "GET"
		}
		url, code := serializeQuery(input.DataRef.URL, [][]queryEntry{mappingEntries(input.DataRef.Params)})
		if code != "" {
			return RequestResult{OK: false, Code: code}
		}
		return successfulRequest(method, url, nil)
	}
	if input.Kind != "rowAction" {
		return RequestResult{OK: false, Code: "INVALID_REQUEST_KIND", Path: "kind"}
	}

	pathValues, failure := resolveRequestMapping(input.RequestMapping.Path, input.Row, "path")
	if failure != nil {
		return *failure
	}
	queryValues, failure := resolveRequestMapping(input.RequestMapping.Query, input.Row, "query")
	if failure != nil {
		return *failure
	}
	bodyValues, failure := resolveRequestMapping(input.RequestMapping.Body, input.Row, "body")
	if failure != nil {
		return *failure
	}

	requestURL := input.Action.URL
	for _, entry := range pathValues {
		if entry.Value == nil {
			return RequestResult{OK: false, Code: "NULL_PATH_VALUE", Path: "requestMapping.path." + entry.Key}
		}
		encoded, ok := encodePathValue(entry.Value)
		if !ok {
			return RequestResult{OK: false, Code: "INVALID_PATH_VALUE", Path: "requestMapping.path." + entry.Key}
		}
		requestURL = strings.ReplaceAll(requestURL, "{"+entry.Key+"}", encoded)
	}

	requestURL, code := serializeQuery(requestURL, [][]queryEntry{queryValues})
	if code != "" {
		return RequestResult{OK: false, Code: code}
	}
	method := input.Action.Method
	if method == "" {
		method = "GET"
	}
	var body map[string]any
	if len(bodyValues) > 0 {
		body = make(map[string]any, len(bodyValues))
		for _, entry := range bodyValues {
			body[entry.Key] = entry.Value
		}
	}
	return successfulRequest(method, requestURL, body)
}

func successfulRequest(method string, requestURL string, body map[string]any) RequestResult {
	return RequestResult{OK: true, Request: &BuiltRequest{Method: method, URL: requestURL, Body: body}}
}

func resolveRequestMapping(mapping map[string]any, row map[string]any, section string) ([]queryEntry, *RequestResult) {
	entries := mappingEntries(mapping)
	for index, entry := range entries {
		value, found := resolveRowValue(entry.Value, row)
		if !found {
			failure := RequestResult{OK: false, Code: "UNRESOLVED_ROW_VALUE", Path: "requestMapping." + section + "." + entry.Key}
			return nil, &failure
		}
		entries[index].Value = value
	}
	return entries, nil
}

func mappingEntries(mapping map[string]any) []queryEntry {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]queryEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, queryEntry{Key: key, Value: mapping[key]})
	}
	return entries
}

func resolveRowValue(value any, row map[string]any) (any, bool) {
	configured, ok := value.(string)
	if !ok {
		return value, true
	}
	match := rowReference.FindStringSubmatch(configured)
	if match == nil {
		return value, true
	}
	var current any = row
	for _, segment := range strings.Split(match[1], ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func encodePathValue(value any) (string, bool) {
	text, tombstone, ok := scalarText(value)
	if !ok || tombstone {
		return "", false
	}
	return percentEncode(text), true
}
