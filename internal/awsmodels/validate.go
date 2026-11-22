package awsmodels

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Model is a validation-only botocore service model (see
// scripts/gen-aws-model-slim.py): shapes plus operation input/output names.
type Model struct {
	Operations map[string]struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	} `json:"operations"`
	Shapes map[string]*ModelShape `json:"shapes"`

	patterns map[string]*regexp.Regexp
}

// ModelShape is one botocore shape.
type ModelShape struct {
	Type      string               `json:"type"`
	Required  []string             `json:"required"`
	Members   map[string]MemberRef `json:"members"`
	Member    *MemberRef           `json:"member"`
	Key       *MemberRef           `json:"key"`
	Value     *MemberRef           `json:"value"`
	Min       *float64             `json:"min"`
	Max       *float64             `json:"max"`
	Pattern   string               `json:"pattern"`
	Enum      []string             `json:"enum"`
	Sensitive bool                 `json:"sensitive"`
}

// MemberRef points a structure member, list member or map key/value at a shape.
type MemberRef struct {
	Shape        string `json:"shape"`
	Location     string `json:"location"`
	LocationName string `json:"locationName"`
}

// LoadModel parses a slim model and precompiles its patterns. Patterns RE2
// can't compile are skipped (botocore patterns are advisory anyway).
func LoadModel(raw []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	m.patterns = map[string]*regexp.Regexp{}
	for name, s := range m.Shapes {
		if s.Pattern == "" {
			continue
		}
		if re, err := regexp.Compile(`^(?:` + s.Pattern + `)$`); err == nil {
			m.patterns[name] = re
		}
	}
	return &m, nil
}

// Validate checks a decoded JSON value against shapeName and returns the
// first problems found (sorted, at most 10), each as "path: reason". Members
// bound to the URI, query string or headers are skipped; callers check those.
func (m *Model) Validate(shapeName string, value any) []string {
	var problems []string
	m.validate(shapeName, value, "", &problems)
	sort.Strings(problems)
	if len(problems) > 10 {
		problems = problems[:10]
	}
	return problems
}

func (m *Model) validate(shapeName string, value any, path string, problems *[]string) {
	if len(*problems) >= 50 {
		return
	}
	s := m.Shapes[shapeName]
	if s == nil || value == nil {
		return
	}
	add := func(format string, args ...any) {
		name := path
		if name == "" {
			name = "input"
		}
		*problems = append(*problems, name+": "+fmt.Sprintf(format, args...))
	}
	switch s.Type {
	case "structure":
		obj, ok := value.(map[string]any)
		if !ok {
			add("must be an object")
			return
		}
		for _, req := range s.Required {
			if ref, ok := s.Members[req]; ok && ref.Location != "" {
				continue
			}
			if v, ok := obj[req]; !ok || v == nil {
				add("missing required member %s", req)
			}
		}
		for key, v := range obj {
			ref, ok := s.Members[key]
			if !ok {
				add("unknown member %s", key)
				continue
			}
			if ref.Location != "" {
				continue
			}
			m.validate(ref.Shape, v, join(path, key), problems)
		}
	case "list":
		items, ok := value.([]any)
		if !ok {
			add("must be a list")
			return
		}
		m.checkSize(s, float64(len(items)), "items", add)
		if s.Member != nil {
			for i, item := range items {
				m.validate(s.Member.Shape, item, fmt.Sprintf("%s[%d]", path, i), problems)
			}
		}
	case "map":
		obj, ok := value.(map[string]any)
		if !ok {
			add("must be an object")
			return
		}
		m.checkSize(s, float64(len(obj)), "entries", add)
		for key, v := range obj {
			if s.Key != nil {
				m.validate(s.Key.Shape, key, join(path, key)+"(key)", problems)
			}
			if s.Value != nil {
				m.validate(s.Value.Shape, v, join(path, key), problems)
			}
		}
	case "string":
		str, ok := value.(string)
		if !ok {
			add("must be a string")
			return
		}
		m.checkSize(s, float64(utf8.RuneCountInString(str)), "characters", add)
		if len(s.Enum) > 0 && !contains(s.Enum, str) {
			add("must be one of %s", strings.Join(s.Enum, ", "))
		}
		if re := m.patterns[shapeName]; re != nil && !re.MatchString(str) {
			add("does not match the required pattern")
		}
	case "integer", "long":
		n, ok := value.(float64)
		if !ok || n != math.Trunc(n) {
			add("must be an integer")
			return
		}
		m.checkRange(s, n, add)
	case "float", "double":
		n, ok := value.(float64)
		if !ok {
			add("must be a number")
			return
		}
		m.checkRange(s, n, add)
	case "boolean":
		if _, ok := value.(bool); !ok {
			add("must be a boolean")
		}
	case "timestamp":
		switch value.(type) {
		case float64, string:
		default:
			add("must be a timestamp")
		}
	case "blob":
		if _, ok := value.(string); !ok {
			add("must be a base64 string")
		}
	}
}

func (m *Model) checkSize(s *ModelShape, n float64, unit string, add func(string, ...any)) {
	if s.Min != nil && n < *s.Min {
		add("must have at least %v %s", *s.Min, unit)
	}
	if s.Max != nil && n > *s.Max {
		add("must have at most %v %s", *s.Max, unit)
	}
}

func (m *Model) checkRange(s *ModelShape, n float64, add func(string, ...any)) {
	if s.Min != nil && n < *s.Min {
		add("must be at least %v", *s.Min)
	}
	if s.Max != nil && n > *s.Max {
		add("must be at most %v", *s.Max)
	}
}

// Conform returns a copy of value holding only the members shapeName
// defines, recursively (unknown keys are dropped).
func (m *Model) Conform(shapeName string, value any) any {
	s := m.Shapes[shapeName]
	if s == nil || value == nil {
		return value
	}
	switch s.Type {
	case "structure":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		out := map[string]any{}
		for key, v := range obj {
			if ref, ok := s.Members[key]; ok && v != nil {
				out[key] = m.Conform(ref.Shape, v)
			}
		}
		return out
	case "list":
		items, ok := value.([]any)
		if !ok {
			return nil
		}
		out := make([]any, 0, len(items))
		for _, item := range items {
			if s.Member != nil {
				item = m.Conform(s.Member.Shape, item)
			}
			out = append(out, item)
		}
		return out
	case "map":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		out := make(map[string]any, len(obj))
		for key, v := range obj {
			if s.Value != nil {
				v = m.Conform(s.Value.Shape, v)
			}
			out[key] = v
		}
		return out
	}
	return value
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
