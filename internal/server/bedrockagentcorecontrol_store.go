package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type bedrockAgentCoreControlStore struct {
	mu sync.Mutex

	nextRuntimeID          int64
	nextRuntimeEndpointID  int64
	nextPolicyGenerationID int64

	tags              map[string]map[string]string
	resourcePolicies  map[string]string
	policyGenerations map[string]map[string]any
}

func newBedrockAgentCoreControlStore() *bedrockAgentCoreControlStore {
	s := &bedrockAgentCoreControlStore{
		nextRuntimeID:          2,
		nextRuntimeEndpointID:  2,
		nextPolicyGenerationID: 2,
		tags:                   map[string]map[string]string{},
		resourcePolicies:       map[string]string{},
		policyGenerations:      map[string]map[string]any{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *bedrockAgentCoreControlStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)
	ctx := bagccMergeMaps(payload, pathParams, query)

	agentRuntimeID := bagccString(ctx, "agentRuntimeId", "runtime-000001")
	agentRuntimeVersion := bagccString(ctx, "agentRuntimeVersion", "v1")
	endpointName := bagccString(ctx, "endpointName", "stackyard-endpoint")
	browserID := bagccString(ctx, "browserId", "browser-000001")
	profileID := bagccString(ctx, "profileId", "profile-000001")
	codeInterpreterID := bagccString(ctx, "codeInterpreterId", "code-000001")
	evaluatorID := bagccString(ctx, "evaluatorId", "evaluator-000001")
	gatewayID := bagccString(ctx, "gatewayIdentifier", "gateway-000001")
	targetID := bagccString(ctx, "targetId", "target-000001")
	memoryID := bagccString(ctx, "memoryId", "memory-000001")
	onlineEvalConfigID := bagccString(ctx, "onlineEvaluationConfigId", "online-eval-000001")
	policyEngineID := bagccString(ctx, "policyEngineId", "policy-engine-000001")
	policyID := bagccString(ctx, "policyId", "policy-000001")
	policyGenerationID := bagccString(ctx, "policyGenerationId", "policy-generation-000001")
	resourceArn := bagccString(ctx, "resourceArn", "arn:aws:bedrock-agentcore:us-east-1:123456789012:resource/stackyard-control")
	workloadIdentityID := bagccString(ctx, "workloadIdentityId", "workload-identity-000001")

	s.ensureTagsLocked(resourceArn)

	switch action {
	case "TagResource":
		tags := bagccMapString(payload["tags"])
		if len(tags) > 0 {
			existing := s.ensureTagsLocked(resourceArn)
			for k, v := range tags {
				existing[k] = v
			}
		}
		return map[string]any{}

	case "UntagResource":
		tagKeys := bagccString(ctx, "tagKeys", "env")
		existing := s.ensureTagsLocked(resourceArn)
		for _, key := range strings.Split(tagKeys, ",") {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			delete(existing, k)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": bagccCloneMapString(s.ensureTagsLocked(resourceArn))}

	case "PutResourcePolicy":
		policy := bagccString(payload, "policy", "{\"Version\":\"2012-10-17\",\"Statement\":[]}")
		s.resourcePolicies[resourceArn] = policy
		return map[string]any{}

	case "GetResourcePolicy":
		policy, ok := s.resourcePolicies[resourceArn]
		if !ok || strings.TrimSpace(policy) == "" {
			policy = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
		}
		return map[string]any{"resourceArn": resourceArn, "policy": policy}

	case "DeleteResourcePolicy":
		delete(s.resourcePolicies, resourceArn)
		return map[string]any{}

	case "StartPolicyGeneration":
		id := fmt.Sprintf("policy-generation-%06d", s.nextPolicyGenerationID)
		s.nextPolicyGenerationID++
		key := policyEngineID + "|" + id
		record := map[string]any{
			"policyGenerationId": id,
			"policyEngineId":     policyEngineID,
			"status":             "IN_PROGRESS",
			"createdAt":          now.Format(time.RFC3339),
		}
		s.policyGenerations[key] = record
		return map[string]any{"policyGeneration": bagccCloneMap(record)}

	case "GetPolicyGeneration":
		key := policyEngineID + "|" + policyGenerationID
		record := s.policyGenerations[key]
		if record == nil {
			record = map[string]any{
				"policyGenerationId": policyGenerationID,
				"policyEngineId":     policyEngineID,
				"status":             "SUCCEEDED",
				"createdAt":          now.Add(-1 * time.Minute).Format(time.RFC3339),
			}
			s.policyGenerations[key] = record
		}
		return map[string]any{"policyGeneration": bagccCloneMap(record)}

	case "ListPolicyGenerations":
		items := make([]any, 0, len(s.policyGenerations))
		for key, record := range s.policyGenerations {
			if !strings.HasPrefix(key, policyEngineID+"|") {
				continue
			}
			items = append(items, bagccCloneMap(record))
		}
		if len(items) == 0 {
			items = append(items, map[string]any{
				"policyGenerationId": "policy-generation-000001",
				"policyEngineId":     policyEngineID,
				"status":             "SUCCEEDED",
			})
		}
		return map[string]any{"policyGenerations": items, "nextToken": ""}

	case "ListPolicyGenerationAssets":
		return map[string]any{
			"policyGenerationAssets": []any{map[string]any{"assetId": "asset-000001", "policyGenerationId": policyGenerationID}},
			"nextToken":              "",
		}

	case "SetTokenVaultCMK":
		return map[string]any{"kmsKeyArn": "arn:aws:kms:us-east-1:123456789012:key/stackyard-token-vault"}

	case "GetTokenVault":
		return map[string]any{"tokenVault": map[string]any{"kmsKeyArn": "arn:aws:kms:us-east-1:123456789012:key/stackyard-token-vault"}}

	case "SynchronizeGatewayTargets":
		return map[string]any{"gatewayIdentifier": gatewayID, "status": "SYNCHRONIZED"}
	}

	if strings.HasPrefix(action, "List") {
		field := bagccListField(action)
		if field == "" {
			return map[string]any{"nextToken": ""}
		}
		item := bagccEntityForAction(action, now, agentRuntimeID, agentRuntimeVersion, endpointName, browserID, profileID, codeInterpreterID, evaluatorID, gatewayID, targetID, memoryID, onlineEvalConfigID, policyEngineID, policyID, workloadIdentityID)
		return map[string]any{field: []any{item}, "nextToken": ""}
	}

	if strings.HasPrefix(action, "Get") {
		field := bagccGetField(action)
		entity := bagccEntityForAction(action, now, agentRuntimeID, agentRuntimeVersion, endpointName, browserID, profileID, codeInterpreterID, evaluatorID, gatewayID, targetID, memoryID, onlineEvalConfigID, policyEngineID, policyID, workloadIdentityID)
		if field == "" {
			return entity
		}
		return map[string]any{field: entity}
	}

	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") {
		if action == "CreateAgentRuntime" {
			agentRuntimeID = fmt.Sprintf("runtime-%06d", s.nextRuntimeID)
			s.nextRuntimeID++
		}
		if action == "CreateAgentRuntimeEndpoint" {
			endpointName = fmt.Sprintf("runtime-endpoint-%06d", s.nextRuntimeEndpointID)
			s.nextRuntimeEndpointID++
		}
		entity := bagccEntityForAction(action, now, agentRuntimeID, agentRuntimeVersion, endpointName, browserID, profileID, codeInterpreterID, evaluatorID, gatewayID, targetID, memoryID, onlineEvalConfigID, policyEngineID, policyID, workloadIdentityID)
		field := bagccMutationField(action)
		if field == "" {
			return entity
		}
		return map[string]any{field: entity}
	}

	if strings.HasPrefix(action, "Delete") {
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *bedrockAgentCoreControlStore) ensureTagsLocked(resourceArn string) map[string]string {
	if tags, ok := s.tags[resourceArn]; ok {
		return tags
	}
	tags := map[string]string{"env": "local", "service": "bedrock-agentcore-control"}
	s.tags[resourceArn] = tags
	return tags
}

func (s *bedrockAgentCoreControlStore) seedLocked(now time.Time) {
	resourceArn := "arn:aws:bedrock-agentcore:us-east-1:123456789012:resource/stackyard-control"
	s.ensureTagsLocked(resourceArn)
	if _, ok := s.resourcePolicies[resourceArn]; !ok {
		s.resourcePolicies[resourceArn] = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
	}
	if len(s.policyGenerations) == 0 {
		s.policyGenerations["policy-engine-000001|policy-generation-000001"] = map[string]any{
			"policyGenerationId": "policy-generation-000001",
			"policyEngineId":     "policy-engine-000001",
			"status":             "SUCCEEDED",
			"createdAt":          now.Add(-5 * time.Minute).Format(time.RFC3339),
		}
	}
}

func bagccListField(action string) string {
	switch action {
	case "ListAgentRuntimeEndpoints":
		return "agentRuntimeEndpoints"
	case "ListAgentRuntimes":
		return "agentRuntimes"
	case "ListAgentRuntimeVersions":
		return "agentRuntimeVersions"
	case "ListApiKeyCredentialProviders":
		return "apiKeyCredentialProviders"
	case "ListBrowserProfiles":
		return "browserProfiles"
	case "ListBrowsers":
		return "browsers"
	case "ListCodeInterpreters":
		return "codeInterpreters"
	case "ListEvaluators":
		return "evaluators"
	case "ListGateways":
		return "gateways"
	case "ListGatewayTargets":
		return "gatewayTargets"
	case "ListMemories":
		return "memories"
	case "ListOauth2CredentialProviders":
		return "oauth2CredentialProviders"
	case "ListOnlineEvaluationConfigs":
		return "onlineEvaluationConfigs"
	case "ListPolicies":
		return "policies"
	case "ListPolicyEngines":
		return "policyEngines"
	case "ListPolicyGenerationAssets":
		return "policyGenerationAssets"
	case "ListPolicyGenerations":
		return "policyGenerations"
	case "ListWorkloadIdentities":
		return "workloadIdentities"
	default:
		return ""
	}
}

func bagccGetField(action string) string {
	switch action {
	case "GetAgentRuntime":
		return "agentRuntime"
	case "GetAgentRuntimeEndpoint":
		return "agentRuntimeEndpoint"
	case "GetApiKeyCredentialProvider":
		return "apiKeyCredentialProvider"
	case "GetBrowser":
		return "browser"
	case "GetBrowserProfile":
		return "browserProfile"
	case "GetCodeInterpreter":
		return "codeInterpreter"
	case "GetEvaluator":
		return "evaluator"
	case "GetGateway":
		return "gateway"
	case "GetGatewayTarget":
		return "gatewayTarget"
	case "GetMemory":
		return "memory"
	case "GetOauth2CredentialProvider":
		return "oauth2CredentialProvider"
	case "GetOnlineEvaluationConfig":
		return "onlineEvaluationConfig"
	case "GetPolicy":
		return "policy"
	case "GetPolicyEngine":
		return "policyEngine"
	case "GetPolicyGeneration":
		return "policyGeneration"
	case "GetTokenVault":
		return "tokenVault"
	case "GetWorkloadIdentity":
		return "workloadIdentity"
	default:
		return ""
	}
}

func bagccMutationField(action string) string {
	switch action {
	case "CreateAgentRuntime", "UpdateAgentRuntime":
		return "agentRuntime"
	case "CreateAgentRuntimeEndpoint", "UpdateAgentRuntimeEndpoint":
		return "agentRuntimeEndpoint"
	case "CreateApiKeyCredentialProvider", "UpdateApiKeyCredentialProvider":
		return "apiKeyCredentialProvider"
	case "CreateBrowser":
		return "browser"
	case "CreateBrowserProfile":
		return "browserProfile"
	case "CreateCodeInterpreter":
		return "codeInterpreter"
	case "CreateEvaluator", "UpdateEvaluator":
		return "evaluator"
	case "CreateGateway", "UpdateGateway":
		return "gateway"
	case "CreateGatewayTarget", "UpdateGatewayTarget":
		return "gatewayTarget"
	case "CreateMemory", "UpdateMemory":
		return "memory"
	case "CreateOauth2CredentialProvider", "UpdateOauth2CredentialProvider":
		return "oauth2CredentialProvider"
	case "CreateOnlineEvaluationConfig", "UpdateOnlineEvaluationConfig":
		return "onlineEvaluationConfig"
	case "CreatePolicy", "UpdatePolicy":
		return "policy"
	case "CreatePolicyEngine", "UpdatePolicyEngine":
		return "policyEngine"
	case "CreateWorkloadIdentity", "UpdateWorkloadIdentity":
		return "workloadIdentity"
	default:
		return ""
	}
}

func bagccEntityForAction(
	action string,
	now time.Time,
	agentRuntimeID, agentRuntimeVersion, endpointName, browserID, profileID, codeInterpreterID,
	evaluatorID, gatewayID, targetID, memoryID, onlineEvalConfigID, policyEngineID, policyID,
	workloadIdentityID string,
) map[string]any {
	createdAt := now.Format(time.RFC3339)

	switch {
	case strings.Contains(action, "AgentRuntimeEndpoint"):
		return map[string]any{
			"agentRuntimeId": agentRuntimeID,
			"endpointName":   endpointName,
			"status":         "ACTIVE",
			"createdAt":      createdAt,
		}
	case strings.Contains(action, "AgentRuntime"):
		return map[string]any{
			"agentRuntimeId":      agentRuntimeID,
			"agentRuntimeVersion": agentRuntimeVersion,
			"status":              "ACTIVE",
			"createdAt":           createdAt,
		}
	case strings.Contains(action, "ApiKeyCredentialProvider"):
		return map[string]any{
			"providerId": "api-key-provider-000001",
			"name":       "stackyard-api-key-provider",
			"status":     "ACTIVE",
		}
	case strings.Contains(action, "Oauth2CredentialProvider"):
		return map[string]any{
			"providerId": "oauth2-provider-000001",
			"name":       "stackyard-oauth2-provider",
			"status":     "ACTIVE",
		}
	case strings.Contains(action, "WorkloadIdentity"):
		return map[string]any{
			"workloadIdentityId": workloadIdentityID,
			"name":               "stackyard-workload-identity",
			"status":             "ACTIVE",
		}
	case strings.Contains(action, "BrowserProfile"):
		return map[string]any{
			"profileId": profileID,
			"name":      "stackyard-browser-profile",
			"status":    "ACTIVE",
		}
	case strings.Contains(action, "Browser"):
		return map[string]any{
			"browserId": browserID,
			"name":      "stackyard-browser",
			"status":    "ACTIVE",
		}
	case strings.Contains(action, "CodeInterpreter"):
		return map[string]any{
			"codeInterpreterId": codeInterpreterID,
			"name":              "stackyard-code-interpreter",
			"status":            "ACTIVE",
		}
	case strings.Contains(action, "Evaluator"):
		return map[string]any{
			"evaluatorId": evaluatorID,
			"name":        "stackyard-evaluator",
			"status":      "ACTIVE",
		}
	case strings.Contains(action, "GatewayTarget"):
		return map[string]any{
			"gatewayIdentifier": gatewayID,
			"targetId":          targetID,
			"status":            "ACTIVE",
		}
	case strings.Contains(action, "Gateway"):
		return map[string]any{
			"gatewayIdentifier": gatewayID,
			"name":              "stackyard-gateway",
			"status":            "ACTIVE",
		}
	case strings.Contains(action, "Memory"):
		return map[string]any{
			"memoryId": memoryID,
			"name":     "stackyard-memory",
			"status":   "ACTIVE",
		}
	case strings.Contains(action, "OnlineEvaluationConfig"):
		return map[string]any{
			"onlineEvaluationConfigId": onlineEvalConfigID,
			"name":                     "stackyard-online-evaluation",
			"status":                   "ACTIVE",
		}
	case strings.Contains(action, "PolicyGeneration"):
		return map[string]any{
			"policyGenerationId": "policy-generation-000001",
			"policyEngineId":     policyEngineID,
			"status":             "SUCCEEDED",
		}
	case strings.Contains(action, "PolicyEngine"):
		return map[string]any{
			"policyEngineId": policyEngineID,
			"name":           "stackyard-policy-engine",
			"status":         "ACTIVE",
		}
	case strings.Contains(action, "Policy"):
		return map[string]any{
			"policyId":       policyID,
			"policyEngineId": policyEngineID,
			"status":         "ACTIVE",
		}
	}

	return map[string]any{"action": action, "status": "SUCCESS"}
}

func bagccMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	for k, v := range pathParams {
		out[k] = v
	}
	for k, values := range query {
		if len(values) > 0 {
			out[k] = values[len(values)-1]
		}
	}
	return out
}

func bagccString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(v))
		if value != "" {
			return value
		}
	}
	return def
}

func bagccMapString(value any) map[string]string {
	out := map[string]string{}
	input, ok := value.(map[string]any)
	if !ok {
		input2, ok2 := value.(map[string]string)
		if !ok2 {
			return out
		}
		for k, v := range input2 {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(v)
		}
		return out
	}
	for k, raw := range input {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprint(raw))
	}
	return out
}

func bagccCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]any, len(keys))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}

func bagccCloneMapString(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}
