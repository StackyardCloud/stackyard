package awsmodels

import (
	"strings"
	"testing"
)

func TestPipesModelValidate(t *testing.T) {
	m := PipesModel()
	ok := map[string]any{
		"Source": "arn:aws:sqs:us-east-1:123456789012:src", "Target": "arn:aws:sqs:us-east-1:123456789012:dst",
		"RoleArn":          "arn:aws:iam::123456789012:role/pipe",
		"SourceParameters": map[string]any{"SqsQueueParameters": map[string]any{"BatchSize": float64(10)}},
	}
	if p := m.Validate("CreatePipeRequest", ok); len(p) != 0 {
		t.Fatalf("valid request rejected: %v", p)
	}
	bad := map[string]any{
		"Source": "nope", "RoleArn": "arn:aws:iam::123456789012:role/pipe", "Bogus": 1,
		"SourceParameters": map[string]any{"SqsQueueParameters": map[string]any{"BatchSize": float64(0)}},
		"DesiredState":     "PAUSED",
	}
	got := strings.Join(m.Validate("CreatePipeRequest", bad), "\n")
	for _, want := range []string{"missing required member Target", "unknown member Bogus", "Source: does not match", "BatchSize: must be at least 1", "DesiredState: must be one of"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	conformed := m.Conform("Pipe", map[string]any{"Name": "p", "Secret": "x"}).(map[string]any)
	if _, leaked := conformed["Secret"]; leaked || conformed["Name"] != "p" {
		t.Fatalf("Conform kept unknown members: %v", conformed)
	}
}
