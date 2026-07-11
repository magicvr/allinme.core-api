package protocol

import (
	"encoding/json"
	"fmt"
)

type ScenarioMeta struct {
	PageID          string `json:"pageId" yaml:"pageId"`
	ProtocolVersion string `json:"protocolVersion" yaml:"protocolVersion"`
}

type ScenarioStep struct {
	Kind  string          `json:"kind"`
	Input json.RawMessage `json:"input"`
}

type ScenarioExecutionInput struct {
	ScenarioPath string         `json:"scenarioPath"`
	ScenarioMeta ScenarioMeta   `json:"scenarioMeta"`
	Steps        []ScenarioStep `json:"steps"`
}

type ScenarioExecutionResult struct {
	PageID          string `json:"pageId"`
	ProtocolVersion string `json:"protocolVersion"`
	Steps           []any  `json:"steps"`
}

func ExecuteScenario(input ScenarioExecutionInput, officialMeta ScenarioMeta) (ScenarioExecutionResult, error) {
	if officialMeta != input.ScenarioMeta {
		return ScenarioExecutionResult{}, fmt.Errorf("scenario metadata mismatch: %s", input.ScenarioPath)
	}
	steps := make([]any, 0, len(input.Steps))
	for _, step := range input.Steps {
		result, err := executeScenarioStep(step)
		if err != nil {
			return ScenarioExecutionResult{}, err
		}
		steps = append(steps, normalizeJSON(result))
	}
	return ScenarioExecutionResult{
		PageID: officialMeta.PageID, ProtocolVersion: officialMeta.ProtocolVersion, Steps: steps,
	}, nil
}

func executeScenarioStep(step ScenarioStep) (any, error) {
	switch step.Kind {
	case "request":
		var input RequestInput
		if err := json.Unmarshal(step.Input, &input); err != nil {
			return nil, err
		}
		return BuildRequest(input), nil
	case "responseMapping":
		var input ResponseMappingInput
		if err := json.Unmarshal(step.Input, &input); err != nil {
			return nil, err
		}
		return MapResponse(input), nil
	case "searchTable":
		var input TableQueryInput
		if err := json.Unmarshal(step.Input, &input); err != nil {
			return nil, err
		}
		return BuildTableQuery(input), nil
	case "action":
		var input ActionOutcomeInput
		if err := json.Unmarshal(step.Input, &input); err != nil {
			return nil, err
		}
		return ProcessActionOutcome(input), nil
	case "upload":
		var input UploadExecutionInput
		if err := json.Unmarshal(step.Input, &input); err != nil {
			return nil, err
		}
		return ExecuteUpload(input), nil
	default:
		return nil, fmt.Errorf("unknown scenario step: %s", step.Kind)
	}
}

func normalizeJSON(value any) any {
	contents, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var normalized any
	if err := json.Unmarshal(contents, &normalized); err != nil {
		panic(err)
	}
	return normalized
}
