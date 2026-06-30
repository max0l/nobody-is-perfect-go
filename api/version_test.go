package api

import (
	"os"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIVersionMatchesProjectVersion(t *testing.T) {
	versionBytes, err := os.ReadFile("../VERSION")
	if err != nil {
		t.Fatalf("ReadFile VERSION returned error: %v", err)
	}
	projectVersion := strings.TrimSpace(string(versionBytes))
	if projectVersion == "" {
		t.Fatal("expected VERSION to be set")
	}

	swagger, err := openapi3.NewLoader().LoadFromFile("../api.yaml")
	if err != nil {
		t.Fatalf("LoadFromFile api.yaml returned error: %v", err)
	}
	if swagger.Info == nil {
		t.Fatal("expected api.yaml info section")
	}
	if swagger.Info.Version != projectVersion {
		t.Fatalf("expected api.yaml version %q to match VERSION, got %q", projectVersion, swagger.Info.Version)
	}
}
