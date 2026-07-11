package protocol_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/protocol"
	"go.yaml.in/yaml/v4"
)

var yamlFence = regexp.MustCompile("(?s)```ya?ml\\s*\\r?\\n(.*?)\\r?\\n```")

type scenarioFixtureCase struct {
	ID       string                           `json:"id"`
	Input    protocol.ScenarioExecutionInput  `json:"input"`
	Expected protocol.ScenarioExecutionResult `json:"expected"`
}

type scenarioFixtureSuite struct {
	FixtureVersion string                `json:"fixtureVersion"`
	Cases          []scenarioFixtureCase `json:"cases"`
}

func TestOfficialScenarioConformance(t *testing.T) {
	fixturesRoot := os.Getenv("SCHEMA_UI_FIXTURES")
	if fixturesRoot == "" {
		fixturesRoot = filepath.Join("..", "..", "..", "schema-ui-docs", "conformance", "fixtures")
	}
	contents, err := os.ReadFile(filepath.Join(fixturesRoot, "scenarios", "cases.json"))
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}

	var suite scenarioFixtureSuite
	if err := json.Unmarshal(contents, &suite); err != nil {
		t.Fatalf("decode fixtures: %v", err)
	}
	if suite.FixtureVersion != "1.0" {
		t.Fatalf("fixtureVersion = %q, want 1.0", suite.FixtureVersion)
	}
	protocolRoot := filepath.Join(fixturesRoot, "..", "..")

	for _, fixture := range suite.Cases {
		t.Run(fixture.ID, func(t *testing.T) {
			officialMeta := readOfficialScenarioMeta(t, protocolRoot, fixture.Input.ScenarioPath)
			actual, err := protocol.ExecuteScenario(fixture.Input, officialMeta)
			if err != nil {
				t.Fatalf("execute scenario: %v", err)
			}
			if !reflect.DeepEqual(actual, fixture.Expected) {
				t.Fatalf("result = %#v, want %#v", actual, fixture.Expected)
			}
		})
	}
}

func readOfficialScenarioMeta(t *testing.T, protocolRoot string, scenarioPath string) protocol.ScenarioMeta {
	t.Helper()
	markdown, err := os.ReadFile(filepath.Join(protocolRoot, filepath.FromSlash(scenarioPath)))
	if err != nil {
		t.Fatalf("read scenario: %v", err)
	}
	match := yamlFence.FindSubmatch(markdown)
	if match == nil {
		t.Fatalf("missing YAML fence: %s", scenarioPath)
	}
	var page struct {
		Meta protocol.ScenarioMeta `yaml:"meta"`
	}
	if err := yaml.Unmarshal(match[1], &page); err != nil {
		t.Fatalf("decode scenario YAML: %v", err)
	}
	return page.Meta
}
