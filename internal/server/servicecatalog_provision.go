package server

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

func svcatRegisterProvisionHandlers() {
	for name, h := range map[string]svcatHandler{
		"ProvisionProduct":                                       (*serviceCatalogStore).provisionProduct,
		"UpdateProvisionedProduct":                               (*serviceCatalogStore).updateProvisioned,
		"TerminateProvisionedProduct":                            (*serviceCatalogStore).terminateProvisioned,
		"ImportAsProvisionedProduct":                             (*serviceCatalogStore).importProvisioned,
		"DescribeProvisionedProduct":                             (*serviceCatalogStore).describeProvisioned,
		"UpdateProvisionedProductProperties":                     (*serviceCatalogStore).updateProvisionedProperties,
		"GetProvisionedProductOutputs":                           (*serviceCatalogStore).provisionedOutputs,
		"ListStackInstancesForProvisionedProduct":                (*serviceCatalogStore).listStackInstances,
		"SearchProvisionedProducts":                              (*serviceCatalogStore).searchProvisioned,
		"ScanProvisionedProducts":                                (*serviceCatalogStore).scanProvisioned,
		"DescribeRecord":                                         (*serviceCatalogStore).describeRecord,
		"ListRecordHistory":                                      (*serviceCatalogStore).listRecordHistory,
		"NotifyProvisionProductEngineWorkflowResult":             (*serviceCatalogStore).notifyProvision,
		"NotifyUpdateProvisionedProductEngineWorkflowResult":     (*serviceCatalogStore).notifyUpdate,
		"NotifyTerminateProvisionedProductEngineWorkflowResult":  (*serviceCatalogStore).notifyTerminate,
		"CreateProvisionedProductPlan":                           (*serviceCatalogStore).createPlan,
		"DescribeProvisionedProductPlan":                         (*serviceCatalogStore).describePlan,
		"ExecuteProvisionedProductPlan":                          (*serviceCatalogStore).executePlan,
		"DeleteProvisionedProductPlan":                           (*serviceCatalogStore).deletePlan,
		"ListProvisionedProductPlans":                            (*serviceCatalogStore).listPlans,
		"CreateServiceAction":                                    (*serviceCatalogStore).createAction,
		"DescribeServiceAction":                                  (*serviceCatalogStore).describeAction,
		"UpdateServiceAction":                                    (*serviceCatalogStore).updateAction,
		"DeleteServiceAction":                                    (*serviceCatalogStore).deleteAction,
		"ListServiceActions":                                     (*serviceCatalogStore).listActions,
		"AssociateServiceActionWithProvisioningArtifact":         (*serviceCatalogStore).associateAction,
		"DisassociateServiceActionFromProvisioningArtifact":      (*serviceCatalogStore).disassociateAction,
		"BatchAssociateServiceActionWithProvisioningArtifact":    (*serviceCatalogStore).batchAssociateActions,
		"BatchDisassociateServiceActionFromProvisioningArtifact": (*serviceCatalogStore).batchDisassociateActions,
		"ListServiceActionsForProvisioningArtifact":              (*serviceCatalogStore).listActionsForArtifact,
		"ListProvisioningArtifactsForServiceAction":              (*serviceCatalogStore).listArtifactsForAction,
		"DescribeServiceActionExecutionParameters":               (*serviceCatalogStore).actionExecutionParameters,
		"ExecuteProvisionedProductServiceAction":                 (*serviceCatalogStore).executeAction,
	} {
		svcatHandlers[name] = h
	}
}

// ---- records ----

func (r *svcatRecord) detail() map[string]any {
	return map[string]any{"RecordId": r.id, "ProvisionedProductName": r.ppName, "Status": r.status, "CreatedTime": svcatEpoch(r.created),
		"UpdatedTime": svcatEpoch(r.updated), "ProvisionedProductType": "CFN_STACK", "RecordType": r.typ, "ProvisionedProductId": r.ppID,
		"ProductId": r.productID, "ProvisioningArtifactId": r.artifactID, "PathId": r.pathID, "RecordErrors": r.errors, "RecordTags": r.tags}
}

// newRecord starts a request on a provisioned product. CloudFormation records
// settle on the next request; Terraform and external records wait for the
// engine's Notify*EngineWorkflowResult call.
func (st *serviceCatalogStore) newRecord(pp *svcatProvisioned, typ string, settle func()) (*svcatRecord, *svcatError) {
	if len(st.records) >= svcatMaxRecords && !st.evictRecordLocked() {
		return nil, svcatErr("LimitExceededException", "too many requests are in progress")
	}
	now := time.Now().UTC()
	r := &svcatRecord{id: svcatNewID("rec-"), ppID: pp.id, ppName: pp.name, typ: typ, status: "IN_PROGRESS", productID: pp.productID,
		artifactID: pp.artifactID, pathID: pp.pathID, created: now, updated: now, errors: []any{}, tags: []any{},
		pending: true, engine: pp.engine, settle: settle}
	for _, t := range pp.tags {
		r.tags = append(r.tags, t)
	}
	st.records[r.id] = r
	st.recordOrder = append(st.recordOrder, r.id)
	pp.lastRecord = r.id
	pp.status = "UNDER_CHANGE"
	return r, nil
}

// evictRecordLocked drops the oldest settled record.
func (st *serviceCatalogStore) evictRecordLocked() bool {
	for i, id := range st.recordOrder {
		if r := st.records[id]; r == nil || !r.pending {
			delete(st.records, id)
			st.recordOrder = append(st.recordOrder[:i:i], st.recordOrder[i+1:]...)
			return true
		}
	}
	return false
}

// settleLocked completes CloudFormation-backed requests made before this one.
// ponytail: settle-on-next-request instead of timers; nothing runs in the background.
func (st *serviceCatalogStore) settleLocked() {
	for _, id := range st.recordOrder {
		if r := st.records[id]; r != nil && r.pending && !r.engine {
			st.completeRecord(r, "SUCCEEDED", "")
		}
	}
	for _, p := range st.plans {
		switch p.status {
		case "CREATE_IN_PROGRESS":
			p.status = "CREATE_SUCCESS"
		case "EXECUTE_IN_PROGRESS":
			p.status = "EXECUTE_SUCCESS"
		}
	}
}

func (st *serviceCatalogStore) completeRecord(r *svcatRecord, status, reason string) {
	r.pending, r.status, r.updated = false, status, time.Now().UTC()
	pp := st.provisioned[r.ppID]
	if status == "FAILED" {
		r.errors = append(r.errors, map[string]any{"Code": "ProvisioningFailed", "Description": reason})
		if pp != nil {
			pp.status, pp.statusMsg = "ERROR", reason
		}
		return
	}
	if r.settle != nil {
		r.settle()
	}
	if pp = st.provisioned[r.ppID]; pp != nil { // terminate removes it
		pp.status, pp.statusMsg = "AVAILABLE", ""
		pp.lastOKRecord = r.id
		if r.typ == "PROVISION_PRODUCT" || r.typ == "UPDATE_PROVISIONED_PRODUCT" {
			pp.lastProvRecord = r.id
		}
	}
}

func (st *serviceCatalogStore) record(id string) (*svcatRecord, *svcatError) {
	r := st.records[id]
	if r == nil {
		return nil, svcatNotFound("Record %s not found", id)
	}
	return r, nil
}

// ---- provisioned products ----

func (pp *svcatProvisioned) detail() map[string]any {
	return map[string]any{"Name": pp.name, "Arn": pp.arn, "Type": "CFN_STACK", "Id": pp.id, "Status": pp.status, "StatusMessage": pp.statusMsg,
		"CreatedTime": svcatEpoch(pp.created), "IdempotencyToken": pp.token, "LastRecordId": pp.lastRecord,
		"LastProvisioningRecordId": pp.lastProvRecord, "LastSuccessfulProvisioningRecordId": pp.lastOKRecord,
		"ProductId": pp.productID, "ProvisioningArtifactId": pp.artifactID, "LaunchRoleArn": pp.props["LAUNCH_ROLE"]}
}

func (st *serviceCatalogStore) attribute(pp *svcatProvisioned) map[string]any {
	d := pp.detail()
	d["Tags"] = pp.tags
	d["PhysicalId"] = pp.physicalID
	d["UserArn"] = pp.props["OWNER"]
	if p := st.products[pp.productID]; p != nil {
		d["ProductName"] = p.name
		if a, err := p.artifact(pp.artifactID, ""); err == nil {
			d["ProvisioningArtifactName"] = a.name
		}
	}
	return d
}

func (st *serviceCatalogStore) provisionedByIDOrName(id, name string) (*svcatProvisioned, *svcatError) {
	if id != "" {
		if pp := st.provisioned[id]; pp != nil {
			return pp, nil
		}
		return nil, svcatNotFound("Provisioned product %s not found", id)
	}
	if name == "" {
		return nil, svcatInvalid("Specify the provisioned product Id or Name")
	}
	for _, pp := range st.provisioned {
		if pp.name == name || pp.arn == name {
			return pp, nil
		}
	}
	return nil, svcatNotFound("Provisioned product %s not found", name)
}

// launchTarget resolves product, artifact and launch path for a provisioning request.
func (st *serviceCatalogStore) launchTarget(in map[string]any) (*svcatProduct, *svcatArtifact, *svcatPortfolio, *svcatError) {
	p, err := st.productByIDOrName(svcatStr(in, "ProductId"), svcatStr(in, "ProductName"))
	if err != nil {
		return nil, nil, nil, err
	}
	a, err := p.artifact(svcatStr(in, "ProvisioningArtifactId"), svcatStr(in, "ProvisioningArtifactName"))
	if err != nil {
		return nil, nil, nil, err
	}
	if !a.active {
		return nil, nil, nil, svcatInvalid("Provisioning artifact %s is not active", a.id)
	}
	port, err := st.resolvePath(p, svcatStr(in, "PathId"), svcatStr(in, "PathName"))
	if err != nil {
		return nil, nil, nil, err
	}
	return p, a, port, nil
}

func svcatStackName(ppID string) string { return "SC-" + svcatAccount + "-" + ppID }

func svcatStackARN(ppID string) string {
	return "arn:aws:cloudformation:" + svcatRegion + ":" + svcatAccount + ":stack/" + svcatStackName(ppID) + "/" + strings.TrimPrefix(ppID, "pp-")
}

// startProvision creates a provisioned product and its PROVISION_PRODUCT record.
func (st *serviceCatalogStore) startProvision(s *Server, name, token string, p *svcatProduct, a *svcatArtifact, port *svcatPortfolio,
	params []any, tags []any) (*svcatRecord, *svcatError) {
	for _, pp := range st.provisioned {
		if pp.name == name {
			return nil, svcatErr("DuplicateResourceException", "Provisioned product %s already exists", name)
		}
	}
	if len(st.provisioned) >= svcatMaxProvisioned {
		return nil, svcatErr("LimitExceededException", "an account can have at most %d provisioned products", svcatMaxProvisioned)
	}
	vals, err := a.tmpl.resolveParams(params, nil)
	if err != nil {
		return nil, err
	}
	if len(tags) > svcatMaxTags {
		return nil, svcatInvalid("a provisioned product can have at most %d tags", svcatMaxTags)
	}
	id := svcatNewID("pp-")
	pp := &svcatProvisioned{id: id, name: name, arn: "arn:aws:servicecatalog:" + svcatRegion + ":" + svcatAccount + ":stack/" + name + "/" + id,
		productID: p.id, artifactID: a.id, pathID: svcatPathID(port.id), token: token, created: time.Now().UTC(), params: vals,
		tags: tags, props: map[string]string{"OWNER": "arn:aws:iam::" + svcatAccount + ":root"},
		engine: p.typ != "CLOUD_FORMATION_TEMPLATE"}
	for _, c := range st.sortedConstraints(port.id, p.id) {
		if c.typ == "LAUNCH" {
			var cp map[string]any
			_ = json.Unmarshal([]byte(c.params), &cp)
			role := svcatStr(cp, "RoleArn")
			if role == "" {
				role = "arn:aws:iam::" + svcatAccount + ":role/" + svcatStr(cp, "LocalRoleName")
			}
			pp.props["LAUNCH_ROLE"] = role
		}
	}
	st.provisioned[id] = pp
	tmpl := a.tmpl
	r, err := st.newRecord(pp, "PROVISION_PRODUCT", func() {
		if pp.engine {
			return
		}
		pp.stackName, pp.physicalID = svcatStackName(pp.id), svcatStackARN(pp.id)
		pp.outputs = tmpl.outputValues(pp.params, pp.physicalID)
		name, arn := pp.stackName, pp.physicalID
		st.post = append(st.post, func() { s.svcatPutStack(name, arn) })
	})
	if err != nil {
		delete(st.provisioned, id)
		return nil, err
	}
	return r, nil
}

func (st *serviceCatalogStore) provisionProduct(s *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	token := svcatStr(in, "ProvisionToken")
	if id, ok := st.svcatIdempotent("ProvisionProduct", token); ok {
		if r := st.records[id]; r != nil {
			return map[string]any{"RecordDetail": r.detail()}, nil
		}
	}
	p, a, port, err := st.launchTarget(in)
	if err != nil {
		return nil, err
	}
	r, err := st.startProvision(s, svcatStr(in, "ProvisionedProductName"), token, p, a, port, svcatList(in, "ProvisioningParameters"), svcatList(in, "Tags"))
	if err != nil {
		return nil, err
	}
	if err := st.rememberToken("ProvisionProduct", token, r.id); err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetail": r.detail()}, nil
}

func (st *serviceCatalogStore) updateProvisioned(s *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	token := svcatStr(in, "UpdateToken")
	if id, ok := st.svcatIdempotent("UpdateProvisionedProduct", token); ok {
		if r := st.records[id]; r != nil {
			return map[string]any{"RecordDetail": r.detail()}, nil
		}
	}
	pp, err := st.provisionedByIDOrName(svcatStr(in, "ProvisionedProductId"), svcatStr(in, "ProvisionedProductName"))
	if err != nil {
		return nil, err
	}
	if pp.status == "UNDER_CHANGE" || pp.status == "PLAN_IN_PROGRESS" {
		return nil, svcatErr("InvalidStateException", "Provisioned product %s is %s", pp.id, pp.status)
	}
	p := st.products[pp.productID]
	if pid, pname := svcatStr(in, "ProductId"), svcatStr(in, "ProductName"); pid != "" || pname != "" {
		if p, err = st.productByIDOrName(pid, pname); err != nil {
			return nil, err
		}
	}
	if p == nil {
		return nil, svcatNotFound("Product %s no longer exists", pp.productID)
	}
	artID, artName := svcatStr(in, "ProvisioningArtifactId"), svcatStr(in, "ProvisioningArtifactName")
	if artID == "" && artName == "" {
		artID = pp.artifactID
	}
	a, err := p.artifact(artID, artName)
	if err != nil {
		return nil, err
	}
	vals, err := a.tmpl.resolveParams(svcatList(in, "ProvisioningParameters"), pp.params)
	if err != nil {
		return nil, err
	}
	pathID := pp.pathID
	if id, name := svcatStr(in, "PathId"), svcatStr(in, "PathName"); id != "" || name != "" {
		port, err := st.resolvePath(p, id, name)
		if err != nil {
			return nil, err
		}
		pathID = svcatPathID(port.id)
	}
	tags := svcatList(in, "Tags")
	if len(tags) > svcatMaxTags {
		return nil, svcatInvalid("a provisioned product can have at most %d tags", svcatMaxTags)
	}
	tmpl := a.tmpl
	r, err := st.newRecord(pp, "UPDATE_PROVISIONED_PRODUCT", func() {
		pp.productID, pp.artifactID, pp.pathID, pp.params = p.id, a.id, pathID, vals
		if tags != nil {
			pp.tags = tags
		}
		if !pp.engine {
			pp.outputs = tmpl.outputValues(vals, pp.physicalID)
		}
	})
	if err != nil {
		return nil, err
	}
	r.productID, r.artifactID, r.pathID = p.id, a.id, pathID
	if err := st.rememberToken("UpdateProvisionedProduct", token, r.id); err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetail": r.detail()}, nil
}

func (st *serviceCatalogStore) terminateProvisioned(s *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	token := svcatStr(in, "TerminateToken")
	if id, ok := st.svcatIdempotent("TerminateProvisionedProduct", token); ok {
		if r := st.records[id]; r != nil {
			return map[string]any{"RecordDetail": r.detail()}, nil
		}
	}
	pp, err := st.provisionedByIDOrName(svcatStr(in, "ProvisionedProductId"), svcatStr(in, "ProvisionedProductName"))
	if err != nil {
		return nil, err
	}
	if pp.status == "UNDER_CHANGE" {
		return nil, svcatErr("InvalidStateException", "Provisioned product %s is UNDER_CHANGE", pp.id)
	}
	retain, _ := in["RetainPhysicalResources"].(bool)
	r, err := st.newRecord(pp, "TERMINATE_PROVISIONED_PRODUCT", func() {
		if pp.stackName != "" && !retain {
			name := pp.stackName
			st.post = append(st.post, func() { s.svcatDeleteStack(name) })
		}
		delete(st.provisioned, pp.id)
		for _, plan := range st.plans {
			if plan.ppID == pp.id {
				plan.ppID = ""
			}
		}
	})
	if err != nil {
		return nil, err
	}
	if err := st.rememberToken("TerminateProvisionedProduct", token, r.id); err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetail": r.detail()}, nil
}

func (st *serviceCatalogStore) importProvisioned(s *Server, in map[string]any, pre *svcatPre) (map[string]any, *svcatError) {
	token := svcatStr(in, "IdempotencyToken")
	if id, ok := st.svcatIdempotent("ImportAsProvisionedProduct", token); ok {
		if r := st.records[id]; r != nil {
			return map[string]any{"RecordDetail": r.detail()}, nil
		}
	}
	if !pre.stackFound {
		return nil, svcatInvalid("PhysicalId %s is not a CloudFormation stack", svcatStr(in, "PhysicalId"))
	}
	for _, pp := range st.provisioned {
		if pp.stackName == pre.stackName {
			return nil, svcatErr("DuplicateResourceException", "Stack %s is already a provisioned product", pre.stackName)
		}
	}
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	a, err := p.artifact(svcatStr(in, "ProvisioningArtifactId"), "")
	if err != nil {
		return nil, err
	}
	port, err := st.resolvePath(p, "", "")
	if err != nil {
		return nil, err
	}
	// Imported stacks keep their parameters; only template defaults are known here.
	r, err := st.startProvision(s, svcatStr(in, "ProvisionedProductName"), token, p, a, port, svcatDefaultParams(a.tmpl), nil)
	if err != nil {
		return nil, err
	}
	pp := st.provisioned[r.ppID]
	stack, physical := pre.stackName, svcatStr(in, "PhysicalId")
	r.typ = "IMPORT_PROVISIONED_PRODUCT"
	r.settle = func() {
		pp.stackName, pp.physicalID = stack, physical
		pp.outputs = a.tmpl.outputValues(pp.params, physical)
	}
	if err := st.rememberToken("ImportAsProvisionedProduct", token, r.id); err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetail": r.detail()}, nil
}

// svcatDefaultParams fills every parameter so an import never fails on a
// missing value it can't know.
func svcatDefaultParams(t *svcatTemplate) []any {
	out := []any{}
	for _, p := range t.params {
		if !p.hasDefault {
			v := ""
			if len(p.allowed) > 0 {
				v = svcatAnyString(p.allowed[0])
			}
			out = append(out, map[string]any{"Key": p.key, "Value": v})
		}
	}
	return out
}

func (st *serviceCatalogStore) describeProvisioned(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	pp, err := st.provisionedByIDOrName(svcatStr(in, "Id"), svcatStr(in, "Name"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProvisionedProductDetail": pp.detail(), "CloudWatchDashboards": []any{}}, nil
}

func (st *serviceCatalogStore) updateProvisionedProperties(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	pp, err := st.provisionedByIDOrName(svcatStr(in, "ProvisionedProductId"), "")
	if err != nil {
		return nil, err
	}
	token := svcatStr(in, "IdempotencyToken")
	if id, ok := st.svcatIdempotent("UpdateProvisionedProductProperties", token); ok {
		if r := st.records[id]; r != nil {
			return map[string]any{"ProvisionedProductId": pp.id, "ProvisionedProductProperties": in["ProvisionedProductProperties"], "RecordId": r.id, "Status": r.status}, nil
		}
	}
	props := svcatMap(in, "ProvisionedProductProperties")
	for k, v := range props {
		s, _ := v.(string)
		switch k {
		case "OWNER", "LAUNCH_ROLE":
			if !svcatPrincipalARN["IAM"].MatchString(s) && !strings.HasSuffix(s, ":root") {
				return nil, svcatInvalid("%s must be an IAM principal ARN", k)
			}
		}
	}
	r, err := st.newRecord(pp, "UPDATE_PROVISIONED_PRODUCT_PROPERTIES", func() {
		for k, v := range props {
			pp.props[k], _ = v.(string)
		}
	})
	if err != nil {
		return nil, err
	}
	r.engine = false // a properties change needs no engine
	if err := st.rememberToken("UpdateProvisionedProductProperties", token, r.id); err != nil {
		return nil, err
	}
	return map[string]any{"ProvisionedProductId": pp.id, "ProvisionedProductProperties": props, "RecordId": r.id, "Status": "CREATED"}, nil
}

func (st *serviceCatalogStore) provisionedOutputs(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	pp, err := st.provisionedByIDOrName(svcatStr(in, "ProvisionedProductId"), svcatStr(in, "ProvisionedProductName"))
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, k := range svcatList(in, "OutputKeys") {
		want[svcatAnyString(k)] = true
	}
	items := []any{}
	for _, o := range pp.outputs {
		if len(want) == 0 || want[svcatStr(svcatAsMap(o), "OutputKey")] {
			items = append(items, o)
		}
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Outputs": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) listStackInstances(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	pp, err := st.provisionedByIDOrName(svcatStr(in, "ProvisionedProductId"), "")
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, c := range st.constraints {
		if c.typ != "STACKSET" || c.product != pp.productID || svcatPathID(c.portfolio) != pp.pathID {
			continue
		}
		var cp map[string]any
		_ = json.Unmarshal([]byte(c.params), &cp)
		// Accounts x regions can be huge; stop at the cap instead of building it all.
	fill:
		for _, acct := range svcatList(cp, "AccountList") {
			for _, region := range svcatList(cp, "RegionList") {
				if len(items) >= svcatMaxStackInstances {
					break fill
				}
				items = append(items, map[string]any{"Account": acct, "Region": region, "StackInstanceStatus": "CURRENT"})
			}
		}
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"StackInstances": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) sortedProvisioned() []*svcatProvisioned {
	out := make([]*svcatProvisioned, 0, len(st.provisioned))
	for _, pp := range st.provisioned {
		out = append(out, pp)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].created.Equal(out[j].created) {
			return out[i].created.Before(out[j].created)
		}
		return out[i].id < out[j].id
	})
	return out
}

func (st *serviceCatalogStore) searchProvisioned(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	var queries []string
	for _, q := range svcatList(svcatMap(in, "Filters"), "SearchQuery") {
		queries = append(queries, strings.ToLower(svcatAnyString(q)))
	}
	var hits []map[string]any
	for _, pp := range st.sortedProvisioned() {
		attr := st.attribute(pp)
		ok := true
		for _, q := range queries {
			field, value, scoped := strings.Cut(q, ":")
			if !scoped {
				hay := strings.ToLower(pp.name + " " + pp.id + " " + pp.arn + " " + pp.status + " " + svcatAnyString(attr["ProductName"]))
				ok = ok && strings.Contains(hay, q)
				continue
			}
			got := ""
			switch field {
			case "name":
				got = pp.name
			case "id":
				got = pp.id
			case "arn":
				got = pp.arn
			case "status":
				got = pp.status
			case "productid":
				got = pp.productID
			case "provisioningartifactid":
				got = pp.artifactID
			case "type":
				got = "CFN_STACK"
			case "lastrecordid":
				got = pp.lastRecord
			}
			ok = ok && strings.EqualFold(got, value)
		}
		if ok {
			hits = append(hits, attr)
		}
	}
	key := map[string]string{"arn": "Arn", "id": "Id", "name": "Name", "lastRecordId": "LastRecordId"}[svcatStr(in, "SortBy")]
	if key != "" {
		desc := svcatStr(in, "SortOrder") == "DESCENDING"
		sort.SliceStable(hits, func(i, j int) bool {
			return (svcatAnyString(hits[i][key]) < svcatAnyString(hits[j][key])) != desc
		})
	}
	items := make([]any, 0, len(hits))
	for _, h := range hits {
		items = append(items, h)
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProvisionedProducts": page, "TotalResultsCount": len(items), "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) scanProvisioned(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	items := []any{}
	for _, pp := range st.sortedProvisioned() {
		items = append(items, pp.detail())
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProvisionedProducts": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) describeRecord(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	r, err := st.record(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	outputs := r.outputs
	if pp := st.provisioned[r.ppID]; pp != nil && r.status == "SUCCEEDED" && (r.typ == "PROVISION_PRODUCT" || r.typ == "UPDATE_PROVISIONED_PRODUCT" || r.typ == "IMPORT_PROVISIONED_PRODUCT") {
		outputs = pp.outputs
	}
	if outputs == nil {
		outputs = []any{}
	}
	page, next, err := svcatPage(outputs, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetail": r.detail(), "RecordOutputs": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) listRecordHistory(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	f := svcatMap(in, "SearchFilter")
	key, value := svcatStr(f, "Key"), svcatStr(f, "Value")
	items := []any{}
	for i := len(st.recordOrder) - 1; i >= 0; i-- { // newest first
		r := st.records[st.recordOrder[i]]
		if r == nil {
			continue
		}
		if (key == "product" && r.productID != value) || (key == "provisionedproduct" && r.ppID != value) {
			continue
		}
		items = append(items, r.detail())
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetails": page, "NextPageToken": next}, nil
}

// notify completes a Terraform / external record of the given type.
// ponytail: WorkflowToken is only checked for presence; Stackyard sends no
// engine SQS messages, so callers pass the record id they got back.
func (st *serviceCatalogStore) notify(in map[string]any, typ string) (map[string]any, *svcatError) {
	r, err := st.record(svcatStr(in, "RecordId"))
	if err != nil {
		return nil, err
	}
	if r.typ != typ {
		return nil, svcatInvalid("Record %s is a %s, not a %s", r.id, r.typ, typ)
	}
	token := svcatStr(in, "IdempotencyToken")
	if !r.pending {
		if r.notifyToken == token {
			return map[string]any{}, nil
		}
		return nil, svcatErr("InvalidStateException", "Record %s is already %s", r.id, r.status)
	}
	r.notifyToken = token
	if outs := svcatList(in, "Outputs"); outs != nil {
		if len(outs) > svcatMaxTemplateDefs {
			return nil, svcatInvalid("at most %d outputs", svcatMaxTemplateDefs)
		}
		r.outputs = outs
		if pp := st.provisioned[r.ppID]; pp != nil {
			pp.outputs = outs
		}
	}
	if pp := st.provisioned[r.ppID]; pp != nil {
		if rid := svcatMap(in, "ResourceIdentifier"); rid != nil {
			pp.physicalID = svcatStr(rid, "UniqueTag")
		}
	}
	st.completeRecord(r, svcatStr(in, "Status"), svcatStr(in, "FailureReason"))
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) notifyProvision(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	return st.notify(in, "PROVISION_PRODUCT")
}

func (st *serviceCatalogStore) notifyUpdate(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	return st.notify(in, "UPDATE_PROVISIONED_PRODUCT")
}

func (st *serviceCatalogStore) notifyTerminate(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	return st.notify(in, "TERMINATE_PROVISIONED_PRODUCT")
}

// ---- plans ----

func (p *svcatPlan) details() map[string]any {
	return map[string]any{"CreatedTime": svcatEpoch(p.created), "PathId": p.pathID, "ProductId": p.productID, "PlanName": p.name, "PlanId": p.id,
		"ProvisionProductId": p.ppID, "ProvisionProductName": p.ppName, "PlanType": p.typ, "ProvisioningArtifactId": p.artifactID,
		"Status": p.status, "UpdatedTime": svcatEpoch(p.updated), "NotificationArns": p.notifications, "ProvisioningParameters": p.params, "Tags": p.tags}
}

func (st *serviceCatalogStore) plan(id string) (*svcatPlan, *svcatError) {
	p := st.plans[id]
	if p == nil {
		return nil, svcatNotFound("Plan %s not found", id)
	}
	return p, nil
}

func (st *serviceCatalogStore) createPlan(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	created := func(p *svcatPlan) map[string]any {
		return map[string]any{"PlanName": p.name, "PlanId": p.id, "ProvisionProductId": p.ppID, "ProvisionedProductName": p.ppName, "ProvisioningArtifactId": p.artifactID}
	}
	token := svcatStr(in, "IdempotencyToken")
	if id, ok := st.svcatIdempotent("CreateProvisionedProductPlan", token); ok {
		if p := st.plans[id]; p != nil {
			return created(p), nil
		}
	}
	prod, a, port, err := st.launchTarget(in)
	if err != nil {
		return nil, err
	}
	if prod.typ != "CLOUD_FORMATION_TEMPLATE" {
		return nil, svcatInvalid("Plans are only supported for CloudFormation products")
	}
	ppName := svcatStr(in, "ProvisionedProductName")
	ppID := ""
	var previous map[string]string
	for _, pp := range st.provisioned {
		if pp.name == ppName {
			ppID, previous = pp.id, pp.params
		}
	}
	if _, err := a.tmpl.resolveParams(svcatList(in, "ProvisioningParameters"), previous); err != nil {
		return nil, err
	}
	if len(st.plans) >= svcatMaxPlans {
		return nil, svcatErr("LimitExceededException", "an account can have at most %d plans", svcatMaxPlans)
	}
	now := time.Now().UTC()
	p := &svcatPlan{id: svcatNewID("plan-"), name: svcatStr(in, "PlanName"), typ: svcatStr(in, "PlanType"), status: "CREATE_IN_PROGRESS",
		productID: prod.id, artifactID: a.id, pathID: svcatPathID(port.id), ppName: ppName, ppID: ppID, created: now, updated: now,
		params: svcatList(in, "ProvisioningParameters"), tags: svcatList(in, "Tags"), notifications: svcatList(in, "NotificationArns")}
	if len(p.tags) > svcatMaxTags {
		return nil, svcatInvalid("a plan can have at most %d tags", svcatMaxTags)
	}
	if err := st.rememberToken("CreateProvisionedProductPlan", token, p.id); err != nil {
		return nil, err
	}
	st.plans[p.id] = p
	return created(p), nil
}

func (st *serviceCatalogStore) describePlan(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.plan(svcatStr(in, "PlanId"))
	if err != nil {
		return nil, err
	}
	changes := []any{}
	if prod := st.products[p.productID]; prod != nil {
		if a, err := prod.artifact(p.artifactID, ""); err == nil {
			action := "Add"
			if p.ppID != "" {
				action = "Modify"
			}
			for _, r := range a.tmpl.resources {
				changes = append(changes, map[string]any{"Action": action, "LogicalResourceId": r.logical, "ResourceType": r.typ})
			}
		}
	}
	page, next, err := svcatPage(changes, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProvisionedProductPlanDetails": p.details(), "ResourceChanges": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) executePlan(s *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	token := svcatStr(in, "IdempotencyToken")
	if id, ok := st.svcatIdempotent("ExecuteProvisionedProductPlan", token); ok {
		if r := st.records[id]; r != nil {
			return map[string]any{"RecordDetail": r.detail()}, nil
		}
	}
	p, err := st.plan(svcatStr(in, "PlanId"))
	if err != nil {
		return nil, err
	}
	if p.status != "CREATE_SUCCESS" {
		return nil, svcatErr("InvalidStateException", "Plan %s is %s; only a CREATE_SUCCESS plan can run", p.id, p.status)
	}
	prod, err := st.product(p.productID)
	if err != nil {
		return nil, err
	}
	a, err := prod.artifact(p.artifactID, "")
	if err != nil {
		return nil, err
	}
	var r *svcatRecord
	if pp := st.provisioned[p.ppID]; pp != nil {
		vals, err := a.tmpl.resolveParams(p.params, pp.params)
		if err != nil {
			return nil, err
		}
		tmpl := a.tmpl
		if r, err = st.newRecord(pp, "UPDATE_PROVISIONED_PRODUCT", func() {
			pp.artifactID, pp.params = a.id, vals
			pp.outputs = tmpl.outputValues(vals, pp.physicalID)
		}); err != nil {
			return nil, err
		}
	} else {
		port := st.portfolios["port-"+strings.TrimPrefix(p.pathID, "lpv3-")]
		if port == nil {
			return nil, svcatNotFound("The plan's launch path no longer exists")
		}
		if r, err = st.startProvision(s, p.ppName, token, prod, a, port, p.params, p.tags); err != nil {
			return nil, err
		}
		p.ppID = r.ppID
	}
	p.status, p.updated = "EXECUTE_IN_PROGRESS", time.Now().UTC()
	if err := st.rememberToken("ExecuteProvisionedProductPlan", token, r.id); err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetail": r.detail()}, nil
}

func (st *serviceCatalogStore) deletePlan(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.plan(svcatStr(in, "PlanId"))
	if err != nil {
		return nil, err
	}
	delete(st.plans, p.id)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) listPlans(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	var ps []*svcatPlan
	for _, p := range st.plans {
		if id := svcatStr(in, "ProvisionProductId"); id == "" || p.ppID == id {
			ps = append(ps, p)
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].created.Before(ps[j].created) || (ps[i].created.Equal(ps[j].created) && ps[i].id < ps[j].id)
	})
	items := make([]any, 0, len(ps))
	for _, p := range ps {
		items = append(items, map[string]any{"PlanName": p.name, "PlanId": p.id, "ProvisionProductId": p.ppID, "ProvisionProductName": p.ppName,
			"PlanType": p.typ, "ProvisioningArtifactId": p.artifactID})
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProvisionedProductPlans": page, "NextPageToken": next}, nil
}

// ---- service actions ----

func (a *svcatAction) summary() map[string]any {
	return map[string]any{"Id": a.id, "Name": a.name, "Description": a.desc, "DefinitionType": a.defType}
}

func (a *svcatAction) detailOut() map[string]any {
	def := map[string]any{}
	for k, v := range a.def {
		def[k] = v
	}
	return map[string]any{"ServiceActionDetail": map[string]any{"ServiceActionSummary": a.summary(), "Definition": def}}
}

// svcatActionParams parses the SSM document's Parameters definition.
func svcatActionParams(def map[string]string) ([]any, *svcatError) {
	raw := def["Parameters"]
	if raw == "" {
		return []any{}, nil
	}
	var params []map[string]any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return nil, svcatInvalid("Definition Parameters must be a JSON list of {Name, Type}")
	}
	out := []any{}
	for _, p := range params {
		typ := svcatStr(p, "Type")
		if svcatStr(p, "Name") == "" || (typ != "TARGET" && typ != "TEXT_VALUE") {
			return nil, svcatInvalid("each Definition parameter needs a Name and a Type of TARGET or TEXT_VALUE")
		}
		out = append(out, map[string]any{"Name": p["Name"], "Type": typ})
	}
	return out, nil
}

func svcatCheckActionDef(def map[string]string) *svcatError {
	if def["Name"] == "" || def["Version"] == "" {
		return svcatInvalid("Definition needs the SSM document Name and Version")
	}
	if r := def["AssumeRole"]; r != "" && r != "LAUNCH_ROLE" && !svcatPrincipalARN["IAM"].MatchString(r) {
		return svcatInvalid("AssumeRole must be LAUNCH_ROLE or an IAM role ARN")
	}
	_, err := svcatActionParams(def)
	return err
}

func svcatStringMap(m map[string]any) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k], _ = v.(string)
	}
	return out
}

func (st *serviceCatalogStore) action(id string) (*svcatAction, *svcatError) {
	a := st.actions[id]
	if a == nil {
		return nil, svcatNotFound("Service action %s not found", id)
	}
	return a, nil
}

func (st *serviceCatalogStore) createAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	if id, ok := st.svcatIdempotent("CreateServiceAction", svcatStr(in, "IdempotencyToken")); ok {
		if a := st.actions[id]; a != nil {
			return a.detailOut(), nil
		}
	}
	def := svcatStringMap(svcatMap(in, "Definition"))
	if err := svcatCheckActionDef(def); err != nil {
		return nil, err
	}
	if len(st.actions) >= svcatMaxActions {
		return nil, svcatErr("LimitExceededException", "an account can have at most %d service actions", svcatMaxActions)
	}
	a := &svcatAction{id: svcatNewID("act-"), name: svcatStr(in, "Name"), desc: svcatStr(in, "Description"), defType: svcatStr(in, "DefinitionType"),
		def: def, assoc: map[[2]string]bool{}}
	if err := st.rememberToken("CreateServiceAction", svcatStr(in, "IdempotencyToken"), a.id); err != nil {
		return nil, err
	}
	st.actions[a.id] = a
	return a.detailOut(), nil
}

func (st *serviceCatalogStore) describeAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	a, err := st.action(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	return a.detailOut(), nil
}

func (st *serviceCatalogStore) updateAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	a, err := st.action(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if d := svcatMap(in, "Definition"); d != nil {
		def := make(map[string]string, len(a.def))
		for k, v := range a.def {
			def[k] = v
		}
		for k, v := range svcatStringMap(d) {
			def[k] = v
		}
		if err := svcatCheckActionDef(def); err != nil {
			return nil, err
		}
		a.def = def
	}
	if v, ok := in["Name"].(string); ok {
		a.name = v
	}
	if v, ok := in["Description"].(string); ok {
		a.desc = v
	}
	return a.detailOut(), nil
}

func (st *serviceCatalogStore) deleteAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	a, err := st.action(svcatStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if len(a.assoc) > 0 {
		return nil, svcatErr("ResourceInUseException", "Service action %s is still associated with provisioning artifacts", a.id)
	}
	delete(st.actions, a.id)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) sortedActions(keep func(*svcatAction) bool) []any {
	var as []*svcatAction
	for _, a := range st.actions {
		if keep == nil || keep(a) {
			as = append(as, a)
		}
	}
	sort.Slice(as, func(i, j int) bool { return as[i].id < as[j].id })
	out := make([]any, 0, len(as))
	for _, a := range as {
		out = append(out, a.summary())
	}
	return out
}

func (st *serviceCatalogStore) listActions(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	page, next, err := svcatPage(st.sortedActions(nil), in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ServiceActionSummaries": page, "NextPageToken": next}, nil
}

// actionTarget checks the action, product and artifact of an association.
func (st *serviceCatalogStore) actionTarget(actionID, productID, artifactID string) (*svcatAction, [2]string, *svcatError) {
	a, err := st.action(actionID)
	if err != nil {
		return nil, [2]string{}, err
	}
	p, err := st.product(productID)
	if err != nil {
		return nil, [2]string{}, err
	}
	if _, err := p.artifact(artifactID, ""); err != nil {
		return nil, [2]string{}, err
	}
	return a, [2]string{p.id, artifactID}, nil
}

func (st *serviceCatalogStore) associateAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	a, key, err := st.actionTarget(svcatStr(in, "ServiceActionId"), svcatStr(in, "ProductId"), svcatStr(in, "ProvisioningArtifactId"))
	if err != nil {
		return nil, err
	}
	if a.assoc[key] {
		return nil, svcatErr("DuplicateResourceException", "Service action %s is already associated", a.id)
	}
	if len(a.assoc) >= svcatMaxArtifacts*10 {
		return nil, svcatErr("LimitExceededException", "too many associations for service action %s", a.id)
	}
	a.assoc[key] = true
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) disassociateAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	a, key, err := st.actionTarget(svcatStr(in, "ServiceActionId"), svcatStr(in, "ProductId"), svcatStr(in, "ProvisioningArtifactId"))
	if err != nil {
		return nil, err
	}
	if !a.assoc[key] {
		return nil, svcatNotFound("Service action %s is not associated with %s/%s", a.id, key[0], key[1])
	}
	delete(a.assoc, key)
	return map[string]any{}, nil
}

func (st *serviceCatalogStore) batchActions(in map[string]any, associate bool) map[string]any {
	failed := []any{}
	for _, item := range svcatList(in, "ServiceActionAssociations") {
		m := svcatAsMap(item)
		args := map[string]any{"ServiceActionId": m["ServiceActionId"], "ProductId": m["ProductId"], "ProvisioningArtifactId": m["ProvisioningArtifactId"]}
		var err *svcatError
		if associate {
			_, err = st.associateAction(nil, args, nil)
		} else {
			_, err = st.disassociateAction(nil, args, nil)
		}
		if err != nil {
			code := map[string]string{"DuplicateResourceException": "DUPLICATE_RESOURCE", "ResourceNotFoundException": "RESOURCE_NOT_FOUND",
				"LimitExceededException": "LIMIT_EXCEEDED", "InvalidParametersException": "INVALID_PARAMETER"}[err.code]
			if code == "" {
				code = "INTERNAL_FAILURE"
			}
			failed = append(failed, map[string]any{"ServiceActionId": m["ServiceActionId"], "ProductId": m["ProductId"],
				"ProvisioningArtifactId": m["ProvisioningArtifactId"], "ErrorCode": code, "ErrorMessage": err.msg})
		}
	}
	return map[string]any{"FailedServiceActionAssociations": failed}
}

func (st *serviceCatalogStore) batchAssociateActions(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	return st.batchActions(in, true), nil
}

func (st *serviceCatalogStore) batchDisassociateActions(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	return st.batchActions(in, false), nil
}

func (st *serviceCatalogStore) listActionsForArtifact(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	p, err := st.product(svcatStr(in, "ProductId"))
	if err != nil {
		return nil, err
	}
	key := [2]string{p.id, svcatStr(in, "ProvisioningArtifactId")}
	if _, err := p.artifact(key[1], ""); err != nil {
		return nil, err
	}
	page, next, err := svcatPage(st.sortedActions(func(a *svcatAction) bool { return a.assoc[key] }), in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ServiceActionSummaries": page, "NextPageToken": next}, nil
}

func (st *serviceCatalogStore) listArtifactsForAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	a, err := st.action(svcatStr(in, "ServiceActionId"))
	if err != nil {
		return nil, err
	}
	keys := make([][2]string, 0, len(a.assoc))
	for k := range a.assoc {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i][0]+keys[i][1] < keys[j][0]+keys[j][1] })
	items := []any{}
	for _, k := range keys {
		p := st.products[k[0]]
		if p == nil {
			continue
		}
		art, err := p.artifact(k[1], "")
		if err != nil {
			continue
		}
		items = append(items, map[string]any{"ProductViewSummary": p.summary(st), "ProvisioningArtifact": map[string]any{
			"Id": art.id, "Name": art.name, "Description": art.desc, "CreatedTime": svcatEpoch(art.created), "Guidance": art.guidance}})
	}
	page, next, err := svcatPage(items, in, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ProvisioningArtifactViews": page, "NextPageToken": next}, nil
}

// actionFor checks the action is associated with the provisioned product's artifact.
func (st *serviceCatalogStore) actionFor(ppID, actionID string) (*svcatProvisioned, *svcatAction, *svcatError) {
	pp, err := st.provisionedByIDOrName(ppID, "")
	if err != nil {
		return nil, nil, err
	}
	a, err := st.action(actionID)
	if err != nil {
		return nil, nil, err
	}
	if !a.assoc[[2]string{pp.productID, pp.artifactID}] {
		return nil, nil, svcatInvalid("Service action %s is not associated with provisioned product %s's artifact", a.id, pp.id)
	}
	return pp, a, nil
}

func (st *serviceCatalogStore) actionExecutionParameters(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	pp, a, err := st.actionFor(svcatStr(in, "ProvisionedProductId"), svcatStr(in, "ServiceActionId"))
	if err != nil {
		return nil, err
	}
	params, err := svcatActionParams(a.def)
	if err != nil {
		return nil, err
	}
	for _, p := range params {
		m := p.(map[string]any)
		m["DefaultValues"] = []any{}
		if m["Type"] == "TARGET" && pp.physicalID != "" {
			m["DefaultValues"] = []any{pp.physicalID}
		}
	}
	return map[string]any{"ServiceActionParameters": params}, nil
}

func (st *serviceCatalogStore) executeAction(_ *Server, in map[string]any, _ *svcatPre) (map[string]any, *svcatError) {
	token := svcatStr(in, "ExecuteToken")
	if id, ok := st.svcatIdempotent("ExecuteProvisionedProductServiceAction", token); ok {
		if r := st.records[id]; r != nil {
			return map[string]any{"RecordDetail": r.detail()}, nil
		}
	}
	pp, a, err := st.actionFor(svcatStr(in, "ProvisionedProductId"), svcatStr(in, "ServiceActionId"))
	if err != nil {
		return nil, err
	}
	if pp.status == "UNDER_CHANGE" {
		return nil, svcatErr("InvalidStateException", "Provisioned product %s is UNDER_CHANGE", pp.id)
	}
	declared, _ := svcatActionParams(a.def)
	names := map[string]bool{}
	for _, d := range declared {
		names[svcatAnyString(d.(map[string]any)["Name"])] = true
	}
	for k := range svcatMap(in, "Parameters") {
		if !names[k] {
			return nil, svcatInvalid("Parameter %s is not defined by service action %s", k, a.id)
		}
	}
	r, err := st.newRecord(pp, "EXECUTE_PROVISIONED_PRODUCT_SERVICE_ACTION", nil)
	if err != nil {
		return nil, err
	}
	r.engine = false // SSM automation, not the provisioning engine
	if err := st.rememberToken("ExecuteProvisionedProductServiceAction", token, r.id); err != nil {
		return nil, err
	}
	return map[string]any{"RecordDetail": r.detail()}, nil
}
