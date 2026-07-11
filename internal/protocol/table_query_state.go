package protocol

type TableQueryState struct {
	Filters  map[string]any `json:"filters"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Sort     *string        `json:"sort"`
}

type TableQueryEvent struct {
	Type    string         `json:"type"`
	Filters map[string]any `json:"filters"`
	Page    int            `json:"page"`
	Sort    *string        `json:"sort"`
}

type TableQueryInput struct {
	BaseURL      string           `json:"baseUrl"`
	StaticParams map[string]any   `json:"staticParams"`
	State        TableQueryState  `json:"state"`
	Event        *TableQueryEvent `json:"event"`
}

type TableQueryResult struct {
	State TableQueryState `json:"state"`
	URL   string          `json:"url"`
	OK    bool            `json:"ok,omitempty"`
	Code  string          `json:"code,omitempty"`
}

func ApplyTableQueryEvent(state TableQueryState, event *TableQueryEvent) TableQueryState {
	result := state
	result.Filters = cloneValues(state.Filters)
	if event == nil {
		return result
	}
	switch event.Type {
	case "submitSearch":
		result.Filters = cloneValues(event.Filters)
		result.Page = 1
	case "clearSearch":
		result.Filters = map[string]any{}
		result.Page = 1
	case "changePage":
		result.Page = event.Page
	case "changeSort":
		result.Page = 1
		result.Sort = event.Sort
	}
	return result
}

func BuildTableQuery(input TableQueryInput) TableQueryResult {
	state := ApplyTableQueryEvent(input.State, input.Event)
	var sortValue any
	if state.Sort != nil {
		sortValue = *state.Sort
	}
	url, code := serializeQuery(input.BaseURL, [][]queryEntry{
		mappingEntries(input.StaticParams),
		mappingEntries(state.Filters),
		{
			{Key: "page", Value: state.Page},
			{Key: "pageSize", Value: state.PageSize},
			{Key: "sort", Value: sortValue},
		},
	})
	if code != "" {
		return TableQueryResult{OK: false, Code: code}
	}
	return TableQueryResult{State: state, URL: url}
}

func cloneValues(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
