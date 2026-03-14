package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type flinkStore struct {
	mu sync.Mutex

	nextApplicationID int64
	nextSnapshotID    int64
	nextOperationID   int64

	applications map[string]*flinkApplication
	snapshots    map[string]map[string]*flinkSnapshot
	operations   map[string][]map[string]any
	tags         map[string]map[string]string
}

type flinkApplication struct {
	Name               string
	ARN                string
	RuntimeEnvironment string
	Status             string
	VersionID          int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type flinkSnapshot struct {
	ApplicationName string
	SnapshotName    string
	Status          string
	CreatedAt       time.Time
}

func newFlinkStore() *flinkStore {
	s := &flinkStore{
		nextApplicationID: 2,
		nextSnapshotID:    1,
		nextOperationID:   1,
		applications:      map[string]*flinkApplication{},
		snapshots:         map[string]map[string]*flinkSnapshot{},
		operations:        map[string][]map[string]any{},
		tags:              map[string]map[string]string{},
	}
	seed := s.ensureApplicationLocked("stackyard-flink-application")
	s.ensureTagMapLocked(seed.ARN)
	return s
}

func (s *flinkStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	applicationName := flinkPayloadString(payload, "ApplicationName", "stackyard-flink-application")
	application := s.ensureApplicationLocked(applicationName)

	resourceARN := flinkPayloadString(payload, "ResourceARN", application.ARN)
	if resourceARN == "" {
		resourceARN = application.ARN
	}
	s.ensureTagMapLocked(resourceARN)

	switch action {
	case "CreateApplication":
		name := flinkPayloadString(payload, "ApplicationName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-flink-application-%06d", s.nextApplicationID)
			s.nextApplicationID++
		}
		application = s.ensureApplicationLocked(name)
		application.RuntimeEnvironment = flinkPayloadString(payload, "RuntimeEnvironment", application.RuntimeEnvironment)
		if application.RuntimeEnvironment == "" {
			application.RuntimeEnvironment = "FLINK-1_18"
		}
		application.Status = "READY"
		application.UpdatedAt = now
		s.addOperationLocked(application.Name, "CreateApplication", "SUCCEEDED")
		return map[string]any{"ApplicationDetail": s.applicationDetailPayload(application)}

	case "DeleteApplication":
		application.Status = "DELETING"
		application.UpdatedAt = now
		s.addOperationLocked(application.Name, "DeleteApplication", "SUCCEEDED")
		return map[string]any{}

	case "DescribeApplication":
		return map[string]any{"ApplicationDetail": s.applicationDetailPayload(application)}

	case "ListApplications":
		names := s.sortedApplicationNamesLocked()
		summaries := make([]any, 0, len(names))
		for _, name := range names {
			app := s.applications[name]
			summaries = append(summaries, s.applicationSummaryPayload(app))
		}
		return map[string]any{"ApplicationSummaries": summaries, "HasMoreApplications": false}

	case "StartApplication":
		application.Status = "RUNNING"
		application.UpdatedAt = now
		s.addOperationLocked(application.Name, "StartApplication", "SUCCEEDED")
		return map[string]any{}

	case "StopApplication":
		application.Status = "READY"
		application.UpdatedAt = now
		s.addOperationLocked(application.Name, "StopApplication", "SUCCEEDED")
		return map[string]any{}

	case "RollbackApplication":
		if application.VersionID > 1 {
			application.VersionID--
		}
		application.UpdatedAt = now
		s.addOperationLocked(application.Name, "RollbackApplication", "SUCCEEDED")
		return map[string]any{
			"ApplicationDetail": s.applicationDetailPayload(application),
		}

	case "UpdateApplication",
		"AddApplicationCloudWatchLoggingOption",
		"AddApplicationInput",
		"AddApplicationInputProcessingConfiguration",
		"AddApplicationOutput",
		"AddApplicationReferenceDataSource",
		"AddApplicationVpcConfiguration",
		"DeleteApplicationCloudWatchLoggingOption",
		"DeleteApplicationInputProcessingConfiguration",
		"DeleteApplicationOutput",
		"DeleteApplicationReferenceDataSource",
		"DeleteApplicationVpcConfiguration",
		"UpdateApplicationMaintenanceConfiguration":
		application.VersionID++
		application.UpdatedAt = now
		s.addOperationLocked(application.Name, action, "SUCCEEDED")
		return s.applicationMutationPayload(action, application)

	case "DescribeApplicationVersion":
		version := flinkPayloadInt64(payload, "ApplicationVersionId", application.VersionID)
		if version <= 0 {
			version = application.VersionID
		}
		return map[string]any{
			"ApplicationVersionDetail": map[string]any{
				"ApplicationARN":       application.ARN,
				"ApplicationName":      application.Name,
				"ApplicationVersionId": version,
				"ApplicationStatus":    application.Status,
				"RuntimeEnvironment":   application.RuntimeEnvironment,
				"CreateTimestamp":      application.CreatedAt.Format(time.RFC3339),
				"UpdateTimestamp":      application.UpdatedAt.Format(time.RFC3339),
			},
		}

	case "ListApplicationVersions":
		if application.VersionID < 1 {
			application.VersionID = 1
		}
		items := make([]any, 0, application.VersionID)
		for version := int64(1); version <= application.VersionID; version++ {
			items = append(items, map[string]any{
				"ApplicationVersionId": version,
				"ApplicationStatus":    application.Status,
			})
		}
		return map[string]any{"ApplicationVersionSummaries": items, "NextToken": ""}

	case "CreateApplicationSnapshot":
		snapshotName := flinkPayloadString(payload, "SnapshotName", "")
		if snapshotName == "" {
			snapshotName = fmt.Sprintf("snapshot-%06d", s.nextSnapshotID)
			s.nextSnapshotID++
		}
		snapshot := s.ensureSnapshotLocked(application.Name, snapshotName)
		snapshot.Status = "READY"
		snapshot.CreatedAt = now
		s.addOperationLocked(application.Name, "CreateApplicationSnapshot", "SUCCEEDED")
		return map[string]any{}

	case "DescribeApplicationSnapshot":
		snapshot := s.resolveSnapshotLocked(application.Name, flinkPayloadString(payload, "SnapshotName", ""))
		return map[string]any{
			"SnapshotDetails": s.snapshotDetailsPayload(application, snapshot),
		}

	case "ListApplicationSnapshots":
		names := s.sortedSnapshotNamesLocked(application.Name)
		items := make([]any, 0, len(names))
		for _, snapshotName := range names {
			snapshot := s.snapshots[application.Name][snapshotName]
			items = append(items, s.snapshotDetailsPayload(application, snapshot))
		}
		return map[string]any{"SnapshotSummaries": items, "NextToken": ""}

	case "DeleteApplicationSnapshot":
		snapshotName := flinkPayloadString(payload, "SnapshotName", "")
		if snapshotName != "" {
			delete(s.ensureSnapshotMapLocked(application.Name), snapshotName)
		}
		s.addOperationLocked(application.Name, "DeleteApplicationSnapshot", "SUCCEEDED")
		return map[string]any{}

	case "ListApplicationOperations":
		ops := s.operations[application.Name]
		items := make([]any, 0, len(ops))
		for _, op := range ops {
			items = append(items, flinkCloneMap(op))
		}
		return map[string]any{"ApplicationOperationInfoList": items, "NextToken": ""}

	case "DescribeApplicationOperation":
		opID := flinkPayloadString(payload, "OperationId", "")
		op := s.findOperationLocked(application.Name, opID)
		if op == nil {
			op = map[string]any{}
		}
		return map[string]any{"ApplicationOperationInfoDetails": flinkCloneMap(op)}

	case "CreateApplicationPresignedUrl":
		return map[string]any{
			"AuthorizedUrl": fmt.Sprintf("https://stackyard.invalid/flink/%s/notebook?token=%s", application.Name, s.nextTokenLocked("token")),
		}

	case "DiscoverInputSchema":
		return map[string]any{
			"InputSchema": map[string]any{
				"RecordFormat": map[string]any{
					"RecordFormatType": "JSON",
					"MappingParameters": map[string]any{
						"JSONMappingParameters": map[string]any{"RecordRowPath": "$"},
					},
				},
				"RecordColumns": []any{
					map[string]any{"Name": "col1", "SqlType": "VARCHAR(16)", "Mapping": "$.col1"},
				},
			},
		}

	case "TagResource":
		tags := s.ensureTagMapLocked(resourceARN)
		for key, value := range flinkPayloadTags(payload) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagMapLocked(resourceARN)
		for _, key := range flinkPayloadTagKeys(payload, "TagKeys") {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"Tags": flinkTagListPayload(s.ensureTagMapLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *flinkStore) ensureApplicationLocked(name string) *flinkApplication {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-flink-application"
	}
	if existing, ok := s.applications[name]; ok {
		return existing
	}
	now := time.Now().UTC()
	app := &flinkApplication{
		Name:               name,
		ARN:                fmt.Sprintf("arn:aws:kinesisanalytics:us-east-1:123456789012:application/%s", name),
		RuntimeEnvironment: "FLINK-1_18",
		Status:             "READY",
		VersionID:          1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.applications[name] = app
	s.ensureSnapshotMapLocked(name)
	if len(s.operations[name]) == 0 {
		s.operations[name] = []map[string]any{}
	}
	return app
}

func (s *flinkStore) ensureSnapshotMapLocked(applicationName string) map[string]*flinkSnapshot {
	if existing, ok := s.snapshots[applicationName]; ok {
		return existing
	}
	created := map[string]*flinkSnapshot{}
	s.snapshots[applicationName] = created
	return created
}

func (s *flinkStore) ensureSnapshotLocked(applicationName, snapshotName string) *flinkSnapshot {
	snapshotName = strings.TrimSpace(snapshotName)
	if snapshotName == "" {
		snapshotName = fmt.Sprintf("snapshot-%06d", s.nextSnapshotID)
		s.nextSnapshotID++
	}
	entries := s.ensureSnapshotMapLocked(applicationName)
	if existing, ok := entries[snapshotName]; ok {
		return existing
	}
	snapshot := &flinkSnapshot{
		ApplicationName: applicationName,
		SnapshotName:    snapshotName,
		Status:          "READY",
		CreatedAt:       time.Now().UTC(),
	}
	entries[snapshotName] = snapshot
	return snapshot
}

func (s *flinkStore) resolveSnapshotLocked(applicationName, snapshotName string) *flinkSnapshot {
	entries := s.ensureSnapshotMapLocked(applicationName)
	snapshotName = strings.TrimSpace(snapshotName)
	if snapshotName != "" {
		if snapshot, ok := entries[snapshotName]; ok {
			return snapshot
		}
	}
	for _, name := range s.sortedSnapshotNamesLocked(applicationName) {
		return entries[name]
	}
	return s.ensureSnapshotLocked(applicationName, "")
}

func (s *flinkStore) sortedApplicationNamesLocked() []string {
	names := make([]string, 0, len(s.applications))
	for name := range s.applications {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		names = append(names, "stackyard-flink-application")
	}
	return names
}

func (s *flinkStore) sortedSnapshotNamesLocked(applicationName string) []string {
	entries := s.ensureSnapshotMapLocked(applicationName)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *flinkStore) addOperationLocked(applicationName, actionName, status string) map[string]any {
	op := map[string]any{
		"OperationId":     fmt.Sprintf("operation-%06d", s.nextOperationID),
		"Operation":       actionName,
		"OperationStatus": status,
		"StartTime":       time.Now().UTC().Format(time.RFC3339),
		"EndTime":         time.Now().UTC().Format(time.RFC3339),
	}
	s.nextOperationID++
	s.operations[applicationName] = append(s.operations[applicationName], op)
	return op
}

func (s *flinkStore) findOperationLocked(applicationName, operationID string) map[string]any {
	operations := s.operations[applicationName]
	operationID = strings.TrimSpace(operationID)
	if operationID != "" {
		for _, op := range operations {
			if strings.EqualFold(flinkPayloadString(op, "OperationId", ""), operationID) {
				return op
			}
		}
	}
	if len(operations) == 0 {
		return s.addOperationLocked(applicationName, "DescribeApplicationOperation", "SUCCEEDED")
	}
	return operations[len(operations)-1]
}

func (s *flinkStore) applicationSummaryPayload(app *flinkApplication) map[string]any {
	if app == nil {
		return map[string]any{}
	}
	return map[string]any{
		"ApplicationName":      app.Name,
		"ApplicationARN":       app.ARN,
		"ApplicationStatus":    app.Status,
		"ApplicationVersionId": app.VersionID,
		"RuntimeEnvironment":   app.RuntimeEnvironment,
	}
}

func (s *flinkStore) applicationDetailPayload(app *flinkApplication) map[string]any {
	if app == nil {
		return map[string]any{}
	}
	return map[string]any{
		"ApplicationName":      app.Name,
		"ApplicationARN":       app.ARN,
		"ApplicationStatus":    app.Status,
		"ApplicationVersionId": app.VersionID,
		"RuntimeEnvironment":   app.RuntimeEnvironment,
		"CreateTimestamp":      app.CreatedAt.Format(time.RFC3339),
		"LastUpdateTimestamp":  app.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *flinkStore) applicationMutationPayload(action string, app *flinkApplication) map[string]any {
	versionID := app.VersionID
	switch action {
	case "UpdateApplication":
		return map[string]any{
			"ApplicationDetail": s.applicationDetailPayload(app),
		}
	case "UpdateApplicationMaintenanceConfiguration":
		return map[string]any{
			"ApplicationARN": app.ARN,
			"ApplicationMaintenanceConfigurationDescription": map[string]any{
				"ApplicationMaintenanceWindowStartTime": "01:00",
				"ApplicationMaintenanceWindowEndTime":   "02:00",
			},
		}
	case "AddApplicationCloudWatchLoggingOption", "DeleteApplicationCloudWatchLoggingOption":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
			"CloudWatchLoggingOptionDescriptions": []any{
				map[string]any{
					"CloudWatchLoggingOptionId": "cwl-000001",
					"LogStreamARN":              "arn:aws:logs:us-east-1:123456789012:log-group:/aws/kinesisanalytics/" + app.Name + ":log-stream:stackyard",
				},
			},
		}
	case "AddApplicationInput":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
			"InputDescriptions": []any{
				map[string]any{"InputId": "1"},
			},
		}
	case "AddApplicationInputProcessingConfiguration":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
			"InputId":              "1",
			"InputProcessingConfigurationDescription": map[string]any{
				"InputLambdaProcessorDescription": map[string]any{
					"ResourceARN": "arn:aws:lambda:us-east-1:123456789012:function:" + app.Name + "-processor",
				},
			},
		}
	case "DeleteApplicationInputProcessingConfiguration":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
		}
	case "AddApplicationOutput":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
			"OutputDescriptions": []any{
				map[string]any{
					"OutputId": "1",
					"DestinationSchema": map[string]any{
						"RecordFormatType": "JSON",
					},
					"KinesisStreamsOutputDescription": map[string]any{
						"ResourceARN": "arn:aws:kinesis:us-east-1:123456789012:stream/" + app.Name,
					},
				},
			},
		}
	case "DeleteApplicationOutput":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
		}
	case "AddApplicationReferenceDataSource":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
			"ReferenceDataSourceDescriptions": []any{
				map[string]any{
					"ReferenceId": "reference-000001",
					"TableName":   "REFERENCE_DATA",
					"S3ReferenceDataSourceDescription": map[string]any{
						"BucketARN": "arn:aws:s3:::stackyard-flink",
						"FileKey":   app.Name + "/reference.json",
					},
				},
			},
		}
	case "DeleteApplicationReferenceDataSource":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
		}
	case "AddApplicationVpcConfiguration":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
			"VpcConfigurationDescription": map[string]any{
				"VpcConfigurationId": "vpc-configuration-000001",
				"VpcId":              "vpc-12345678",
				"SubnetIds":          []any{"subnet-12345678"},
				"SecurityGroupIds":   []any{"sg-12345678"},
			},
		}
	case "DeleteApplicationVpcConfiguration":
		return map[string]any{
			"ApplicationARN":       app.ARN,
			"ApplicationVersionId": versionID,
		}
	default:
		return map[string]any{}
	}
}

func (s *flinkStore) snapshotDetailsPayload(app *flinkApplication, snapshot *flinkSnapshot) map[string]any {
	if snapshot == nil {
		return map[string]any{}
	}
	return map[string]any{
		"SnapshotName":              snapshot.SnapshotName,
		"SnapshotStatus":            snapshot.Status,
		"ApplicationVersionId":      app.VersionID,
		"SnapshotCreationTimestamp": snapshot.CreatedAt.Format(time.RFC3339),
	}
}

func (s *flinkStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = "arn:aws:kinesisanalytics:us-east-1:123456789012:application/stackyard-flink-application"
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	created := map[string]string{"stackyard": "true"}
	s.tags[resourceARN] = created
	return created
}

func (s *flinkStore) nextTokenLocked(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		prefix = "token"
	}
	token := fmt.Sprintf("%s-%06d", prefix, s.nextOperationID)
	s.nextOperationID++
	return token
}

func flinkTagListPayload(tags map[string]string) []any {
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

func flinkPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	value, ok := raw.(string)
	if !ok {
		return def
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func flinkPayloadInt64(payload map[string]any, key string, def int64) int64 {
	if payload == nil {
		return def
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	switch v := raw.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return parsed
		}
	case jsonNumberLike:
		parsed, err := strconv.ParseInt(v.String(), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return def
}

type jsonNumberLike interface {
	String() string
}

func flinkPayloadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload["Tags"]
	if !ok || raw == nil {
		return out
	}

	switch v := raw.(type) {
	case map[string]any:
		for key, value := range v {
			if text, ok := value.(string); ok {
				out[strings.TrimSpace(key)] = strings.TrimSpace(text)
			}
		}
	case []any:
		for _, item := range v {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := flinkPayloadString(entry, "Key", "")
			value := flinkPayloadString(entry, "Value", "")
			if key != "" {
				out[key] = value
			}
		}
	}
	return out
}

func flinkPayloadTagKeys(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	array, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(array))
	for _, item := range array {
		text, ok := item.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func flinkCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
