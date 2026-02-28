package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	supplyChainDefaultRegion    = "us-east-1"
	supplyChainDefaultAccountID = "123456789012"
)

type supplyChainStore struct {
	mu sync.Mutex

	nextInstanceID int64
	nextEventID    int64
	nextJobID      int64
	nextExecID     int64

	clientTokenToInstanceID map[string]string
	instances               map[string]map[string]any
	namespaces              map[string]map[string]any
	datasets                map[string]map[string]any
	flows                   map[string]map[string]any
	flowExecutions          map[string]map[string]map[string]any
	events                  map[string]map[string]any
	eventsByInstance        map[string][]string
	bomJobs                 map[string]map[string]any
	tags                    map[string]map[string]string
}

func newSupplyChainStore() *supplyChainStore {
	s := &supplyChainStore{
		nextInstanceID:          2,
		nextEventID:             2,
		nextJobID:               2,
		nextExecID:              2,
		clientTokenToInstanceID: map[string]string{},
		instances:               map[string]map[string]any{},
		namespaces:              map[string]map[string]any{},
		datasets:                map[string]map[string]any{},
		flows:                   map[string]map[string]any{},
		flowExecutions:          map[string]map[string]map[string]any{},
		events:                  map[string]map[string]any{},
		eventsByInstance:        map[string][]string{},
		bomJobs:                 map[string]map[string]any{},
		tags:                    map[string]map[string]string{},
	}
	now := time.Now().UTC()
	inst := s.ensureInstanceLocked("scn-instance-000001", now)
	s.ensureNamespaceLocked(inst["instanceId"].(string), "default", now)
	s.ensureDatasetLocked(inst["instanceId"].(string), "default", "orders", now)
	s.ensureFlowLocked(inst["instanceId"].(string), "orders-flow", now)
	s.ensureEventLocked(inst["instanceId"].(string), "event-000001", now)
	s.ensureBOMJobLocked(inst["instanceId"].(string), "bom-000001", now)
	s.ensureTagsLocked(supplyChainInstanceARN(inst["instanceId"].(string)))
	return s
}

func (s *supplyChainStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := supplyChainMergeMaps(payload, pathParams, query)

	instanceID := supplyChainString(ctx, []string{"instanceId", "InstanceId"}, "scn-instance-000001")
	instanceName := supplyChainString(ctx, []string{"instanceName", "instanceNameFilter", "InstanceName"}, "stackyard-supply-chain")
	namespaceName := supplyChainString(ctx, []string{"namespace", "Namespace"}, "default")
	resourceName := supplyChainString(ctx, []string{"name", "Name"}, "resource")
	flowName := supplyChainString(ctx, []string{"flowName", "FlowName", "name", "Name"}, "orders-flow")
	eventID := supplyChainString(ctx, []string{"eventId", "EventId"}, "event-000001")
	executionID := supplyChainString(ctx, []string{"executionId", "ExecutionId"}, "exec-000001")
	jobID := supplyChainString(ctx, []string{"jobId", "JobId"}, "bom-000001")
	resourceARN := supplyChainString(ctx, []string{"resourceArn", "ResourceArn"}, supplyChainInstanceARN(instanceID))

	switch action {
	case "CreateInstance":
		clientToken := supplyChainString(ctx, []string{"clientToken", "ClientToken"}, "")
		if clientToken != "" {
			if existing := strings.TrimSpace(s.clientTokenToInstanceID[clientToken]); existing != "" {
				inst := s.ensureInstanceLocked(existing, now)
				return map[string]any{"instance": supplyChainCloneMap(inst)}
			}
		}
		newID := fmt.Sprintf("scn-instance-%06d", s.nextInstanceID)
		s.nextInstanceID++
		inst := s.ensureInstanceLocked(newID, now)
		if name := supplyChainString(ctx, []string{"instanceName", "InstanceName"}, ""); name != "" {
			inst["instanceName"] = name
			instanceName = name
		} else {
			inst["instanceName"] = instanceName
		}
		if desc := supplyChainString(ctx, []string{"instanceDescription", "InstanceDescription"}, ""); desc != "" {
			inst["instanceDescription"] = desc
		}
		if webDNS := supplyChainString(ctx, []string{"webAppDnsDomain", "WebAppDnsDomain"}, ""); webDNS != "" {
			inst["webAppDnsDomain"] = webDNS
		}
		inst["instanceState"] = "Active"
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		if clientToken != "" {
			s.clientTokenToInstanceID[clientToken] = newID
		}
		s.mergeTagsLocked(supplyChainMapString(payload["tags"]), supplyChainInstanceARN(newID))
		return map[string]any{"instance": supplyChainCloneMap(inst)}

	case "GetInstance":
		return map[string]any{"instance": supplyChainCloneMap(s.ensureInstanceLocked(instanceID, now))}

	case "ListInstances":
		out := []any{}
		stateFilter := strings.ToLower(strings.TrimSpace(supplyChainString(ctx, []string{"instanceStateFilter"}, "")))
		nameFilter := strings.ToLower(strings.TrimSpace(supplyChainString(ctx, []string{"instanceNameFilter"}, "")))
		for _, id := range s.sortedInstanceIDsLocked() {
			inst := s.instances[id]
			if inst == nil {
				continue
			}
			if stateFilter != "" {
				state := strings.ToLower(strings.TrimSpace(supplyChainString(inst, []string{"instanceState"}, "")))
				if state != strings.ToLower(stateFilter) {
					continue
				}
			}
			if nameFilter != "" {
				name := strings.ToLower(strings.TrimSpace(supplyChainString(inst, []string{"instanceName"}, "")))
				if !strings.Contains(name, nameFilter) {
					continue
				}
			}
			out = append(out, supplyChainCloneMap(inst))
		}
		return map[string]any{"instances": out, "nextToken": ""}

	case "UpdateInstance":
		inst := s.ensureInstanceLocked(instanceID, now)
		if name := supplyChainString(ctx, []string{"instanceName", "InstanceName"}, ""); name != "" {
			inst["instanceName"] = name
		}
		if desc := supplyChainString(ctx, []string{"instanceDescription", "InstanceDescription"}, ""); desc != "" {
			inst["instanceDescription"] = desc
		}
		if state := supplyChainString(ctx, []string{"instanceState", "InstanceState"}, ""); state != "" {
			inst["instanceState"] = state
		}
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{"instance": supplyChainCloneMap(inst)}

	case "DeleteInstance":
		inst := s.ensureInstanceLocked(instanceID, now)
		inst["instanceState"] = "Delete"
		inst["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "CreateDataLakeNamespace":
		ns := s.ensureNamespaceLocked(instanceID, resourceName, now)
		ns["status"] = "ACTIVE"
		ns["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{"dataLakeNamespace": supplyChainCloneMap(ns)}

	case "GetDataLakeNamespace":
		return map[string]any{"dataLakeNamespace": supplyChainCloneMap(s.ensureNamespaceLocked(instanceID, resourceName, now))}

	case "ListDataLakeNamespaces":
		out := []any{}
		for _, key := range s.sortedKeysForPrefixLocked(s.namespaces, instanceID+"|") {
			out = append(out, supplyChainCloneMap(s.namespaces[key]))
		}
		return map[string]any{"dataLakeNamespaces": out, "nextToken": ""}

	case "UpdateDataLakeNamespace":
		ns := s.ensureNamespaceLocked(instanceID, resourceName, now)
		ns["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{"dataLakeNamespace": supplyChainCloneMap(ns)}

	case "DeleteDataLakeNamespace":
		ns := s.ensureNamespaceLocked(instanceID, resourceName, now)
		ns["status"] = "DELETED"
		ns["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "CreateDataLakeDataset":
		ds := s.ensureDatasetLocked(instanceID, namespaceName, resourceName, now)
		ds["status"] = "ACTIVE"
		ds["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{"dataLakeDataset": supplyChainCloneMap(ds)}

	case "GetDataLakeDataset":
		return map[string]any{"dataLakeDataset": supplyChainCloneMap(s.ensureDatasetLocked(instanceID, namespaceName, resourceName, now))}

	case "ListDataLakeDatasets":
		out := []any{}
		prefix := instanceID + "|" + namespaceName + "|"
		for _, key := range s.sortedKeysForPrefixLocked(s.datasets, prefix) {
			out = append(out, supplyChainCloneMap(s.datasets[key]))
		}
		return map[string]any{"dataLakeDatasets": out, "nextToken": ""}

	case "UpdateDataLakeDataset":
		ds := s.ensureDatasetLocked(instanceID, namespaceName, resourceName, now)
		ds["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{"dataLakeDataset": supplyChainCloneMap(ds)}

	case "DeleteDataLakeDataset":
		ds := s.ensureDatasetLocked(instanceID, namespaceName, resourceName, now)
		ds["status"] = "DELETED"
		ds["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "CreateDataIntegrationFlow":
		flow := s.ensureFlowLocked(instanceID, resourceName, now)
		flow["status"] = "ACTIVE"
		flow["lastModifiedTime"] = now.Format(time.RFC3339)
		exec := s.newExecutionLocked(instanceID, resourceName, now)
		return map[string]any{
			"dataIntegrationFlow": supplyChainCloneMap(flow),
			"execution":           supplyChainCloneMap(exec),
		}

	case "GetDataIntegrationFlow":
		return map[string]any{"dataIntegrationFlow": supplyChainCloneMap(s.ensureFlowLocked(instanceID, resourceName, now))}

	case "ListDataIntegrationFlows":
		out := []any{}
		for _, key := range s.sortedKeysForPrefixLocked(s.flows, instanceID+"|") {
			out = append(out, supplyChainCloneMap(s.flows[key]))
		}
		return map[string]any{"dataIntegrationFlows": out, "nextToken": ""}

	case "UpdateDataIntegrationFlow":
		flow := s.ensureFlowLocked(instanceID, resourceName, now)
		flow["lastModifiedTime"] = now.Format(time.RFC3339)
		exec := s.newExecutionLocked(instanceID, resourceName, now)
		return map[string]any{
			"dataIntegrationFlow": supplyChainCloneMap(flow),
			"execution":           supplyChainCloneMap(exec),
		}

	case "DeleteDataIntegrationFlow":
		flow := s.ensureFlowLocked(instanceID, resourceName, now)
		flow["status"] = "DELETED"
		flow["lastModifiedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "SendDataIntegrationEvent":
		event := s.newEventLocked(instanceID, now)
		if eventType := supplyChainString(ctx, []string{"eventType", "EventType"}, "DataSetLoad"); eventType != "" {
			event["eventType"] = eventType
		}
		return map[string]any{"dataIntegrationEvent": supplyChainCloneMap(event)}

	case "GetDataIntegrationEvent":
		return map[string]any{"dataIntegrationEvent": supplyChainCloneMap(s.ensureEventLocked(instanceID, eventID, now))}

	case "ListDataIntegrationEvents":
		out := []any{}
		for _, id := range s.eventsByInstance[instanceID] {
			if event := s.events[id]; event != nil {
				out = append(out, supplyChainCloneMap(event))
			}
		}
		if len(out) == 0 {
			out = append(out, supplyChainCloneMap(s.ensureEventLocked(instanceID, eventID, now)))
		}
		return map[string]any{"dataIntegrationEvents": out, "nextToken": ""}

	case "GetDataIntegrationFlowExecution":
		return map[string]any{"dataIntegrationFlowExecution": supplyChainCloneMap(s.ensureExecutionLocked(instanceID, flowName, executionID, now))}

	case "ListDataIntegrationFlowExecutions":
		out := []any{}
		flowKey := supplyChainFlowKey(instanceID, flowName)
		execs := s.flowExecutions[flowKey]
		keys := make([]string, 0, len(execs))
		for id := range execs {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			out = append(out, supplyChainCloneMap(execs[id]))
		}
		if len(out) == 0 {
			out = append(out, supplyChainCloneMap(s.newExecutionLocked(instanceID, flowName, now)))
		}
		return map[string]any{"dataIntegrationFlowExecutions": out, "nextToken": ""}

	case "CreateBillOfMaterialsImportJob":
		job := s.newBOMJobLocked(instanceID, now)
		return map[string]any{"billOfMaterialsImportJob": supplyChainCloneMap(job)}

	case "GetBillOfMaterialsImportJob":
		return map[string]any{"billOfMaterialsImportJob": supplyChainCloneMap(s.ensureBOMJobLocked(instanceID, jobID, now))}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range supplyChainMapString(ctx["tags"]) {
			tags[key] = value
		}
		for key, value := range supplyChainMapString(ctx["Tags"]) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range supplyChainStringSlice(ctx["tagKeys"]) {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(tags, key)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": supplyChainCloneMapString(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *supplyChainStore) ensureInstanceLocked(instanceID string, now time.Time) map[string]any {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	if inst := s.instances[instanceID]; inst != nil {
		return inst
	}
	inst := map[string]any{
		"instanceId":          instanceID,
		"instanceName":        "stackyard-supply-chain",
		"instanceDescription": "stackyard supply chain instance",
		"instanceState":       "Active",
		"awsAccountId":        supplyChainDefaultAccountID,
		"createdTime":         now.Format(time.RFC3339),
		"lastModifiedTime":    now.Format(time.RFC3339),
		"webAppDnsDomain":     "stackyard",
		"instanceArn":         supplyChainInstanceARN(instanceID),
	}
	s.instances[instanceID] = inst
	return inst
}

func (s *supplyChainStore) ensureNamespaceLocked(instanceID, namespace string, now time.Time) map[string]any {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = "default"
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	key := instanceID + "|" + namespace
	if ns := s.namespaces[key]; ns != nil {
		return ns
	}
	ns := map[string]any{
		"instanceId":       instanceID,
		"name":             namespace,
		"status":           "ACTIVE",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
		"namespaceArn":     supplyChainNamespaceARN(instanceID, namespace),
	}
	s.namespaces[key] = ns
	return ns
}

func (s *supplyChainStore) ensureDatasetLocked(instanceID, namespace, name string, now time.Time) map[string]any {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = "default"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "orders"
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	key := instanceID + "|" + namespace + "|" + name
	if ds := s.datasets[key]; ds != nil {
		return ds
	}
	ds := map[string]any{
		"instanceId":       instanceID,
		"namespace":        namespace,
		"name":             name,
		"status":           "ACTIVE",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
		"datasetArn":       supplyChainDatasetARN(instanceID, namespace, name),
	}
	s.datasets[key] = ds
	return ds
}

func (s *supplyChainStore) ensureFlowLocked(instanceID, name string, now time.Time) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "orders-flow"
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	key := supplyChainFlowKey(instanceID, name)
	if flow := s.flows[key]; flow != nil {
		return flow
	}
	flow := map[string]any{
		"instanceId":       instanceID,
		"name":             name,
		"status":           "ACTIVE",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
		"flowArn":          supplyChainFlowARN(instanceID, name),
	}
	s.flows[key] = flow
	return flow
}

func (s *supplyChainStore) ensureExecutionLocked(instanceID, flowName, executionID string, now time.Time) map[string]any {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	flowName = strings.TrimSpace(flowName)
	if flowName == "" {
		flowName = "orders-flow"
	}
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		executionID = "exec-000001"
	}
	flowKey := supplyChainFlowKey(instanceID, flowName)
	if s.flowExecutions[flowKey] == nil {
		s.flowExecutions[flowKey] = map[string]map[string]any{}
	}
	if exec := s.flowExecutions[flowKey][executionID]; exec != nil {
		return exec
	}
	exec := map[string]any{
		"instanceId":       instanceID,
		"flowName":         flowName,
		"executionId":      executionID,
		"status":           "SUCCEEDED",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	s.flowExecutions[flowKey][executionID] = exec
	return exec
}

func (s *supplyChainStore) newExecutionLocked(instanceID, flowName string, now time.Time) map[string]any {
	id := fmt.Sprintf("exec-%06d", s.nextExecID)
	s.nextExecID++
	return s.ensureExecutionLocked(instanceID, flowName, id, now)
}

func (s *supplyChainStore) ensureEventLocked(instanceID, eventID string, now time.Time) map[string]any {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		eventID = "event-000001"
	}
	if event := s.events[eventID]; event != nil {
		return event
	}
	event := map[string]any{
		"instanceId":       instanceID,
		"eventId":          eventID,
		"eventType":        "DataSetLoad",
		"status":           "SUCCEEDED",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	s.events[eventID] = event
	s.eventsByInstance[instanceID] = append(s.eventsByInstance[instanceID], eventID)
	return event
}

func (s *supplyChainStore) newEventLocked(instanceID string, now time.Time) map[string]any {
	id := fmt.Sprintf("event-%06d", s.nextEventID)
	s.nextEventID++
	return s.ensureEventLocked(instanceID, id, now)
}

func (s *supplyChainStore) ensureBOMJobLocked(instanceID, jobID string, now time.Time) map[string]any {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		jobID = "bom-000001"
	}
	key := instanceID + "|" + jobID
	if job := s.bomJobs[key]; job != nil {
		return job
	}
	job := map[string]any{
		"instanceId":       instanceID,
		"jobId":            jobID,
		"status":           "IN_PROGRESS",
		"createdTime":      now.Format(time.RFC3339),
		"lastModifiedTime": now.Format(time.RFC3339),
	}
	s.bomJobs[key] = job
	return job
}

func (s *supplyChainStore) newBOMJobLocked(instanceID string, now time.Time) map[string]any {
	id := fmt.Sprintf("bom-%06d", s.nextJobID)
	s.nextJobID++
	return s.ensureBOMJobLocked(instanceID, id, now)
}

func (s *supplyChainStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = supplyChainInstanceARN("scn-instance-000001")
	}
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	tags := map[string]string{
		"env":     "local",
		"service": "supplychain",
	}
	s.tags[resourceARN] = tags
	return tags
}

func (s *supplyChainStore) mergeTagsLocked(tags map[string]string, resourceARN string) {
	if len(tags) == 0 {
		return
	}
	existing := s.ensureTagsLocked(resourceARN)
	for key, value := range tags {
		existing[key] = value
	}
}

func (s *supplyChainStore) sortedInstanceIDsLocked() []string {
	keys := make([]string, 0, len(s.instances))
	for key := range s.instances {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *supplyChainStore) sortedKeysForPrefixLocked(collection map[string]map[string]any, prefix string) []string {
	keys := []string{}
	for key := range collection {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func supplyChainFlowKey(instanceID, name string) string {
	return strings.TrimSpace(instanceID) + "|" + strings.TrimSpace(name)
}

func supplyChainMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
		out[strings.ToLower(strings.TrimSpace(key))] = value
	}
	for key, values := range query {
		if len(values) == 1 {
			out[key] = values[0]
		} else if len(values) > 1 {
			list := make([]any, 0, len(values))
			for _, value := range values {
				list = append(list, value)
			}
			out[key] = list
		}
	}
	return out
}

func supplyChainString(source map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		for sourceKey, raw := range source {
			if !strings.EqualFold(strings.TrimSpace(sourceKey), strings.TrimSpace(key)) {
				continue
			}
			switch value := raw.(type) {
			case string:
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					return trimmed
				}
			case []any:
				for _, entry := range value {
					if text, ok := entry.(string); ok {
						if trimmed := strings.TrimSpace(text); trimmed != "" {
							return trimmed
						}
					}
				}
			case fmt.Stringer:
				if trimmed := strings.TrimSpace(value.String()); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func supplyChainStringSlice(raw any) []string {
	out := []string{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range out {
			if existing == value {
				return
			}
		}
		out = append(out, value)
	}
	switch value := raw.(type) {
	case string:
		for _, part := range strings.Split(value, ",") {
			appendValue(part)
		}
	case []string:
		for _, part := range value {
			appendValue(part)
		}
	case []any:
		for _, part := range value {
			if text, ok := part.(string); ok {
				appendValue(text)
			}
		}
	}
	return out
}

func supplyChainMapString(raw any) map[string]string {
	out := map[string]string{}
	switch value := raw.(type) {
	case map[string]string:
		for key, entry := range value {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(entry)
		}
	case map[string]any:
		for key, entry := range value {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			text, _ := entry.(string)
			out[key] = strings.TrimSpace(text)
		}
	}
	return out
}

func supplyChainCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = supplyChainCloneAny(value)
	}
	return out
}

func supplyChainCloneAny(raw any) any {
	switch value := raw.(type) {
	case map[string]any:
		return supplyChainCloneMap(value)
	case map[string]string:
		return supplyChainCloneMapString(value)
	case []any:
		out := make([]any, 0, len(value))
		for _, entry := range value {
			out = append(out, supplyChainCloneAny(entry))
		}
		return out
	default:
		return raw
	}
}

func supplyChainCloneMapString(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func supplyChainInstanceARN(instanceID string) string {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	return fmt.Sprintf("arn:aws:scn:%s:%s:instance/%s", supplyChainDefaultRegion, supplyChainDefaultAccountID, instanceID)
}

func supplyChainNamespaceARN(instanceID, namespace string) string {
	instanceID = strings.TrimSpace(instanceID)
	namespace = strings.TrimSpace(namespace)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	if namespace == "" {
		namespace = "default"
	}
	return fmt.Sprintf("arn:aws:scn:%s:%s:instance/%s/namespace/%s", supplyChainDefaultRegion, supplyChainDefaultAccountID, instanceID, namespace)
}

func supplyChainDatasetARN(instanceID, namespace, name string) string {
	instanceID = strings.TrimSpace(instanceID)
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	if namespace == "" {
		namespace = "default"
	}
	if name == "" {
		name = "orders"
	}
	return fmt.Sprintf(
		"arn:aws:scn:%s:%s:instance/%s/namespace/%s/dataset/%s",
		supplyChainDefaultRegion,
		supplyChainDefaultAccountID,
		instanceID,
		namespace,
		name,
	)
}

func supplyChainFlowARN(instanceID, name string) string {
	instanceID = strings.TrimSpace(instanceID)
	name = strings.TrimSpace(name)
	if instanceID == "" {
		instanceID = "scn-instance-000001"
	}
	if name == "" {
		name = "orders-flow"
	}
	return fmt.Sprintf("arn:aws:scn:%s:%s:instance/%s/flow/%s", supplyChainDefaultRegion, supplyChainDefaultAccountID, instanceID, name)
}
