package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// sarTemplate is what the Serverless Application Repository reads from an
// application version's SAM / CloudFormation template.
type sarTemplate struct {
	params       []map[string]any // ParameterDefinition shapes
	capabilities []string
	dependencies []sarDependency
	resources    int
}

type sarDependency struct {
	applicationID, semanticVersion string
}

const (
	sarMaxTemplateBytes = 1 << 20 // 1 MiB of template text
	sarMaxTemplateDefs  = 200     // parameters or resources per template
)

// Resource types that need CAPABILITY_RESOURCE_POLICY when deployed through the
// repository. ponytail: a representative set, not AWS's full list.
var sarResourcePolicyTypes = map[string]bool{
	"AWS::Lambda::Permission":                    true,
	"AWS::SNS::TopicPolicy":                      true,
	"AWS::SQS::QueuePolicy":                      true,
	"AWS::S3::BucketPolicy":                      true,
	"AWS::ApplicationAutoScaling::ScalingPolicy": true,
	"AWS::Lambda::LayerVersionPermission":        true,
}

// IAM properties that give a resource a custom name (CAPABILITY_NAMED_IAM).
var sarNamedIAMProps = []string{"RoleName", "UserName", "GroupName", "ManagedPolicyName", "InstanceProfileName"}

// sarParseTemplate validates a template body and extracts parameter
// definitions, required capabilities and nested-application dependencies.
// JSON templates are parsed; YAML templates get a structural check only.
// ponytail: no YAML library in go.mod, so YAML templates expose no parameters,
// capabilities or dependencies.
func sarParseTemplate(body string) (*sarTemplate, *sarError) {
	if strings.TrimSpace(body) == "" {
		return nil, sarBadRequest("The template is empty.")
	}
	if len(body) > sarMaxTemplateBytes {
		return nil, sarBadRequest("The template exceeds %d bytes.", sarMaxTemplateBytes)
	}
	t := &sarTemplate{params: []map[string]any{}, capabilities: []string{}}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		if strings.HasPrefix(body, "Resources:") || strings.Contains(body, "\nResources:") {
			return t, nil
		}
		return nil, sarBadRequest("The template is neither a JSON nor a YAML SAM/CloudFormation template.")
	}
	res, ok := doc["Resources"].(map[string]any)
	if !ok || len(res) == 0 {
		return nil, sarBadRequest("The template must contain a non-empty Resources section.")
	}
	params, _ := doc["Parameters"].(map[string]any)
	if len(res) > sarMaxTemplateDefs || len(params) > sarMaxTemplateDefs {
		return nil, sarBadRequest("The template has more than %d resources or parameters.", sarMaxTemplateDefs)
	}
	t.resources = len(res)

	refs := map[string][]string{} // parameter -> resources that Ref it
	caps := map[string]bool{}
	if _, ok := doc["Transform"]; ok {
		caps["CAPABILITY_AUTO_EXPAND"] = true
	}
	for _, name := range sarSortedKeys(res) {
		r, _ := res[name].(map[string]any)
		typ, _ := r["Type"].(string)
		if typ == "" {
			return nil, sarBadRequest("Resource %s has no Type.", name)
		}
		props, _ := r["Properties"].(map[string]any)
		switch {
		case strings.HasPrefix(typ, "AWS::IAM::"):
			caps["CAPABILITY_IAM"] = true
			for _, p := range sarNamedIAMProps {
				if _, ok := props[p]; ok {
					caps["CAPABILITY_NAMED_IAM"] = true
				}
			}
		case typ == "AWS::Serverless::Application" || typ == "AWS::CloudFormation::Stack":
			caps["CAPABILITY_AUTO_EXPAND"] = true
		}
		if sarResourcePolicyTypes[typ] {
			caps["CAPABILITY_RESOURCE_POLICY"] = true
		}
		if typ == "AWS::Serverless::Application" {
			if loc, ok := props["Location"].(map[string]any); ok {
				app, _ := loc["ApplicationId"].(string)
				ver, _ := loc["SemanticVersion"].(string)
				if app != "" && ver != "" {
					t.dependencies = append(t.dependencies, sarDependency{app, ver})
				}
			}
		}
		for p := range sarCollectRefs(r, 0) {
			refs[p] = append(refs[p], name)
		}
	}
	for _, c := range []string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM", "CAPABILITY_AUTO_EXPAND", "CAPABILITY_RESOURCE_POLICY"} {
		if caps[c] {
			t.capabilities = append(t.capabilities, c)
		}
	}
	for _, name := range sarSortedKeys(params) {
		p, ok := params[name].(map[string]any)
		if !ok {
			return nil, sarBadRequest("Parameter %s must be an object.", name)
		}
		def := map[string]any{"name": name, "referencedByResources": append([]string{}, refs[name]...)}
		for src, dst := range map[string]string{"Type": "type", "Description": "description", "AllowedPattern": "allowedPattern", "ConstraintDescription": "constraintDescription"} {
			if v, ok := p[src].(string); ok {
				def[dst] = v
			}
		}
		if v, ok := p["Default"]; ok {
			def["defaultValue"] = fmt.Sprint(v)
		}
		if av, ok := p["AllowedValues"].([]any); ok {
			vals := []string{}
			for _, a := range av {
				vals = append(vals, fmt.Sprint(a))
			}
			def["allowedValues"] = vals
		}
		for src, dst := range map[string]string{"MinLength": "minLength", "MaxLength": "maxLength", "MinValue": "minValue", "MaxValue": "maxValue"} {
			if n, ok := sarNumber(p[src]); ok {
				def[dst] = n
			}
		}
		switch v := p["NoEcho"].(type) {
		case bool:
			def["noEcho"] = v
		case string:
			def["noEcho"] = strings.EqualFold(v, "true")
		}
		t.params = append(t.params, def)
	}
	return t, nil
}

// sarCollectRefs returns the names used in {"Ref": name} anywhere under v.
// Depth is bounded so a deeply nested template can't recurse without limit.
func sarCollectRefs(v any, depth int) map[string]bool {
	out := map[string]bool{}
	var walk func(any, int)
	walk = func(v any, depth int) {
		if depth > 64 {
			return
		}
		switch t := v.(type) {
		case map[string]any:
			if ref, ok := t["Ref"].(string); ok && len(t) == 1 {
				out[ref] = true
			}
			for _, child := range t {
				walk(child, depth+1)
			}
		case []any:
			for _, child := range t {
				walk(child, depth+1)
			}
		}
	}
	walk(v, depth)
	return out
}

func sarNumber(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case string:
		var x int64
		if _, err := fmt.Sscan(n, &x); err == nil {
			return x, true
		}
	}
	return 0, false
}

func sarSortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
