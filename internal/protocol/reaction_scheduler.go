package protocol

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	reactionCondition  = regexp.MustCompile(`^\$deps\.([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*(==|!=)\s*(true|false|null|-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?|'[^']*')$`)
	reactionDependency = regexp.MustCompile(`\$deps\.([A-Za-z_][A-Za-z0-9_]*)`)
)

type ReactionBranch struct {
	Value    any
	HasValue bool
}

func (branch *ReactionBranch) UnmarshalJSON(contents []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(contents, &object); err != nil {
		return err
	}
	rawValue, exists := object["value"]
	branch.HasValue = exists
	if exists {
		return json.Unmarshal(rawValue, &branch.Value)
	}
	return nil
}

type ReactionDefinition struct {
	When      string          `json:"when"`
	Fulfill   *ReactionBranch `json:"fulfill"`
	Otherwise *ReactionBranch `json:"otherwise"`
}

type ReactionField struct {
	Field     string               `json:"field"`
	Reactions []ReactionDefinition `json:"reactions"`
}

type ReactionObserver struct {
	ID   string `json:"id"`
	When string `json:"when"`
}

type ReactionScheduleInput struct {
	InitialValues map[string]any     `json:"initialValues"`
	MaxRounds     int                `json:"maxRounds"`
	Fields        []ReactionField    `json:"fields"`
	Observers     []ReactionObserver `json:"observers"`
}

type ReactionCommit struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type ReactionRound struct {
	Round        int              `json:"round"`
	Snapshot     map[string]any   `json:"snapshot"`
	Observations map[string]bool  `json:"observations"`
	Commits      []ReactionCommit `json:"commits"`
}

type ReactionWarning struct {
	Code  string `json:"code"`
	Field string `json:"field"`
	Count int    `json:"count"`
}

type ReactionScheduleResult struct {
	OK               bool              `json:"ok"`
	Values           map[string]any    `json:"values"`
	Rounds           []ReactionRound   `json:"rounds,omitempty"`
	Warnings         []ReactionWarning `json:"warnings,omitempty"`
	Code             string            `json:"code,omitempty"`
	MaxRounds        int               `json:"maxRounds,omitempty"`
	RoundCount       int               `json:"roundCount,omitempty"`
	DependencyFields []string          `json:"dependencyFields,omitempty"`
}

func RunReactionSchedule(input ReactionScheduleInput) ReactionScheduleResult {
	values := cloneJSON(input.InitialValues).(map[string]any)
	maxRounds := input.MaxRounds
	if maxRounds == 0 {
		maxRounds = 10
	}
	dependencyFields := collectReactionDependencyFields(input)
	warnings := make([]ReactionWarning, 0)
	warnedFields := make(map[string]struct{})
	rounds := make([]ReactionRound, 0)

	for round := 1; round <= maxRounds; round++ {
		snapshot := cloneJSON(values).(map[string]any)
		observations := make(map[string]bool, len(input.Observers))
		for _, observer := range input.Observers {
			observations[observer.ID] = evaluateReactionCondition(observer.When, snapshot)
		}

		pending := make(map[string]any)
		pendingOrder := make([]string, 0)
		for _, field := range input.Fields {
			valueWriteCount := 0
			for _, reaction := range field.Reactions {
				branch := reaction.Otherwise
				if evaluateReactionCondition(reaction.When, snapshot) {
					branch = reaction.Fulfill
				}
				if branch != nil && branch.HasValue {
					if _, exists := pending[field.Field]; !exists {
						pendingOrder = append(pendingOrder, field.Field)
					}
					pending[field.Field] = cloneJSON(branch.Value)
					valueWriteCount++
				}
			}
			if valueWriteCount > 1 {
				if _, warned := warnedFields[field.Field]; !warned {
					warnings = append(warnings, ReactionWarning{Code: "MULTIPLE_VALUE_WRITES", Field: field.Field, Count: valueWriteCount})
					warnedFields[field.Field] = struct{}{}
				}
			}
		}

		commits := make([]ReactionCommit, 0)
		for _, field := range pendingOrder {
			value := pending[field]
			if !reflect.DeepEqual(values[field], value) {
				values[field] = cloneJSON(value)
				commits = append(commits, ReactionCommit{Field: field, Value: cloneJSON(value)})
			}
		}
		rounds = append(rounds, ReactionRound{Round: round, Snapshot: snapshot, Observations: observations, Commits: commits})

		schedulesNextRound := false
		for _, commit := range commits {
			if _, exists := dependencyFields[commit.Field]; exists {
				schedulesNextRound = true
				break
			}
		}
		if !schedulesNextRound {
			return ReactionScheduleResult{OK: true, Values: values, Rounds: rounds, Warnings: warnings}
		}
	}

	fields := make([]string, 0, len(dependencyFields))
	for field := range dependencyFields {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return ReactionScheduleResult{
		OK: false, Code: "REACTION_LOOP_LIMIT", MaxRounds: maxRounds, Values: values,
		RoundCount: maxRounds, DependencyFields: fields,
	}
}

func evaluateReactionCondition(expression string, snapshot map[string]any) bool {
	match := reactionCondition.FindStringSubmatch(expression)
	if match == nil {
		panic(fmt.Sprintf("unsupported reference expression: %s", expression))
	}
	left, found := readReactionPath(snapshot, match[1])
	if !found {
		return false
	}
	equal := reflect.DeepEqual(left, parseReactionLiteral(match[3]))
	if match[2] == "==" {
		return equal
	}
	return !equal
}

func readReactionPath(values map[string]any, path string) (any, bool) {
	var current any = values
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

func parseReactionLiteral(token string) any {
	switch token {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if strings.HasPrefix(token, "'") {
		return strings.TrimSuffix(strings.TrimPrefix(token, "'"), "'")
	}
	value, _ := strconv.ParseFloat(token, 64)
	return value
}

func collectReactionDependencyFields(input ReactionScheduleInput) map[string]struct{} {
	fields := make(map[string]struct{})
	for _, field := range input.Fields {
		for _, reaction := range field.Reactions {
			for _, match := range reactionDependency.FindAllStringSubmatch(reaction.When, -1) {
				fields[match[1]] = struct{}{}
			}
		}
	}
	for _, observer := range input.Observers {
		for _, match := range reactionDependency.FindAllStringSubmatch(observer.When, -1) {
			fields[match[1]] = struct{}{}
		}
	}
	return fields
}

func cloneJSON(value any) any {
	contents, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var cloned any
	if err := json.Unmarshal(contents, &cloned); err != nil {
		panic(err)
	}
	return cloned
}
