package protocol

type ActionFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ActionErrorBody struct {
	Message string             `json:"message"`
	Errors  []ActionFieldError `json:"errors"`
}

type ActionTransport struct {
	Type   string           `json:"type"`
	Status int              `json:"status"`
	Body   *ActionErrorBody `json:"body"`
}

type OutcomeBehavior struct {
	Behavior string `json:"behavior"`
	Message  string `json:"message"`
	URL      string `json:"url"`
}

type ActionContext struct {
	TableID string `json:"tableId"`
}

type ActionOutcomeInput struct {
	Transport ActionTransport  `json:"transport"`
	OnSuccess *OutcomeBehavior `json:"onSuccess"`
	OnError   *OutcomeBehavior `json:"onError"`
	Context   ActionContext    `json:"context"`
}

type ActionEvent struct {
	Type      string             `json:"type"`
	Status    int                `json:"status,omitempty"`
	Message   string             `json:"message,omitempty"`
	URL       string             `json:"url,omitempty"`
	TableID   string             `json:"tableId,omitempty"`
	Display   *string            `json:"display,omitempty"`
	Retryable bool               `json:"retryable,omitempty"`
	Errors    []ActionFieldError `json:"errors,omitempty"`
}

type ActionOutcomeResult struct {
	OK     bool          `json:"ok"`
	Events []ActionEvent `json:"events"`
}

func ProcessActionOutcome(input ActionOutcomeInput) ActionOutcomeResult {
	transport := input.Transport
	if transport.Type == "success" {
		events := []ActionEvent{{Type: "requestSucceeded", Status: transport.Status}}
		if input.OnSuccess != nil {
			events = append(events, actionBehaviorEvent(*input.OnSuccess, input.Context))
		}
		return ActionOutcomeResult{OK: true, Events: events}
	}

	if transport.Type == "httpError" {
		body := ActionErrorBody{}
		if transport.Body != nil {
			body = *transport.Body
		}
		if transport.Status == 401 || transport.Status == 403 {
			var display *string
			if transport.Status == 403 {
				display = stringPointer("无权限访问")
			}
			return ActionOutcomeResult{OK: false, Events: []ActionEvent{
				{Type: "authFailure", Status: transport.Status},
				{Type: "errorState", Display: display},
			}}
		}

		if transport.Status == 400 && len(body.Errors) > 0 {
			events := []ActionEvent{{Type: "fieldErrors", Errors: body.Errors}}
			if input.OnError != nil && input.OnError.Behavior == "toast" {
				events = append(events, actionBehaviorEvent(*input.OnError, input.Context))
			} else if body.Message != "" {
				events = append(events, ActionEvent{Type: "toast", Message: body.Message})
			}
			return ActionOutcomeResult{OK: false, Events: events}
		}

		var display *string
		switch {
		case transport.Status == 404:
			display = stringPointer("资源不存在")
		case transport.Status >= 500:
			display = stringPointer("系统异常，请稍后重试")
		case body.Message != "":
			display = stringPointer(body.Message)
		}
		events := []ActionEvent{{Type: "errorState", Display: display}}
		if input.OnError != nil {
			events = append(events, actionBehaviorEvent(*input.OnError, input.Context))
		}
		return ActionOutcomeResult{OK: false, Events: events}
	}

	if transport.Type == "abort" {
		return ActionOutcomeResult{OK: false, Events: []ActionEvent{}}
	}
	display := "网络异常，请检查网络连接"
	if transport.Type == "timeout" {
		display = "请求超时，请稍后重试"
	}
	events := []ActionEvent{{Type: "errorState", Display: &display, Retryable: true}}
	if input.OnError != nil {
		events = append(events, actionBehaviorEvent(*input.OnError, input.Context))
	}
	return ActionOutcomeResult{OK: false, Events: events}
}

func actionBehaviorEvent(behavior OutcomeBehavior, context ActionContext) ActionEvent {
	switch behavior.Behavior {
	case "toast":
		return ActionEvent{Type: "toast", Message: behavior.Message}
	case "navigate":
		return ActionEvent{Type: "navigate", URL: behavior.URL}
	case "reload":
		if context.TableID != "" {
			return ActionEvent{Type: "reloadTable", TableID: context.TableID}
		}
		return ActionEvent{Type: "reloadCurrentData"}
	default:
		return ActionEvent{Type: "closeModal"}
	}
}

func stringPointer(value string) *string {
	return &value
}
