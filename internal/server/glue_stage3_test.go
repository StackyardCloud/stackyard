package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// Schema Registry, sessions and statements, Data Quality, tags, resource
// policies and the smaller resource types.

func TestGlueStage3SchemaRegistry(t *testing.T) {
	_, ts := newGlueTestServer(t)
	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"string"}]}`
	addOptional := `{"type":"record","name":"User","fields":[{"name":"id","type":"string"},{"name":"email","type":"string","default":""}]}`
	addRequired := `{"type":"record","name":"User","fields":[{"name":"id","type":"string"},{"name":"age","type":"int"}]}`
	q := strconv.Quote

	glueCall(t, ts, http.StatusOK, "CreateRegistry", `{"RegistryName":"events","Description":"event schemas"}`)
	if code := glueErrorType(t, ts, "CreateRegistry", `{"RegistryName":"events"}`); code != "AlreadyExistsException" {
		t.Fatalf("duplicate registry: %s", code)
	}
	if code := glueErrorType(t, ts, "CreateSchema", `{"RegistryId":{"RegistryName":"events"},"SchemaName":"user","DataFormat":"AVRO","SchemaDefinition":"{not json"}`); code != "InvalidInputException" {
		t.Fatalf("invalid definition: %s", code)
	}
	created := glueCall(t, ts, http.StatusOK, "CreateSchema", `{"RegistryId":{"RegistryName":"events"},"SchemaName":"user","DataFormat":"AVRO","Compatibility":"BACKWARD","SchemaDefinition":`+q(v1)+`}`)
	if created["SchemaVersionStatus"] != "AVAILABLE" || created["LatestSchemaVersion"] != 1.0 || !strings.HasSuffix(created["SchemaArn"].(string), ":schema/events/user") {
		t.Fatalf("create schema: %v", created)
	}
	schemaID := `{"RegistryName":"events","SchemaName":"user"}`

	// BACKWARD allows a new field with a default, not one without.
	ok := glueCall(t, ts, http.StatusOK, "RegisterSchemaVersion", `{"SchemaId":`+schemaID+`,"SchemaDefinition":`+q(addOptional)+`}`)
	if ok["Status"] != "AVAILABLE" || ok["VersionNumber"] != 2.0 {
		t.Fatalf("compatible version: %v", ok)
	}
	again := glueCall(t, ts, http.StatusOK, "RegisterSchemaVersion", `{"SchemaId":`+schemaID+`,"SchemaDefinition":`+q(addOptional)+`}`)
	if again["SchemaVersionId"] != ok["SchemaVersionId"] {
		t.Fatalf("re-registering returns the same version: %v", again)
	}
	bad := glueCall(t, ts, http.StatusOK, "RegisterSchemaVersion", `{"SchemaId":`+schemaID+`,"SchemaDefinition":`+q(addRequired)+`}`)
	if bad["Status"] != "FAILURE" {
		t.Fatalf("incompatible version: %v", bad)
	}
	latest := glueCall(t, ts, http.StatusOK, "GetSchemaVersion", `{"SchemaId":`+schemaID+`,"SchemaVersionNumber":{"LatestVersion":true}}`)
	if latest["VersionNumber"] != 2.0 || latest["SchemaDefinition"] != addOptional {
		t.Fatalf("latest version: %v", latest)
	}
	if byDef := glueCall(t, ts, http.StatusOK, "GetSchemaByDefinition", `{"SchemaId":`+schemaID+`,"SchemaDefinition":`+q(v1)+`}`); byDef["SchemaVersionId"] == nil {
		t.Fatalf("by definition: %v", byDef)
	}
	diff := glueCall(t, ts, http.StatusOK, "GetSchemaVersionsDiff", `{"SchemaId":`+schemaID+`,"FirstSchemaVersionNumber":{"VersionNumber":1},"SecondSchemaVersionNumber":{"VersionNumber":2},"SchemaDiffType":"SYNTAX_DIFF"}`)
	if !strings.Contains(diff["Diff"].(string), "+") {
		t.Fatalf("diff: %v", diff)
	}

	// Metadata on a version.
	glueCall(t, ts, http.StatusOK, "PutSchemaVersionMetadata", `{"SchemaVersionId":"`+ok["SchemaVersionId"].(string)+`","MetadataKeyValue":{"MetadataKey":"owner","MetadataValue":"team-a"}}`)
	if code := glueErrorType(t, ts, "PutSchemaVersionMetadata", `{"SchemaVersionId":"`+ok["SchemaVersionId"].(string)+`","MetadataKeyValue":{"MetadataKey":"owner","MetadataValue":"team-a"}}`); code != "AlreadyExistsException" {
		t.Fatalf("duplicate metadata: %s", code)
	}
	meta := glueCall(t, ts, http.StatusOK, "QuerySchemaVersionMetadata", `{"SchemaVersionId":"`+ok["SchemaVersionId"].(string)+`"}`)["MetadataInfoMap"].(map[string]any)
	if meta["owner"].(map[string]any)["MetadataValue"] != "team-a" {
		t.Fatalf("metadata: %v", meta)
	}
	glueCall(t, ts, http.StatusOK, "RemoveSchemaVersionMetadata", `{"SchemaVersionId":"`+ok["SchemaVersionId"].(string)+`","MetadataKeyValue":{"MetadataKey":"owner","MetadataValue":"team-a"}}`)

	// The checkpoint (version 1) can't be deleted; others can.
	errs := glueCall(t, ts, http.StatusOK, "DeleteSchemaVersions", `{"SchemaId":`+schemaID+`,"Versions":"1-3"}`)["SchemaVersionErrors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["VersionNumber"] != 1.0 {
		t.Fatalf("delete versions: %v", errs)
	}
	if code := glueErrorType(t, ts, "DeleteSchemaVersions", `{"SchemaId":`+schemaID+`,"Versions":"3-1"}`); code != "InvalidInputException" {
		t.Fatalf("bad range: %s", code)
	}

	glueCall(t, ts, http.StatusOK, "UpdateSchema", `{"SchemaId":`+schemaID+`,"Compatibility":"DISABLED"}`)
	if code := glueErrorType(t, ts, "RegisterSchemaVersion", `{"SchemaId":`+schemaID+`,"SchemaDefinition":`+q(addRequired)+`}`); code != "InvalidInputException" {
		t.Fatalf("registering with DISABLED: %s", code)
	}
	if v := glueCall(t, ts, http.StatusOK, "CheckSchemaVersionValidity", `{"DataFormat":"JSON","SchemaDefinition":"{\"type\":\"object\"}"}`); v["Valid"] != true {
		t.Fatalf("validity: %v", v)
	}

	// A schema without RegistryId lands in default-registry.
	glueCall(t, ts, http.StatusOK, "CreateSchema", `{"SchemaName":"loose","DataFormat":"JSON"}`)
	if reg := glueCall(t, ts, http.StatusOK, "GetRegistry", `{"RegistryId":{"RegistryName":"default-registry"}}`); reg["Status"] != "AVAILABLE" {
		t.Fatalf("default registry: %v", reg)
	}
	if out := glueCall(t, ts, http.StatusOK, "DeleteRegistry", `{"RegistryId":{"RegistryName":"events"}}`); out["Status"] != "DELETING" {
		t.Fatalf("delete registry: %v", out)
	}
	if code := glueErrorType(t, ts, "GetSchema", `{"SchemaId":`+schemaID+`}`); code != "EntityNotFoundException" {
		t.Fatalf("schema after its registry was deleted: %s", code)
	}
}

func TestGlueStage3SessionsAndStatements(t *testing.T) {
	_, ts := newGlueTestServer(t)
	s := glueCall(t, ts, http.StatusOK, "CreateSession", `{"Id":"s1","Role":"arn:aws:iam::123456789012:role/glue","Command":{"Name":"glueetl","PythonVersion":"3"}}`)["Session"].(map[string]any)
	if s["Status"] != "PROVISIONING" {
		t.Fatalf("create session: %v", s)
	}
	id := glueCall(t, ts, http.StatusOK, "RunStatement", `{"SessionId":"s1","Code":"print(1)"}`)["Id"]
	if id != 0.0 {
		t.Fatalf("first statement id: %v", id)
	}
	glueCall(t, ts, http.StatusOK, "GetStatement", `{"SessionId":"s1","Id":0}`)
	st := glueCall(t, ts, http.StatusOK, "GetStatement", `{"SessionId":"s1","Id":0}`)["Statement"].(map[string]any)
	if st["State"] != "AVAILABLE" || st["Output"].(map[string]any)["Status"] != "ok" {
		t.Fatalf("statement: %v", st)
	}
	if code := glueErrorType(t, ts, "CancelStatement", `{"SessionId":"s1","Id":0}`); code != "IllegalSessionStateException" {
		t.Fatalf("cancelling a finished statement: %s", code)
	}
	ids := glueCall(t, ts, http.StatusOK, "ListSessions", `{}`)["Ids"].([]any)
	if len(ids) != 1 || ids[0] != "s1" {
		t.Fatalf("session ids: %v", ids)
	}
	glueCall(t, ts, http.StatusOK, "StopSession", `{"Id":"s1"}`)
	if code := glueErrorType(t, ts, "RunStatement", `{"SessionId":"s1","Code":"1"}`); code != "IllegalSessionStateException" {
		t.Fatalf("statement on a stopped session: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "DeleteSession", `{"Id":"s1"}`)
	if code := glueErrorType(t, ts, "GetSession", `{"Id":"s1"}`); code != "EntityNotFoundException" {
		t.Fatalf("deleted session: %s", code)
	}
}

func TestGlueStage3DataQuality(t *testing.T) {
	_, ts := newGlueTestServer(t)
	if code := glueErrorType(t, ts, "CreateDataQualityRuleset", `{"Name":"rs","Ruleset":"not dqdl"}`); code != "InvalidInputException" {
		t.Fatalf("invalid ruleset: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "CreateDataQualityRuleset", `{"Name":"rs","Ruleset":"Rules = [ RowCount > 0, IsComplete \"id\" ]"}`)
	src := `{"GlueTable":{"DatabaseName":"d","TableName":"t"}}`
	if code := glueErrorType(t, ts, "StartDataQualityRulesetEvaluationRun", `{"DataSource":`+src+`,"Role":"r","RulesetNames":["missing"]}`); code != "EntityNotFoundException" {
		t.Fatalf("evaluation with a missing ruleset: %s", code)
	}
	run := glueCall(t, ts, http.StatusOK, "StartDataQualityRulesetEvaluationRun", `{"DataSource":`+src+`,"Role":"r","RulesetNames":["rs"]}`)["RunId"].(string)
	glueCall(t, ts, http.StatusOK, "GetDataQualityRulesetEvaluationRun", `{"RunId":"`+run+`"}`)
	done := glueCall(t, ts, http.StatusOK, "GetDataQualityRulesetEvaluationRun", `{"RunId":"`+run+`"}`)
	results := done["ResultIds"].([]any)
	if done["Status"] != "SUCCEEDED" || len(results) != 1 {
		t.Fatalf("evaluation run: %v", done)
	}
	res := glueCall(t, ts, http.StatusOK, "GetDataQualityResult", `{"ResultId":"`+results[0].(string)+`"}`)
	if res["Score"] != 1.0 || len(res["RuleResults"].([]any)) != 2 {
		t.Fatalf("result: %v", res)
	}
	got := glueCall(t, ts, http.StatusOK, "BatchGetDataQualityResult", `{"ResultIds":["`+results[0].(string)+`","dqresult-nope"]}`)
	if len(got["Results"].([]any)) != 1 || len(got["ResultsNotFound"].([]any)) != 1 {
		t.Fatalf("batch results: %v", got)
	}
	if code := glueErrorType(t, ts, "CancelDataQualityRulesetEvaluationRun", `{"RunId":"`+run+`"}`); code != "IllegalStateException" {
		t.Fatalf("cancelling a finished run: %s", code)
	}

	rec := glueCall(t, ts, http.StatusOK, "StartDataQualityRuleRecommendationRun", `{"DataSource":`+src+`,"Role":"r","CreatedRulesetName":"recommended"}`)["RunId"].(string)
	glueCall(t, ts, http.StatusOK, "GetDataQualityRuleRecommendationRun", `{"RunId":"`+rec+`"}`)
	glueCall(t, ts, http.StatusOK, "GetDataQualityRuleRecommendationRun", `{"RunId":"`+rec+`"}`)
	if rs := glueCall(t, ts, http.StatusOK, "GetDataQualityRuleset", `{"Name":"recommended"}`); rs["RecommendationRunId"] != rec {
		t.Fatalf("recommended ruleset: %v", rs)
	}
}

func TestGlueStage3TagsPoliciesAndSettings(t *testing.T) {
	_, ts := newGlueTestServer(t)
	glueCall(t, ts, http.StatusOK, "CreateJob", `{"Name":"tagged","Role":"r","Command":{"Name":"glueetl"},"Tags":{"a":"1"}}`)
	arn := "arn:aws:glue:us-east-1:123456789012:job/tagged"
	glueCall(t, ts, http.StatusOK, "TagResource", `{"ResourceArn":"`+arn+`","TagsToAdd":{"b":"2"}}`)
	glueCall(t, ts, http.StatusOK, "UntagResource", `{"ResourceArn":"`+arn+`","TagsToRemove":["a"]}`)
	tags := glueCall(t, ts, http.StatusOK, "GetTags", `{"ResourceArn":"`+arn+`"}`)["Tags"].(map[string]any)
	if len(tags) != 1 || tags["b"] != "2" {
		t.Fatalf("tags: %v", tags)
	}
	if code := glueErrorType(t, ts, "TagResource", `{"ResourceArn":"arn:aws:glue:us-east-1:123456789012:job/ghost","TagsToAdd":{"b":"2"}}`); code != "EntityNotFoundException" {
		t.Fatalf("tagging a missing resource: %s", code)
	}
	if code := glueErrorType(t, ts, "GetTags", `{"ResourceArn":"arn:aws:s3:::bucket"}`); code != "InvalidInputException" {
		t.Fatalf("non-Glue ARN: %s", code)
	}

	policy := `{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"AWS\":\"*\"},\"Action\":\"glue:GetTable\",\"Resource\":\"*\"}]}`
	if code := glueErrorType(t, ts, "PutResourcePolicy", `{"PolicyInJson":"{}"}`); code != "InvalidInputException" {
		t.Fatalf("policy without a Statement: %s", code)
	}
	hash := glueCall(t, ts, http.StatusOK, "PutResourcePolicy", `{"PolicyInJson":"`+policy+`","PolicyExistsCondition":"NOT_EXIST"}`)["PolicyHash"].(string)
	if code := glueErrorType(t, ts, "PutResourcePolicy", `{"PolicyInJson":"`+policy+`","PolicyExistsCondition":"NOT_EXIST"}`); code != "ConditionCheckFailureException" {
		t.Fatalf("NOT_EXIST with a policy present: %s", code)
	}
	if got := glueCall(t, ts, http.StatusOK, "GetResourcePolicy", `{}`); got["PolicyHash"] != hash {
		t.Fatalf("policy: %v", got)
	}
	if code := glueErrorType(t, ts, "DeleteResourcePolicy", `{"PolicyHashCondition":"stale"}`); code != "ConditionCheckFailureException" {
		t.Fatalf("delete with a stale hash: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "DeleteResourcePolicy", `{"PolicyHashCondition":"`+hash+`"}`)
	if code := glueErrorType(t, ts, "GetResourcePolicy", `{}`); code != "EntityNotFoundException" {
		t.Fatalf("policy after delete: %s", code)
	}

	glueCall(t, ts, http.StatusOK, "PutDataCatalogEncryptionSettings", `{"DataCatalogEncryptionSettings":{"EncryptionAtRest":{"CatalogEncryptionMode":"SSE-KMS","SseAwsKmsKeyId":"alias/x"}}}`)
	enc := glueCall(t, ts, http.StatusOK, "GetDataCatalogEncryptionSettings", `{}`)["DataCatalogEncryptionSettings"].(map[string]any)
	if enc["EncryptionAtRest"].(map[string]any)["CatalogEncryptionMode"] != "SSE-KMS" {
		t.Fatalf("encryption settings: %v", enc)
	}

	if code := glueErrorType(t, ts, "GetGlueIdentityCenterConfiguration", `{}`); code != "EntityNotFoundException" {
		t.Fatalf("missing identity center config: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "CreateGlueIdentityCenterConfiguration", `{"InstanceArn":"arn:aws:sso:::instance/ssoins-1"}`)
	if code := glueErrorType(t, ts, "CreateGlueIdentityCenterConfiguration", `{"InstanceArn":"arn:aws:sso:::instance/ssoins-1"}`); code != "AlreadyExistsException" {
		t.Fatalf("second identity center config: %s", code)
	}

	if ct := glueCall(t, ts, http.StatusOK, "DescribeConnectionType", `{"ConnectionType":"JDBC"}`); ct["Description"] == nil {
		t.Fatalf("built-in connection type: %v", ct)
	}
	if code := glueErrorType(t, ts, "DeleteConnectionType", `{"ConnectionType":"JDBC"}`); code != "InvalidInputException" {
		t.Fatalf("deleting a built-in type: %s", code)
	}
}

func TestGlueStage3TaskRunsOptimizersAndIntegrations(t *testing.T) {
	_, ts := newGlueTestServer(t)
	glueCall(t, ts, http.StatusOK, "CreateDatabase", `{"DatabaseInput":{"Name":"d"}}`)
	glueCall(t, ts, http.StatusOK, "CreateTable", `{"DatabaseName":"d","TableInput":{"Name":"t"}}`)

	run := glueCall(t, ts, http.StatusOK, "StartColumnStatisticsTaskRun", `{"DatabaseName":"d","TableName":"t","Role":"r"}`)["ColumnStatisticsTaskRunId"].(string)
	if code := glueErrorType(t, ts, "StartColumnStatisticsTaskRun", `{"DatabaseName":"d","TableName":"t","Role":"r"}`); code != "ColumnStatisticsTaskRunningException" {
		t.Fatalf("second statistics run: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "StopColumnStatisticsTaskRun", `{"DatabaseName":"d","TableName":"t"}`)
	if code := glueErrorType(t, ts, "StopColumnStatisticsTaskRun", `{"DatabaseName":"d","TableName":"t"}`); code != "ColumnStatisticsTaskNotRunningException" {
		t.Fatalf("stopping with nothing running: %s", code)
	}
	if r := glueCall(t, ts, http.StatusOK, "GetColumnStatisticsTaskRun", `{"ColumnStatisticsTaskRunId":"`+run+`"}`)["ColumnStatisticsTaskRun"].(map[string]any); r["Status"] != "STOPPED" {
		t.Fatalf("statistics run: %v", r)
	}

	glueCall(t, ts, http.StatusOK, "CreateTableOptimizer", `{"CatalogId":"123456789012","DatabaseName":"d","TableName":"t","Type":"compaction","TableOptimizerConfiguration":{"roleArn":"arn:aws:iam::123456789012:role/glue","enabled":true}}`)
	opt := glueCall(t, ts, http.StatusOK, "GetTableOptimizer", `{"CatalogId":"123456789012","DatabaseName":"d","TableName":"t","Type":"compaction"}`)["TableOptimizer"].(map[string]any)
	if opt["type"] != "compaction" || opt["configuration"].(map[string]any)["enabled"] != true {
		t.Fatalf("optimizer: %v", opt)
	}
	batch := glueCall(t, ts, http.StatusOK, "BatchGetTableOptimizer", `{"Entries":[{"catalogId":"123456789012","databaseName":"d","tableName":"t","type":"compaction"},{"catalogId":"123456789012","databaseName":"d","tableName":"t","type":"retention"}]}`)
	if len(batch["TableOptimizers"].([]any)) != 1 || len(batch["Failures"].([]any)) != 1 {
		t.Fatalf("batch optimizers: %v", batch)
	}

	integ := glueCall(t, ts, http.StatusOK, "CreateIntegration", `{"IntegrationName":"zetl","SourceArn":"arn:aws:dynamodb:us-east-1:123456789012:table/src","TargetArn":"arn:aws:glue:us-east-1:123456789012:database/d"}`)
	if integ["Status"] != "ACTIVE" || integ["IntegrationArn"] == nil {
		t.Fatalf("integration: %v", integ)
	}
	inbound := glueCall(t, ts, http.StatusOK, "DescribeInboundIntegrations", `{"TargetArn":"arn:aws:glue:us-east-1:123456789012:database/d"}`)["InboundIntegrations"].([]any)
	if len(inbound) != 1 {
		t.Fatalf("inbound: %v", inbound)
	}
	glueCall(t, ts, http.StatusOK, "ModifyIntegration", `{"IntegrationIdentifier":"zetl","Description":"changed"}`)
	if out := glueCall(t, ts, http.StatusOK, "DeleteIntegration", `{"IntegrationIdentifier":"`+integ["IntegrationArn"].(string)+`"}`); out["Description"] != "changed" {
		t.Fatalf("delete integration: %v", out)
	}
	if code := glueErrorType(t, ts, "DeleteIntegration", `{"IntegrationIdentifier":"zetl"}`); code != "IntegrationNotFoundFault" {
		t.Fatalf("deleting a missing integration: %s", code)
	}
}
