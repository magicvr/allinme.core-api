package protocol

import (
	"math"
	"strings"
)

type ResponseMapping struct {
	List  string `json:"list"`
	Total string `json:"total"`
}

type ResponseMappingInput struct {
	Component         string           `json:"component"`
	PaginationMode    string           `json:"paginationMode"`
	DatasourceMapping *ResponseMapping `json:"datasourceMapping"`
	LocalMapping      *ResponseMapping `json:"localMapping"`
	Response          any              `json:"response"`
}

type MappedResponse struct {
	List  []any    `json:"list"`
	Total *float64 `json:"total,omitempty"`
}

type ResponseMappingResult struct {
	OK   bool            `json:"ok"`
	Data *MappedResponse `json:"data,omitempty"`
	Code string          `json:"code,omitempty"`
	Path string          `json:"path,omitempty"`
}

func MapResponse(input ResponseMappingInput) ResponseMappingResult {
	mapping := input.DatasourceMapping
	if input.LocalMapping != nil {
		mapping = input.LocalMapping
	}

	if input.Component == "chart" && mapping == nil {
		list, ok := input.Response.([]any)
		if !ok {
			return responseMappingFailure("RESPONSE_MAPPING_TYPE_MISMATCH", "$")
		}
		return successfulResponseMapping(list, nil)
	}

	listPath := "list"
	if mapping != nil && mapping.List != "" {
		listPath = mapping.List
	}
	listValue, found := readResponsePath(input.Response, listPath)
	if !found {
		return responseMappingFailure("RESPONSE_MAPPING_PATH_MISSING", listPath)
	}
	list, ok := listValue.([]any)
	if !ok {
		return responseMappingFailure("RESPONSE_MAPPING_TYPE_MISMATCH", listPath)
	}

	if input.Component != "table" || input.PaginationMode != "server" {
		return successfulResponseMapping(list, nil)
	}
	totalPath := "total"
	if mapping != nil && mapping.Total != "" {
		totalPath = mapping.Total
	}
	totalValue, found := readResponsePath(input.Response, totalPath)
	if !found {
		return responseMappingFailure("RESPONSE_MAPPING_PATH_MISSING", totalPath)
	}
	total, ok := totalValue.(float64)
	if !ok || math.IsNaN(total) || math.IsInf(total, 0) {
		return responseMappingFailure("RESPONSE_MAPPING_TYPE_MISMATCH", totalPath)
	}
	return successfulResponseMapping(list, &total)
}

func readResponsePath(response any, path string) (any, bool) {
	current := response
	for _, segment := range strings.Split(path, ".") {
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

func responseMappingFailure(code string, path string) ResponseMappingResult {
	return ResponseMappingResult{OK: false, Code: code, Path: path}
}

func successfulResponseMapping(list []any, total *float64) ResponseMappingResult {
	return ResponseMappingResult{OK: true, Data: &MappedResponse{List: list, Total: total}}
}
