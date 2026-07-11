package protocol_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/protocol"
)

type responseMappingFixtureCase struct {
	ID       string                         `json:"id"`
	Input    protocol.ResponseMappingInput  `json:"input"`
	Expected protocol.ResponseMappingResult `json:"expected"`
}

type responseMappingFixtureSuite struct {
	FixtureVersion string                       `json:"fixtureVersion"`
	Cases          []responseMappingFixtureCase `json:"cases"`
}

func TestResponseMappingConformance(t *testing.T) {
	fixturesRoot := os.Getenv("SCHEMA_UI_FIXTURES")
	if fixturesRoot == "" {
		fixturesRoot = filepath.Join("..", "..", "..", "schema-ui-docs", "conformance", "fixtures")
	}
	contents, err := os.ReadFile(filepath.Join(fixturesRoot, "response-mapping", "cases.json"))
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}

	var suite responseMappingFixtureSuite
	if err := json.Unmarshal(contents, &suite); err != nil {
		t.Fatalf("decode fixtures: %v", err)
	}
	if suite.FixtureVersion != "1.0" {
		t.Fatalf("fixtureVersion = %q, want 1.0", suite.FixtureVersion)
	}

	for _, fixture := range suite.Cases {
		t.Run(fixture.ID, func(t *testing.T) {
			actual := protocol.MapResponse(fixture.Input)
			if !reflect.DeepEqual(actual, fixture.Expected) {
				t.Fatalf("result = %#v, want %#v", actual, fixture.Expected)
			}
		})
	}
}
