package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type osisDestination struct{ service, endpoint string }

var osisHostURL = regexp.MustCompile(`https?://[^\s"',\]]+`)

// osisCheckConfig structurally checks a Data Prepper pipeline configuration:
// a top-level `version`, and at least one pipeline with both a source and a
// sink. JSON bodies are checked as JSON; anything else as indented YAML.
// ponytail: no YAML library is in go.mod, so this is a line-based structural
// check (keys, indentation, tabs), not a full YAML parse; plugin options
// aren't validated. Swap in a YAML parser if real parsing is ever needed.
func osisCheckConfig(body string) []string {
	if strings.TrimSpace(body) == "" {
		return []string{"The pipeline configuration is empty."}
	}
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		return osisCheckJSONConfig(body)
	}
	type node struct {
		name       string
		childDepth int
		keys       map[string]bool
	}
	var problems []string
	hasVersion := false
	var pipelines []*node
	var cur *node
	for i, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " "), "\t") {
			problems = append(problems, fmt.Sprintf("Line %d: tabs can't be used for indentation in YAML.", i+1))
			continue
		}
		text := strings.TrimRight(line, " \r")
		trimmed := strings.TrimSpace(text)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "---" {
			continue
		}
		depth := len(text) - len(strings.TrimLeft(text, " "))
		key, _, isKey := strings.Cut(strings.TrimPrefix(trimmed, "- "), ":")
		key = strings.Trim(strings.TrimSpace(key), `"'`)
		if depth == 0 {
			if !isKey || strings.HasPrefix(trimmed, "- ") {
				problems = append(problems, fmt.Sprintf("Line %d: top-level entries must be `key:` mappings.", i+1))
				cur = nil
				continue
			}
			if key == "version" {
				hasVersion = true
				cur = nil
				continue
			}
			cur = &node{name: key, keys: map[string]bool{}}
			pipelines = append(pipelines, cur)
			continue
		}
		if cur == nil || !isKey {
			continue
		}
		if cur.childDepth == 0 {
			cur.childDepth = depth
		}
		if depth == cur.childDepth {
			cur.keys[key] = true
		}
	}
	if !hasVersion {
		problems = append(problems, "The pipeline configuration must set a top-level version (for example `version: \"2\"`).")
	}
	if len(pipelines) == 0 {
		problems = append(problems, "The pipeline configuration must define at least one pipeline.")
	}
	for _, p := range pipelines {
		for _, need := range []string{"source", "sink"} {
			if !p.keys[need] {
				problems = append(problems, fmt.Sprintf("Pipeline %s must define a %s.", p.name, need))
			}
		}
	}
	return problems
}

func osisCheckJSONConfig(body string) []string {
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		return []string{"The pipeline configuration isn't valid JSON: " + err.Error()}
	}
	var problems []string
	if _, ok := doc["version"]; !ok {
		problems = append(problems, "The pipeline configuration must set a top-level version.")
	}
	names := []string{}
	for name := range doc {
		if name != "version" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		problems = append(problems, "The pipeline configuration must define at least one pipeline.")
	}
	for _, name := range names {
		p, _ := doc[name].(map[string]any)
		for _, need := range []string{"source", "sink"} {
			if p == nil || p[need] == nil {
				problems = append(problems, fmt.Sprintf("Pipeline %s must define a %s.", name, need))
			}
		}
	}
	return problems
}

// osisDestinations reads OpenSearch sink hosts out of a configuration.
func osisDestinations(body string) []osisDestination {
	seen := map[string]bool{}
	var out []osisDestination
	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, "hosts") && !strings.Contains(line, "http") {
			continue
		}
		for _, url := range osisHostURL.FindAllString(line, 8) {
			if !seen[url] && len(out) < 20 {
				seen[url] = true
				out = append(out, osisDestination{service: "OpenSearch", endpoint: url})
			}
		}
	}
	return out
}

type osisBlueprint struct {
	name, display, description, service, useCase string
	yaml                                         string
}

// osisBlueprints is a small embedded catalog of real Data Prepper configurations.
var osisBlueprints = []osisBlueprint{
	{
		name: "AWS-ApacheLogPipeline", display: "Apache log pipeline", service: "OpenSearch", useCase: "Log analytics",
		description: "Parse Apache access logs received over HTTP and index them in OpenSearch.",
		yaml: `version: "2"
apache-log-pipeline:
  source:
    http:
      path: "/${pipelineName}/logs"
  processor:
    - grok:
        match:
          log: ["%{COMMONAPACHELOG_DATATYPED}"]
  sink:
    - opensearch:
        hosts: ["https://search-mydomain.us-east-1.es.amazonaws.com"]
        index: "apache_logs"
        aws:
          region: "us-east-1"
          sts_role_arn: "arn:aws:iam::123456789012:role/pipeline-role"
`,
	},
	{
		name: "AWS-OtelTracesPipeline", display: "OpenTelemetry traces pipeline", service: "OpenSearch", useCase: "Trace analytics",
		description: "Receive OpenTelemetry traces and write raw spans and the service map to OpenSearch.",
		yaml: `version: "2"
otel-trace-pipeline:
  source:
    otel_trace_source:
      path: "/${pipelineName}/v1/traces"
  sink:
    - pipeline:
        name: "raw-trace-pipeline"
raw-trace-pipeline:
  source:
    pipeline:
      name: "otel-trace-pipeline"
  processor:
    - otel_traces:
  sink:
    - opensearch:
        hosts: ["https://search-mydomain.us-east-1.es.amazonaws.com"]
        index_type: "trace-analytics-raw"
        aws:
          region: "us-east-1"
          sts_role_arn: "arn:aws:iam::123456789012:role/pipeline-role"
`,
	},
	{
		name: "AWS-S3SourcePipeline", display: "S3 source pipeline", service: "S3", useCase: "Log analytics",
		description: "Read new S3 objects announced on an SQS queue and index each line in OpenSearch.",
		yaml: `version: "2"
s3-log-pipeline:
  source:
    s3:
      notification_type: "sqs"
      codec:
        newline:
      sqs:
        queue_url: "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue"
      aws:
        region: "us-east-1"
        sts_role_arn: "arn:aws:iam::123456789012:role/pipeline-role"
  sink:
    - opensearch:
        hosts: ["https://search-mydomain.us-east-1.es.amazonaws.com"]
        index: "s3_logs"
        aws:
          region: "us-east-1"
          sts_role_arn: "arn:aws:iam::123456789012:role/pipeline-role"
`,
	},
}

func osisFindBlueprint(name string) *osisBlueprint {
	for i := range osisBlueprints {
		if osisBlueprints[i].name == name {
			return &osisBlueprints[i]
		}
	}
	return nil
}

// osisYAMLToJSON converts one of the catalog's YAML bodies (mappings, block
// lists and flow scalars only) to indented JSON for Format=JSON.
// ponytail: handles the catalog's YAML subset only; it isn't a general parser.
func osisYAMLToJSON(src string) string {
	var lines []string
	for _, l := range strings.Split(src, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	// Recursive descent over indentation: parse(i, indent) reads the block at
	// line i and returns it with the index of the first line after it.
	var parse func(i, indent int) (any, int)
	parse = func(i, indent int) (any, int) {
		if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "- ") {
			var list []any
			for i < len(lines) {
				line := lines[i]
				d := len(line) - len(strings.TrimLeft(line, " "))
				t := strings.TrimSpace(line)
				if d != indent || !strings.HasPrefix(t, "- ") {
					break
				}
				// "- key:" opens a mapping item whose keys sit at d+2.
				lines[i] = strings.Repeat(" ", d+2) + strings.TrimPrefix(t, "- ")
				v, next := parse(i, d+2)
				list = append(list, v)
				i = next
			}
			return list, i
		}
		obj := orderedObject{}
		for i < len(lines) {
			line := lines[i]
			d := len(line) - len(strings.TrimLeft(line, " "))
			if d < indent {
				break
			}
			if d > indent {
				i++
				continue
			}
			key, rest, _ := strings.Cut(strings.TrimSpace(line), ":")
			rest = strings.TrimSpace(rest)
			i++
			if rest != "" {
				obj = append(obj, orderedField{key, osisScalar(rest)})
				continue
			}
			if i < len(lines) {
				nd := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
				if nd > d {
					v, next := parse(i, nd)
					obj = append(obj, orderedField{key, v})
					i = next
					continue
				}
			}
			obj = append(obj, orderedField{key, nil})
		}
		return obj, i
	}
	v, _ := parse(0, 0)
	raw, _ := json.MarshalIndent(v, "", "  ")
	return string(raw)
}

type orderedField struct {
	key   string
	value any
}

// orderedObject marshals as a JSON object keeping insertion order.
type orderedObject []orderedField

func (o orderedObject) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteString("{")
	for i, f := range o {
		if i > 0 {
			b.WriteString(",")
		}
		k, _ := json.Marshal(f.key)
		v, err := json.Marshal(f.value)
		if err != nil {
			return nil, err
		}
		b.Write(k)
		b.WriteString(":")
		b.Write(v)
	}
	b.WriteString("}")
	return []byte(b.String()), nil
}

func osisScalar(s string) any {
	if strings.HasPrefix(s, "[") {
		var list []any
		if json.Unmarshal([]byte(s), &list) == nil {
			return list
		}
	}
	return strings.Trim(s, `"`)
}
