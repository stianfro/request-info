package requestinfo_test

import (
	"os"
	"strings"
	"testing"
)

func TestProjectDescriptorSetsGoBuildTarget(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("project.toml")
	if err != nil {
		t.Fatalf("read project.toml: %v", err)
	}

	contents := string(data)
	for _, want := range []string{
		`schema-version = "0.2"`,
		`name = "BP_GO_TARGETS"`,
		`value = "./cmd/request-info"`,
	} {
		if !strings.Contains(contents, want) {
			t.Fatalf("project.toml missing %q", want)
		}
	}
}
