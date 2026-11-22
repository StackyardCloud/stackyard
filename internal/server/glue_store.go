package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/services/athena"
)

// glueStore keeps every Glue resource as a record (the JSON members it was
// created with, flattened from the *Input member) under a resource kind.
// Most operations are plain create/get/list/update/delete on a kind and are
// served from the glueOps table; operations with real behaviour beyond CRUD
// (job runs, crawler states, table versions, schema versions, ...) are
// handled in glueHandlers.
type glueStore struct {
	mu       sync.Mutex
	records  map[string]map[string]glueRecord // kind -> identity key -> record
	tags     map[string]map[string]string     // resource ARN -> tags
	policies map[string]glueRecord            // resource ARN ("" = catalog) -> policy
	singles  map[string]glueRecord            // singleton settings by name
	athena   *athena.Service                  // the default catalog is mirrored into Athena's AwsDataCatalog
	seq      int
}

type glueRecord = map[string]any

type glueError struct {
	code string
	msg  string
}

func glueErr(code, format string, args ...any) *glueError {
	return &glueError{code: code, msg: fmt.Sprintf(format, args...)}
}

func glueNotFound(format string, args ...any) *glueError {
	return glueErr("EntityNotFoundException", format, args...)
}

func glueInvalid(format string, args ...any) *glueError {
	return glueErr("InvalidInputException", format, args...)
}

const (
	glueRegion  = "us-east-1"
	glueAccount = "123456789012"
	// glueMaxRecordsPerKind bounds each kind so one caller can't grow the
	// store without limit.
	glueMaxRecordsPerKind = 10000
	glueDefaultPageSize   = 100
)

// glueKind describes one resource kind.
type glueKind struct {
	fields       []string // identity members of the record, in order
	input        string   // request member flattened into the record on create/update
	parent       string   // kind that must exist first
	parentFields []string // record members naming the parent's identity
	idPrefix     string   // generate the last identity member when the request has none
	arnType      string   // ARN resource type (arn:aws:glue:...:<arnType>/<identity>)
	notFound     string   // error code for a missing record (default EntityNotFoundException)
	exists       string   // error code for a duplicate (default AlreadyExistsException)
}

var glueKinds = map[string]glueKind{
	"database":         {fields: []string{"Name"}, input: "DatabaseInput", arnType: "database"},
	"table":            {fields: []string{"DatabaseName", "Name"}, input: "TableInput", parent: "database", parentFields: []string{"DatabaseName"}, arnType: "table"},
	"partition":        {fields: []string{"DatabaseName", "TableName", "Values"}, input: "PartitionInput", parent: "table", parentFields: []string{"DatabaseName", "TableName"}},
	"udf":              {fields: []string{"DatabaseName", "FunctionName"}, input: "FunctionInput", parent: "database", parentFields: []string{"DatabaseName"}, arnType: "userDefinedFunction"},
	"connection":       {fields: []string{"Name"}, input: "ConnectionInput", arnType: "connection"},
	"catalog":          {fields: []string{"Name"}, input: "CatalogInput", arnType: "catalog"},
	"classifier":       {fields: []string{"Name"}},
	"crawler":          {fields: []string{"Name"}, arnType: "crawler"},
	"job":              {fields: []string{"Name"}, arnType: "job"},
	"jobrun":           {fields: []string{"JobName", "Id"}, parent: "job", parentFields: []string{"JobName"}},
	"trigger":          {fields: []string{"Name"}, arnType: "trigger"},
	"workflow":         {fields: []string{"Name"}, arnType: "workflow"},
	"workflowrun":      {fields: []string{"Name", "WorkflowRunId"}, parent: "workflow", parentFields: []string{"Name"}},
	"blueprint":        {fields: []string{"Name"}, arnType: "blueprint"},
	"blueprintrun":     {fields: []string{"BlueprintName", "RunId"}, parent: "blueprint", parentFields: []string{"BlueprintName"}},
	"devendpoint":      {fields: []string{"EndpointName"}, arnType: "devEndpoint"},
	"secconfig":        {fields: []string{"Name"}},
	"mltransform":      {fields: []string{"TransformId"}, idPrefix: "tfm-", arnType: "mlTransform"},
	"mltaskrun":        {fields: []string{"TransformId", "TaskRunId"}, parent: "mltransform", parentFields: []string{"TransformId"}, idPrefix: "tsk-"},
	"session":          {fields: []string{"Id"}, arnType: "session"},
	"statement":        {fields: []string{"SessionId", "Id"}, parent: "session", parentFields: []string{"SessionId"}},
	"dqruleset":        {fields: []string{"Name"}, arnType: "dataQualityRuleset"},
	"dqevalconfig":     {fields: []string{"Name"}},
	"dqevalrun":        {fields: []string{"RunId"}, idPrefix: "dqrun-"},
	"dqrecrun":         {fields: []string{"RunId"}, idPrefix: "dqrun-"},
	"dqresult":         {fields: []string{"ResultId"}, idPrefix: "dqresult-"},
	"customentitytype": {fields: []string{"Name"}, arnType: "customEntityType"},
	"tableoptimizer":   {fields: []string{"DatabaseName", "TableName", "Type"}, parent: "table", parentFields: []string{"DatabaseName", "TableName"}},
	"colstatsettings":  {fields: []string{"DatabaseName", "TableName"}, parent: "table", parentFields: []string{"DatabaseName", "TableName"}},
	"colstatsrun":      {fields: []string{"ColumnStatisticsTaskRunId"}, idPrefix: "csr-"},
	"usageprofile":     {fields: []string{"Name"}, arnType: "usageProfile"},
	"integration":      {fields: []string{"IntegrationArn"}},
	"intresprop":       {fields: []string{"ResourceArn"}},
	"inttableprop":     {fields: []string{"ResourceArn", "TableName"}},
	"glossary":         {fields: []string{"Id"}, idPrefix: "gls-"},
	"glossaryterm":     {fields: []string{"Id"}, idPrefix: "glt-", parent: "glossary", parentFields: []string{"GlossaryId"}},
	"asset":            {fields: []string{"Id"}},
	"assettype":        {fields: []string{"Id"}},
	"formtype":         {fields: []string{"Id"}},
	"attachment":       {fields: []string{"AssetIdentifier", "AttachmentName"}, parent: "asset", parentFields: []string{"AssetIdentifier"}},
	"connectiontype":   {fields: []string{"ConnectionType"}},
	"mvrefreshrun":     {fields: []string{"MaterializedViewRefreshTaskRunId"}, idPrefix: "mvr-"},
	"registry":         {fields: []string{"RegistryName"}, arnType: "registry"},
	"schema":           {fields: []string{"RegistryName", "SchemaName"}, parent: "registry", parentFields: []string{"RegistryName"}, arnType: "schema"},
	"upgradeanalysis":  {fields: []string{"jobName", "analysisId"}, parent: "job", parentFields: []string{"jobName"}},
}

// glueOp maps an operation onto a generic verb over a kind.
type glueOp struct {
	kind    string
	verb    string   // create | get | list | names | delete | update | batchget
	keys    []string // request paths for the identity (list: filters on same-named record members)
	out     string   // response member holding the record / list
	input   string   // update: request member merged into the record (default kind.input)
	listIn  string   // batchget: request member listing identities (last identity member)
	missing string   // batchget: response member listing identities not found
	name    string   // names: record member listed (default last identity member)
}

var glueOps = map[string]glueOp{
	"CreateDatabase": {kind: "database", verb: "create"},
	"GetDatabase":    {kind: "database", verb: "get", keys: []string{"Name"}, out: "Database"},
	"GetDatabases":   {kind: "database", verb: "list", out: "DatabaseList"},
	"UpdateDatabase": {kind: "database", verb: "update", keys: []string{"Name"}},
	"DeleteDatabase": {kind: "database", verb: "delete", keys: []string{"Name"}},

	"GetTable":    {kind: "table", verb: "get", keys: []string{"DatabaseName", "Name"}, out: "Table"},
	"GetTables":   {kind: "table", verb: "list", keys: []string{"DatabaseName"}, out: "TableList"},
	"DeleteTable": {kind: "table", verb: "delete", keys: []string{"DatabaseName", "Name"}},

	"CreatePartition": {kind: "partition", verb: "create", keys: []string{"DatabaseName", "TableName", "PartitionInput.Values"}},
	"GetPartition":    {kind: "partition", verb: "get", keys: []string{"DatabaseName", "TableName", "PartitionValues"}, out: "Partition"},
	"GetPartitions":   {kind: "partition", verb: "list", keys: []string{"DatabaseName", "TableName"}, out: "Partitions"},
	"UpdatePartition": {kind: "partition", verb: "update", keys: []string{"DatabaseName", "TableName", "PartitionValueList"}},
	"DeletePartition": {kind: "partition", verb: "delete", keys: []string{"DatabaseName", "TableName", "PartitionValues"}},

	"CreateUserDefinedFunction": {kind: "udf", verb: "create", keys: []string{"DatabaseName", "FunctionInput.FunctionName"}},
	"GetUserDefinedFunction":    {kind: "udf", verb: "get", keys: []string{"DatabaseName", "FunctionName"}, out: "UserDefinedFunction"},
	"UpdateUserDefinedFunction": {kind: "udf", verb: "update", keys: []string{"DatabaseName", "FunctionName"}},
	"DeleteUserDefinedFunction": {kind: "udf", verb: "delete", keys: []string{"DatabaseName", "FunctionName"}},

	"CreateConnection": {kind: "connection", verb: "create", keys: []string{"ConnectionInput.Name"}},
	"GetConnection":    {kind: "connection", verb: "get", keys: []string{"Name"}, out: "Connection"},
	"GetConnections":   {kind: "connection", verb: "list", out: "ConnectionList"},
	"UpdateConnection": {kind: "connection", verb: "update", keys: []string{"Name"}},
	"DeleteConnection": {kind: "connection", verb: "delete", keys: []string{"ConnectionName"}},

	"CreateCatalog": {kind: "catalog", verb: "create", keys: []string{"Name"}},
	"GetCatalog":    {kind: "catalog", verb: "get", keys: []string{"CatalogId"}, out: "Catalog"},
	"GetCatalogs":   {kind: "catalog", verb: "list", out: "CatalogList"},
	"UpdateCatalog": {kind: "catalog", verb: "update", keys: []string{"CatalogId"}},
	"DeleteCatalog": {kind: "catalog", verb: "delete", keys: []string{"CatalogId"}},

	"GetClassifier":    {kind: "classifier", verb: "get", keys: []string{"Name"}, out: "Classifier"},
	"GetClassifiers":   {kind: "classifier", verb: "list", out: "Classifiers"},
	"DeleteClassifier": {kind: "classifier", verb: "delete", keys: []string{"Name"}},

	"CreateCrawler":    {kind: "crawler", verb: "create"},
	"GetCrawler":       {kind: "crawler", verb: "get", keys: []string{"Name"}, out: "Crawler"},
	"GetCrawlers":      {kind: "crawler", verb: "list", out: "Crawlers"},
	"ListCrawlers":     {kind: "crawler", verb: "names", out: "CrawlerNames"},
	"BatchGetCrawlers": {kind: "crawler", verb: "batchget", listIn: "CrawlerNames", out: "Crawlers", missing: "CrawlersNotFound"},

	"CreateJob":    {kind: "job", verb: "create"},
	"GetJob":       {kind: "job", verb: "get", keys: []string{"JobName"}, out: "Job"},
	"GetJobs":      {kind: "job", verb: "list", out: "Jobs"},
	"ListJobs":     {kind: "job", verb: "names", out: "JobNames"},
	"BatchGetJobs": {kind: "job", verb: "batchget", listIn: "JobNames", out: "Jobs", missing: "JobsNotFound"},
	"UpdateJob":    {kind: "job", verb: "update", keys: []string{"JobName"}, input: "JobUpdate"},
	"DeleteJob":    {kind: "job", verb: "delete", keys: []string{"JobName"}},

	"GetJobRun":  {kind: "jobrun", verb: "get", keys: []string{"JobName", "RunId"}, out: "JobRun"},
	"GetJobRuns": {kind: "jobrun", verb: "list", keys: []string{"JobName"}, out: "JobRuns"},

	"CreateTrigger":    {kind: "trigger", verb: "create"},
	"GetTrigger":       {kind: "trigger", verb: "get", keys: []string{"Name"}, out: "Trigger"},
	"GetTriggers":      {kind: "trigger", verb: "list", out: "Triggers"},
	"ListTriggers":     {kind: "trigger", verb: "names", out: "TriggerNames"},
	"BatchGetTriggers": {kind: "trigger", verb: "batchget", listIn: "TriggerNames", out: "Triggers", missing: "TriggersNotFound"},
	"UpdateTrigger":    {kind: "trigger", verb: "update", keys: []string{"Name"}, input: "TriggerUpdate", out: "Trigger"},
	"DeleteTrigger":    {kind: "trigger", verb: "delete", keys: []string{"Name"}},

	"CreateWorkflow":    {kind: "workflow", verb: "create"},
	"GetWorkflow":       {kind: "workflow", verb: "get", keys: []string{"Name"}, out: "Workflow"},
	"ListWorkflows":     {kind: "workflow", verb: "names", out: "Workflows"},
	"BatchGetWorkflows": {kind: "workflow", verb: "batchget", listIn: "Names", out: "Workflows", missing: "MissingWorkflows"},
	"UpdateWorkflow":    {kind: "workflow", verb: "update", keys: []string{"Name"}},

	"GetWorkflowRun":  {kind: "workflowrun", verb: "get", keys: []string{"Name", "RunId"}, out: "Run"},
	"GetWorkflowRuns": {kind: "workflowrun", verb: "list", keys: []string{"Name"}, out: "Runs"},

	"CreateBlueprint":    {kind: "blueprint", verb: "create"},
	"GetBlueprint":       {kind: "blueprint", verb: "get", keys: []string{"Name"}, out: "Blueprint"},
	"ListBlueprints":     {kind: "blueprint", verb: "names", out: "Blueprints"},
	"BatchGetBlueprints": {kind: "blueprint", verb: "batchget", listIn: "Names", out: "Blueprints", missing: "MissingBlueprints"},
	"UpdateBlueprint":    {kind: "blueprint", verb: "update", keys: []string{"Name"}},
	"DeleteBlueprint":    {kind: "blueprint", verb: "delete", keys: []string{"Name"}},
	"GetBlueprintRun":    {kind: "blueprintrun", verb: "get", keys: []string{"BlueprintName", "RunId"}, out: "BlueprintRun"},
	"GetBlueprintRuns":   {kind: "blueprintrun", verb: "list", keys: []string{"BlueprintName"}, out: "BlueprintRuns"},

	"CreateDevEndpoint":    {kind: "devendpoint", verb: "create"},
	"GetDevEndpoint":       {kind: "devendpoint", verb: "get", keys: []string{"EndpointName"}, out: "DevEndpoint"},
	"GetDevEndpoints":      {kind: "devendpoint", verb: "list", out: "DevEndpoints"},
	"ListDevEndpoints":     {kind: "devendpoint", verb: "names", out: "DevEndpointNames"},
	"BatchGetDevEndpoints": {kind: "devendpoint", verb: "batchget", listIn: "DevEndpointNames", out: "DevEndpoints", missing: "DevEndpointsNotFound"},
	"DeleteDevEndpoint":    {kind: "devendpoint", verb: "delete", keys: []string{"EndpointName"}},

	"CreateSecurityConfiguration": {kind: "secconfig", verb: "create"},
	"GetSecurityConfiguration":    {kind: "secconfig", verb: "get", keys: []string{"Name"}, out: "SecurityConfiguration"},
	"GetSecurityConfigurations":   {kind: "secconfig", verb: "list", out: "SecurityConfigurations"},
	"DeleteSecurityConfiguration": {kind: "secconfig", verb: "delete", keys: []string{"Name"}},

	"CreateMLTransform": {kind: "mltransform", verb: "create"},
	"GetMLTransform":    {kind: "mltransform", verb: "get", keys: []string{"TransformId"}},
	"GetMLTransforms":   {kind: "mltransform", verb: "list", out: "Transforms"},
	"ListMLTransforms":  {kind: "mltransform", verb: "names", out: "TransformIds"},
	"UpdateMLTransform": {kind: "mltransform", verb: "update", keys: []string{"TransformId"}},
	"DeleteMLTransform": {kind: "mltransform", verb: "delete", keys: []string{"TransformId"}},
	"GetMLTaskRun":      {kind: "mltaskrun", verb: "get", keys: []string{"TransformId", "TaskRunId"}},
	"GetMLTaskRuns":     {kind: "mltaskrun", verb: "list", keys: []string{"TransformId"}, out: "TaskRuns"},

	"GetSession":     {kind: "session", verb: "get", keys: []string{"Id"}, out: "Session"},
	"DeleteSession":  {kind: "session", verb: "delete", keys: []string{"Id"}},
	"GetStatement":   {kind: "statement", verb: "get", keys: []string{"SessionId", "Id"}, out: "Statement"},
	"ListStatements": {kind: "statement", verb: "list", keys: []string{"SessionId"}, out: "Statements"},

	"CreateDataQualityRuleset": {kind: "dqruleset", verb: "create"},
	"GetDataQualityRuleset":    {kind: "dqruleset", verb: "get", keys: []string{"Name"}},
	"ListDataQualityRulesets":  {kind: "dqruleset", verb: "list", out: "Rulesets"},
	"UpdateDataQualityRuleset": {kind: "dqruleset", verb: "update", keys: []string{"Name"}},
	"DeleteDataQualityRuleset": {kind: "dqruleset", verb: "delete", keys: []string{"Name"}},

	"CreateDataQualityEvaluationConfig": {kind: "dqevalconfig", verb: "create"},
	"GetDataQualityEvaluationConfig":    {kind: "dqevalconfig", verb: "get", keys: []string{"Name"}},
	"ListDataQualityEvaluationConfigs":  {kind: "dqevalconfig", verb: "list", out: "DataQualityEvaluationConfigs"},
	"UpdateDataQualityEvaluationConfig": {kind: "dqevalconfig", verb: "update", keys: []string{"Name"}},
	"DeleteDataQualityEvaluationConfig": {kind: "dqevalconfig", verb: "delete", keys: []string{"Name"}},

	"GetDataQualityRulesetEvaluationRun":      {kind: "dqevalrun", verb: "get", keys: []string{"RunId"}},
	"ListDataQualityRulesetEvaluationRuns":    {kind: "dqevalrun", verb: "list", out: "Runs"},
	"BatchGetDataQualityRulesetEvaluationRun": {kind: "dqevalrun", verb: "batchget", listIn: "RunIds", out: "Runs", missing: "RunsNotFound"},
	"GetDataQualityRuleRecommendationRun":     {kind: "dqrecrun", verb: "get", keys: []string{"RunId"}},
	"ListDataQualityRuleRecommendationRuns":   {kind: "dqrecrun", verb: "list", out: "Runs"},
	"GetDataQualityResult":                    {kind: "dqresult", verb: "get", keys: []string{"ResultId"}},
	"GetDataQualityResultV2":                  {kind: "dqresult", verb: "get", keys: []string{"ResultId"}},
	"ListDataQualityResults":                  {kind: "dqresult", verb: "list", out: "Results"},
	"ListDataQualityResultsV2":                {kind: "dqresult", verb: "list", out: "Results"},
	"BatchGetDataQualityResult":               {kind: "dqresult", verb: "batchget", listIn: "ResultIds", out: "Results", missing: "ResultsNotFound"},
	"BatchGetDataQualityResultV2":             {kind: "dqresult", verb: "batchget", listIn: "ResultIds", out: "Results", missing: "ResultsNotFound"},

	"CreateCustomEntityType":    {kind: "customentitytype", verb: "create"},
	"GetCustomEntityType":       {kind: "customentitytype", verb: "get", keys: []string{"Name"}},
	"ListCustomEntityTypes":     {kind: "customentitytype", verb: "list", out: "CustomEntityTypes"},
	"BatchGetCustomEntityTypes": {kind: "customentitytype", verb: "batchget", listIn: "Names", out: "CustomEntityTypes", missing: "CustomEntityTypesNotFound"},
	"DeleteCustomEntityType":    {kind: "customentitytype", verb: "delete", keys: []string{"Name"}},

	"CreateTableOptimizer": {kind: "tableoptimizer", verb: "create", keys: []string{"DatabaseName", "TableName", "Type"}},
	"GetTableOptimizer":    {kind: "tableoptimizer", verb: "get", keys: []string{"DatabaseName", "TableName", "Type"}, out: "TableOptimizer"},
	"UpdateTableOptimizer": {kind: "tableoptimizer", verb: "update", keys: []string{"DatabaseName", "TableName", "Type"}},
	"DeleteTableOptimizer": {kind: "tableoptimizer", verb: "delete", keys: []string{"DatabaseName", "TableName", "Type"}},

	"CreateColumnStatisticsTaskSettings": {kind: "colstatsettings", verb: "create", keys: []string{"DatabaseName", "TableName"}},
	"GetColumnStatisticsTaskSettings":    {kind: "colstatsettings", verb: "get", keys: []string{"DatabaseName", "TableName"}, out: "ColumnStatisticsTaskSettings"},
	"UpdateColumnStatisticsTaskSettings": {kind: "colstatsettings", verb: "update", keys: []string{"DatabaseName", "TableName"}},
	"DeleteColumnStatisticsTaskSettings": {kind: "colstatsettings", verb: "delete", keys: []string{"DatabaseName", "TableName"}},
	"GetColumnStatisticsTaskRun":         {kind: "colstatsrun", verb: "get", keys: []string{"ColumnStatisticsTaskRunId"}, out: "ColumnStatisticsTaskRun"},
	"GetColumnStatisticsTaskRuns":        {kind: "colstatsrun", verb: "list", keys: []string{"DatabaseName", "TableName"}, out: "ColumnStatisticsTaskRuns"},
	"ListColumnStatisticsTaskRuns":       {kind: "colstatsrun", verb: "names", out: "ColumnStatisticsTaskRunIds"},

	"CreateUsageProfile": {kind: "usageprofile", verb: "create"},
	"GetUsageProfile":    {kind: "usageprofile", verb: "get", keys: []string{"Name"}},
	"ListUsageProfiles":  {kind: "usageprofile", verb: "list", out: "Profiles"},
	"UpdateUsageProfile": {kind: "usageprofile", verb: "update", keys: []string{"Name"}},
	"DeleteUsageProfile": {kind: "usageprofile", verb: "delete", keys: []string{"Name"}},

	"DescribeIntegrations": {kind: "integration", verb: "list", out: "Integrations"},

	"CreateIntegrationResourceProperty": {kind: "intresprop", verb: "create"},
	"GetIntegrationResourceProperty":    {kind: "intresprop", verb: "get", keys: []string{"ResourceArn"}},
	"UpdateIntegrationResourceProperty": {kind: "intresprop", verb: "update", keys: []string{"ResourceArn"}},
	"DeleteIntegrationResourceProperty": {kind: "intresprop", verb: "delete", keys: []string{"ResourceArn"}},
	"ListIntegrationResourceProperties": {kind: "intresprop", verb: "list", out: "IntegrationResourcePropertyList"},

	"CreateIntegrationTableProperties": {kind: "inttableprop", verb: "create"},
	"GetIntegrationTableProperties":    {kind: "inttableprop", verb: "get", keys: []string{"ResourceArn", "TableName"}},
	"UpdateIntegrationTableProperties": {kind: "inttableprop", verb: "update", keys: []string{"ResourceArn", "TableName"}},
	"DeleteIntegrationTableProperties": {kind: "inttableprop", verb: "delete", keys: []string{"ResourceArn", "TableName"}},
	"ListIntegrationTableProperties":   {kind: "inttableprop", verb: "list", out: "IntegrationTablePropertiesList"},

	"CreateGlossary":     {kind: "glossary", verb: "create"},
	"GetGlossary":        {kind: "glossary", verb: "get", keys: []string{"Identifier"}},
	"ListGlossaries":     {kind: "glossary", verb: "list", out: "Items"},
	"UpdateGlossary":     {kind: "glossary", verb: "update", keys: []string{"Identifier"}},
	"GetGlossaryTerm":    {kind: "glossaryterm", verb: "get", keys: []string{"Identifier"}},
	"UpdateGlossaryTerm": {kind: "glossaryterm", verb: "update", keys: []string{"Identifier"}},
	"DeleteGlossaryTerm": {kind: "glossaryterm", verb: "delete", keys: []string{"Identifier"}},

	"GetAsset":         {kind: "asset", verb: "get", keys: []string{"Identifier"}},
	"UpdateAsset":      {kind: "asset", verb: "update", keys: []string{"Identifier"}},
	"DeleteAsset":      {kind: "asset", verb: "delete", keys: []string{"Identifier"}},
	"SearchAssets":     {kind: "asset", verb: "list", out: "Items"},
	"GetAssetType":     {kind: "assettype", verb: "get", keys: []string{"Identifier"}},
	"ListAssetTypes":   {kind: "assettype", verb: "list", out: "Items"},
	"DeleteAssetType":  {kind: "assettype", verb: "delete", keys: []string{"Identifier"}},
	"GetFormType":      {kind: "formtype", verb: "get", keys: []string{"Identifier"}},
	"ListFormTypes":    {kind: "formtype", verb: "list", out: "Items"},
	"DeleteFormType":   {kind: "formtype", verb: "delete", keys: []string{"Identifier"}},
	"DeleteAttachment": {kind: "attachment", verb: "delete", keys: []string{"AssetIdentifier", "AttachmentName"}},

	"DescribeConnectionType": {kind: "connectiontype", verb: "get", keys: []string{"ConnectionType"}},
	"ListConnectionTypes":    {kind: "connectiontype", verb: "list", out: "ConnectionTypes"},
	"DeleteConnectionType":   {kind: "connectiontype", verb: "delete", keys: []string{"ConnectionType"}},

	"GetMaterializedViewRefreshTaskRun":   {kind: "mvrefreshrun", verb: "get", keys: []string{"MaterializedViewRefreshTaskRunId"}, out: "MaterializedViewRefreshTaskRun"},
	"ListMaterializedViewRefreshTaskRuns": {kind: "mvrefreshrun", verb: "list", keys: []string{"DatabaseName", "TableName"}, out: "MaterializedViewRefreshTaskRuns"},

	"GetJobUpgradeAnalysis": {kind: "upgradeanalysis", verb: "get", keys: []string{"jobName", "analysisId"}},
}

func newGlueStore(ath *athena.Service) *glueStore {
	st := &glueStore{
		records:  map[string]map[string]glueRecord{},
		tags:     map[string]map[string]string{},
		policies: map[string]glueRecord{},
		singles:  map[string]glueRecord{},
		athena:   ath,
	}
	// Athena's AwsDataCatalog ships with a "default" database; Glue has it too.
	_, _ = st.create("database", glueRecord{"Name": "default", "Description": "Default Hive database"}, nil)
	for _, t := range glueBuiltinConnectionTypes {
		_, _ = st.create("connectiontype", glueWrap(t, ""), nil)
	}
	return st
}

// handle runs one operation. The input has already passed model validation.
// handle runs op and shapes its result to outShape while still holding the
// lock: handler results can alias stored records, so they must not be read
// after unlock (a concurrent write would be a fatal map race).
func (st *glueStore) handle(op, outShape string, in glueRecord) (any, *glueError) {
	// Request members never start with "_"; those names are the store's own
	// bookkeeping (_next, _reads, _indexes, ...), so a caller can't set them.
	for k := range in {
		if strings.HasPrefix(k, "_") {
			delete(in, k)
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	var out glueRecord
	var err *glueError
	if h, ok := glueHandlers[op]; ok {
		out, err = h(st, in)
	} else if spec, ok := glueOps[op]; ok {
		out, err = st.generic(spec, in)
	} else {
		return nil, glueErr("InternalFailure", "operation %s has no handler", op)
	}
	if err != nil || outShape == "" {
		return glueRecord{}, err
	}
	return glueConform(outShape, out), nil
}

func (st *glueStore) generic(spec glueOp, in glueRecord) (glueRecord, *glueError) {
	switch spec.verb {
	case "create":
		rec, err := st.createFrom(spec, in)
		if err != nil {
			return nil, err
		}
		return glueWrap(rec, spec.out), nil
	case "get":
		rec, err := st.find(spec.kind, glueKeyValues(in, spec.keys))
		if err != nil {
			return nil, err
		}
		return glueWrap(rec, spec.out), nil
	case "list", "names":
		return st.list(spec, in)
	case "update":
		rec, err := st.find(spec.kind, glueKeyValues(in, spec.keys))
		if err != nil {
			return nil, err
		}
		st.merge(spec.kind, rec, in, glueFirst(spec.input, glueKinds[spec.kind].input), spec.keys)
		return glueMerge(glueWrap(rec, spec.out), in), nil
	case "delete":
		rec, err := st.find(spec.kind, glueKeyValues(in, spec.keys))
		if err != nil {
			return nil, err
		}
		st.remove(spec.kind, rec)
		return glueMerge(rec, in), nil
	case "batchget":
		names, _ := in[spec.listIn].([]any)
		found, missing := []any{}, []any{}
		for _, n := range names {
			if rec, err := st.find(spec.kind, []any{n}); err == nil {
				found = append(found, rec)
			} else {
				missing = append(missing, n)
			}
		}
		return glueRecord{spec.out: found, spec.missing: missing}, nil
	}
	return glueRecord{}, nil
}

// createFrom builds a record from a create request and stores it.
func (st *glueStore) createFrom(spec glueOp, in glueRecord) (glueRecord, *glueError) {
	kind := glueKinds[spec.kind]
	rec := glueRecord{}
	for k, v := range in {
		if k != kind.input && k != "Tags" && k != "ClientToken" {
			rec[k] = v
		}
	}
	if nested, ok := in[kind.input].(map[string]any); ok {
		for k, v := range nested {
			rec[k] = v
		}
	}
	if len(spec.keys) > 0 {
		for i, v := range glueKeyValues(in, spec.keys) {
			if v != nil {
				rec[kind.fields[i]] = v
			}
		}
	}
	return st.create(spec.kind, rec, glueTags(in["Tags"]))
}

// create stores rec under its kind, generating the id when the kind allows.
func (st *glueStore) create(kindName string, rec glueRecord, tags map[string]string) (glueRecord, *glueError) {
	kind := glueKinds[kindName]
	last := kind.fields[len(kind.fields)-1]
	if rec[last] == nil && kind.idPrefix != "" {
		rec[last] = st.newID(kind.idPrefix)
	}
	ids := make([]any, len(kind.fields))
	for i, f := range kind.fields {
		if rec[f] == nil {
			return nil, glueInvalid("%s is required", f)
		}
		ids[i] = rec[f]
	}
	if kind.parent != "" {
		pids := make([]any, len(kind.parentFields))
		for i, f := range kind.parentFields {
			pids[i] = rec[f]
		}
		if _, err := st.find(kind.parent, pids); err != nil {
			return nil, err
		}
	}
	bucket := st.records[kindName]
	if bucket == nil {
		bucket = map[string]glueRecord{}
		st.records[kindName] = bucket
	}
	key := glueKey(ids)
	if _, exists := bucket[key]; exists {
		return nil, glueErr(glueFirst(kind.exists, "AlreadyExistsException"), "%s %s already exists", kindName, glueIDText(ids))
	}
	if len(bucket) >= glueMaxRecordsPerKind {
		return nil, glueErr("ResourceNumberLimitExceededException", "too many %s resources", kindName)
	}
	now := glueEpoch(time.Now())
	for _, f := range glueCreatedFields {
		if rec[f] == nil {
			rec[f] = now
		}
	}
	glueTouch(rec)
	bucket[key] = rec
	if kind.arnType != "" && len(tags) > 0 {
		st.tags[st.arn(kindName, rec)] = tags
	}
	return rec, nil
}

// Response shapes name their timestamps differently; each record carries all
// the spellings and shaping keeps the ones the response has.
var (
	glueCreatedFields = []string{"CreateTime", "CreatedOn", "CreationTime", "CreatedTime", "CreatedTimestamp", "CreatedAt", "CreateDate", "StartedOn"}
	glueUpdatedFields = []string{"UpdateTime", "LastModifiedOn", "LastUpdated", "UpdatedTime", "LastModifiedTimestamp", "UpdatedAt", "LastUpdatedOn"}
)

func glueTouch(rec glueRecord) {
	now := glueEpoch(time.Now())
	for _, f := range glueUpdatedFields {
		rec[f] = now
	}
}

func (st *glueStore) find(kindName string, ids []any) (glueRecord, *glueError) {
	kind := glueKinds[kindName]
	for _, id := range ids {
		if id == nil {
			return nil, glueInvalid("%s identity is required", kindName)
		}
	}
	if rec, ok := st.records[kindName][glueKey(ids)]; ok {
		return rec, nil
	}
	return nil, glueErr(glueFirst(kind.notFound, "EntityNotFoundException"), "%s %s not found", kindName, glueIDText(ids))
}

// remove deletes rec and, recursively, every child record and its tags.
func (st *glueStore) remove(kindName string, rec glueRecord) {
	kind := glueKinds[kindName]
	ids := make([]any, len(kind.fields))
	for i, f := range kind.fields {
		ids[i] = rec[f]
	}
	delete(st.records[kindName], glueKey(ids))
	if kind.arnType != "" {
		delete(st.tags, st.arn(kindName, rec))
	}
	for childName, child := range glueKinds {
		if child.parent != kindName {
			continue
		}
		for _, c := range st.records[childName] {
			match := true
			for i, f := range child.parentFields {
				if glueKey([]any{c[f]}) != glueKey([]any{ids[i]}) {
					match = false
					break
				}
			}
			if match {
				st.remove(childName, c)
			}
		}
	}
}

// merge applies an update request: the nested input member plus top-level
// members other than the identity.
func (st *glueStore) merge(kindName string, rec, in glueRecord, inputMember string, keys []string) {
	skip := map[string]bool{"Tags": true, "ClientToken": true, inputMember: true}
	for _, k := range keys {
		skip[strings.SplitN(k, ".", 2)[0]] = true
	}
	for k, v := range in {
		if !skip[k] {
			rec[k] = v
		}
	}
	if nested, ok := in[inputMember].(map[string]any); ok {
		kind := glueKinds[kindName]
		for k, v := range nested {
			if !glueContains(kind.fields, k) {
				rec[k] = v
			}
		}
	}
	glueTouch(rec)
}

// list returns a page of records (or of identities, for "names") filtered by
// the request members named in spec.keys and by Tags when given.
func (st *glueStore) list(spec glueOp, in glueRecord) (glueRecord, *glueError) {
	kind := glueKinds[spec.kind]
	if kind.parent != "" && len(spec.keys) == len(kind.parentFields) {
		if _, err := st.find(kind.parent, glueKeyValues(in, spec.keys)); err != nil {
			return nil, err
		}
	}
	filters := map[string]any{}
	for _, k := range spec.keys {
		if v := glueAt(in, k); v != nil {
			filters[k] = v
		}
	}
	wantTags := glueTags(in["Tags"])
	keys := make([]string, 0, len(st.records[spec.kind]))
	for key, rec := range st.records[spec.kind] {
		ok := true
		for f, v := range filters {
			if glueKey([]any{rec[f]}) != glueKey([]any{v}) {
				ok = false
				break
			}
		}
		if ok && len(wantTags) > 0 && kind.arnType != "" {
			have := st.tags[st.arn(spec.kind, rec)]
			for tk, tv := range wantTags {
				if have[tk] != tv {
					ok = false
					break
				}
			}
		}
		if ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	page, next, err := gluePage(keys, in)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	nameField := glueFirst(spec.name, kind.fields[len(kind.fields)-1])
	for _, key := range page {
		rec := st.records[spec.kind][key]
		if spec.verb == "names" {
			items = append(items, rec[nameField])
		} else {
			items = append(items, rec)
		}
	}
	out := glueRecord{spec.out: items}
	if next != "" {
		out["NextToken"] = next
		out["Marker"] = next
	}
	return out, nil
}

// gluePage applies NextToken/Marker and MaxResults/MaxRecords.
func gluePage(keys []string, in glueRecord) ([]string, string, *glueError) {
	start := 0
	if tok, _ := glueFirstValue(in, "NextToken", "Marker").(string); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(keys) {
			return nil, "", glueInvalid("invalid NextToken")
		}
		start = int(n)
	}
	size := glueDefaultPageSize
	if v, ok := glueFirstValue(in, "MaxResults", "MaxRecords").(float64); ok && v > 0 && v < float64(size) {
		size = int(v)
	}
	end := min(start+size, len(keys))
	next := ""
	if end < len(keys) {
		next = strconv.Itoa(end)
	}
	return keys[start:end], next, nil
}

func (st *glueStore) arn(kindName string, rec glueRecord) string {
	kind := glueKinds[kindName]
	parts := make([]string, len(kind.fields))
	for i, f := range kind.fields {
		parts[i] = fmt.Sprint(rec[f])
	}
	return fmt.Sprintf("arn:aws:glue:%s:%s:%s/%s", glueRegion, glueAccount, kind.arnType, strings.Join(parts, "/"))
}

func (st *glueStore) newID(prefix string) string {
	st.seq++
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s%s%d", prefix, hex.EncodeToString(b), st.seq)
}

func (st *glueStore) all(kindName string) []glueRecord {
	keys := make([]string, 0, len(st.records[kindName]))
	for k := range st.records[kindName] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]glueRecord, 0, len(keys))
	for _, k := range keys {
		out = append(out, st.records[kindName][k])
	}
	return out
}

// glueWrap returns rec's members plus rec itself under member (when set), so
// one record answers both flat and wrapped response shapes.
func glueWrap(rec glueRecord, member string) glueRecord {
	out := glueRecord{}
	for k, v := range rec {
		out[k] = v
	}
	if member != "" {
		out[member] = rec
	}
	return out
}

func glueMerge(a, b glueRecord) glueRecord {
	out := glueRecord{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

// glueAt reads a dotted request path.
func glueAt(in glueRecord, path string) any {
	var cur any = in
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	return cur
}

func glueKeyValues(in glueRecord, paths []string) []any {
	out := make([]any, len(paths))
	for i, p := range paths {
		out[i] = glueAt(in, p)
	}
	return out
}

func glueKey(ids []any) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		if s, ok := id.(string); ok {
			parts[i] = s
			continue
		}
		b, _ := json.Marshal(id)
		parts[i] = string(b)
	}
	return strings.Join(parts, "\x1f")
}

func glueIDText(ids []any) string {
	return strings.ReplaceAll(glueKey(ids), "\x1f", "/")
}

func glueTags(v any) map[string]string {
	m, _ := v.(map[string]any)
	out := map[string]string{}
	for k, val := range m {
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}

func glueStr(in glueRecord, key string) string {
	s, _ := in[key].(string)
	return s
}

func glueFirst(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func glueFirstValue(in glueRecord, keys ...string) any {
	for _, k := range keys {
		if v, ok := in[k]; ok && v != nil {
			return v
		}
	}
	return nil
}
