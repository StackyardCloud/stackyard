package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type iotStore struct {
	mu   sync.Mutex
	next int64
	tags map[string]map[string]string
}

func newIoTStore() *iotStore {
	return &iotStore{
		next: 1,
		tags: map[string]map[string]string{
			"arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing": {
				"seed": "true",
			},
		},
	}
}

func (s *iotStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "TagResource":
		arn := iotResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing"
		}
		incoming := iotExtractTagMap(iotValue(payload, "tags"))
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
		arn := iotResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing"
		}
		current := s.tags[arn]
		if current == nil {
			return map[string]any{}
		}
		for _, key := range iotStringSlice(iotValue(payload, "tagKeys")) {
			delete(current, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(current, key)
			}
		}
		s.tags[arn] = current
		return map[string]any{}

	case "ListTagsForResource":
		arn := iotResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing"
		}
		tags := iotCloneStringMap(s.tags[arn])
		return map[string]any{"tags": iotTagList(tags)}

	case "DescribeEndpoint":
		return map[string]any{"endpointAddress": "a3qEXAMPLEffp-ats.iot.us-east-1.amazonaws.com"}
	case "DescribeEventConfigurations":
		return map[string]any{"eventConfigurations": map[string]any{}}
	case "GetLoggingOptions":
		return map[string]any{"roleArn": "arn:aws:iam::123456789012:role/stackyard-iot", "logLevel": "ERROR"}
	case "GetV2LoggingOptions":
		return map[string]any{"defaultLogLevel": "ERROR", "disableAllLogs": false}
	case "CreateKeysAndCertificate":
		id := s.nextID("cert")
		return map[string]any{
			"certificateArn": fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:cert/%s", id),
			"certificateId":  id,
			"certificatePem": "-----BEGIN CERTIFICATE-----\\nSTACKYARD\\n-----END CERTIFICATE-----",
			"keyPair": map[string]any{
				"PublicKey":  "STACKYARD_PUBLIC_KEY",
				"PrivateKey": "STACKYARD_PRIVATE_KEY",
			},
		}
	case "CreateCertificateFromCsr":
		id := s.nextID("cert")
		return map[string]any{
			"certificateArn": fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:cert/%s", id),
			"certificateId":  id,
			"certificatePem": "-----BEGIN CERTIFICATE-----\\nSTACKYARD\\n-----END CERTIFICATE-----",
		}
	case "GetRegistrationCode":
		return map[string]any{"registrationCode": "stackyard-registration-code"}
	}

	if strings.HasPrefix(action, "List") {
		key := iotListKey(action)
		item := map[string]any{
			"name":               "stackyard",
			"thingName":          "stackyard-thing",
			"thingArn":           "arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing",
			"thingGroupName":     "stackyard-group",
			"thingTypeName":      "stackyard-type",
			"policyName":         "stackyard-policy",
			"jobId":              "stackyard-job",
			"certificateId":      "stackyard-cert",
			"target":             "arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing",
			"creationDate":       now,
			"lastModifiedDate":   now,
			"lastUpdatedDate":    now,
			"defaultVersionId":   "1",
			"templateArn":        "arn:aws:iot:us-east-1:123456789012:provisioningtemplate/stackyard-template",
			"scheduledAuditName": "stackyard-audit",
		}
		return map[string]any{key: []any{item}, "nextToken": "", "nextMarker": ""}
	}

	if strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "Get") {
		id := iotResolveID(payload, pathParams)
		return map[string]any{
			"id":                 id,
			"name":               id,
			"arn":                iotARNFor(action, id),
			"thingName":          id,
			"thingArn":           fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:thing/%s", id),
			"thingTypeName":      id,
			"thingTypeArn":       fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:thingtype/%s", id),
			"thingGroupName":     id,
			"thingGroupArn":      fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:thinggroup/%s", id),
			"policyName":         id,
			"policyArn":          fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:policy/%s", id),
			"jobId":              id,
			"targetArn":          fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:thing/%s", id),
			"description":        "stackyard " + action,
			"status":             "ACTIVE",
			"creationDate":       now,
			"lastModifiedDate":   now,
			"lastUpdatedDate":    now,
			"defaultVersionId":   "1",
			"attributes":         map[string]any{"stackyard": "true"},
			"version":            1,
			"generationId":       "1",
			"documentParameters": map[string]any{},
		}
	}

	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Register") {
		id := iotResolveID(payload, pathParams)
		if id == "" {
			id = s.nextID("resource")
		}
		return map[string]any{
			"id":                      id,
			"name":                    id,
			"arn":                     iotARNFor(action, id),
			"thingName":               id,
			"thingArn":                fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:thing/%s", id),
			"thingGroupName":          id,
			"thingGroupArn":           fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:thinggroup/%s", id),
			"thingTypeName":           id,
			"thingTypeArn":            fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:thingtype/%s", id),
			"billingGroupName":        id,
			"billingGroupArn":         fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:billinggroup/%s", id),
			"policyName":              id,
			"policyArn":               fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:policy/%s", id),
			"jobId":                   id,
			"jobArn":                  fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:job/%s", id),
			"jobTemplateId":           id,
			"jobTemplateArn":          fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:jobtemplate/%s", id),
			"certificateArn":          fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:cert/%s", id),
			"certificateId":           id,
			"authorizerName":          id,
			"authorizerArn":           fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:authorizer/%s", id),
			"domainConfigurationName": id,
			"domainConfigurationArn":  fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:domainconfiguration/%s", id),
			"topicRuleArn":            fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:rule/%s", id),
			"topicRuleName":           id,
			"streamId":                id,
			"streamArn":               fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:stream/%s", id),
			"roleAlias":               id,
			"roleAliasArn":            fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:rolealias/%s", id),
			"securityProfileName":     id,
			"securityProfileArn":      fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:securityprofile/%s", id),
			"auditTaskId":             id,
			"taskId":                  id,
			"creationDate":            now,
			"lastModifiedDate":        now,
			"version":                 1,
		}
	}

	if strings.HasPrefix(action, "Update") ||
		strings.HasPrefix(action, "Delete") ||
		strings.HasPrefix(action, "Attach") ||
		strings.HasPrefix(action, "Detach") ||
		strings.HasPrefix(action, "Set") ||
		strings.HasPrefix(action, "Cancel") ||
		strings.HasPrefix(action, "Start") ||
		strings.HasPrefix(action, "Stop") ||
		strings.HasPrefix(action, "Transfer") ||
		strings.HasPrefix(action, "Put") ||
		strings.HasPrefix(action, "Enable") ||
		strings.HasPrefix(action, "Disable") ||
		strings.HasPrefix(action, "Reject") ||
		strings.HasPrefix(action, "Accept") ||
		strings.HasPrefix(action, "Confirm") ||
		strings.HasPrefix(action, "Validate") ||
		strings.HasPrefix(action, "Test") ||
		strings.HasPrefix(action, "Search") ||
		strings.HasPrefix(action, "Deprecate") ||
		strings.HasPrefix(action, "Clear") ||
		strings.HasPrefix(action, "Remove") ||
		strings.HasPrefix(action, "Associate") ||
		strings.HasPrefix(action, "Disassociate") ||
		strings.HasPrefix(action, "Replace") {
		return map[string]any{
			"id":        iotResolveID(payload, pathParams),
			"status":    "SUCCESS",
			"timestamp": now,
		}
	}

	return map[string]any{"operation": action, "status": "SUCCESS", "timestamp": now}
}

func (s *iotStore) nextID(prefix string) string {
	s.next++
	return fmt.Sprintf("%s-%06d", prefix, s.next)
}

func iotListKey(action string) string {
	keys := map[string]string{
		"ListTagsForResource":                   "tags",
		"ListThings":                            "things",
		"ListThingGroups":                       "thingGroups",
		"ListThingGroupsForThing":               "thingGroups",
		"ListThingsInThingGroup":                "things",
		"ListThingsInBillingGroup":              "things",
		"ListThingTypes":                        "thingTypes",
		"ListPolicies":                          "policies",
		"ListPolicyVersions":                    "policyVersions",
		"ListPrincipalPolicies":                 "policies",
		"ListAttachedPolicies":                  "policies",
		"ListTargetsForPolicy":                  "targets",
		"ListTargetsForSecurityProfile":         "securityProfileTargets",
		"ListPrincipalThings":                   "things",
		"ListPrincipalThingsV2":                 "principalThingObjects",
		"ListCertificates":                      "certificates",
		"ListCACertificates":                    "certificates",
		"ListCertificatesByCA":                  "certificates",
		"ListOutgoingCertificates":              "outgoingCertificates",
		"ListBillingGroups":                     "billingGroups",
		"ListJobs":                              "jobs",
		"ListJobTemplates":                      "jobTemplates",
		"ListManagedJobTemplates":               "managedJobTemplates",
		"ListJobExecutionsForJob":               "executionSummaries",
		"ListJobExecutionsForThing":             "executionSummaries",
		"ListTopicRules":                        "rules",
		"ListTopicRuleDestinations":             "destinationSummaries",
		"ListAuthorizers":                       "authorizers",
		"ListDomainConfigurations":              "domainConfigurations",
		"ListDimensions":                        "dimensionNames",
		"ListStreams":                           "streams",
		"ListAuditTasks":                        "tasks",
		"ListAuditFindings":                     "findings",
		"ListAuditSuppressions":                 "suppressions",
		"ListAuditMitigationActionsTasks":       "tasks",
		"ListAuditMitigationActionsExecutions":  "actionsExecutions",
		"ListDetectMitigationActionsTasks":      "tasks",
		"ListDetectMitigationActionsExecutions": "actionsExecutions",
		"ListScheduledAudits":                   "scheduledAudits",
		"ListSecurityProfiles":                  "securityProfileIdentifiers",
		"ListSecurityProfilesForTarget":         "securityProfileTargetMappings",
		"ListActiveViolations":                  "activeViolations",
		"ListViolationEvents":                   "violationEvents",
		"ListRelatedResourcesForAuditFinding":   "relatedResources",
		"ListCustomMetrics":                     "metricNames",
		"ListFleetMetrics":                      "fleetMetrics",
		"ListMetricValues":                      "metricDatumList",
		"ListMitigationActions":                 "actionIdentifiers",
		"ListIndices":                           "indexNames",
		"ListProvisioningTemplates":             "templates",
		"ListProvisioningTemplateVersions":      "versions",
		"ListThingRegistrationTasks":            "taskIds",
		"ListThingRegistrationTaskReports":      "resourceLinks",
		"ListRoleAliases":                       "roleAliases",
		"ListPackageVersions":                   "packageVersionSummaries",
		"ListPackages":                          "packageSummaries",
		"ListCommands":                          "commandSummaries",
		"ListCommandExecutions":                 "commandExecutions",
		"ListCertificateProviders":              "certificateProviders",
		"ListSbomValidationResults":             "summaries",
	}
	if key := keys[action]; key != "" {
		return key
	}
	return "items"
}

func iotResolveID(payload map[string]any, pathParams map[string]string) string {
	keys := []string{
		"id",
		"name",
		"thingName",
		"thingGroupName",
		"thingTypeName",
		"billingGroupName",
		"policyName",
		"certificateId",
		"caCertificateId",
		"authorizerName",
		"domainConfigurationName",
		"jobId",
		"jobTemplateId",
		"streamId",
		"ruleName",
		"target",
		"templateName",
		"templateArn",
		"scheduledAuditName",
		"securityProfileName",
		"taskId",
		"auditTaskId",
		"actionName",
		"dimensionName",
		"metricName",
		"roleAlias",
		"indexName",
		"packageName",
		"versionName",
		"commandId",
		"executionId",
	}
	for _, key := range keys {
		if v := iotPathParam(pathParams, key, ""); v != "" {
			return v
		}
	}
	for _, key := range keys {
		if v := iotDefaultString(payload, key, ""); v != "" {
			return v
		}
	}
	return "stackyard"
}

func iotARNFor(action, id string) string {
	typeByAction := map[string]string{
		"ThingGroup":           "thinggroup",
		"ThingType":            "thingtype",
		"Thing":                "thing",
		"BillingGroup":         "billinggroup",
		"Policy":               "policy",
		"Certificate":          "cert",
		"Authorizer":           "authorizer",
		"DomainConfiguration":  "domainconfiguration",
		"JobTemplate":          "jobtemplate",
		"Job":                  "job",
		"TopicRule":            "rule",
		"Stream":               "stream",
		"MitigationAction":     "mitigationaction",
		"SecurityProfile":      "securityprofile",
		"ScheduledAudit":       "scheduledaudit",
		"Dimension":            "dimension",
		"FleetMetric":          "fleetmetric",
		"RoleAlias":            "rolealias",
		"ProvisioningTemplate": "provisioningtemplate",
		"PackageVersion":       "packageversion",
		"Package":              "package",
		"CommandExecution":     "commandexecution",
		"Command":              "command",
	}
	for marker, resourceType := range typeByAction {
		if strings.Contains(action, marker) {
			return fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:%s/%s", resourceType, id)
		}
	}
	return fmt.Sprintf("arn:aws:iot:us-east-1:123456789012:resource/%s", id)
}

func iotResolveResourceARN(payload map[string]any, pathParams map[string]string, query url.Values) string {
	if value := strings.TrimSpace(iotDefaultString(payload, "resourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(iotPathParam(pathParams, "resourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(query.Get("resourceArn")); value != "" {
		return value
	}
	return ""
}

func iotValue(payload map[string]any, key string) any {
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

func iotDefaultString(payload map[string]any, key, fallback string) string {
	value := iotValue(payload, key)
	text := strings.TrimSpace(iotToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func iotPathParam(pathParams map[string]string, key, fallback string) string {
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

func iotToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func iotStringSlice(value any) []string {
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
			text := strings.TrimSpace(iotToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(iotToString(v))
		if text == "" || text == "<nil>" {
			return nil
		}
		return []string{text}
	}
}

func iotExtractTagMap(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for k, val := range v {
			k = strings.TrimSpace(k)
			if k != "" {
				out[k] = val
			}
		}
	case map[string]any:
		for k, raw := range v {
			k = strings.TrimSpace(k)
			if k != "" {
				out[k] = iotToString(raw)
			}
		}
	case []any:
		for _, item := range v {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(iotDefaultString(entry, "key", ""))
			if key == "" {
				continue
			}
			out[key] = iotDefaultString(entry, "value", "")
		}
	}
	return out
}

func iotCloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func iotTagList(in map[string]string) []map[string]string {
	if len(in) == 0 {
		return []map[string]string{}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]string{"key": key, "value": in[key]})
	}
	return out
}
