package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type iotSiteWiseStore struct {
	mu   sync.Mutex
	next int64
	tags map[string]map[string]string
}

func newIoTSiteWiseStore() *iotSiteWiseStore {
	return &iotSiteWiseStore{
		next: 1,
		tags: map[string]map[string]string{
			"arn:aws:iotsitewise:us-east-1:123456789012:asset/stackyard-asset": {
				"seed": "true",
			},
		},
	}
}

func (s *iotSiteWiseStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "TagResource":
		arn := iotSiteWiseResolveResourceARN(payload, query)
		if arn == "" {
			arn = "arn:aws:iotsitewise:us-east-1:123456789012:asset/stackyard-asset"
		}
		incoming := iotSiteWiseExtractTagMap(iotSiteWiseValue(payload, "tags"))
		current := s.tags[arn]
		if current == nil {
			current = map[string]string{}
		}
		for k, v := range incoming {
			current[k] = v
		}
		s.tags[arn] = current
		return map[string]any{}

	case "UntagResource":
		arn := iotSiteWiseResolveResourceARN(payload, query)
		if arn == "" {
			arn = "arn:aws:iotsitewise:us-east-1:123456789012:asset/stackyard-asset"
		}
		current := s.tags[arn]
		if current == nil {
			return map[string]any{}
		}
		for _, key := range iotSiteWiseStringSlice(iotSiteWiseValue(payload, "tagKeys")) {
			delete(current, key)
		}
		for _, key := range query["tagKeys"] {
			if strings.TrimSpace(key) != "" {
				delete(current, strings.TrimSpace(key))
			}
		}
		s.tags[arn] = current
		return map[string]any{}

	case "ListTagsForResource":
		arn := iotSiteWiseResolveResourceARN(payload, query)
		if arn == "" {
			arn = "arn:aws:iotsitewise:us-east-1:123456789012:asset/stackyard-asset"
		}
		return map[string]any{"tags": iotSiteWiseCloneStringMap(s.tags[arn])}

	case "CreateAccessPolicy":
		id := iotSiteWiseDefaultString(payload, "accessPolicyId", s.nextID("ap"))
		return map[string]any{
			"accessPolicyId":  id,
			"accessPolicyArn": "arn:aws:iotsitewise:us-east-1:123456789012:access-policy/" + id,
			"accessPolicyStatus": map[string]any{
				"state": "ACTIVE",
			},
		}
	case "CreateAsset":
		id := iotSiteWiseDefaultString(payload, "assetId", s.nextID("asset"))
		return map[string]any{
			"assetId":  id,
			"assetArn": "arn:aws:iotsitewise:us-east-1:123456789012:asset/" + id,
			"assetStatus": map[string]any{
				"state": "ACTIVE",
			},
		}
	case "CreateAssetModel":
		id := iotSiteWiseDefaultString(payload, "assetModelId", s.nextID("asset-model"))
		return map[string]any{
			"assetModelId":  id,
			"assetModelArn": "arn:aws:iotsitewise:us-east-1:123456789012:asset-model/" + id,
			"assetModelStatus": map[string]any{
				"state": "ACTIVE",
			},
		}
	case "CreateAssetModelCompositeModel":
		id := iotSiteWiseDefaultString(payload, "assetModelCompositeModelId", s.nextID("asset-model-composite"))
		return map[string]any{
			"assetModelCompositeModelId": id,
			"assetModelCompositeModelPath": []any{
				map[string]any{"id": iotSiteWisePathParam(pathParams, "assetModelId", "stackyard-asset-model")},
			},
			"assetModelCompositeModelStatus": map[string]any{"state": "ACTIVE"},
		}
	case "CreateBulkImportJob":
		id := iotSiteWiseDefaultString(payload, "jobId", s.nextID("job"))
		return map[string]any{"jobId": id, "jobStatus": "COMPLETED"}
	case "CreateComputationModel":
		id := iotSiteWiseDefaultString(payload, "computationModelId", s.nextID("computation-model"))
		return map[string]any{"computationModelId": id, "computationModelArn": "arn:aws:iotsitewise:us-east-1:123456789012:computation-model/" + id, "computationModelStatus": map[string]any{"state": "ACTIVE"}}
	case "CreateDashboard":
		id := iotSiteWiseDefaultString(payload, "dashboardId", s.nextID("dashboard"))
		return map[string]any{"dashboardId": id, "dashboardArn": "arn:aws:iotsitewise:us-east-1:123456789012:dashboard/" + id}
	case "CreateDataset":
		id := iotSiteWiseDefaultString(payload, "datasetId", s.nextID("dataset"))
		return map[string]any{"datasetId": id, "datasetArn": "arn:aws:iotsitewise:us-east-1:123456789012:dataset/" + id, "datasetStatus": map[string]any{"state": "ACTIVE"}}
	case "CreateGateway":
		id := iotSiteWiseDefaultString(payload, "gatewayId", s.nextID("gateway"))
		return map[string]any{"gatewayId": id, "gatewayArn": "arn:aws:iotsitewise:us-east-1:123456789012:gateway/" + id}
	case "CreatePortal":
		id := iotSiteWiseDefaultString(payload, "portalId", s.nextID("portal"))
		return map[string]any{"portalId": id, "portalArn": "arn:aws:iotsitewise:us-east-1:123456789012:portal/" + id, "portalStartUrl": "https://example.stackyard.local/portals/" + id}
	case "CreateProject":
		id := iotSiteWiseDefaultString(payload, "projectId", s.nextID("project"))
		return map[string]any{"projectId": id, "projectArn": "arn:aws:iotsitewise:us-east-1:123456789012:project/" + id}

	case "BatchAssociateProjectAssets", "BatchDisassociateProjectAssets":
		return map[string]any{"errors": []any{}}
	case "BatchGetAssetPropertyAggregates":
		return map[string]any{"errorEntries": []any{}, "successEntries": []any{}}
	case "BatchGetAssetPropertyValue":
		return map[string]any{"errorEntries": []any{}, "successEntries": []any{}}
	case "BatchGetAssetPropertyValueHistory":
		return map[string]any{"errorEntries": []any{}, "successEntries": []any{}}
	case "BatchPutAssetPropertyValue":
		return map[string]any{"errorEntries": []any{}}

	case "ExecuteAction":
		return map[string]any{"executionId": s.nextID("execution"), "status": "SUCCEEDED", "completionDate": now}
	case "ExecuteQuery":
		return map[string]any{"columns": []any{}, "rows": []any{}, "nextToken": ""}
	case "InvokeAssistant":
		return map[string]any{"conversationId": "stackyard-conversation", "response": map[string]any{"text": "stackyard"}}

	case "GetAssetPropertyAggregates":
		return map[string]any{"aggregatedValues": []any{}, "nextToken": ""}
	case "GetAssetPropertyValue":
		return map[string]any{"propertyValue": map[string]any{"value": map[string]any{"stringValue": "stackyard"}, "timestamp": map[string]any{"timeInSeconds": now.Unix(), "offsetInNanos": 0}, "quality": "GOOD"}}
	case "GetAssetPropertyValueHistory":
		return map[string]any{"assetPropertyValueHistory": []any{}, "nextToken": ""}
	case "GetInterpolatedAssetPropertyValues":
		return map[string]any{"interpolatedAssetPropertyValues": []any{}, "nextToken": ""}

	case "DescribeDefaultEncryptionConfiguration":
		return map[string]any{"encryptionType": "SITEWISE_DEFAULT_ENCRYPTION", "kmsKeyArn": "", "configurationStatus": map[string]any{"state": "ENABLED"}}
	case "DescribeLoggingOptions":
		return map[string]any{"loggingOptions": map[string]any{"level": "ERROR"}}
	case "DescribeStorageConfiguration":
		return map[string]any{"storageType": "SITEWISE_DEFAULT_STORAGE"}
	}

	if strings.HasPrefix(action, "List") {
		key := iotSiteWiseListKey(action)
		if key == "tags" {
			return map[string]any{"tags": map[string]any{}, "nextToken": ""}
		}
		entry := map[string]any{"id": "stackyard", "name": "stackyard", "arn": "arn:aws:iotsitewise:us-east-1:123456789012:stackyard/stackyard"}
		return map[string]any{key: []any{entry}, "nextToken": ""}
	}
	if strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "Get") {
		id := iotSiteWiseResolveEntityID(action, payload, pathParams)
		return map[string]any{
			"operation":      action,
			"id":             id,
			"arn":            iotSiteWiseARNFor(action, id),
			"creationDate":   now,
			"lastUpdateDate": now,
			"status": map[string]any{
				"state": "ACTIVE",
			},
		}
	}
	if strings.HasPrefix(action, "Create") {
		id := iotSiteWiseResolveEntityID(action, payload, pathParams)
		if id == "" {
			id = s.nextID("resource")
		}
		return map[string]any{"id": id, "arn": iotSiteWiseARNFor(action, id), "status": map[string]any{"state": "ACTIVE"}}
	}
	if strings.HasPrefix(action, "Delete") {
		return map[string]any{"operation": action, "state": "DELETING", "requestDate": now}
	}
	if strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Put") || strings.HasPrefix(action, "Associate") || strings.HasPrefix(action, "Disassociate") || strings.HasPrefix(action, "Batch") {
		return map[string]any{"operation": action, "state": "ACTIVE", "lastUpdateDate": now}
	}
	return map[string]any{"operation": action, "status": "SUCCESS", "timestamp": now}
}

func (s *iotSiteWiseStore) nextID(prefix string) string {
	s.next++
	return fmt.Sprintf("%s-%06d", prefix, s.next)
}

func iotSiteWiseListKey(action string) string {
	keys := map[string]string{
		"ListAccessPolicies":                     "accessPolicySummaries",
		"ListActions":                            "actionSummaries",
		"ListAssetModelCompositeModels":          "assetModelCompositeModelSummaries",
		"ListAssetModelProperties":               "assetModelPropertySummaries",
		"ListAssetModels":                        "assetModelSummaries",
		"ListAssetProperties":                    "assetPropertySummaries",
		"ListAssetRelationships":                 "assetRelationshipSummaries",
		"ListAssets":                             "assetSummaries",
		"ListAssociatedAssets":                   "assetSummaries",
		"ListBulkImportJobs":                     "jobSummaries",
		"ListCompositionRelationships":           "compositionRelationshipSummaries",
		"ListComputationModelDataBindingUsages":  "computationModelDataBindingUsageSummaries",
		"ListComputationModelResolveToResources": "resolveToResources",
		"ListComputationModels":                  "computationModelSummaries",
		"ListDashboards":                         "dashboardSummaries",
		"ListDatasets":                           "datasetSummaries",
		"ListExecutions":                         "executionSummaries",
		"ListGateways":                           "gatewaySummaries",
		"ListInterfaceRelationships":             "interfaceRelationshipSummaries",
		"ListPortals":                            "portalSummaries",
		"ListProjectAssets":                      "assetIds",
		"ListProjects":                           "projectSummaries",
		"ListTagsForResource":                    "tags",
		"ListTimeSeries":                         "timeSeriesSummaries",
	}
	if key := keys[action]; key != "" {
		return key
	}
	return "summaries"
}

func iotSiteWiseResolveEntityID(action string, payload map[string]any, pathParams map[string]string) string {
	keys := []string{
		"accessPolicyId",
		"assetId",
		"assetModelId",
		"assetModelCompositeModelId",
		"assetCompositeModelId",
		"propertyId",
		"jobId",
		"computationModelId",
		"dashboardId",
		"datasetId",
		"gatewayId",
		"portalId",
		"projectId",
		"actionId",
		"executionId",
		"interfaceAssetModelId",
		"capabilityNamespace",
	}
	for _, key := range keys {
		if v := iotSiteWisePathParam(pathParams, key, ""); v != "" {
			return v
		}
	}
	for _, key := range keys {
		if v := iotSiteWiseDefaultString(payload, key, ""); v != "" {
			return v
		}
	}
	switch action {
	case "DescribeAccessPolicy", "UpdateAccessPolicy", "DeleteAccessPolicy":
		return "stackyard-access-policy"
	case "DescribeAsset", "UpdateAsset", "DeleteAsset":
		return "stackyard-asset"
	case "DescribeAssetModel", "UpdateAssetModel", "DeleteAssetModel":
		return "stackyard-asset-model"
	case "DescribeComputationModel", "UpdateComputationModel", "DeleteComputationModel":
		return "stackyard-computation-model"
	case "DescribeDashboard", "UpdateDashboard", "DeleteDashboard":
		return "stackyard-dashboard"
	case "DescribeDataset", "UpdateDataset", "DeleteDataset":
		return "stackyard-dataset"
	case "DescribeGateway", "UpdateGateway", "DeleteGateway":
		return "stackyard-gateway"
	case "DescribePortal", "UpdatePortal", "DeletePortal":
		return "stackyard-portal"
	case "DescribeProject", "UpdateProject", "DeleteProject":
		return "stackyard-project"
	case "DescribeExecution":
		return "stackyard-execution"
	case "DescribeAction":
		return "stackyard-action"
	default:
		return "stackyard"
	}
}

func iotSiteWiseARNFor(action, id string) string {
	typeByAction := map[string]string{
		"AccessPolicy":      "access-policy",
		"Asset":             "asset",
		"AssetModel":        "asset-model",
		"ComputationModel":  "computation-model",
		"Dashboard":         "dashboard",
		"Dataset":           "dataset",
		"Gateway":           "gateway",
		"Portal":            "portal",
		"Project":           "project",
		"Execution":         "execution",
		"Action":            "action",
		"BulkImportJob":     "job",
		"TimeSeries":        "timeseries",
		"StorageConfig":     "configuration",
		"EncryptionConfig":  "configuration",
		"LoggingOptions":    "configuration",
		"InterfaceRelation": "asset-model-interface-relationship",
	}
	for marker, resourceType := range typeByAction {
		if strings.Contains(action, marker) {
			return fmt.Sprintf("arn:aws:iotsitewise:us-east-1:123456789012:%s/%s", resourceType, id)
		}
	}
	return fmt.Sprintf("arn:aws:iotsitewise:us-east-1:123456789012:resource/%s", id)
}

func iotSiteWiseResolveResourceARN(payload map[string]any, query url.Values) string {
	if value := strings.TrimSpace(iotSiteWiseDefaultString(payload, "resourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(query.Get("resourceArn")); value != "" {
		return value
	}
	return ""
}

func iotSiteWiseValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return value
		}
	}
	return nil
}

func iotSiteWiseDefaultString(payload map[string]any, key, fallback string) string {
	value := iotSiteWiseValue(payload, key)
	text := strings.TrimSpace(iotSiteWiseToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func iotSiteWiseToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func iotSiteWisePathParam(pathParams map[string]string, key, fallback string) string {
	if pathParams == nil {
		return fallback
	}
	if value, ok := pathParams[key]; ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return fallback
}

func iotSiteWiseExtractTagMap(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(iotSiteWiseToString(val))
		}
	case map[string]string:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
	}
	return out
}

func iotSiteWiseCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = input[key]
	}
	return out
}

func iotSiteWiseStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			text := strings.TrimSpace(iotSiteWiseToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(v)}
	default:
		return nil
	}
}
