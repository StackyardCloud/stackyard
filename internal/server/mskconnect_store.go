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
	mskConnectDefaultRegion    = "us-east-1"
	mskConnectDefaultAccountID = "123456789012"

	mskConnectSeedConnectorARN          = "arn:aws:kafkaconnect:us-east-1:123456789012:connector/stackyard-connector/00000000-0000-0000-0000-000000000001-1"
	mskConnectSeedCustomPluginARN       = "arn:aws:kafkaconnect:us-east-1:123456789012:custom-plugin/stackyard-plugin/00000000-0000-0000-0000-000000000002-2"
	mskConnectSeedWorkerConfigARN       = "arn:aws:kafkaconnect:us-east-1:123456789012:worker-configuration/stackyard-worker/00000000-0000-0000-0000-000000000003-3"
	mskConnectSeedConnectorOperationARN = "arn:aws:kafkaconnect:us-east-1:123456789012:connector-operation/stackyard-connector/00000000-0000-0000-0000-000000000004"
)

type mskConnectStore struct {
	mu sync.Mutex

	nextConnectorSerial          int64
	nextCustomPluginSerial       int64
	nextWorkerConfigSerial       int64
	nextConnectorOperationSerial int64

	connectors            map[string]*mskConnectConnector
	customPlugins         map[string]*mskConnectCustomPlugin
	workerConfigs         map[string]*mskConnectWorkerConfiguration
	connectorOperations   map[string]*mskConnectConnectorOperation
	operationsByConnector map[string][]string
	tags                  map[string]map[string]string
}

type mskConnectConnector struct {
	Arn                     string
	Name                    string
	State                   string
	CreationTime            string
	CurrentVersion          string
	KafkaConnectVersion     string
	ServiceExecutionRoleArn string
}

type mskConnectCustomPlugin struct {
	Arn          string
	Name         string
	State        string
	CreationTime string
	Revision     int
}

type mskConnectWorkerConfiguration struct {
	Arn                   string
	Name                  string
	CreationTime          string
	LatestRevision        int
	PropertiesFileContent string
}

type mskConnectConnectorOperation struct {
	Arn           string
	ConnectorArn  string
	OperationType string
	State         string
	CreationTime  string
	EndTime       string
}

func newMSKConnectStore() *mskConnectStore {
	now := time.Now().UTC()
	s := &mskConnectStore{
		nextConnectorSerial:          2,
		nextCustomPluginSerial:       2,
		nextWorkerConfigSerial:       2,
		nextConnectorOperationSerial: 2,
		connectors:                   map[string]*mskConnectConnector{},
		customPlugins:                map[string]*mskConnectCustomPlugin{},
		workerConfigs:                map[string]*mskConnectWorkerConfiguration{},
		connectorOperations:          map[string]*mskConnectConnectorOperation{},
		operationsByConnector:        map[string][]string{},
		tags:                         map[string]map[string]string{},
	}

	connector := s.ensureConnectorLocked(mskConnectSeedConnectorARN, "stackyard-connector", now)
	plugin := s.ensureCustomPluginLocked(mskConnectSeedCustomPluginARN, "stackyard-plugin", now)
	worker := s.ensureWorkerConfigurationLocked(mskConnectSeedWorkerConfigARN, "stackyard-worker", now)
	_ = plugin
	_ = worker
	_ = s.ensureConnectorOperationLocked(mskConnectSeedConnectorOperationARN, connector.Arn, "CREATE_CONNECTOR", "RUNNING", now)

	s.ensureTagSetLocked(connector.Arn)
	s.tags[connector.Arn]["env"] = "coverage"
	s.tags[connector.Arn]["service"] = "mskconnect"
	return s
}

func (s *mskConnectStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := mskConnectMergeMaps(payload, pathParams, query)
	now := time.Now().UTC()

	connectorArn := mskConnectStringAny(ctx, []string{"connectorArn", "ConnectorArn"}, mskConnectSeedConnectorARN)
	customPluginArn := mskConnectStringAny(ctx, []string{"customPluginArn", "CustomPluginArn"}, mskConnectSeedCustomPluginARN)
	workerConfigArn := mskConnectStringAny(ctx, []string{"workerConfigurationArn", "WorkerConfigurationArn"}, mskConnectSeedWorkerConfigARN)
	resourceArn := mskConnectStringAny(ctx, []string{"resourceArn", "ResourceArn"}, connectorArn)
	connectorOpArn := mskConnectStringAny(ctx, []string{"connectorOperationArn", "ConnectorOperationArn"}, mskConnectSeedConnectorOperationARN)
	connectorName := mskConnectStringAny(ctx, []string{"connectorName", "ConnectorName"}, "stackyard-connector")
	namePrefix := mskConnectStringAny(ctx, []string{"namePrefix", "connectorNamePrefix", "NamePrefix", "ConnectorNamePrefix"}, "")

	connector := s.ensureConnectorLocked(connectorArn, connectorName, now)
	plugin := s.ensureCustomPluginLocked(customPluginArn, "stackyard-plugin", now)
	worker := s.ensureWorkerConfigurationLocked(workerConfigArn, "stackyard-worker", now)
	op := s.ensureConnectorOperationLocked(connectorOpArn, connector.Arn, "CREATE_CONNECTOR", "RUNNING", now)

	s.ensureTagSetLocked(resourceArn)

	s.applyMutationsLocked(action, payload, connector, plugin, worker, now)

	switch action {
	case "CreateConnector":
		created := s.createConnectorLocked(payload, now)
		createdOp := s.ensureConnectorOperationLocked("", created.Arn, "CREATE_CONNECTOR", "RUNNING", now)
		return map[string]any{
			"connectorArn":          created.Arn,
			"connectorName":         created.Name,
			"connectorState":        created.State,
			"creationTime":          created.CreationTime,
			"currentVersion":        created.CurrentVersion,
			"connectorOperationArn": createdOp.Arn,
		}
	case "DescribeConnector":
		return s.describeConnectorPayload(connector)
	case "ListConnectors":
		items := make([]any, 0, len(s.connectors))
		for _, arn := range s.sortedConnectorARNsLocked() {
			current := s.connectors[arn]
			if namePrefix != "" && !strings.HasPrefix(strings.ToLower(current.Name), strings.ToLower(namePrefix)) {
				continue
			}
			items = append(items, map[string]any{
				"connectorArn":   current.Arn,
				"connectorName":  current.Name,
				"connectorState": current.State,
				"creationTime":   current.CreationTime,
			})
		}
		return map[string]any{"connectors": items, "nextToken": ""}
	case "UpdateConnector":
		updated := s.ensureConnectorLocked(connectorArn, connectorName, now)
		updated.CurrentVersion = mskConnectNextVersion(updated.CurrentVersion)
		updated.State = "RUNNING"
		updated.CreationTime = updated.CreationTime
		updatedOp := s.ensureConnectorOperationLocked("", updated.Arn, "UPDATE_CONNECTOR", "RUNNING", now)
		return map[string]any{
			"connectorArn":          updated.Arn,
			"connectorState":        updated.State,
			"currentVersion":        updated.CurrentVersion,
			"connectorOperationArn": updatedOp.Arn,
		}
	case "DeleteConnector":
		deleted := s.ensureConnectorLocked(connectorArn, connectorName, now)
		deleted.State = "DELETING"
		deleted.CurrentVersion = mskConnectNextVersion(deleted.CurrentVersion)
		deletedOp := s.ensureConnectorOperationLocked("", deleted.Arn, "DELETE_CONNECTOR", "RUNNING", now)
		return map[string]any{
			"connectorArn":          deleted.Arn,
			"connectorState":        deleted.State,
			"connectorOperationArn": deletedOp.Arn,
		}
	case "ListConnectorOperations":
		ops := make([]any, 0)
		for _, opArn := range s.operationsByConnector[connector.Arn] {
			if current := s.connectorOperations[opArn]; current != nil {
				ops = append(ops, s.connectorOperationPayload(current))
			}
		}
		if len(ops) == 0 {
			seed := s.ensureConnectorOperationLocked("", connector.Arn, "CREATE_CONNECTOR", "RUNNING", now)
			ops = append(ops, s.connectorOperationPayload(seed))
		}
		return map[string]any{"connectorOperations": ops, "nextToken": ""}
	case "DescribeConnectorOperation":
		return s.connectorOperationPayload(op)
	case "CreateCustomPlugin":
		created := s.createCustomPluginLocked(payload, now)
		return map[string]any{
			"customPluginArn":   created.Arn,
			"customPluginState": created.State,
			"name":              created.Name,
			"revision":          created.Revision,
		}
	case "DescribeCustomPlugin":
		return map[string]any{
			"customPluginArn":   plugin.Arn,
			"customPluginState": plugin.State,
			"creationTime":      plugin.CreationTime,
			"name":              plugin.Name,
			"latestRevision": map[string]any{
				"revision": plugin.Revision,
			},
		}
	case "ListCustomPlugins":
		items := make([]any, 0, len(s.customPlugins))
		for _, arn := range s.sortedCustomPluginARNsLocked() {
			current := s.customPlugins[arn]
			if namePrefix != "" && !strings.HasPrefix(strings.ToLower(current.Name), strings.ToLower(namePrefix)) {
				continue
			}
			items = append(items, map[string]any{
				"customPluginArn":   current.Arn,
				"customPluginState": current.State,
				"creationTime":      current.CreationTime,
				"name":              current.Name,
				"latestRevision": map[string]any{
					"revision": current.Revision,
				},
			})
		}
		return map[string]any{"customPlugins": items, "nextToken": ""}
	case "DeleteCustomPlugin":
		plugin.State = "DELETING"
		return map[string]any{
			"customPluginArn":   plugin.Arn,
			"customPluginState": plugin.State,
		}
	case "CreateWorkerConfiguration":
		created := s.createWorkerConfigurationLocked(payload, now)
		return map[string]any{
			"workerConfigurationArn": created.Arn,
			"creationTime":           created.CreationTime,
			"latestRevision": map[string]any{
				"revision": created.LatestRevision,
			},
		}
	case "DescribeWorkerConfiguration":
		return map[string]any{
			"workerConfigurationArn": worker.Arn,
			"creationTime":           worker.CreationTime,
			"name":                   worker.Name,
			"latestRevision": map[string]any{
				"revision": worker.LatestRevision,
			},
		}
	case "ListWorkerConfigurations":
		items := make([]any, 0, len(s.workerConfigs))
		for _, arn := range s.sortedWorkerConfigARNsLocked() {
			current := s.workerConfigs[arn]
			if namePrefix != "" && !strings.HasPrefix(strings.ToLower(current.Name), strings.ToLower(namePrefix)) {
				continue
			}
			items = append(items, map[string]any{
				"workerConfigurationArn": current.Arn,
				"creationTime":           current.CreationTime,
				"name":                   current.Name,
				"latestRevision": map[string]any{
					"revision": current.LatestRevision,
				},
			})
		}
		return map[string]any{"workerConfigurations": items, "nextToken": ""}
	case "DeleteWorkerConfiguration":
		return map[string]any{
			"workerConfigurationArn": worker.Arn,
		}
	case "TagResource":
		tagSet := s.ensureTagSetLocked(resourceArn)
		for key, value := range mskConnectTagsFromAny(payload["tags"]) {
			tagSet[key] = value
		}
		for key, value := range mskConnectTagsFromAny(payload["Tags"]) {
			tagSet[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		tagSet := s.ensureTagSetLocked(resourceArn)
		for _, key := range mskConnectTagKeys(ctx, query) {
			delete(tagSet, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": mskConnectCloneStringMap(s.ensureTagSetLocked(resourceArn))}
	default:
		return map[string]any{}
	}
}

func (s *mskConnectStore) applyMutationsLocked(action string, payload map[string]any, connector *mskConnectConnector, plugin *mskConnectCustomPlugin, worker *mskConnectWorkerConfiguration, now time.Time) {
	if connector != nil {
		if version := mskConnectStringAny(payload, []string{"kafkaConnectVersion", "KafkaConnectVersion"}, ""); version != "" {
			connector.KafkaConnectVersion = version
		}
		if roleArn := mskConnectStringAny(payload, []string{"serviceExecutionRoleArn", "ServiceExecutionRoleArn"}, ""); roleArn != "" {
			connector.ServiceExecutionRoleArn = roleArn
		}
	}
	if plugin != nil {
		if name := mskConnectStringAny(payload, []string{"name", "Name"}, ""); name != "" {
			plugin.Name = name
		}
	}
	if worker != nil {
		if content := mskConnectStringAny(payload, []string{"propertiesFileContent", "PropertiesFileContent"}, ""); content != "" {
			worker.PropertiesFileContent = content
		}
	}
	_ = action
	_ = now
}

func (s *mskConnectStore) createConnectorLocked(payload map[string]any, now time.Time) *mskConnectConnector {
	name := mskConnectStringAny(payload, []string{"connectorName", "ConnectorName"}, "")
	if name == "" {
		name = fmt.Sprintf("stackyard-connector-%06d", s.nextConnectorSerial)
	}
	arn := mskConnectConnectorARN(name, s.nextConnectorSerial)
	s.nextConnectorSerial++
	created := s.ensureConnectorLocked(arn, name, now)
	created.State = "RUNNING"
	created.CurrentVersion = "1"
	if version := mskConnectStringAny(payload, []string{"kafkaConnectVersion", "KafkaConnectVersion"}, ""); version != "" {
		created.KafkaConnectVersion = version
	}
	if roleArn := mskConnectStringAny(payload, []string{"serviceExecutionRoleArn", "ServiceExecutionRoleArn"}, ""); roleArn != "" {
		created.ServiceExecutionRoleArn = roleArn
	}
	s.ensureTagSetLocked(created.Arn)
	return created
}

func (s *mskConnectStore) createCustomPluginLocked(payload map[string]any, now time.Time) *mskConnectCustomPlugin {
	name := mskConnectStringAny(payload, []string{"name", "Name"}, "")
	if name == "" {
		name = fmt.Sprintf("stackyard-plugin-%06d", s.nextCustomPluginSerial)
	}
	arn := mskConnectCustomPluginARN(name, s.nextCustomPluginSerial)
	s.nextCustomPluginSerial++
	created := s.ensureCustomPluginLocked(arn, name, now)
	created.State = "ACTIVE"
	created.Revision = 1
	return created
}

func (s *mskConnectStore) createWorkerConfigurationLocked(payload map[string]any, now time.Time) *mskConnectWorkerConfiguration {
	name := mskConnectStringAny(payload, []string{"name", "Name"}, "")
	if name == "" {
		name = fmt.Sprintf("stackyard-worker-%06d", s.nextWorkerConfigSerial)
	}
	arn := mskConnectWorkerConfigurationARN(name, s.nextWorkerConfigSerial)
	s.nextWorkerConfigSerial++
	created := s.ensureWorkerConfigurationLocked(arn, name, now)
	created.LatestRevision = 1
	if content := mskConnectStringAny(payload, []string{"propertiesFileContent", "PropertiesFileContent"}, ""); content != "" {
		created.PropertiesFileContent = content
	}
	return created
}

func (s *mskConnectStore) ensureConnectorLocked(arn, name string, now time.Time) *mskConnectConnector {
	resolvedARN := strings.TrimSpace(arn)
	resolvedName := strings.TrimSpace(name)
	if resolvedName == "" {
		resolvedName = mskConnectNameFromARN(resolvedARN)
	}
	if resolvedName == "" {
		resolvedName = "stackyard-connector"
	}
	if resolvedARN == "" {
		resolvedARN = mskConnectConnectorARN(resolvedName, s.nextConnectorSerial)
		s.nextConnectorSerial++
	}

	if existing := s.connectors[resolvedARN]; existing != nil {
		if existing.Name == "" {
			existing.Name = resolvedName
		}
		return existing
	}

	created := &mskConnectConnector{
		Arn:                     resolvedARN,
		Name:                    resolvedName,
		State:                   "RUNNING",
		CreationTime:            now.Format(time.RFC3339),
		CurrentVersion:          "1",
		KafkaConnectVersion:     "2.7.1",
		ServiceExecutionRoleArn: "arn:aws:iam::123456789012:role/stackyard-msk-connect",
	}
	s.connectors[resolvedARN] = created
	if s.operationsByConnector[resolvedARN] == nil {
		s.operationsByConnector[resolvedARN] = []string{}
	}
	return created
}

func (s *mskConnectStore) ensureCustomPluginLocked(arn, name string, now time.Time) *mskConnectCustomPlugin {
	resolvedARN := strings.TrimSpace(arn)
	resolvedName := strings.TrimSpace(name)
	if resolvedName == "" {
		resolvedName = mskConnectNameFromARN(resolvedARN)
	}
	if resolvedName == "" {
		resolvedName = "stackyard-plugin"
	}
	if resolvedARN == "" {
		resolvedARN = mskConnectCustomPluginARN(resolvedName, s.nextCustomPluginSerial)
		s.nextCustomPluginSerial++
	}

	if existing := s.customPlugins[resolvedARN]; existing != nil {
		if existing.Name == "" {
			existing.Name = resolvedName
		}
		return existing
	}

	created := &mskConnectCustomPlugin{
		Arn:          resolvedARN,
		Name:         resolvedName,
		State:        "ACTIVE",
		CreationTime: now.Format(time.RFC3339),
		Revision:     1,
	}
	s.customPlugins[resolvedARN] = created
	return created
}

func (s *mskConnectStore) ensureWorkerConfigurationLocked(arn, name string, now time.Time) *mskConnectWorkerConfiguration {
	resolvedARN := strings.TrimSpace(arn)
	resolvedName := strings.TrimSpace(name)
	if resolvedName == "" {
		resolvedName = mskConnectNameFromARN(resolvedARN)
	}
	if resolvedName == "" {
		resolvedName = "stackyard-worker"
	}
	if resolvedARN == "" {
		resolvedARN = mskConnectWorkerConfigurationARN(resolvedName, s.nextWorkerConfigSerial)
		s.nextWorkerConfigSerial++
	}

	if existing := s.workerConfigs[resolvedARN]; existing != nil {
		if existing.Name == "" {
			existing.Name = resolvedName
		}
		return existing
	}

	created := &mskConnectWorkerConfiguration{
		Arn:                   resolvedARN,
		Name:                  resolvedName,
		CreationTime:          now.Format(time.RFC3339),
		LatestRevision:        1,
		PropertiesFileContent: "offset.flush.interval.ms=1000",
	}
	s.workerConfigs[resolvedARN] = created
	return created
}

func (s *mskConnectStore) ensureConnectorOperationLocked(arn, connectorArn, operationType, state string, now time.Time) *mskConnectConnectorOperation {
	resolvedConnectorARN := strings.TrimSpace(connectorArn)
	if resolvedConnectorARN == "" {
		resolvedConnectorARN = mskConnectSeedConnectorARN
	}
	resolvedARN := strings.TrimSpace(arn)
	if resolvedARN == "" {
		resolvedARN = mskConnectConnectorOperationARN(mskConnectNameFromARN(resolvedConnectorARN), s.nextConnectorOperationSerial)
		s.nextConnectorOperationSerial++
	}
	if operationType == "" {
		operationType = "CREATE_CONNECTOR"
	}
	if state == "" {
		state = "RUNNING"
	}

	if existing := s.connectorOperations[resolvedARN]; existing != nil {
		return existing
	}

	created := &mskConnectConnectorOperation{
		Arn:           resolvedARN,
		ConnectorArn:  resolvedConnectorARN,
		OperationType: operationType,
		State:         state,
		CreationTime:  now.Format(time.RFC3339),
		EndTime:       now.Format(time.RFC3339),
	}
	s.connectorOperations[resolvedARN] = created
	s.operationsByConnector[resolvedConnectorARN] = appendUniqueString(s.operationsByConnector[resolvedConnectorARN], resolvedARN)
	return created
}

func (s *mskConnectStore) ensureTagSetLocked(resourceArn string) map[string]string {
	resolved := strings.TrimSpace(resourceArn)
	if resolved == "" {
		resolved = mskConnectSeedConnectorARN
	}
	if s.tags[resolved] == nil {
		s.tags[resolved] = map[string]string{"stackyard": "true", "service": "mskconnect"}
	}
	return s.tags[resolved]
}

func (s *mskConnectStore) describeConnectorPayload(connector *mskConnectConnector) map[string]any {
	if connector == nil {
		return map[string]any{}
	}
	return map[string]any{
		"connectorArn":            connector.Arn,
		"connectorName":           connector.Name,
		"connectorState":          connector.State,
		"creationTime":            connector.CreationTime,
		"currentVersion":          connector.CurrentVersion,
		"kafkaConnectVersion":     connector.KafkaConnectVersion,
		"serviceExecutionRoleArn": connector.ServiceExecutionRoleArn,
	}
}

func (s *mskConnectStore) connectorOperationPayload(op *mskConnectConnectorOperation) map[string]any {
	if op == nil {
		return map[string]any{}
	}
	return map[string]any{
		"connectorOperationArn":   op.Arn,
		"connectorArn":            op.ConnectorArn,
		"operationType":           op.OperationType,
		"connectorOperationState": op.State,
		"creationTime":            op.CreationTime,
		"endTime":                 op.EndTime,
	}
}

func (s *mskConnectStore) sortedConnectorARNsLocked() []string {
	keys := make([]string, 0, len(s.connectors))
	for key := range s.connectors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *mskConnectStore) sortedCustomPluginARNsLocked() []string {
	keys := make([]string, 0, len(s.customPlugins))
	for key := range s.customPlugins {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *mskConnectStore) sortedWorkerConfigARNsLocked() []string {
	keys := make([]string, 0, len(s.workerConfigs))
	for key := range s.workerConfigs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mskConnectConnectorARN(name string, serial int64) string {
	return fmt.Sprintf(
		"arn:aws:kafkaconnect:%s:%s:connector/%s/%012d-1",
		mskConnectDefaultRegion,
		mskConnectDefaultAccountID,
		strings.TrimSpace(name),
		serial,
	)
}

func mskConnectCustomPluginARN(name string, serial int64) string {
	return fmt.Sprintf(
		"arn:aws:kafkaconnect:%s:%s:custom-plugin/%s/%012d-1",
		mskConnectDefaultRegion,
		mskConnectDefaultAccountID,
		strings.TrimSpace(name),
		serial,
	)
}

func mskConnectWorkerConfigurationARN(name string, serial int64) string {
	return fmt.Sprintf(
		"arn:aws:kafkaconnect:%s:%s:worker-configuration/%s/%012d-1",
		mskConnectDefaultRegion,
		mskConnectDefaultAccountID,
		strings.TrimSpace(name),
		serial,
	)
}

func mskConnectConnectorOperationARN(connectorName string, serial int64) string {
	name := strings.TrimSpace(connectorName)
	if name == "" {
		name = "stackyard-connector"
	}
	return fmt.Sprintf(
		"arn:aws:kafkaconnect:%s:%s:connector-operation/%s/%012d",
		mskConnectDefaultRegion,
		mskConnectDefaultAccountID,
		name,
		serial,
	)
}

func mskConnectNameFromARN(arn string) string {
	parts := strings.Split(strings.TrimSpace(arn), ":")
	if len(parts) < 6 {
		return ""
	}
	resource := strings.TrimSpace(parts[5])
	segments := strings.Split(resource, "/")
	if len(segments) < 2 {
		return ""
	}
	return strings.TrimSpace(segments[1])
}

func mskConnectNextVersion(current string) string {
	value := strings.TrimSpace(current)
	if value == "" {
		return "1"
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return "1"
	}
	return strconv.Itoa(parsed + 1)
}

func mskConnectMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
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
			for _, value := range values {
				arr = append(arr, value)
			}
			out[key] = arr
		}
	}
	return out
}

func mskConnectStringAny(in map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		raw, ok := in[key]
		if !ok || raw == nil {
			continue
		}
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
	return fallback
}

func mskConnectTagKeys(ctx map[string]any, query url.Values) []string {
	keys := []string{}
	appendToken := func(raw string) {
		for _, token := range strings.Split(raw, ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				keys = append(keys, token)
			}
		}
	}

	for _, queryValue := range query["tagKeys"] {
		appendToken(queryValue)
	}

	for _, field := range []string{"tagKeys", "TagKeys"} {
		raw, ok := ctx[field]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case string:
			appendToken(value)
		case []string:
			for _, entry := range value {
				appendToken(entry)
			}
		case []any:
			for _, entry := range value {
				appendToken(fmt.Sprintf("%v", entry))
			}
		default:
			appendToken(fmt.Sprintf("%v", value))
		}
	}

	return mskConnectUniqueStrings(keys)
}

func mskConnectTagsFromAny(raw any) map[string]string {
	out := map[string]string{}
	switch value := raw.(type) {
	case map[string]string:
		for key, item := range value {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(item)
		}
	case map[string]any:
		for key, item := range value {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(fmt.Sprintf("%v", item))
		}
	}
	return out
}

func mskConnectCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func mskConnectUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		token := strings.TrimSpace(value)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}
