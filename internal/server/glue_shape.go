package server

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Glue requests are validated and responses shaped against the embedded
// botocore model (internal/awsmodels/glue/model.json), so every one of the
// ~300 operations checks required members, types, enums and min/max bounds,
// and every response carries exactly the model's members with the right types.

func glueShape(name string) *awsmodels.Shape {
	if name == "" {
		return nil
	}
	return awsmodels.GlueModel().Shapes[name]
}

// glueValidate returns "" or a message describing the first invalid member.
func glueValidate(shapeName string, value any, path string) string {
	sh := glueShape(shapeName)
	if sh == nil || sh.Document {
		return ""
	}
	label := path
	if label == "" {
		label = "input"
	}
	switch sh.Type {
	case "structure":
		m, ok := value.(map[string]any)
		if !ok {
			if value == nil && path == "" {
				m = map[string]any{}
			} else {
				return label + " must be an object"
			}
		}
		for _, req := range sh.Required {
			if v, ok := m[req]; !ok || v == nil {
				return fmt.Sprintf("%s is required", glueJoinPath(path, req))
			}
		}
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			member, known := sh.Members[k]
			if !known || m[k] == nil {
				continue // newer SDKs may send members this model predates
			}
			if msg := glueValidate(member, m[k], glueJoinPath(path, k)); msg != "" {
				return msg
			}
		}
	case "list":
		l, ok := value.([]any)
		if !ok {
			return label + " must be a list"
		}
		if msg := glueBound(sh, float64(len(l)), label+" length"); msg != "" {
			return msg
		}
		for i, item := range l {
			if msg := glueValidate(sh.Member, item, fmt.Sprintf("%s[%d]", label, i)); msg != "" {
				return msg
			}
		}
	case "map":
		mm, ok := value.(map[string]any)
		if !ok {
			return label + " must be a map"
		}
		if msg := glueBound(sh, float64(len(mm)), label+" size"); msg != "" {
			return msg
		}
		keys := make([]string, 0, len(mm))
		for k := range mm {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if msg := glueValidate(sh.Key, k, label+" key"); msg != "" {
				return msg
			}
			if msg := glueValidate(sh.Value, mm[k], label+"."+k); msg != "" {
				return msg
			}
		}
	case "string":
		s, ok := value.(string)
		if !ok {
			return label + " must be a string"
		}
		if len(sh.Enum) > 0 && !glueContains(sh.Enum, s) {
			return fmt.Sprintf("%s must be one of %v", label, sh.Enum)
		}
		return glueBound(sh, float64(utf8.RuneCountInString(s)), label+" length")
	case "integer", "long":
		n, ok := value.(float64)
		if !ok || n != math.Trunc(n) {
			return label + " must be an integer"
		}
		return glueBound(sh, n, label)
	case "double", "float":
		n, ok := value.(float64)
		if !ok {
			return label + " must be a number"
		}
		return glueBound(sh, n, label)
	case "boolean":
		if _, ok := value.(bool); !ok {
			return label + " must be a boolean"
		}
	case "blob":
		if _, ok := value.(string); !ok {
			return label + " must be base64 text"
		}
	case "timestamp":
		switch value.(type) {
		case float64, string:
		default:
			return label + " must be a timestamp"
		}
	}
	return ""
}

func glueBound(sh *awsmodels.Shape, n float64, label string) string {
	if sh.Min != nil && n < *sh.Min {
		return fmt.Sprintf("%s must be at least %v", label, *sh.Min)
	}
	if sh.Max != nil && n > *sh.Max {
		return fmt.Sprintf("%s must be at most %v", label, *sh.Max)
	}
	return ""
}

// glueConform shapes value to the named shape: model members only, values
// converted to the member's wire type, and required members filled with a
// zero value when the source lacks them.
func glueConform(shapeName string, value any) any {
	sh := glueShape(shapeName)
	if sh == nil || sh.Document {
		return value
	}
	switch sh.Type {
	case "structure":
		src, _ := value.(map[string]any)
		out := map[string]any{}
		for name, member := range sh.Members {
			if v, ok := src[name]; ok && v != nil {
				if c := glueConform(member, v); c != nil {
					out[name] = c
				}
			}
		}
		for _, req := range sh.Required {
			if _, ok := out[req]; !ok {
				out[req] = glueConform(sh.Members[req], nil)
			}
		}
		return out
	case "list":
		src, ok := value.([]any)
		if !ok {
			if value != nil {
				return nil
			}
			return []any{}
		}
		out := make([]any, 0, len(src))
		for _, item := range src {
			if c := glueConform(sh.Member, item); c != nil {
				out = append(out, c)
			}
		}
		return out
	case "map":
		src, ok := value.(map[string]any)
		if !ok {
			if value != nil {
				return nil
			}
			return map[string]any{}
		}
		out := make(map[string]any, len(src))
		for k, v := range src {
			if c := glueConform(sh.Value, v); c != nil {
				out[k] = c
			}
		}
		return out
	case "string":
		switch v := value.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(v)
		case nil:
			if len(sh.Enum) > 0 {
				return sh.Enum[0]
			}
			return ""
		}
		return nil
	case "integer", "long":
		switch v := value.(type) {
		case float64:
			return math.Trunc(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case nil:
			return float64(0)
		}
		return nil
	case "double", "float":
		switch v := value.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case nil:
			return float64(0)
		}
		return nil
	case "boolean":
		if b, ok := value.(bool); ok {
			return b
		}
		if value == nil {
			return false
		}
		return nil
	case "timestamp":
		switch v := value.(type) {
		case float64:
			return v
		case time.Time:
			return glueEpoch(v)
		case string:
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return glueEpoch(t)
			}
			return nil
		case nil:
			return glueEpoch(time.Now())
		}
		return nil
	case "blob":
		if s, ok := value.(string); ok {
			return s
		}
		if value == nil {
			return ""
		}
		return nil
	}
	return value
}

func glueEpoch(t time.Time) float64 {
	return float64(t.UnixMilli()) / 1000
}

func glueJoinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func glueContains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
