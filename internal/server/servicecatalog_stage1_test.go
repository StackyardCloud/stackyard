package server

import (
	"fmt"
	"strings"
	"testing"
)

const svcatTestTemplate = `{"Parameters":{"Env":{"Type":"String","AllowedValues":["dev","prod"]},"Size":{"Type":"Number","Default":"1"}},
"Resources":{"Bucket":{"Type":"AWS::S3::Bucket"}},
"Outputs":{"EnvOut":{"Description":"env","Value":{"Ref":"Env"}},"Fixed":{"Value":"hello"}}}`

// svcatSeed puts the test template in Stackyard's S3.
func svcatSeed(t *testing.T, srv *Server) {
	t.Helper()
	if _, err := srv.s3.CreateBucket("sc-templates"); err != nil && !strings.Contains(err.Error(), "exist") {
		t.Fatal(err)
	}
	if _, err := srv.s3.PutObject("sc-templates", "app.json", "application/json", []byte(svcatTestTemplate), nil, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestServiceCatalogStage1ProvisioningLifecycle(t *testing.T) {
	srv, ts := newServiceCatalogTestServer(t)
	svcatSeed(t, srv)
	port := svcatOK(t, ts, "CreatePortfolio", `{"DisplayName":"Platform","ProviderName":"IT","IdempotencyToken":"port1","Tags":[{"Key":"team","Value":"core"}]}`)
	portID := port["PortfolioDetail"].(map[string]any)["Id"].(string)
	// Idempotent: the same token returns the same portfolio.
	again := svcatOK(t, ts, "CreatePortfolio", `{"DisplayName":"Platform","ProviderName":"IT","IdempotencyToken":"port1"}`)
	if again["PortfolioDetail"].(map[string]any)["Id"] != portID {
		t.Fatalf("idempotency token made a second portfolio")
	}

	// The template must exist in S3.
	if code := svcatErrCode(t, ts, "CreateProduct", `{"Name":"App","Owner":"IT","ProductType":"CLOUD_FORMATION_TEMPLATE","IdempotencyToken":"bad",
		"ProvisioningArtifactParameters":{"Info":{"LoadTemplateFromURL":"https://sc-templates.s3.amazonaws.com/missing.json"}}}`); code != "InvalidParametersException" {
		t.Fatalf("missing template: %s", code)
	}
	prod := svcatOK(t, ts, "CreateProduct", `{"Name":"App","Owner":"IT","ProductType":"CLOUD_FORMATION_TEMPLATE","IdempotencyToken":"prod1",
		"ProvisioningArtifactParameters":{"Name":"v1","Info":{"LoadTemplateFromURL":"https://sc-templates.s3.amazonaws.com/app.json"}}}`)
	prodID := prod["ProductViewDetail"].(map[string]any)["ProductViewSummary"].(map[string]any)["ProductId"].(string)
	paID := prod["ProvisioningArtifactDetail"].(map[string]any)["Id"].(string)

	params := svcatOK(t, ts, "DescribeProvisioningArtifact", fmt.Sprintf(`{"ProductId":%q,"ProvisioningArtifactId":%q,"IncludeProvisioningArtifactParameters":true,"Verbose":true}`, prodID, paID))
	if got := len(params["ProvisioningArtifactParameters"].([]any)); got != 2 {
		t.Fatalf("template parameters: %d", got)
	}

	// No launch path until the product is in a portfolio.
	provision := func(name, token, extra string) (int, map[string]any) {
		return svcatCall(t, ts, "ProvisionProduct", fmt.Sprintf(`{"ProductId":%q,"ProvisioningArtifactId":%q,"ProvisionedProductName":%q,"ProvisionToken":%q%s}`,
			prodID, paID, name, token, extra))
	}
	if status, out := provision("app1", "t1", `,"ProvisioningParameters":[{"Key":"Env","Value":"dev"}]`); status != 400 || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("provision without launch path: %d %v", status, out)
	}
	svcatOK(t, ts, "AssociateProductWithPortfolio", fmt.Sprintf(`{"ProductId":%q,"PortfolioId":%q}`, prodID, portID))

	// Parameters are checked against the template.
	if status, out := provision("app1", "t0", ``); status != 400 || out["__type"] != "InvalidParametersException" {
		t.Fatalf("missing required parameter: %d %v", status, out)
	}
	if status, out := provision("app1", "t0b", `,"ProvisioningParameters":[{"Key":"Env","Value":"qa"}]`); status != 400 || out["__type"] != "InvalidParametersException" {
		t.Fatalf("disallowed value: %d %v", status, out)
	}
	status, out := provision("app1", "t1", `,"ProvisioningParameters":[{"Key":"Env","Value":"dev"}]`)
	if status != 200 {
		t.Fatalf("provision: %d %v", status, out)
	}
	rec := out["RecordDetail"].(map[string]any)
	if rec["Status"] != "IN_PROGRESS" {
		t.Fatalf("new record: %v", rec["Status"])
	}
	recID, ppID := rec["RecordId"].(string), rec["ProvisionedProductId"].(string)
	if status, out := provision("app1", "t1", `,"ProvisioningParameters":[{"Key":"Env","Value":"dev"}]`); status != 200 || out["RecordDetail"].(map[string]any)["RecordId"] != recID {
		t.Fatalf("ProvisionToken idempotency: %d %v", status, out)
	}
	if status, out := provision("app1", "t2", `,"ProvisioningParameters":[{"Key":"Env","Value":"dev"}]`); status != 400 || out["__type"] != "DuplicateResourceException" {
		t.Fatalf("duplicate name: %d %v", status, out)
	}

	// The record settles, the stack exists in CloudFormation, outputs resolve.
	desc := svcatOK(t, ts, "DescribeRecord", fmt.Sprintf(`{"Id":%q}`, recID))
	if desc["RecordDetail"].(map[string]any)["Status"] != "SUCCEEDED" {
		t.Fatalf("record did not settle: %v", desc["RecordDetail"])
	}
	outputs := map[string]any{}
	for _, o := range desc["RecordOutputs"].([]any) {
		m := o.(map[string]any)
		outputs[m["OutputKey"].(string)] = m["OutputValue"]
	}
	if outputs["EnvOut"] != "dev" || outputs["Fixed"] != "hello" || !strings.HasPrefix(fmt.Sprint(outputs["CloudformationStackARN"]), "arn:aws:cloudformation:") {
		t.Fatalf("outputs: %v", outputs)
	}
	if name, ok := srv.svcatFindStack(svcatStackName(ppID)); !ok || name == "" {
		t.Fatalf("CloudFormation stack not registered")
	}
	pp := svcatOK(t, ts, "DescribeProvisionedProduct", fmt.Sprintf(`{"Id":%q}`, ppID))["ProvisionedProductDetail"].(map[string]any)
	if pp["Status"] != "AVAILABLE" || pp["LastSuccessfulProvisioningRecordId"] != recID {
		t.Fatalf("provisioned product: %v", pp)
	}

	// Products in use can't be deleted.
	if code := svcatErrCode(t, ts, "DeleteProduct", fmt.Sprintf(`{"Id":%q}`, prodID)); code != "ResourceInUseException" {
		t.Fatalf("delete product in use: %s", code)
	}

	// Update keeps previous values, then terminate removes the stack.
	up := svcatOK(t, ts, "UpdateProvisionedProduct", fmt.Sprintf(`{"ProvisionedProductId":%q,"UpdateToken":"u1","ProvisioningParameters":[{"Key":"Env","UsePreviousValue":true},{"Key":"Size","Value":"3"}]}`, ppID))
	upID := up["RecordDetail"].(map[string]any)["RecordId"].(string)
	svcatOK(t, ts, "DescribeRecord", fmt.Sprintf(`{"Id":%q}`, upID))
	srv.serviceCatalog.mu.Lock()
	gotParams := srv.serviceCatalog.provisioned[ppID].params
	srv.serviceCatalog.mu.Unlock()
	if gotParams["Env"] != "dev" || gotParams["Size"] != "3" {
		t.Fatalf("updated params: %v", gotParams)
	}
	svcatOK(t, ts, "TerminateProvisionedProduct", fmt.Sprintf(`{"ProvisionedProductId":%q,"TerminateToken":"x1"}`, ppID))
	svcatOK(t, ts, "ListRecordHistory", `{}`)
	if code := svcatErrCode(t, ts, "DescribeProvisionedProduct", fmt.Sprintf(`{"Id":%q}`, ppID)); code != "ResourceNotFoundException" {
		t.Fatalf("terminated product still there: %s", code)
	}
	if _, ok := srv.svcatFindStack(svcatStackName(ppID)); ok {
		t.Fatalf("CloudFormation stack not deleted")
	}
}

func TestServiceCatalogStage1ConstraintsTagOptionsSharesActions(t *testing.T) {
	srv, ts := newServiceCatalogTestServer(t)
	svcatSeed(t, srv)
	portID := svcatOK(t, ts, "CreatePortfolio", `{"DisplayName":"P","ProviderName":"IT","IdempotencyToken":"p"}`)["PortfolioDetail"].(map[string]any)["Id"].(string)
	prod := svcatOK(t, ts, "CreateProduct", `{"Name":"App","Owner":"IT","ProductType":"CLOUD_FORMATION_TEMPLATE","IdempotencyToken":"prod",
		"ProvisioningArtifactParameters":{"Info":{"LoadTemplateFromURL":"https://s3.amazonaws.com/sc-templates/app.json"}}}`)
	prodID := prod["ProductViewDetail"].(map[string]any)["ProductViewSummary"].(map[string]any)["ProductId"].(string)
	paID := prod["ProvisioningArtifactDetail"].(map[string]any)["Id"].(string)

	launch := func(token, params string) string {
		return fmt.Sprintf(`{"PortfolioId":%q,"ProductId":%q,"Type":"LAUNCH","Parameters":%q,"IdempotencyToken":%q}`, portID, prodID, params, token)
	}
	if code := svcatErrCode(t, ts, "CreateConstraint", launch("c0", `{"RoleArn":"arn:aws:iam::123456789012:role/launch"}`)); code != "InvalidParametersException" {
		t.Fatalf("constraint on a product not in the portfolio: %s", code)
	}
	svcatOK(t, ts, "AssociateProductWithPortfolio", fmt.Sprintf(`{"ProductId":%q,"PortfolioId":%q}`, prodID, portID))
	if code := svcatErrCode(t, ts, "CreateConstraint", launch("c1", `{"RoleArn":"x","LocalRoleName":"y"}`)); code != "InvalidParametersException" {
		t.Fatalf("LAUNCH with both roles: %s", code)
	}
	svcatOK(t, ts, "CreateConstraint", launch("c2", `{"RoleArn":"arn:aws:iam::123456789012:role/launch"}`))
	if code := svcatErrCode(t, ts, "CreateConstraint", launch("c3", `{"LocalRoleName":"other"}`)); code != "DuplicateResourceException" {
		t.Fatalf("second LAUNCH constraint: %s", code)
	}
	if code := svcatErrCode(t, ts, "DisassociateProductFromPortfolio", fmt.Sprintf(`{"ProductId":%q,"PortfolioId":%q}`, prodID, portID)); code != "ResourceInUseException" {
		t.Fatalf("disassociate with constraints: %s", code)
	}

	// Tag options: unique key=value, one value per key per resource, in-use protection.
	tagID := svcatOK(t, ts, "CreateTagOption", `{"Key":"cost","Value":"a"}`)["TagOptionDetail"].(map[string]any)["Id"].(string)
	if code := svcatErrCode(t, ts, "CreateTagOption", `{"Key":"cost","Value":"a"}`); code != "DuplicateResourceException" {
		t.Fatalf("duplicate tag option: %s", code)
	}
	tag2 := svcatOK(t, ts, "CreateTagOption", `{"Key":"cost","Value":"b"}`)["TagOptionDetail"].(map[string]any)["Id"].(string)
	svcatOK(t, ts, "AssociateTagOptionWithResource", fmt.Sprintf(`{"ResourceId":%q,"TagOptionId":%q}`, portID, tagID))
	if code := svcatErrCode(t, ts, "AssociateTagOptionWithResource", fmt.Sprintf(`{"ResourceId":%q,"TagOptionId":%q}`, portID, tag2)); code != "DuplicateResourceException" {
		t.Fatalf("second value for a key: %s", code)
	}
	if code := svcatErrCode(t, ts, "DeleteTagOption", fmt.Sprintf(`{"Id":%q}`, tagID)); code != "ResourceInUseException" {
		t.Fatalf("delete tag option in use: %s", code)
	}

	// Organization shares need Organizations access.
	org := fmt.Sprintf(`{"PortfolioId":%q,"OrganizationNode":{"Type":"ORGANIZATION","Value":"o-abcdefghij"}}`, portID)
	if code := svcatErrCode(t, ts, "CreatePortfolioShare", org); code != "OperationNotSupportedException" {
		t.Fatalf("org share without access: %s", code)
	}
	svcatOK(t, ts, "EnableAWSOrganizationsAccess", `{}`)
	tok := svcatOK(t, ts, "CreatePortfolioShare", org)["PortfolioShareToken"].(string)
	if st := svcatOK(t, ts, "DescribePortfolioShareStatus", fmt.Sprintf(`{"PortfolioShareToken":%q}`, tok)); st["Status"] != "COMPLETED" {
		t.Fatalf("share status: %v", st)
	}
	svcatOK(t, ts, "CreatePortfolioShare", fmt.Sprintf(`{"PortfolioId":%q,"AccountId":"111122223333"}`, portID))
	if acc := svcatOK(t, ts, "ListPortfolioAccess", fmt.Sprintf(`{"PortfolioId":%q}`, portID)); len(acc["AccountIds"].([]any)) != 1 {
		t.Fatalf("portfolio access: %v", acc)
	}
	if code := svcatErrCode(t, ts, "DisableAWSOrganizationsAccess", `{}`); code != "ResourceInUseException" {
		t.Fatalf("disable org access while shared: %s", code)
	}

	// Service actions must be associated with the provisioned product's artifact.
	act := svcatOK(t, ts, "CreateServiceAction", `{"Name":"Restart","DefinitionType":"SSM_AUTOMATION","IdempotencyToken":"a1",
		"Definition":{"Name":"AWS-RestartEC2Instance","Version":"1","Parameters":"[{\"Name\":\"InstanceId\",\"Type\":\"TARGET\"}]"}}`)
	actID := act["ServiceActionDetail"].(map[string]any)["ServiceActionSummary"].(map[string]any)["Id"].(string)
	rec := svcatOK(t, ts, "ProvisionProduct", fmt.Sprintf(`{"ProductId":%q,"ProvisioningArtifactId":%q,"ProvisionedProductName":"svc","ProvisionToken":"pt",
		"ProvisioningParameters":[{"Key":"Env","Value":"prod"}]}`, prodID, paID))["RecordDetail"].(map[string]any)
	ppID := rec["ProvisionedProductId"].(string)
	exec := fmt.Sprintf(`{"ProvisionedProductId":%q,"ServiceActionId":%q,"ExecuteToken":"e1"}`, ppID, actID)
	if code := svcatErrCode(t, ts, "ExecuteProvisionedProductServiceAction", exec); code != "InvalidParametersException" {
		t.Fatalf("unassociated action: %s", code)
	}
	svcatOK(t, ts, "AssociateServiceActionWithProvisioningArtifact", fmt.Sprintf(`{"ProductId":%q,"ProvisioningArtifactId":%q,"ServiceActionId":%q}`, prodID, paID, actID))
	params := svcatOK(t, ts, "DescribeServiceActionExecutionParameters", fmt.Sprintf(`{"ProvisionedProductId":%q,"ServiceActionId":%q}`, ppID, actID))
	sap := params["ServiceActionParameters"].([]any)
	if len(sap) != 1 || len(sap[0].(map[string]any)["DefaultValues"].([]any)) != 1 {
		t.Fatalf("execution parameters: %v", params)
	}
	if r := svcatOK(t, ts, "ExecuteProvisionedProductServiceAction", exec)["RecordDetail"].(map[string]any); r["RecordType"] != "EXECUTE_PROVISIONED_PRODUCT_SERVICE_ACTION" {
		t.Fatalf("execute: %v", r)
	}
	if code := svcatErrCode(t, ts, "DeleteServiceAction", fmt.Sprintf(`{"Id":%q}`, actID)); code != "ResourceInUseException" {
		t.Fatalf("delete associated action: %s", code)
	}
	if code := svcatErrCode(t, ts, "DeletePortfolio", fmt.Sprintf(`{"Id":%q}`, portID)); code != "ResourceInUseException" {
		t.Fatalf("delete portfolio in use: %s", code)
	}
}

func TestServiceCatalogStage1PlansAndEngineWorkflow(t *testing.T) {
	srv, ts := newServiceCatalogTestServer(t)
	svcatSeed(t, srv)
	if _, err := srv.s3.PutObject("sc-templates", "tf.tar.gz", "application/gzip", []byte("not a cloudformation template"), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	portID := svcatOK(t, ts, "CreatePortfolio", `{"DisplayName":"P","ProviderName":"IT","IdempotencyToken":"p"}`)["PortfolioDetail"].(map[string]any)["Id"].(string)
	mk := func(typ, key, token string) (string, string) {
		out := svcatOK(t, ts, "CreateProduct", fmt.Sprintf(`{"Name":%q,"Owner":"IT","ProductType":%q,"IdempotencyToken":%q,
			"ProvisioningArtifactParameters":{"Info":{"LoadTemplateFromURL":"https://sc-templates.s3.amazonaws.com/%s"}}}`, token, typ, token, key))
		id := out["ProductViewDetail"].(map[string]any)["ProductViewSummary"].(map[string]any)["ProductId"].(string)
		svcatOK(t, ts, "AssociateProductWithPortfolio", fmt.Sprintf(`{"ProductId":%q,"PortfolioId":%q}`, id, portID))
		return id, out["ProvisioningArtifactDetail"].(map[string]any)["Id"].(string)
	}
	cfnID, cfnPA := mk("CLOUD_FORMATION_TEMPLATE", "app.json", "cfn")

	// A plan lists the template's resources and executes once created.
	plan := svcatOK(t, ts, "CreateProvisionedProductPlan", fmt.Sprintf(`{"PlanName":"p1","PlanType":"CLOUDFORMATION","ProductId":%q,"ProvisioningArtifactId":%q,
		"ProvisionedProductName":"planned","IdempotencyToken":"pl","ProvisioningParameters":[{"Key":"Env","Value":"dev"}]}`, cfnID, cfnPA))
	planID := plan["PlanId"].(string)
	desc := svcatOK(t, ts, "DescribeProvisionedProductPlan", fmt.Sprintf(`{"PlanId":%q}`, planID))
	if len(desc["ResourceChanges"].([]any)) != 1 || desc["ProvisionedProductPlanDetails"].(map[string]any)["Status"] != "CREATE_SUCCESS" {
		t.Fatalf("plan: %v", desc)
	}
	svcatOK(t, ts, "ExecuteProvisionedProductPlan", fmt.Sprintf(`{"PlanId":%q,"IdempotencyToken":"ex"}`, planID))
	if pp := svcatOK(t, ts, "DescribeProvisionedProduct", `{"Name":"planned"}`)["ProvisionedProductDetail"].(map[string]any); pp["Status"] != "AVAILABLE" {
		t.Fatalf("plan-provisioned product: %v", pp)
	}

	// Terraform records wait for the engine to notify.
	tfID, tfPA := mk("TERRAFORM_OPEN_SOURCE", "tf.tar.gz", "tf")
	rec := svcatOK(t, ts, "ProvisionProduct", fmt.Sprintf(`{"ProductId":%q,"ProvisioningArtifactId":%q,"ProvisionedProductName":"tf1","ProvisionToken":"tf1"}`, tfID, tfPA))["RecordDetail"].(map[string]any)
	recID := rec["RecordId"].(string)
	if r := svcatOK(t, ts, "DescribeRecord", fmt.Sprintf(`{"Id":%q}`, recID))["RecordDetail"].(map[string]any); r["Status"] != "IN_PROGRESS" {
		t.Fatalf("terraform record settled without the engine: %v", r["Status"])
	}
	notify := fmt.Sprintf(`{"WorkflowToken":"wf","RecordId":%q,"Status":"SUCCEEDED","IdempotencyToken":"n1","Outputs":[{"OutputKey":"url","OutputValue":"https://x"}]}`, recID)
	if code := svcatErrCode(t, ts, "NotifyUpdateProvisionedProductEngineWorkflowResult", notify); code != "InvalidParametersException" {
		t.Fatalf("notify of the wrong record type: %s", code)
	}
	svcatOK(t, ts, "NotifyProvisionProductEngineWorkflowResult", notify)
	svcatOK(t, ts, "NotifyProvisionProductEngineWorkflowResult", notify) // same token: idempotent
	if r := svcatOK(t, ts, "DescribeRecord", fmt.Sprintf(`{"Id":%q}`, recID)); r["RecordDetail"].(map[string]any)["Status"] != "SUCCEEDED" || len(r["RecordOutputs"].([]any)) != 1 {
		t.Fatalf("notified record: %v", r)
	}
	if code := svcatErrCode(t, ts, "NotifyProvisionProductEngineWorkflowResult", strings.Replace(notify, `"n1"`, `"n2"`, 1)); code != "InvalidStateException" {
		t.Fatalf("second notify: %s", code)
	}

	// Search finds both provisioned products; a scoped query narrows to one.
	all := svcatOK(t, ts, "SearchProvisionedProducts", `{}`)
	one := svcatOK(t, ts, "SearchProvisionedProducts", `{"Filters":{"SearchQuery":["name:tf1"]}}`)
	if all["TotalResultsCount"].(float64) != 2 || one["TotalResultsCount"].(float64) != 1 {
		t.Fatalf("search: all=%v one=%v", all["TotalResultsCount"], one["TotalResultsCount"])
	}
}
