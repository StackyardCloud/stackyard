package server

import (
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type appConfigStore struct {
	mu     sync.Mutex
	nextID int64
	tags   map[string]map[string]string

	// The control-plane responses above are fabricated, but AppConfig Data
	// needs real content, so these record what was created and deployed.
	apps        map[string]string                   // application ID -> name
	envs        map[string]map[string]string        // application ID -> environment ID -> name
	profiles    map[string]map[string]string        // application ID -> profile ID -> name
	hosted      map[string]bool                     // appID/profileID of profiles with LocationUri "hosted"
	versions    map[string][]appConfigHostedVersion // appID/profileID -> versions, number = index+1
	deployments map[string][]appConfigDeployment    // appID/envID -> deployments, number = index+1
	sessions    map[string]appConfigDataSession     // configuration token -> session state
}

type appConfigHostedVersion struct {
	Content     []byte
	ContentType string
	Label       string
	Description string
	Deleted     bool // kept as a tombstone so later version numbers don't shift
}

type appConfigDeployment struct {
	ProfileID  string
	Version    string
	RolledBack bool
}

func newAppConfigStore() *appConfigStore {
	// The seed matches what the fabricated control plane already reports:
	// app-000001/env-000001/cp-000001 with deployment 1 of hosted version 1.
	return &appConfigStore{
		nextID: 2, // 1 is the seed below; created IDs must not collide with it
		tags: map[string]map[string]string{
			"arn:aws:appconfig:us-east-1:123456789012:application/app-000001": {
				"seed": "true",
			},
		},
		apps:     map[string]string{"app-000001": "stackyard-app"},
		envs:     map[string]map[string]string{"app-000001": {"env-000001": "stackyard-env"}},
		profiles: map[string]map[string]string{"app-000001": {"cp-000001": "stackyard-config-profile"}},
		hosted:   map[string]bool{"app-000001/cp-000001": true},
		versions: map[string][]appConfigHostedVersion{
			"app-000001/cp-000001": {{Content: []byte(`{"stackyard":true}`), ContentType: "application/json"}},
		},
		deployments: map[string][]appConfigDeployment{
			"app-000001/env-000001": {{ProfileID: "cp-000001", Version: "1"}},
		},
		sessions: map[string]appConfigDataSession{},
	}
}

// appConfigFault is an error the control plane returns instead of a result.
type appConfigFault struct {
	Status        int
	Code, Message string
}

func (s *appConfigStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) (map[string]any, *appConfigFault) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fault := s.validateDataPlaneLocked(action, payload, pathParams); fault != nil {
		return nil, fault
	}
	result := s.handleLocked(action, payload, pathParams, query)
	s.recordDataPlaneLocked(action, payload, pathParams, result)
	return result, nil
}

func (s *appConfigStore) handleLocked(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	now := time.Now().UTC()

	switch action {
	case "CreateApplication", "UpdateApplication", "GetApplication":
		appID := s.resolveOrNewIDLocked(payload, pathParams, "ApplicationId", "app-")
		return map[string]any{
			"Id":          appID,
			"Name":        appConfigDefaultString(payload, "Name", "stackyard-app"),
			"Description": appConfigDefaultString(payload, "Description", "Stackyard AppConfig application"),
		}

	case "ListApplications":
		return map[string]any{
			"Items": []any{
				map[string]any{"Id": "app-000001", "Name": "stackyard-app"},
			},
			"NextToken": "",
		}

	case "DeleteApplication":
		return map[string]any{}

	case "CreateEnvironment", "UpdateEnvironment", "GetEnvironment":
		appID := s.resolveID(payload, pathParams, "ApplicationId", "app-000001")
		envID := s.resolveOrNewIDLocked(payload, pathParams, "EnvironmentId", "env-")
		return map[string]any{
			"ApplicationId": appID,
			"Id":            envID,
			"Name":          appConfigDefaultString(payload, "Name", "stackyard-env"),
			"Description":   appConfigDefaultString(payload, "Description", "Stackyard AppConfig environment"),
			"State":         "READY_FOR_DEPLOYMENT",
		}

	case "ListEnvironments":
		appID := s.resolveID(payload, pathParams, "ApplicationId", "app-000001")
		return map[string]any{
			"Items": []any{
				map[string]any{"ApplicationId": appID, "Id": "env-000001", "Name": "stackyard-env", "State": "READY_FOR_DEPLOYMENT"},
			},
			"NextToken": "",
		}

	case "DeleteEnvironment":
		return map[string]any{}

	case "CreateConfigurationProfile", "UpdateConfigurationProfile", "GetConfigurationProfile":
		appID := s.resolveID(payload, pathParams, "ApplicationId", "app-000001")
		profileID := s.resolveOrNewIDLocked(payload, pathParams, "ConfigurationProfileId", "cp-")
		return map[string]any{
			"ApplicationId": appID,
			"Id":            profileID,
			"Name":          appConfigDefaultString(payload, "Name", "stackyard-config-profile"),
			"LocationUri":   appConfigDefaultString(payload, "LocationUri", "hosted"),
		}

	case "ListConfigurationProfiles":
		appID := s.resolveID(payload, pathParams, "ApplicationId", "app-000001")
		return map[string]any{
			"Items": []any{
				map[string]any{"ApplicationId": appID, "Id": "cp-000001", "Name": "stackyard-config-profile", "LocationUri": "hosted"},
			},
			"NextToken": "",
		}

	case "DeleteConfigurationProfile":
		return map[string]any{}

	case "CreateDeploymentStrategy", "UpdateDeploymentStrategy", "GetDeploymentStrategy":
		strategyID := s.resolveOrNewIDLocked(payload, pathParams, "DeploymentStrategyId", "ds-")
		return map[string]any{
			"Id":                          strategyID,
			"Name":                        appConfigDefaultString(payload, "Name", "stackyard-deploy-strategy"),
			"DeploymentDurationInMinutes": int64(0),
			"FinalBakeTimeInMinutes":      int64(0),
			"GrowthFactor":                float64(25),
			"ReplicateTo":                 "NONE",
		}

	case "ListDeploymentStrategies":
		return map[string]any{
			"Items": []any{
				map[string]any{
					"Id":                          "ds-000001",
					"Name":                        "stackyard-deploy-strategy",
					"DeploymentDurationInMinutes": int64(0),
					"FinalBakeTimeInMinutes":      int64(0),
					"GrowthFactor":                float64(25),
					"ReplicateTo":                 "NONE",
				},
			},
			"NextToken": "",
		}

	case "DeleteDeploymentStrategy":
		return map[string]any{}

	case "CreateHostedConfigurationVersion", "GetHostedConfigurationVersion":
		appID := s.resolveID(payload, pathParams, "ApplicationId", "app-000001")
		profileID := s.resolveID(payload, pathParams, "ConfigurationProfileId", "cp-000001")
		version := s.resolveID(payload, pathParams, "VersionNumber", "1")
		versionNum, _ := strconv.ParseInt(version, 10, 64)
		if versionNum <= 0 {
			versionNum = 1
		}
		return map[string]any{
			"ApplicationId":          appID,
			"ConfigurationProfileId": profileID,
			"VersionNumber":          versionNum,
			"ContentType":            appConfigDefaultString(payload, "ContentType", "application/json"),
		}

	case "ListHostedConfigurationVersions":
		return map[string]any{
			"Items": []any{
				map[string]any{"VersionNumber": int64(1), "ContentType": "application/json", "Description": "seed"},
			},
			"NextToken": "",
		}

	case "DeleteHostedConfigurationVersion":
		return map[string]any{}

	case "StartDeployment", "GetDeployment", "StopDeployment":
		appID := s.resolveID(payload, pathParams, "ApplicationId", "app-000001")
		envID := s.resolveID(payload, pathParams, "EnvironmentId", "env-000001")
		profileID := s.resolveID(payload, pathParams, "ConfigurationProfileId", "cp-000001")
		strategyID := s.resolveID(payload, pathParams, "DeploymentStrategyId", "ds-000001")
		deploymentNumRaw := s.resolveID(payload, pathParams, "DeploymentNumber", "1")
		deploymentNum, _ := strconv.ParseInt(deploymentNumRaw, 10, 64)
		if deploymentNum <= 0 {
			deploymentNum = 1
		}
		state := "DEPLOYING"
		if action == "GetDeployment" {
			state = "COMPLETE"
		}
		if action == "StopDeployment" {
			state = "ROLLED_BACK"
		}
		return map[string]any{
			"ApplicationId":          appID,
			"EnvironmentId":          envID,
			"ConfigurationProfileId": profileID,
			"DeploymentStrategyId":   strategyID,
			"DeploymentNumber":       deploymentNum,
			"State":                  state,
			"StartedAt":              now,
			"CompletedAt":            now,
		}

	case "ListDeployments":
		return map[string]any{
			"Items": []any{
				map[string]any{
					"DeploymentNumber": int64(1),
					"State":            "COMPLETE",
					"StartedAt":        now,
					"CompletedAt":      now,
				},
			},
			"NextToken": "",
		}

	case "CreateExtension", "UpdateExtension", "GetExtension":
		extID := s.resolveOrNewIDLocked(payload, pathParams, "ExtensionIdentifier", "ext-")
		return map[string]any{
			"Id":            extID,
			"Name":          appConfigDefaultString(payload, "Name", "stackyard-extension"),
			"Description":   appConfigDefaultString(payload, "Description", "Stackyard AppConfig extension"),
			"VersionNumber": int64(1),
		}

	case "ListExtensions":
		return map[string]any{
			"Items": []any{
				map[string]any{"Id": "ext-000001", "Name": "stackyard-extension", "VersionNumber": int64(1)},
			},
			"NextToken": "",
		}

	case "DeleteExtension":
		return map[string]any{}

	case "CreateExtensionAssociation", "UpdateExtensionAssociation", "GetExtensionAssociation":
		assocID := s.resolveOrNewIDLocked(payload, pathParams, "ExtensionAssociationId", "exa-")
		return map[string]any{
			"Id":                     assocID,
			"ExtensionArn":           appConfigDefaultString(payload, "ExtensionIdentifier", "ext-000001"),
			"ResourceArn":            appConfigDefaultString(payload, "ResourceIdentifier", "arn:aws:appconfig:us-east-1:123456789012:application/app-000001"),
			"Arn":                    fmt.Sprintf("arn:aws:appconfig:us-east-1:123456789012:extensionassociation/%s", assocID),
			"ExtensionVersionNumber": int64(1),
		}

	case "ListExtensionAssociations":
		return map[string]any{
			"Items": []any{
				map[string]any{
					"Id":           "exa-000001",
					"ExtensionArn": "ext-000001",
					"ResourceArn":  "arn:aws:appconfig:us-east-1:123456789012:application/app-000001",
				},
			},
			"NextToken": "",
		}

	case "DeleteExtensionAssociation":
		return map[string]any{}

	case "GetAccountSettings", "UpdateAccountSettings":
		return map[string]any{
			"DeletionProtection": map[string]any{
				"Enabled": false,
			},
		}

	case "GetConfiguration":
		return map[string]any{
			"ConfigurationVersion": "1",
			"ContentType":          "application/json",
			"Content":              "{\"stackyard\":true}",
		}

	case "ValidateConfiguration":
		return map[string]any{}

	case "TagResource":
		arn := s.resolveID(payload, pathParams, "ResourceArn", "arn:aws:appconfig:us-east-1:123456789012:application/app-000001")
		tagMap := appConfigStringMap(payload["Tags"])
		if len(tagMap) == 0 {
			tagMap = appConfigStringMap(payload["tags"])
		}
		if len(tagMap) == 0 {
			tagMap = map[string]string{"stackyard": "true"}
		}
		if _, ok := s.tags[arn]; !ok {
			s.tags[arn] = map[string]string{}
		}
		for k, v := range tagMap {
			s.tags[arn][k] = v
		}
		return map[string]any{}

	case "UntagResource":
		arn := s.resolveID(payload, pathParams, "ResourceArn", "arn:aws:appconfig:us-east-1:123456789012:application/app-000001")
		keys := appConfigStringSlice(payload["TagKeys"])
		if len(keys) == 0 {
			keys = appConfigStringSlice(payload["tagKeys"])
		}
		if tags, ok := s.tags[arn]; ok {
			for _, k := range keys {
				delete(tags, k)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		arn := s.resolveID(payload, pathParams, "ResourceArn", "arn:aws:appconfig:us-east-1:123456789012:application/app-000001")
		if _, ok := s.tags[arn]; !ok {
			s.tags[arn] = map[string]string{"stackyard": "true"}
		}
		return map[string]any{"Tags": maps.Clone(s.tags[arn])}
	}

	return map[string]any{}
}

func (s *appConfigStore) nextTokenLocked(width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%0*d", width, id)
}

// resolveOrNewIDLocked takes the counter only when no ID was given, so reads
// and updates never consume IDs (replaying only the journaled writes must
// reproduce the same IDs).
func (s *appConfigStore) resolveOrNewIDLocked(payload map[string]any, pathParams map[string]string, key, prefix string) string {
	if id := s.resolveID(payload, pathParams, key, ""); id != "" {
		return id
	}
	return prefix + s.nextTokenLocked(6)
}

func (s *appConfigStore) resolveID(payload map[string]any, pathParams map[string]string, key string, fallback string) string {
	if pathParams != nil {
		if value, ok := pathParams[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if payload != nil {
		for k, v := range payload {
			if strings.EqualFold(k, key) {
				if value := strings.TrimSpace(fmt.Sprintf("%v", v)); value != "" {
					return value
				}
			}
		}
	}
	return fallback
}

func appConfigDefaultString(payload map[string]any, key string, fallback string) string {
	if payload != nil {
		for k, v := range payload {
			if strings.EqualFold(k, key) {
				if value := strings.TrimSpace(fmt.Sprintf("%v", v)); value != "" {
					return value
				}
			}
		}
	}
	return fallback
}

func appConfigStringMap(v any) map[string]string {
	out := map[string]string{}
	switch x := v.(type) {
	case map[string]string:
		for k, v := range x {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	case map[string]any:
		for k, v := range x {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return out
}

func appConfigStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case []string:
		out := make([]string, 0, len(x))
		for _, item := range x {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			value := strings.TrimSpace(fmt.Sprintf("%v", item))
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}
