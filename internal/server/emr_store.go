package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type emrStore struct {
	mu sync.Mutex

	nextID int64
	tags   map[string]map[string]string
}

func newEMRStore() *emrStore {
	return &emrStore{
		nextID: 2,
		tags: map[string]map[string]string{
			"j-0000000000001": {
				"stackyard": "true",
				"service":   "emr",
			},
		},
	}
}

func (s *emrStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	clusterID := emrPayloadString(payload, "ClusterId", "j-0000000000001")
	if clusterID == "" {
		clusterID = "j-0000000000001"
	}

	switch action {
	case "AddTags":
		for _, id := range emrPayloadStringSlice(payload, "ResourceId") {
			tags := s.ensureTagsLocked(id)
			for key, value := range emrPayloadTags(payload) {
				tags[key] = value
			}
		}
		if len(emrPayloadStringSlice(payload, "ResourceId")) == 0 {
			tags := s.ensureTagsLocked(clusterID)
			for key, value := range emrPayloadTags(payload) {
				tags[key] = value
			}
		}
		return map[string]any{}

	case "RemoveTags":
		keys := emrPayloadStringSlice(payload, "TagKeys")
		for _, id := range emrPayloadStringSlice(payload, "ResourceId") {
			tags := s.ensureTagsLocked(id)
			for _, key := range keys {
				delete(tags, key)
			}
		}
		if len(emrPayloadStringSlice(payload, "ResourceId")) == 0 {
			tags := s.ensureTagsLocked(clusterID)
			for _, key := range keys {
				delete(tags, key)
			}
		}
		return map[string]any{}

	case "RunJobFlow":
		id := s.nextTokenLocked("j", 13)
		s.ensureTagsLocked(id)
		return map[string]any{"JobFlowId": id}

	case "StartNotebookExecution":
		return map[string]any{"NotebookExecutionId": s.nextTokenLocked("ex", 12)}

	case "CreatePersistentAppUI":
		return map[string]any{"PersistentAppUIId": s.nextTokenLocked("pui", 10)}

	case "CreateStudio":
		return map[string]any{"StudioId": s.nextTokenLocked("es", 12), "Url": "https://studio.stackyard.local"}

	case "CreateStudioSessionMapping":
		return map[string]any{"IdentityId": emrPayloadString(payload, "IdentityName", "stackyard-user")}

	case "DescribeCluster":
		return map[string]any{"Cluster": s.clusterShape(clusterID, now)}

	case "DescribeStep":
		stepID := emrPayloadString(payload, "StepId", "s-0000000000001")
		return map[string]any{"Step": s.stepShape(stepID, now)}

	case "DescribeStudio":
		studioID := emrPayloadString(payload, "StudioId", "es-0000000000001")
		return map[string]any{"Studio": s.studioShape(studioID, now)}

	case "DescribeSecurityConfiguration":
		name := emrPayloadString(payload, "Name", "stackyard-security-config")
		return map[string]any{
			"Name":                  name,
			"SecurityConfiguration": `{"EncryptionConfiguration":{"EnableAtRestEncryption":true}}`,
			"CreationDateTime":      now.Format(time.RFC3339),
		}

	case "GetAutoTerminationPolicy":
		return map[string]any{"AutoTerminationPolicy": map[string]any{"IdleTimeout": 3600}}

	case "GetBlockPublicAccessConfiguration":
		return map[string]any{
			"BlockPublicAccessConfiguration": map[string]any{
				"BlockPublicSecurityGroupRules":          true,
				"PermittedPublicSecurityGroupRuleRanges": []any{map[string]any{"MinRange": 22, "MaxRange": 22}},
			},
		}

	case "GetClusterSessionCredentials":
		return map[string]any{
			"Credentials": map[string]any{
				"AccessKeyId":     "ASIASTACKYARDEMREXAMPLE",
				"SecretAccessKey": "stackyard-session-secret",
				"SessionToken":    "stackyard-session-token",
				"Expiration":      now.Add(1 * time.Hour).Format(time.RFC3339),
			},
		}

	case "GetManagedScalingPolicy":
		return map[string]any{"ManagedScalingPolicy": map[string]any{"ComputeLimits": map[string]any{"UnitType": "Instances", "MinimumCapacityUnits": 1, "MaximumCapacityUnits": 10}}}

	case "GetOnClusterAppUIPresignedURL", "GetPersistentAppUIPresignedURL":
		return map[string]any{"PresignedURL": "https://stackyard.local/emr/ui"}

	case "GetStudioSessionMapping":
		return map[string]any{"SessionMappingDetail": map[string]any{"IdentityType": "USER", "IdentityName": "stackyard-user", "StudioId": "es-0000000000001"}}

	case "ListClusters", "DescribeJobFlows":
		return map[string]any{"Clusters": []any{s.clusterShape(clusterID, now)}, "Marker": ""}

	case "ListInstanceGroups":
		return map[string]any{"InstanceGroups": []any{map[string]any{"Id": "ig-0000000000001", "Name": "Core", "Market": "ON_DEMAND", "InstanceGroupType": "CORE", "RequestedInstanceCount": 2, "RunningInstanceCount": 2}}, "Marker": ""}

	case "ListInstanceFleets":
		return map[string]any{"InstanceFleets": []any{map[string]any{"Id": "if-0000000000001", "Name": "TaskFleet", "Status": map[string]any{"State": "RUNNING"}}}, "Marker": ""}

	case "ListInstances":
		return map[string]any{"Instances": []any{map[string]any{"Id": "i-0000000000001", "Ec2InstanceId": "i-0stackyard0001", "Status": map[string]any{"State": "RUNNING"}}}, "Marker": ""}

	case "ListSteps":
		return map[string]any{"Steps": []any{s.stepShape("s-0000000000001", now)}, "Marker": ""}

	case "ListSecurityConfigurations":
		return map[string]any{"SecurityConfigurations": []any{map[string]any{"Name": "stackyard-security-config", "CreationDateTime": now.Format(time.RFC3339)}}, "NextToken": ""}

	case "ListStudios":
		return map[string]any{"Studios": []any{s.studioShape("es-0000000000001", now)}, "Marker": ""}

	case "ListStudioSessionMappings":
		return map[string]any{"SessionMappings": []any{map[string]any{"IdentityType": "USER", "IdentityName": "stackyard-user", "StudioId": "es-0000000000001"}}, "Marker": ""}

	case "ListNotebookExecutions":
		return map[string]any{"NotebookExecutions": []any{map[string]any{"NotebookExecutionId": "ex-0000000000001", "Status": "RUNNING", "EditorId": "user-000001", "ExecutionEngine": map[string]any{"Id": clusterID}}}, "Marker": ""}

	case "ListBootstrapActions":
		return map[string]any{"BootstrapActions": []any{map[string]any{"Name": "stackyard-bootstrap", "ScriptPath": "s3://stackyard-emr/bootstrap.sh"}}, "Marker": ""}

	case "ListReleaseLabels":
		return map[string]any{"ReleaseLabels": []any{"emr-7.0.0", "emr-7.1.0"}, "Marker": ""}

	case "ListSupportedInstanceTypes":
		return map[string]any{"SupportedInstanceTypes": []any{"m5.xlarge", "m5.2xlarge", "r5.xlarge"}, "Marker": ""}
	}

	switch {
	case strings.HasPrefix(action, "List"):
		return map[string]any{"Items": []any{map[string]any{"Id": s.nextTokenLocked("item", 10), "Status": "ACTIVE"}}, "Marker": ""}
	case strings.HasPrefix(action, "Describe"), strings.HasPrefix(action, "Get"):
		return map[string]any{"Action": action, "Status": "ACTIVE", "Timestamp": now.Format(time.RFC3339)}
	case strings.HasPrefix(action, "Create"), strings.HasPrefix(action, "Put"), strings.HasPrefix(action, "Set"),
		strings.HasPrefix(action, "Modify"), strings.HasPrefix(action, "Update"), strings.HasPrefix(action, "Add"),
		strings.HasPrefix(action, "Start"), strings.HasPrefix(action, "Run"):
		return map[string]any{"Action": action, "Status": "OK", "RequestId": s.nextTokenLocked("req", 10)}
	case strings.HasPrefix(action, "Stop"), strings.HasPrefix(action, "Delete"), strings.HasPrefix(action, "Remove"),
		strings.HasPrefix(action, "Cancel"), strings.HasPrefix(action, "Terminate"):
		return map[string]any{"Action": action, "Status": "OK"}
	}

	return map[string]any{"Action": action, "Status": "OK"}
}

func (s *emrStore) clusterShape(clusterID string, now time.Time) map[string]any {
	return map[string]any{
		"Id":           clusterID,
		"Name":         "stackyard-emr-cluster",
		"ReleaseLabel": "emr-7.0.0",
		"Status": map[string]any{
			"State": "WAITING",
			"Timeline": map[string]any{
				"CreationDateTime": now.Add(-15 * time.Minute).Format(time.RFC3339),
			},
		},
		"NormalizedInstanceHours": 3,
		"ServiceRole":             "EMR_DefaultRole",
		"Tags":                    emrTagsList(s.ensureTagsLocked(clusterID)),
	}
}

func (s *emrStore) stepShape(stepID string, now time.Time) map[string]any {
	return map[string]any{
		"Id":   stepID,
		"Name": "stackyard-step",
		"Status": map[string]any{
			"State": "COMPLETED",
			"Timeline": map[string]any{
				"CreationDateTime": now.Add(-10 * time.Minute).Format(time.RFC3339),
				"EndDateTime":      now.Add(-2 * time.Minute).Format(time.RFC3339),
			},
		},
	}
}

func (s *emrStore) studioShape(studioID string, now time.Time) map[string]any {
	return map[string]any{
		"StudioId":              studioID,
		"Name":                  "stackyard-emr-studio",
		"Url":                   "https://studio.stackyard.local",
		"CreationTime":          now.Add(-1 * time.Hour).Format(time.RFC3339),
		"DefaultS3Location":     "s3://stackyard-emr/studio/",
		"EngineSecurityGroupId": "sg-0000000000001",
	}
}

func (s *emrStore) ensureTagsLocked(resource string) map[string]string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		resource = "j-0000000000001"
	}
	if tags, ok := s.tags[resource]; ok {
		return tags
	}
	s.tags[resource] = map[string]string{"service": "emr"}
	return s.tags[resource]
}

func (s *emrStore) nextTokenLocked(prefix string, width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%0*d", prefix, width, id)
}

func emrPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func emrPayloadString(payload map[string]any, key, fallback string) string {
	value, ok := emrPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	out := strings.TrimSpace(fmt.Sprintf("%v", value))
	if out == "" {
		return fallback
	}
	return out
}

func emrPayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := emrPayloadValue(payload, key)
	if !ok {
		return nil
	}
	switch raw := value.(type) {
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			s := strings.TrimSpace(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", value))
		if s == "" {
			return nil
		}
		return []string{s}
	}
}

func emrPayloadTags(payload map[string]any) map[string]string {
	value, ok := emrPayloadValue(payload, "Tags")
	if !ok {
		return nil
	}

	result := map[string]string{}
	switch raw := value.(type) {
	case map[string]any:
		for key, val := range raw {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			result[k] = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	case []any:
		for _, item := range raw {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(fmt.Sprintf("%v", entry["Key"]))
			if key == "" {
				continue
			}
			result[key] = strings.TrimSpace(fmt.Sprintf("%v", entry["Value"]))
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func emrTagsList(tags map[string]string) []any {
	if tags == nil {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}
