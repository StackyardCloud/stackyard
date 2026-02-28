package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type ssmSAPStore struct {
	mu sync.Mutex

	nextAppID       int64
	nextComponentID int64
	nextDatabaseID  int64
	nextOperationID int64
	nextCheckOpID   int64

	applications                 map[string]map[string]any
	components                   map[string]map[string]any
	databases                    map[string]map[string]any
	operations                   map[string]map[string]any
	operationEvents              map[string][]map[string]any
	configCheckDefinitions       []map[string]any
	configCheckOperations        map[string]map[string]any
	resourcePermissions          map[string]map[string]any
	tags                         map[string]map[string]string
	subCheckResults              map[string][]map[string]any
	subCheckRuleResults          map[string][]map[string]any
	latestConfigurationOperation string
}

func newSSMSAPStore() *ssmSAPStore {
	now := time.Now().UTC()
	applicationID := "app-000001"
	applicationARN := ssmsapApplicationARN(applicationID)
	componentID := "component-000001"
	databaseID := "database-000001"
	operationID := "operation-000001"
	checkOpID := "check-operation-000001"
	permissionID := "perm-000001"

	s := &ssmSAPStore{
		nextAppID:       2,
		nextComponentID: 2,
		nextDatabaseID:  2,
		nextOperationID: 2,
		nextCheckOpID:   2,
		applications: map[string]map[string]any{
			applicationID: {
				"ApplicationId":           applicationID,
				"ApplicationArn":          applicationARN,
				"Type":                    "HANA",
				"Status":                  "ACTIVATED",
				"DiscoveryStatus":         "SUCCESS",
				"Tags":                    map[string]any{"stackyard": "true"},
				"LastUpdated":             now,
				"Credentials":             []any{map[string]any{"DatabaseName": "HDB", "CredentialType": "ADMIN"}},
				"ApplicationVersion":      "2.0",
				"Sid":                     "HDB",
				"ResourceType":            "application",
				"ReplicationStatus":       "HEALTHY",
				"Resilience":              map[string]any{"HsrTier": "PRIMARY"},
				"ConfigurationCheckState": "READY",
			},
		},
		components: map[string]map[string]any{
			componentID: {
				"ComponentId":   componentID,
				"ApplicationId": applicationID,
				"ComponentType": "HANA",
				"Status":        "RUNNING",
				"Hostname":      "hana-primary.stackyard.local",
				"Sid":           "HDB",
				"Tags":          map[string]any{"stackyard": "true"},
				"ResourceType":  "component",
				"LastUpdated":   now,
			},
		},
		databases: map[string]map[string]any{
			databaseID: {
				"DatabaseId":        databaseID,
				"ApplicationId":     applicationID,
				"ComponentId":       componentID,
				"Status":            "AVAILABLE",
				"DatabaseName":      "SYSTEMDB",
				"DatabaseType":      "HANA",
				"ConnectionDetails": map[string]any{"Host": "hana-primary.stackyard.local", "Port": 30013},
				"LastUpdated":       now,
				"ResourceType":      "database",
			},
		},
		operations: map[string]map[string]any{
			operationID: {
				"OperationId":   operationID,
				"Type":          "REGISTER_APPLICATION",
				"Status":        "SUCCEEDED",
				"ApplicationId": applicationID,
				"StartTime":     now.Add(-5 * time.Minute),
				"EndTime":       now.Add(-4 * time.Minute),
			},
		},
		operationEvents: map[string][]map[string]any{
			operationID: {
				{
					"OperationId": operationID,
					"Type":        "INFO",
					"Description": "Application registration completed",
					"Timestamp":   now.Add(-4 * time.Minute),
				},
			},
		},
		configCheckDefinitions: []map[string]any{
			{
				"ConfigurationCheckDefinitionId": "check-def-000001",
				"Name":                           "HANAConnectivity",
				"Description":                    "Validates HANA network and credential readiness",
				"Tags":                           map[string]any{"stackyard": "true"},
			},
		},
		configCheckOperations: map[string]map[string]any{
			checkOpID: {
				"ConfigurationCheckOperationId": checkOpID,
				"Status":                        "SUCCEEDED",
				"ApplicationId":                 applicationID,
				"StartTime":                     now.Add(-3 * time.Minute),
				"EndTime":                       now.Add(-2 * time.Minute),
			},
		},
		resourcePermissions: map[string]map[string]any{
			permissionID: {
				"ResourcePermissionId": permissionID,
				"ActionType":           "RESTORE_DATABASE",
				"ResourceArn":          applicationARN,
				"SourceResourceArn":    "arn:aws:backup:us-east-1:123456789012:recovery-point:stackyard",
				"LastUpdated":          now,
			},
		},
		tags: map[string]map[string]string{
			applicationARN: {"stackyard": "true"},
		},
		subCheckResults: map[string][]map[string]any{
			checkOpID: {
				{
					"Name":        "NetworkReachability",
					"Status":      "SUCCESS",
					"Description": "SAP hosts reachable",
				},
			},
		},
		subCheckRuleResults: map[string][]map[string]any{
			checkOpID: {
				{
					"Name":        "HANA_PORT_OPEN",
					"Status":      "SUCCESS",
					"Severity":    "LOW",
					"Description": "HANA SQL port reachable",
				},
			},
		},
		latestConfigurationOperation: checkOpID,
	}

	return s
}

func (s *ssmSAPStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "DeleteResourcePermission":
		permID := ssmsapStringAny(payload, "ResourcePermissionId", s.firstResourcePermissionIDLocked())
		delete(s.resourcePermissions, permID)
		return map[string]any{}

	case "DeregisterApplication":
		appID := ssmsapStringAny(payload, "ApplicationId", s.firstApplicationIDLocked())
		delete(s.applications, appID)
		for id, comp := range s.components {
			if ssmsapStringAny(comp, "ApplicationId", "") == appID {
				delete(s.components, id)
			}
		}
		for id, db := range s.databases {
			if ssmsapStringAny(db, "ApplicationId", "") == appID {
				delete(s.databases, id)
			}
		}
		return map[string]any{}

	case "GetApplication":
		appID := ssmsapStringAny(payload, "ApplicationId", s.firstApplicationIDLocked())
		app := s.ensureApplicationLocked(appID)
		return map[string]any{"Application": ssmsapCloneMap(app)}

	case "GetComponent":
		componentID := ssmsapStringAny(payload, "ComponentId", s.firstComponentIDLocked())
		comp := s.ensureComponentLocked(componentID, s.firstApplicationIDLocked())
		return map[string]any{"Component": ssmsapCloneMap(comp)}

	case "GetConfigurationCheckOperation":
		checkID := ssmsapStringAny(payload, "ConfigurationCheckOperationId", s.latestConfigurationOperation)
		if checkID == "" {
			checkID = s.firstConfigCheckOperationIDLocked()
		}
		op := s.ensureConfigCheckOperationLocked(checkID, s.firstApplicationIDLocked())
		return map[string]any{"ConfigurationCheckOperation": ssmsapCloneMap(op)}

	case "GetDatabase":
		databaseID := ssmsapStringAny(payload, "DatabaseId", s.firstDatabaseIDLocked())
		db := s.ensureDatabaseLocked(databaseID, s.firstApplicationIDLocked(), s.firstComponentIDLocked())
		return map[string]any{"Database": ssmsapCloneMap(db)}

	case "GetOperation":
		opID := ssmsapStringAny(payload, "OperationId", s.firstOperationIDLocked())
		op := s.ensureOperationLocked(opID, "GENERIC_OPERATION", s.firstApplicationIDLocked(), "SUCCEEDED")
		return map[string]any{"Operation": ssmsapCloneMap(op)}

	case "GetResourcePermission":
		permID := ssmsapStringAny(payload, "ResourcePermissionId", s.firstResourcePermissionIDLocked())
		perm := s.ensureResourcePermissionLocked(permID)
		return map[string]any{"ResourcePermission": ssmsapCloneMap(perm)}

	case "ListApplications":
		ids := make([]string, 0, len(s.applications))
		for id := range s.applications {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, ssmsapApplicationSummary(s.applications[id]))
		}
		return map[string]any{"Applications": out, "NextToken": ""}

	case "ListComponents":
		appID := ssmsapStringAny(payload, "ApplicationId", "")
		ids := make([]string, 0, len(s.components))
		for id, component := range s.components {
			if appID != "" && ssmsapStringAny(component, "ApplicationId", "") != appID {
				continue
			}
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, ssmsapComponentSummary(s.components[id]))
		}
		return map[string]any{"Components": out, "NextToken": ""}

	case "ListConfigurationCheckDefinitions":
		out := make([]any, 0, len(s.configCheckDefinitions))
		for _, definition := range s.configCheckDefinitions {
			out = append(out, ssmsapCloneMap(definition))
		}
		return map[string]any{"ConfigurationCheckDefinitions": out, "NextToken": ""}

	case "ListConfigurationCheckOperations":
		ids := make([]string, 0, len(s.configCheckOperations))
		for id := range s.configCheckOperations {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, ssmsapCloneMap(s.configCheckOperations[id]))
		}
		return map[string]any{"ConfigurationCheckOperations": out, "NextToken": ""}

	case "ListDatabases":
		appID := ssmsapStringAny(payload, "ApplicationId", "")
		ids := make([]string, 0, len(s.databases))
		for id, database := range s.databases {
			if appID != "" && ssmsapStringAny(database, "ApplicationId", "") != appID {
				continue
			}
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, ssmsapDatabaseSummary(s.databases[id]))
		}
		return map[string]any{"Databases": out, "NextToken": ""}

	case "ListOperationEvents":
		opID := ssmsapStringAny(payload, "OperationId", s.firstOperationIDLocked())
		events := s.operationEvents[opID]
		if len(events) == 0 {
			events = []map[string]any{{
				"OperationId":    opID,
				"Type":           "INFO",
				"Description":    "No operation events recorded",
				"Timestamp":      now,
				"ResourceStatus": "UNKNOWN",
			}}
		}
		out := make([]any, 0, len(events))
		for _, event := range events {
			out = append(out, ssmsapCloneMap(event))
		}
		return map[string]any{"OperationEvents": out, "NextToken": ""}

	case "ListOperations":
		ids := make([]string, 0, len(s.operations))
		for id := range s.operations {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, ssmsapCloneMap(s.operations[id]))
		}
		return map[string]any{"Operations": out, "NextToken": ""}

	case "ListSubCheckResults":
		checkID := ssmsapStringAny(payload, "ConfigurationCheckOperationId", s.latestConfigurationOperation)
		results := s.subCheckResults[checkID]
		out := make([]any, 0, len(results))
		for _, r := range results {
			out = append(out, ssmsapCloneMap(r))
		}
		return map[string]any{"SubCheckResults": out, "NextToken": ""}

	case "ListSubCheckRuleResults":
		checkID := ssmsapStringAny(payload, "ConfigurationCheckOperationId", s.latestConfigurationOperation)
		results := s.subCheckRuleResults[checkID]
		out := make([]any, 0, len(results))
		for _, r := range results {
			out = append(out, ssmsapCloneMap(r))
		}
		return map[string]any{"SubCheckRuleResults": out, "NextToken": ""}

	case "ListTagsForResource":
		resourceARN := ssmsapStringAny(pathParamsToAny(pathParams), "resourceArn", ssmsapFirstKeyOrDefault(s.tags, ssmsapApplicationARN(s.firstApplicationIDLocked())))
		s.ensureTagsLocked(resourceARN)
		return map[string]any{"Tags": ssmsapCloneStringMap(s.tags[resourceARN])}

	case "PutResourcePermission":
		permissionID := ssmsapStringAny(payload, "ResourcePermissionId", fmt.Sprintf("perm-%06d", s.nextOperationIDLocked()))
		permission := s.ensureResourcePermissionLocked(permissionID)
		for key, value := range payload {
			permission[key] = value
		}
		permission["LastUpdated"] = now
		return map[string]any{}

	case "RegisterApplication":
		appID := ssmsapStringAny(payload, "ApplicationId", "")
		if appID == "" {
			appID = fmt.Sprintf("app-%06d", s.nextAppIDLocked())
		}
		app := s.ensureApplicationLocked(appID)
		for key, value := range payload {
			app[key] = value
		}
		app["ApplicationId"] = appID
		app["ApplicationArn"] = ssmsapApplicationARN(appID)
		app["Status"] = "ACTIVATED"
		app["LastUpdated"] = now
		s.ensureTagsLocked(ssmsapApplicationARN(appID))
		operationID := s.createOperationLocked("REGISTER_APPLICATION", appID, "SUCCEEDED")
		return map[string]any{"ApplicationId": appID, "OperationId": operationID}

	case "StartApplication":
		appID := ssmsapStringAny(payload, "ApplicationId", s.firstApplicationIDLocked())
		opID := s.createOperationLocked("START_APPLICATION", appID, "IN_PROGRESS")
		s.operations[opID]["Status"] = "SUCCEEDED"
		s.operations[opID]["EndTime"] = now
		return map[string]any{"OperationId": opID}

	case "StartApplicationRefresh":
		appID := ssmsapStringAny(payload, "ApplicationId", s.firstApplicationIDLocked())
		opID := s.createOperationLocked("START_APPLICATION_REFRESH", appID, "IN_PROGRESS")
		s.operations[opID]["Status"] = "SUCCEEDED"
		s.operations[opID]["EndTime"] = now
		return map[string]any{"OperationId": opID}

	case "StartConfigurationChecks":
		appID := ssmsapStringAny(payload, "ApplicationId", s.firstApplicationIDLocked())
		checkID := fmt.Sprintf("check-operation-%06d", s.nextCheckOperationIDLocked())
		checkOp := s.ensureConfigCheckOperationLocked(checkID, appID)
		for key, value := range payload {
			checkOp[key] = value
		}
		checkOp["ConfigurationCheckOperationId"] = checkID
		checkOp["ApplicationId"] = appID
		checkOp["Status"] = "SUCCEEDED"
		checkOp["StartTime"] = now
		checkOp["EndTime"] = now
		s.latestConfigurationOperation = checkID
		s.subCheckResults[checkID] = []map[string]any{{
			"Name":        "ConfigurationChecks",
			"Status":      "SUCCESS",
			"Description": "Configuration checks completed successfully",
		}}
		s.subCheckRuleResults[checkID] = []map[string]any{{
			"Name":        "HANA_INSTANCE_HEALTH",
			"Status":      "SUCCESS",
			"Severity":    "LOW",
			"Description": "All required SAP instances are healthy",
		}}
		return map[string]any{"ConfigurationCheckOperationId": checkID, "OperationId": checkID}

	case "StopApplication":
		appID := ssmsapStringAny(payload, "ApplicationId", s.firstApplicationIDLocked())
		opID := s.createOperationLocked("STOP_APPLICATION", appID, "IN_PROGRESS")
		s.operations[opID]["Status"] = "SUCCEEDED"
		s.operations[opID]["EndTime"] = now
		return map[string]any{"OperationId": opID}

	case "TagResource":
		resourceARN := ssmsapStringAny(pathParamsToAny(pathParams), "resourceArn", ssmsapApplicationARN(s.firstApplicationIDLocked()))
		s.ensureTagsLocked(resourceARN)
		for key, value := range ssmsapMapString(payload, "Tags") {
			s.tags[resourceARN][key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := ssmsapStringAny(pathParamsToAny(pathParams), "resourceArn", ssmsapApplicationARN(s.firstApplicationIDLocked()))
		s.ensureTagsLocked(resourceARN)
		tagKeys := ssmsapQueryOrPayloadStrings(query, payload, "tagKeys", "TagKeys")
		for _, key := range tagKeys {
			delete(s.tags[resourceARN], key)
		}
		return map[string]any{}

	case "UpdateApplicationSettings":
		appID := ssmsapStringAny(payload, "ApplicationId", s.firstApplicationIDLocked())
		app := s.ensureApplicationLocked(appID)
		settings := ssmsapMapAny(payload, "ApplicationSettings")
		if len(settings) > 0 {
			app["ApplicationSettings"] = settings
		}
		for key, value := range payload {
			if strings.EqualFold(key, "ApplicationSettings") {
				continue
			}
			app[key] = value
		}
		app["LastUpdated"] = now
		operationID := s.createOperationLocked("UPDATE_APPLICATION_SETTINGS", appID, "SUCCEEDED")
		return map[string]any{"OperationId": operationID}
	}

	return map[string]any{}
}

func (s *ssmSAPStore) ensureApplicationLocked(applicationID string) map[string]any {
	id := strings.TrimSpace(applicationID)
	if id == "" {
		id = s.firstApplicationIDLocked()
	}
	if app := s.applications[id]; app != nil {
		return app
	}
	app := map[string]any{
		"ApplicationId":   id,
		"ApplicationArn":  ssmsapApplicationARN(id),
		"Type":            "HANA",
		"Status":          "ACTIVATED",
		"DiscoveryStatus": "SUCCESS",
		"Tags":            map[string]any{"stackyard": "true"},
		"LastUpdated":     time.Now().UTC(),
	}
	s.applications[id] = app
	return app
}

func (s *ssmSAPStore) ensureComponentLocked(componentID, applicationID string) map[string]any {
	id := strings.TrimSpace(componentID)
	if id == "" {
		id = s.firstComponentIDLocked()
	}
	if component := s.components[id]; component != nil {
		return component
	}
	component := map[string]any{
		"ComponentId":   id,
		"ApplicationId": applicationID,
		"ComponentType": "HANA",
		"Status":        "RUNNING",
		"LastUpdated":   time.Now().UTC(),
	}
	s.components[id] = component
	return component
}

func (s *ssmSAPStore) ensureDatabaseLocked(databaseID, applicationID, componentID string) map[string]any {
	id := strings.TrimSpace(databaseID)
	if id == "" {
		id = s.firstDatabaseIDLocked()
	}
	if database := s.databases[id]; database != nil {
		return database
	}
	database := map[string]any{
		"DatabaseId":    id,
		"ApplicationId": applicationID,
		"ComponentId":   componentID,
		"DatabaseType":  "HANA",
		"Status":        "AVAILABLE",
		"LastUpdated":   time.Now().UTC(),
	}
	s.databases[id] = database
	return database
}

func (s *ssmSAPStore) ensureOperationLocked(operationID, operationType, applicationID, status string) map[string]any {
	id := strings.TrimSpace(operationID)
	if id == "" {
		id = s.firstOperationIDLocked()
	}
	if op := s.operations[id]; op != nil {
		return op
	}
	op := map[string]any{
		"OperationId":   id,
		"Type":          operationType,
		"ApplicationId": applicationID,
		"Status":        status,
		"StartTime":     time.Now().UTC(),
	}
	s.operations[id] = op
	return op
}

func (s *ssmSAPStore) ensureConfigCheckOperationLocked(operationID, applicationID string) map[string]any {
	id := strings.TrimSpace(operationID)
	if id == "" {
		id = s.firstConfigCheckOperationIDLocked()
	}
	if op := s.configCheckOperations[id]; op != nil {
		return op
	}
	op := map[string]any{
		"ConfigurationCheckOperationId": id,
		"ApplicationId":                 applicationID,
		"Status":                        "SUCCEEDED",
		"StartTime":                     time.Now().UTC(),
		"EndTime":                       time.Now().UTC(),
	}
	s.configCheckOperations[id] = op
	return op
}

func (s *ssmSAPStore) ensureResourcePermissionLocked(permissionID string) map[string]any {
	id := strings.TrimSpace(permissionID)
	if id == "" {
		id = s.firstResourcePermissionIDLocked()
	}
	if perm := s.resourcePermissions[id]; perm != nil {
		return perm
	}
	perm := map[string]any{
		"ResourcePermissionId": id,
		"ActionType":           "RESTORE_DATABASE",
		"ResourceArn":          ssmsapApplicationARN(s.firstApplicationIDLocked()),
		"SourceResourceArn":    "arn:aws:backup:us-east-1:123456789012:recovery-point:stackyard",
		"LastUpdated":          time.Now().UTC(),
	}
	s.resourcePermissions[id] = perm
	return perm
}

func (s *ssmSAPStore) ensureTagsLocked(resourceARN string) {
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
}

func (s *ssmSAPStore) createOperationLocked(operationType, applicationID, status string) string {
	opID := fmt.Sprintf("operation-%06d", s.nextOperationIDLocked())
	op := map[string]any{
		"OperationId":   opID,
		"Type":          operationType,
		"ApplicationId": applicationID,
		"Status":        status,
		"StartTime":     time.Now().UTC(),
	}
	s.operations[opID] = op
	s.operationEvents[opID] = []map[string]any{{
		"OperationId":    opID,
		"Type":           "INFO",
		"Description":    strings.ReplaceAll(strings.ToLower(operationType), "_", " ") + " operation accepted",
		"Timestamp":      time.Now().UTC(),
		"ResourceStatus": status,
	}}
	return opID
}

func (s *ssmSAPStore) firstApplicationIDLocked() string {
	ids := make([]string, 0, len(s.applications))
	for id := range s.applications {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "app-000001"
	}
	sort.Strings(ids)
	return ids[0]
}

func (s *ssmSAPStore) firstComponentIDLocked() string {
	ids := make([]string, 0, len(s.components))
	for id := range s.components {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "component-000001"
	}
	sort.Strings(ids)
	return ids[0]
}

func (s *ssmSAPStore) firstDatabaseIDLocked() string {
	ids := make([]string, 0, len(s.databases))
	for id := range s.databases {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "database-000001"
	}
	sort.Strings(ids)
	return ids[0]
}

func (s *ssmSAPStore) firstOperationIDLocked() string {
	ids := make([]string, 0, len(s.operations))
	for id := range s.operations {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "operation-000001"
	}
	sort.Strings(ids)
	return ids[0]
}

func (s *ssmSAPStore) firstConfigCheckOperationIDLocked() string {
	ids := make([]string, 0, len(s.configCheckOperations))
	for id := range s.configCheckOperations {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "check-operation-000001"
	}
	sort.Strings(ids)
	return ids[0]
}

func (s *ssmSAPStore) firstResourcePermissionIDLocked() string {
	ids := make([]string, 0, len(s.resourcePermissions))
	for id := range s.resourcePermissions {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "perm-000001"
	}
	sort.Strings(ids)
	return ids[0]
}

func (s *ssmSAPStore) nextAppIDLocked() int64 {
	id := s.nextAppID
	s.nextAppID++
	return id
}

func (s *ssmSAPStore) nextOperationIDLocked() int64 {
	id := s.nextOperationID
	s.nextOperationID++
	return id
}

func (s *ssmSAPStore) nextCheckOperationIDLocked() int64 {
	id := s.nextCheckOpID
	s.nextCheckOpID++
	return id
}

func ssmsapApplicationARN(applicationID string) string {
	id := strings.TrimSpace(applicationID)
	if id == "" {
		id = "app-000001"
	}
	if strings.HasPrefix(id, "arn:") {
		return id
	}
	return fmt.Sprintf("arn:aws:ssm-sap:us-east-1:123456789012:application/%s", id)
}

func ssmsapApplicationSummary(application map[string]any) map[string]any {
	return map[string]any{
		"ApplicationId":   ssmsapStringAny(application, "ApplicationId", "app-000001"),
		"ApplicationArn":  ssmsapStringAny(application, "ApplicationArn", ssmsapApplicationARN("app-000001")),
		"Type":            ssmsapStringAny(application, "Type", "HANA"),
		"Status":          ssmsapStringAny(application, "Status", "ACTIVATED"),
		"DiscoveryStatus": ssmsapStringAny(application, "DiscoveryStatus", "SUCCESS"),
	}
}

func ssmsapComponentSummary(component map[string]any) map[string]any {
	return map[string]any{
		"ComponentId":   ssmsapStringAny(component, "ComponentId", "component-000001"),
		"ApplicationId": ssmsapStringAny(component, "ApplicationId", "app-000001"),
		"ComponentType": ssmsapStringAny(component, "ComponentType", "HANA"),
		"Status":        ssmsapStringAny(component, "Status", "RUNNING"),
	}
}

func ssmsapDatabaseSummary(database map[string]any) map[string]any {
	return map[string]any{
		"DatabaseId":    ssmsapStringAny(database, "DatabaseId", "database-000001"),
		"ApplicationId": ssmsapStringAny(database, "ApplicationId", "app-000001"),
		"DatabaseType":  ssmsapStringAny(database, "DatabaseType", "HANA"),
		"Status":        ssmsapStringAny(database, "Status", "AVAILABLE"),
	}
}

func ssmsapStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if str, ok := v.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					return str
				}
			}
			break
		}
	}
	return fallback
}

func ssmsapMapAny(values map[string]any, key string) map[string]any {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if out, ok := v.(map[string]any); ok {
				return ssmsapCloneMap(out)
			}
			break
		}
	}
	return map[string]any{}
}

func ssmsapMapString(values map[string]any, key string) map[string]string {
	out := map[string]string{}
	for k, v := range values {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if raw, ok := v.(map[string]any); ok {
			for rk, rv := range raw {
				if str, ok := rv.(string); ok {
					out[strings.TrimSpace(rk)] = strings.TrimSpace(str)
				}
			}
		}
		if raw, ok := v.(map[string]string); ok {
			for rk, rv := range raw {
				out[strings.TrimSpace(rk)] = strings.TrimSpace(rv)
			}
		}
	}
	return out
}

func ssmsapQueryOrPayloadStrings(query url.Values, payload map[string]any, queryKey, payloadKey string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range query[queryKey] {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(payloadKey)) {
			continue
		}
		switch typed := v.(type) {
		case []any:
			for _, element := range typed {
				if s, ok := element.(string); ok {
					s = strings.TrimSpace(s)
					if s == "" {
						continue
					}
					if _, exists := seen[s]; exists {
						continue
					}
					seen[s] = struct{}{}
					out = append(out, s)
				}
			}
		case []string:
			for _, element := range typed {
				element = strings.TrimSpace(element)
				if element == "" {
					continue
				}
				if _, exists := seen[element]; exists {
					continue
				}
				seen[element] = struct{}{}
				out = append(out, element)
			}
		}
	}
	return out
}

func ssmsapCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out[k] = ssmsapCloneMap(tv)
		case map[string]string:
			copyMap := make(map[string]string, len(tv))
			for mk, mv := range tv {
				copyMap[mk] = mv
			}
			out[k] = copyMap
		case []any:
			out[k] = ssmsapCloneSlice(tv)
		default:
			out[k] = tv
		}
	}
	return out
}

func ssmsapCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out = append(out, ssmsapCloneMap(tv))
		case []any:
			out = append(out, ssmsapCloneSlice(tv))
		default:
			out = append(out, tv)
		}
	}
	return out
}

func ssmsapCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func ssmsapFirstKeyOrDefault[V any](values map[string]V, fallback string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return fallback
	}
	sort.Strings(keys)
	return keys[0]
}

func pathParamsToAny(pathParams map[string]string) map[string]any {
	out := make(map[string]any, len(pathParams))
	for k, v := range pathParams {
		out[k] = v
	}
	return out
}
