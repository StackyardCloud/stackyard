package server

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.SecurityInsights (Microsoft Sentinel) resources on a Log
// Analytics workspace, served by the shared ARM handler as extension
// resources of the workspace. Sentinel stores configuration only: rules
// never run, no alerts are raised, connectors ingest nothing and
// automation rules never fire. Incidents, bookmarks and watchlists are
// kept as written; see provider_azure_securityinsights_incidents.go.
func init() {
	registerAzureARMResourceType(azureSentinelType("onboardingStates", azureARMResourceType{
		ValidateName: azureSingletonName("default"),
		// The handler checks the workspace before the PUT takes the store
		// lock; check again under it, so a concurrent workspace delete
		// can't leave an orphaned onboarding state.
		Linked: func(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
			if s.azureARMResources[azureARMKey(azureSentinelWorkspace(rec.ID))] == nil {
				return nil, nil, "The workspace was deleted."
			}
			return nil, nil, ""
		},
	}))
	registerAzureARMResourceType(azureSentinelType("alertRules", azureARMResourceType{
		Validate:           azureSentinelAlertRuleValidate,
		ReadOnlyProperties: azureSentinelAlertRuleProperties,
		Linked:             azureSentinelOnboarded(nil),
	}))
	registerAzureARMResourceType(azureSentinelType("alertRules/actions", azureARMResourceType{
		Validate:            azureSentinelActionValidate,
		WriteOnlyProperties: []string{"triggerUri"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if m := azureSentinelWorkflowID.FindStringSubmatch(azureARMString(rec.Properties["triggerUri"])); m != nil {
				return map[string]any{"workflowId": m[1]}
			}
			return nil
		},
	}))
	registerAzureARMResourceType(azureSentinelType("automationRules", azureARMResourceType{
		Validate: azureSentinelAutomationRuleValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"createdTimeUtc": rec.CreatedAt.Format(time.RFC3339), "lastModifiedTimeUtc": rec.UpdatedAt.Format(time.RFC3339),
				"createdBy": azureSentinelClient, "lastModifiedBy": azureSentinelClient,
			}
		},
		Linked: azureSentinelOnboarded(nil),
	}))
	registerAzureARMResourceType(azureSentinelType("dataConnectors", azureARMResourceType{
		Validate: azureSentinelDataConnectorValidate,
		Linked:   azureSentinelOnboarded(nil),
	}))
	// Metadata is the one Sentinel type with PATCH, so it skips the
	// onboarding check, which runs as a Linked hook.
	metadata := azureSentinelType("metadata", azureARMResourceType{Validate: azureSentinelMetadataValidate})
	metadata.NoPatch = false
	registerAzureARMResourceType(metadata)
	registerAzureARMView(azureSentinelNamespace, "alertRuleTemplates", azureSentinelAlertRuleTemplates)
	azureSentinelRegisterIncidents()
}

const azureSentinelNamespace = "Microsoft.SecurityInsights"

var (
	azureSentinelWorkspaceScope = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.OperationalInsights/workspaces/[^/]+$`)
	azureSentinelWorkflowID     = regexp.MustCompile(`(?i)/workflows/([^/?]+)/`)
	azureSentinelDuration       = regexp.MustCompile(`^P(?:(\d{1,5})D)?(?:T(?:(\d{1,6})H)?(?:(\d{1,7})M)?(?:(\d{1,9})S)?)?$`)
	// azureSentinelClient is the ClientInfo of every change: the emulator
	// has no signed-in user.
	azureSentinelClient = map[string]any{"name": "stackyard"}

	azureSentinelSeverities     = []string{"High", "Medium", "Low", "Informational"}
	azureSentinelTriggerOps     = []string{"GreaterThan", "LessThan", "Equal", "NotEqual"}
	azureSentinelMatchingMethod = []string{"AllEntities", "AnyAlert", "Selected"}
	azureSentinelAggregations   = []string{"SingleAlert", "AlertPerResult"}
	azureSentinelEntityTypes    = []string{"Account", "Host", "IP", "Malware", "File", "Process", "CloudApplication", "DNS", "AzureResource",
		"FileHash", "RegistryKey", "RegistryValue", "SecurityGroup", "URL", "Mailbox", "MailCluster", "MailMessage", "SubmissionMail"}
	azureSentinelTactics = []string{"Reconnaissance", "ResourceDevelopment", "InitialAccess", "Execution", "Persistence", "PrivilegeEscalation",
		"DefenseEvasion", "CredentialAccess", "Discovery", "LateralMovement", "Collection", "Exfiltration", "CommandAndControl", "Impact",
		"PreAttack", "ImpairProcessControl", "InhibitResponseFunction"}
	azureSentinelProducts = []string{"Microsoft Cloud App Security", "Azure Security Center", "Azure Advanced Threat Protection",
		"Azure Active Directory Identity Protection", "Azure Security Center for IoT"}
	azureSentinelMetadataKinds = []string{"DataConnector", "DataType", "Workbook", "WorkbookTemplate", "Playbook", "PlaybookTemplate",
		"AnalyticsRuleTemplate", "AnalyticsRule", "HuntingQuery", "InvestigationQuery", "Parser", "Watchlist", "WatchlistTemplate", "Solution",
		"AzureFunction", "LogicAppsCustomConnector", "AutomationRule", "ResourcesDataConnector", "Notebook", "Standalone", "SummaryRule"}
)

// Fusion rules can only come from the built-in Fusion template, the one
// alert rule template the emulator lists.
const azureSentinelFusionTemplate = "f71aba3d-28fb-450b-b192-4e76a83015c8"

var azureSentinelFusionProperties = map[string]any{
	"displayName": "Advanced Multistage Attack Detection",
	"description": "Microsoft Sentinel uses Fusion, a correlation engine based on scalable machine learning algorithms, to automatically detect multistage attacks by identifying combinations of anomalous behaviors and suspicious activities that are observed at various stages of the kill chain.",
	"severity":    "High",
	"tactics":     []any{"Collection", "CommandAndControl", "CredentialAccess", "DefenseEvasion", "Discovery", "Execution", "Exfiltration", "Impact", "InitialAccess", "LateralMovement", "Persistence", "PrivilegeEscalation"},
	"techniques":  []any{},
}

// azureSentinelType registers typ as a Sentinel type on a stored Log
// Analytics workspace. Sentinel has no PATCH, except for metadata.
func azureSentinelType(path string, typ azureARMResourceType) azureARMResourceType {
	typ.Namespace, typ.Type, typ.Extension, typ.ScopeStored, typ.NoPatch = azureSentinelNamespace, path, true, true, true
	typ.ValidateScope = func(scope string) string {
		if !azureSentinelWorkspaceScope.MatchString(scope) {
			return "Microsoft Sentinel resources belong to a Log Analytics workspace (Microsoft.OperationalInsights/workspaces)."
		}
		return ""
	}
	return typ
}

// azureSentinelOnboarded rejects a PUT on a workspace without Sentinel's
// onboarding state, then returns extra's server-owned properties, if any.
// It runs as a Linked hook, under the store lock.
func azureSentinelOnboarded(extra func(s *Server, rec *azureARMResource) map[string]any) func(*Server, *azureARMResource) (map[string]any, []*azureARMResource, string) {
	return func(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
		workspace := azureSentinelWorkspace(rec.ID)
		if s.azureARMResources[azureARMKey(workspace+"/providers/"+azureSentinelNamespace+"/onboardingStates/default")] == nil {
			return nil, nil, "Workspace '" + workspace[strings.LastIndex(workspace, "/")+1:] + "' is not onboarded to Microsoft Sentinel. Create its onboarding state 'default' first."
		}
		if extra == nil {
			return nil, nil, ""
		}
		return extra(s, rec), nil, ""
	}
}

// azureSentinelWorkspace returns the workspace ID a Sentinel resource ID is
// under. It compares segments with EqualFold rather than searching a
// lower-cased copy, whose byte offsets can differ for non-ASCII names.
func azureSentinelWorkspace(id string) string {
	seg := strings.Split(id, "/")
	for i := len(seg) - 2; i > 0; i-- {
		if strings.EqualFold(seg[i], "providers") && strings.EqualFold(seg[i+1], azureSentinelNamespace) {
			return strings.Join(seg[:i], "/")
		}
	}
	return id
}

// azureSentinelDurationOf parses an ISO 8601 duration of days, hours,
// minutes and seconds, such as "PT5H" or "P1DT12H".
func azureSentinelDurationOf(raw any) (time.Duration, bool) {
	text, _ := raw.(string)
	m := azureSentinelDuration.FindStringSubmatch(text)
	if m == nil || text == "P" || strings.HasSuffix(text, "T") {
		return 0, false
	}
	var total time.Duration
	for i, unit := range []time.Duration{24 * time.Hour, time.Hour, time.Minute, time.Second} {
		if n, err := strconv.Atoi(m[i+1]); err == nil {
			total += time.Duration(n) * unit
		}
	}
	return total, true
}

// azureSentinelDurationIn checks that props[key] is a duration in [least, most].
func azureSentinelDurationIn(props map[string]any, key string, least, most time.Duration, label string) string {
	d, ok := azureSentinelDurationOf(props[key])
	if !ok || d < least || d > most {
		return "properties." + key + " must be an ISO 8601 duration " + label + "."
	}
	return ""
}

func azureSentinelRequireBool(props map[string]any, keys ...string) string {
	for _, key := range keys {
		if _, ok := props[key].(bool); !ok {
			return "properties." + key + " is required and must be true or false."
		}
	}
	return ""
}

func azureSentinelRequireString(props map[string]any, keys ...string) string {
	for _, key := range keys {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required."
		}
	}
	return ""
}

// azureSentinelEach checks that every item of props[key], an optional
// array, is one of values.
func azureSentinelEach(props map[string]any, key string, values []string) string {
	for _, item := range azureARMSlice(props[key]) {
		if !slices.Contains(values, azureARMString(item)) {
			return "properties." + key + " items must be one of " + strings.Join(values, ", ") + "."
		}
	}
	return ""
}

// azureSentinelOneOf checks that props[key] is one of values, if present
// or required.
func azureSentinelOneOf(props map[string]any, key string, required bool, values []string) string {
	raw, present := props[key]
	if (present || required) && !slices.Contains(values, azureARMString(raw)) {
		return "properties." + key + " must be one of " + strings.Join(values, ", ") + "."
	}
	return ""
}

func azureSentinelFirst(checks ...string) string {
	for _, msg := range checks {
		if msg != "" {
			return msg
		}
	}
	return ""
}

func azureSentinelAlertRuleValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch azureARMString(payload["kind"]) {
	case "Scheduled":
		return azureSentinelScheduledRuleValidate(props)
	case "MicrosoftSecurityIncidentCreation":
		return azureSentinelFirst(
			azureSentinelRequireString(props, "displayName"),
			azureSentinelRequireBool(props, "enabled"),
			azureSentinelOneOf(props, "productFilter", true, azureSentinelProducts),
			azureSentinelEach(props, "severitiesFilter", azureSentinelSeverities),
		)
	case "Fusion":
		if !strings.EqualFold(azureARMString(props["alertRuleTemplateName"]), azureSentinelFusionTemplate) {
			return "properties.alertRuleTemplateName must be the Fusion template, " + azureSentinelFusionTemplate + "."
		}
		return azureSentinelRequireBool(props, "enabled")
	default:
		return "kind must be Scheduled, MicrosoftSecurityIncidentCreation or Fusion."
	}
}

// azureSentinelScheduledRuleValidate applies the limits of the analytics
// rule wizard: queries run every 5 minutes to 14 days over a lookback of
// 5 minutes to 14 days that is at least the interval, suppression lasts up
// to 24 hours, and a rule maps up to 10 entities.
func azureSentinelScheduledRuleValidate(props map[string]any) string {
	const least, most = 5 * time.Minute, 14 * 24 * time.Hour
	msg := azureSentinelFirst(
		azureSentinelRequireString(props, "displayName", "query"),
		azureSentinelRequireBool(props, "enabled", "suppressionEnabled"),
		azureSentinelDurationIn(props, "suppressionDuration", 0, 24*time.Hour, "of at most 24 hours"),
		azureSentinelDurationIn(props, "queryFrequency", least, most, "from 5 minutes to 14 days"),
		azureSentinelDurationIn(props, "queryPeriod", least, most, "from 5 minutes to 14 days"),
		azureSentinelOneOf(props, "severity", true, azureSentinelSeverities),
		azureSentinelOneOf(props, "triggerOperator", true, azureSentinelTriggerOps),
		azureSentinelEach(props, "tactics", azureSentinelTactics),
	)
	if msg != "" {
		return msg
	}
	frequency, _ := azureSentinelDurationOf(props["queryFrequency"])
	if period, _ := azureSentinelDurationOf(props["queryPeriod"]); period < frequency {
		return "properties.queryPeriod must be at least properties.queryFrequency."
	}
	if !azureWholeNumber(props["triggerThreshold"], 0, 2147483647) {
		return "properties.triggerThreshold is required and must be a whole number of at least 0."
	}
	if settings, present := props["eventGroupingSettings"]; present {
		if msg := azureSentinelOneOf(azureARMObject(settings), "aggregationKind", false, azureSentinelAggregations); msg != "" {
			return strings.Replace(msg, "properties.", "properties.eventGroupingSettings.", 1)
		}
	}
	mappings := azureARMSlice(props["entityMappings"])
	if len(mappings) > 10 {
		return "properties.entityMappings can map at most 10 entities."
	}
	for _, raw := range mappings {
		mapping := azureARMObject(raw)
		if !slices.Contains(azureSentinelEntityTypes, azureARMString(mapping["entityType"])) || len(azureARMSlice(mapping["fieldMappings"])) == 0 {
			return "properties.entityMappings items need an entityType Microsoft Sentinel recognizes and at least one fieldMappings item."
		}
	}
	if raw, present := props["incidentConfiguration"]; present {
		return azureSentinelIncidentConfigurationValidate(azureARMObject(raw))
	}
	return ""
}

func azureSentinelIncidentConfigurationValidate(config map[string]any) string {
	if _, ok := config["createIncident"].(bool); !ok {
		return "properties.incidentConfiguration.createIncident is required and must be true or false."
	}
	raw, present := config["groupingConfiguration"]
	if !present {
		return ""
	}
	grouping := azureARMObject(raw)
	msg := azureSentinelFirst(
		azureSentinelRequireBool(grouping, "enabled", "reopenClosedIncident"),
		azureSentinelDurationIn(grouping, "lookbackDuration", time.Second, 14*24*time.Hour, "of up to 14 days"),
		azureSentinelOneOf(grouping, "matchingMethod", true, azureSentinelMatchingMethod),
		azureSentinelEach(grouping, "groupByEntities", azureSentinelEntityTypes),
		azureSentinelEach(grouping, "groupByAlertDetails", []string{"DisplayName", "Severity"}),
	)
	return strings.Replace(msg, "properties.", "properties.incidentConfiguration.groupingConfiguration.", 1)
}

// azureSentinelAlertRuleProperties fills a Fusion rule's read-only
// details from its template.
func azureSentinelAlertRuleProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{"lastModifiedUtc": rec.UpdatedAt.Format(time.RFC3339)}
	if rec.Kind == "Fusion" {
		props = azureARMMerge(props, azureSentinelFusionProperties)
	}
	return props
}

func azureSentinelAlertRuleTemplates(s *Server, scope string) []map[string]any {
	prefix := azureARMKey(scope + "/providers/" + azureSentinelNamespace + "/alertRules/")
	created := len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return rec.Kind == "Fusion" && strings.HasPrefix(azureARMKey(rec.ID), prefix)
	}))
	props := azureARMMerge(azureSentinelFusionProperties, map[string]any{
		"alertRulesCreatedByTemplateCount": created, "status": "Available",
		"createdDateUTC": "2019-07-25T00:00:00Z", "lastUpdatedDateUTC": "2019-07-25T00:00:00Z",
		"requiredDataConnectors": []any{},
	})
	return []map[string]any{{
		"id":   scope + "/providers/" + azureSentinelNamespace + "/alertRuleTemplates/" + azureSentinelFusionTemplate,
		"name": azureSentinelFusionTemplate, "type": azureSentinelNamespace + "/AlertRuleTemplates", "kind": "Fusion", "properties": props,
	}}
}

func azureSentinelActionValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.Contains(strings.ToLower(azureARMString(props["logicAppResourceId"])), "/providers/microsoft.logic/workflows/") {
		return "properties.logicAppResourceId must be the ID of a Logic App workflow (Microsoft.Logic/workflows)."
	}
	if !strings.HasPrefix(azureARMString(props["triggerUri"]), "https://") {
		return "properties.triggerUri is required and must be the workflow's https trigger callback URL."
	}
	return ""
}

func azureSentinelAutomationRuleValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if name := azureARMString(props["displayName"]); strings.TrimSpace(name) == "" || len([]rune(name)) > 500 {
		return "properties.displayName is required and can be at most 500 characters."
	}
	if !azureWholeNumber(props["order"], 1, 1000) {
		return "properties.order must be a whole number from 1 to 1000."
	}
	logic, ok := props["triggeringLogic"].(map[string]any)
	if !ok {
		return "properties.triggeringLogic is required."
	}
	msg := azureSentinelFirst(
		azureSentinelRequireBool(logic, "isEnabled"),
		azureSentinelOneOf(logic, "triggersOn", true, []string{"Incidents", "Alerts"}),
		azureSentinelOneOf(logic, "triggersWhen", true, []string{"Created", "Updated"}),
	)
	if msg != "" {
		return strings.Replace(msg, "properties.", "properties.triggeringLogic.", 1)
	}
	if raw, present := logic["expirationTimeUtc"]; present {
		if _, err := time.Parse(time.RFC3339, azureARMString(raw)); err != nil {
			return "properties.triggeringLogic.expirationTimeUtc must be an RFC 3339 date and time."
		}
	}
	conditions := azureARMSlice(logic["conditions"])
	if len(conditions) > 50 {
		return "properties.triggeringLogic.conditions can have at most 50 items."
	}
	for _, raw := range conditions {
		if !slices.Contains([]string{"Property", "PropertyArray", "PropertyChanged", "PropertyArrayChanged", "Boolean"}, azureARMString(azureARMObject(raw)["conditionType"])) {
			return "properties.triggeringLogic.conditions items need a conditionType of Property, PropertyArray, PropertyChanged, PropertyArrayChanged or Boolean."
		}
	}
	actions := azureARMSlice(props["actions"])
	if len(actions) == 0 || len(actions) > 20 {
		return "properties.actions must have 1 to 20 items."
	}
	alerts := logic["triggersOn"] == "Alerts"
	for _, raw := range actions {
		if msg := azureSentinelAutomationActionValidate(azureARMObject(raw), alerts); msg != "" {
			return msg
		}
	}
	return ""
}

// azureSentinelAutomationActionValidate checks one action. Rules that
// trigger on alerts can only run playbooks.
func azureSentinelAutomationActionValidate(action map[string]any, alerts bool) string {
	if !azureWholeNumber(action["order"], 0, 2147483647) {
		return "properties.actions items need a whole-number order."
	}
	config := azureARMObject(action["actionConfiguration"])
	kind := azureARMString(action["actionType"])
	switch {
	case alerts && kind != "RunPlaybook":
		return "Automation rules that trigger on alerts can only run playbooks (actionType RunPlaybook)."
	case kind == "RunPlaybook":
		if !strings.Contains(strings.ToLower(azureARMString(config["logicAppResourceId"])), "/providers/microsoft.logic/workflows/") {
			return "RunPlaybook actions need actionConfiguration.logicAppResourceId, the ID of a Logic App workflow."
		}
	case kind == "AddIncidentTask":
		if strings.TrimSpace(azureARMString(config["title"])) == "" {
			return "AddIncidentTask actions need actionConfiguration.title."
		}
	case kind == "ModifyProperties":
		msg := azureSentinelFirst(
			azureSentinelOneOf(config, "severity", false, azureSentinelSeverities),
			azureSentinelOneOf(config, "status", false, azureSentinelIncidentStatuses),
			azureSentinelOneOf(config, "classification", false, azureSentinelClassifications),
			azureSentinelOneOf(config, "classificationReason", false, azureSentinelClassificationReasons),
		)
		return strings.Replace(msg, "properties.", "ModifyProperties actionConfiguration.", 1)
	default:
		return "properties.actions items need an actionType of ModifyProperties, RunPlaybook or AddIncidentTask."
	}
	return ""
}

// azureSentinelConnectorRules lists, for each data connector kind, whether
// it needs properties.tenantId and properties.dataTypes, and any other
// required properties.
var azureSentinelConnectorRules = map[string]struct {
	tenant, dataTypes bool
	other             []string
}{
	"AzureActiveDirectory":                          {tenant: true},
	"AzureAdvancedThreatProtection":                 {tenant: true},
	"MicrosoftDefenderAdvancedThreatProtection":     {tenant: true},
	"AzureSecurityCenter":                           {},
	"AmazonWebServicesCloudTrail":                   {dataTypes: true},
	"MicrosoftCloudAppSecurity":                     {tenant: true, dataTypes: true},
	"MicrosoftThreatIntelligence":                   {tenant: true, dataTypes: true},
	"Office365":                                     {tenant: true, dataTypes: true},
	"ThreatIntelligence":                            {tenant: true, dataTypes: true},
	"PremiumMicrosoftDefenderForThreatIntelligence": {tenant: true, dataTypes: true, other: []string{"lookbackPeriod"}},
	"RestApiPoller":                                 {other: []string{"connectorDefinitionName", "auth", "request"}},
}

func azureSentinelDataConnectorValidate(payload map[string]any) string {
	kind := azureARMString(payload["kind"])
	rules, ok := azureSentinelConnectorRules[kind]
	if !ok {
		kinds := make([]string, 0, len(azureSentinelConnectorRules))
		for k := range azureSentinelConnectorRules {
			kinds = append(kinds, k)
		}
		slices.Sort(kinds)
		return "kind must be one of " + strings.Join(kinds, ", ") + "."
	}
	props := azureARMObject(payload["properties"])
	if _, err := uuid.Parse(azureARMString(props["tenantId"])); rules.tenant && err != nil {
		return "properties.tenantId is required and must be a tenant GUID."
	}
	for _, key := range rules.other {
		if props[key] == nil {
			return "properties." + key + " is required."
		}
	}
	dataTypes, present := props["dataTypes"].(map[string]any)
	if rules.dataTypes && !present {
		return "properties.dataTypes is required."
	}
	for name, raw := range dataTypes {
		if state, ok := azureARMObject(raw)["state"]; ok && state != "Enabled" && state != "Disabled" {
			return "properties.dataTypes." + name + ".state must be Enabled or Disabled."
		}
	}
	return ""
}

func azureSentinelMetadataValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.HasPrefix(strings.ToLower(azureARMString(props["parentId"])), "/subscriptions/") {
		return "properties.parentId is required and must be the resource ID of the content's parent, such as the workspace."
	}
	return azureSentinelOneOf(props, "kind", true, azureSentinelMetadataKinds)
}
