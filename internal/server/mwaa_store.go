package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	mwaaDefaultRegion    = "us-east-1"
	mwaaDefaultAccountID = "123456789012"
)

type mwaaStore struct {
	mu sync.Mutex

	nextEnvironmentID int64
	environments      map[string]*mwaaEnvironment
	tags              map[string]map[string]string
}

type mwaaEnvironment struct {
	Name             string
	ARN              string
	Status           string
	CreatedAt        string
	UpdatedAt        string
	AirflowVersion   string
	EnvironmentClass string
	ExecutionRoleArn string
	KmsKey           string
	SourceBucketArn  string
	DagS3Path        string
	WebserverURL     string
	MinWorkers       int
	MaxWorkers       int
}

func newMWAAStore() *mwaaStore {
	now := time.Now().UTC().Format(time.RFC3339)
	seed := &mwaaEnvironment{
		Name:             "stackyard-environment",
		ARN:              mwaaEnvironmentARN("stackyard-environment"),
		Status:           "AVAILABLE",
		CreatedAt:        now,
		UpdatedAt:        now,
		AirflowVersion:   "2.10.3",
		EnvironmentClass: "mw1.small",
		ExecutionRoleArn: "arn:aws:iam::123456789012:role/stackyard-mwaa-role",
		KmsKey:           "arn:aws:kms:us-east-1:123456789012:key/stackyard-mwaa",
		SourceBucketArn:  "arn:aws:s3:::stackyard-mwaa",
		DagS3Path:        "dags",
		WebserverURL:     "https://stackyard-environment.airflow.stackyard.local",
		MinWorkers:       1,
		MaxWorkers:       5,
	}

	return &mwaaStore{
		nextEnvironmentID: 2,
		environments: map[string]*mwaaEnvironment{
			seed.Name: seed,
		},
		tags: map[string]map[string]string{
			seed.ARN: {"stackyard": "true", "service": "mwaa", "env": "coverage"},
		},
	}
}

func (s *mwaaStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := mwaaMergeMaps(payload, pathParams, query)
	environmentName := mwaaStringAny(ctx, []string{"Name", "name", "EnvironmentName", "environmentName"}, "stackyard-environment")
	env := s.ensureEnvironmentLocked(environmentName)
	resourceARN := mwaaStringAny(ctx, []string{"ResourceArn", "resourceArn"}, env.ARN)

	s.ensureTagSetLocked(env.ARN)
	s.ensureTagSetLocked(resourceARN)

	s.applyPayloadMutationsLocked(action, payload, env)

	switch action {
	case "CreateEnvironment":
		return map[string]any{"Arn": env.ARN}
	case "GetEnvironment":
		return map[string]any{"Environment": s.environmentPayload(env)}
	case "ListEnvironments":
		items := make([]any, 0, len(s.environments))
		for _, name := range s.sortedEnvironmentNamesLocked() {
			current := s.environments[name]
			items = append(items, map[string]any{
				"Name":   current.Name,
				"Arn":    current.ARN,
				"Status": current.Status,
			})
		}
		return map[string]any{"Environments": items, "NextToken": ""}
	case "UpdateEnvironment":
		return map[string]any{"Arn": env.ARN}
	case "DeleteEnvironment":
		env.Status = "DELETED"
		env.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}
	case "CreateCliToken":
		return map[string]any{
			"CliToken":          "cli-token-" + env.Name,
			"WebServerHostname": mwaaWebserverHostname(env.Name),
		}
	case "CreateWebLoginToken":
		return map[string]any{
			"WebToken":          "web-token-" + env.Name,
			"WebServerHostname": mwaaWebserverHostname(env.Name),
			"IamIdentity":       "arn:aws:sts::123456789012:assumed-role/stackyard-mwaa-user/stackyard",
		}
	case "InvokeRestApi":
		method := strings.ToUpper(mwaaStringAny(payload, []string{"Method", "method"}, "GET"))
		path := mwaaStringAny(payload, []string{"Path", "path"}, "/health")
		response := map[string]any{"ok": true, "method": method, "path": path, "environment": env.Name}
		responseJSON, _ := json.Marshal(response)
		return map[string]any{
			"RestApiStatusCode": 200,
			"RestApiResponse":   string(responseJSON),
		}
	case "PublishMetrics":
		metricCount := len(mwaaAnyArray(payload, []string{"MetricData", "metricData"}))
		return map[string]any{"AcceptedMetrics": metricCount}
	case "TagResource":
		tagSet := s.ensureTagSetLocked(resourceARN)
		for key, value := range mwaaTagsFromAny(payload["Tags"]) {
			tagSet[key] = value
		}
		for key, value := range mwaaTagsFromAny(payload["tags"]) {
			tagSet[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		tagSet := s.ensureTagSetLocked(resourceARN)
		for _, key := range mwaaTagKeys(ctx, query) {
			delete(tagSet, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"Tags": mwaaCloneStringMap(s.ensureTagSetLocked(resourceARN))}
	default:
		return map[string]any{}
	}
}

func (s *mwaaStore) ensureEnvironmentLocked(name string) *mwaaEnvironment {
	envName := strings.TrimSpace(name)
	if envName == "" {
		envName = "stackyard-environment"
	}
	if existing, ok := s.environments[envName]; ok {
		return existing
	}

	now := time.Now().UTC().Format(time.RFC3339)
	created := &mwaaEnvironment{
		Name:             envName,
		ARN:              mwaaEnvironmentARN(envName),
		Status:           "AVAILABLE",
		CreatedAt:        now,
		UpdatedAt:        now,
		AirflowVersion:   "2.10.3",
		EnvironmentClass: "mw1.small",
		ExecutionRoleArn: "arn:aws:iam::123456789012:role/stackyard-mwaa-role",
		KmsKey:           "arn:aws:kms:us-east-1:123456789012:key/stackyard-mwaa",
		SourceBucketArn:  "arn:aws:s3:::stackyard-mwaa",
		DagS3Path:        "dags",
		WebserverURL:     "https://" + envName + ".airflow.stackyard.local",
		MinWorkers:       1,
		MaxWorkers:       5,
	}
	s.environments[envName] = created
	s.ensureTagSetLocked(created.ARN)
	if strings.HasPrefix(envName, "stackyard-environment-") {
		s.nextEnvironmentID++
	}
	return created
}

func (s *mwaaStore) ensureTagSetLocked(resourceARN string) map[string]string {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		arn = mwaaEnvironmentARN("stackyard-environment")
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{"stackyard": "true", "service": "mwaa"}
	}
	return s.tags[arn]
}

func (s *mwaaStore) sortedEnvironmentNamesLocked() []string {
	keys := make([]string, 0, len(s.environments))
	for key := range s.environments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *mwaaStore) environmentPayload(env *mwaaEnvironment) map[string]any {
	if env == nil {
		env = s.ensureEnvironmentLocked("stackyard-environment")
	}
	return map[string]any{
		"Name":             env.Name,
		"Arn":              env.ARN,
		"Status":           env.Status,
		"CreatedAt":        env.CreatedAt,
		"WebserverUrl":     env.WebserverURL,
		"AirflowVersion":   env.AirflowVersion,
		"EnvironmentClass": env.EnvironmentClass,
		"ExecutionRoleArn": env.ExecutionRoleArn,
		"KmsKey":           env.KmsKey,
		"SourceBucketArn":  env.SourceBucketArn,
		"DagS3Path":        env.DagS3Path,
		"LastUpdate": map[string]any{
			"CreatedAt": env.UpdatedAt,
			"Status":    "SUCCESS",
		},
		"MaxWorkers": env.MaxWorkers,
		"MinWorkers": env.MinWorkers,
	}
}

func (s *mwaaStore) applyPayloadMutationsLocked(action string, payload map[string]any, env *mwaaEnvironment) {
	if env == nil {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if action == "CreateEnvironment" {
		env.Status = "CREATING"
	}
	if action == "UpdateEnvironment" {
		env.Status = "UPDATING"
	}

	if airflowVersion := mwaaStringAny(payload, []string{"AirflowVersion", "airflowVersion"}, ""); airflowVersion != "" {
		env.AirflowVersion = airflowVersion
	}
	if envClass := mwaaStringAny(payload, []string{"EnvironmentClass", "environmentClass"}, ""); envClass != "" {
		env.EnvironmentClass = envClass
	}
	if roleArn := mwaaStringAny(payload, []string{"ExecutionRoleArn", "executionRoleArn"}, ""); roleArn != "" {
		env.ExecutionRoleArn = roleArn
	}
	if kmsKey := mwaaStringAny(payload, []string{"KmsKey", "kmsKey"}, ""); kmsKey != "" {
		env.KmsKey = kmsKey
	}
	if bucketArn := mwaaStringAny(payload, []string{"SourceBucketArn", "sourceBucketArn"}, ""); bucketArn != "" {
		env.SourceBucketArn = bucketArn
	}
	if dagPath := mwaaStringAny(payload, []string{"DagS3Path", "dagS3Path"}, ""); dagPath != "" {
		env.DagS3Path = dagPath
	}
	if minWorkers, ok := mwaaIntAny(payload, []string{"MinWorkers", "minWorkers"}); ok {
		env.MinWorkers = minWorkers
	}
	if maxWorkers, ok := mwaaIntAny(payload, []string{"MaxWorkers", "maxWorkers"}); ok {
		env.MaxWorkers = maxWorkers
	}

	if action == "CreateEnvironment" || action == "UpdateEnvironment" {
		env.Status = "AVAILABLE"
		env.UpdatedAt = now
	}
	if action == "PublishMetrics" || action == "InvokeRestApi" || action == "CreateCliToken" || action == "CreateWebLoginToken" {
		env.UpdatedAt = now
	}
}

func mwaaEnvironmentARN(name string) string {
	return fmt.Sprintf("arn:aws:airflow:%s:%s:environment/%s", mwaaDefaultRegion, mwaaDefaultAccountID, strings.TrimSpace(name))
}

func mwaaWebserverHostname(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "stackyard-environment"
	}
	return base + ".airflow.stackyard.local"
}

func mwaaMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 1 {
			out[key] = values[0]
		} else if len(values) > 1 {
			arr := make([]any, 0, len(values))
			for _, v := range values {
				arr = append(arr, v)
			}
			out[key] = arr
		}
	}
	return out
}

func mwaaStringAny(in map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		if raw, ok := in[key]; ok && raw != nil {
			switch value := raw.(type) {
			case string:
				trimmed := strings.TrimSpace(value)
				if trimmed != "" {
					return trimmed
				}
			case json.Number:
				trimmed := strings.TrimSpace(value.String())
				if trimmed != "" {
					return trimmed
				}
			default:
				trimmed := strings.TrimSpace(fmt.Sprintf("%v", raw))
				if trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return fallback
}

func mwaaIntAny(in map[string]any, keys []string) (int, bool) {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case int:
			return value, true
		case int64:
			return int(value), true
		case float64:
			return int(value), true
		case json.Number:
			parsed, err := value.Int64()
			if err == nil {
				return int(parsed), true
			}
			flt, err := value.Float64()
			if err == nil {
				return int(flt), true
			}
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			parsed, err := strconv.Atoi(trimmed)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func mwaaAnyArray(in map[string]any, keys []string) []any {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case []any:
			return value
		case []map[string]any:
			out := make([]any, 0, len(value))
			for _, item := range value {
				out = append(out, item)
			}
			return out
		}
	}
	return nil
}

func mwaaTagsFromAny(raw any) map[string]string {
	out := map[string]string{}
	value, ok := raw.(map[string]any)
	if !ok || value == nil {
		if typed, typedOK := raw.(map[string]string); typedOK {
			for key, val := range typed {
				k := strings.TrimSpace(key)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(val)
			}
		}
		return out
	}
	for key, val := range value {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprintf("%v", val))
	}
	return out
}

func mwaaTagKeys(ctx map[string]any, query url.Values) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	appendKey := func(key string) {
		k := strings.TrimSpace(key)
		if k == "" {
			return
		}
		if _, exists := seen[k]; exists {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}

	for _, keyName := range []string{"TagKeys", "tagKeys"} {
		raw, ok := ctx[keyName]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case []any:
			for _, item := range value {
				appendKey(fmt.Sprintf("%v", item))
			}
		case []string:
			for _, item := range value {
				appendKey(item)
			}
		case string:
			for _, part := range strings.Split(value, ",") {
				appendKey(part)
			}
		default:
			appendKey(fmt.Sprintf("%v", value))
		}
	}

	for _, value := range query["tagKeys"] {
		for _, part := range strings.Split(value, ",") {
			appendKey(part)
		}
	}

	return out
}

func mwaaCloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
