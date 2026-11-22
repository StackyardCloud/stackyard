package server

import (
	"net/url"
	"regexp"
	"sort"
)

var baVpcConfigIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{30}[a-z0-9]$`)

func bedrockAgentKBHandlers() map[string]baHandler {
	return map[string]baHandler{
		"CreateKnowledgeBase":          (*bedrockAgentStore).createKnowledgeBase,
		"GetKnowledgeBase":             (*bedrockAgentStore).getKnowledgeBase,
		"UpdateKnowledgeBase":          (*bedrockAgentStore).updateKnowledgeBase,
		"DeleteKnowledgeBase":          (*bedrockAgentStore).deleteKnowledgeBase,
		"ListKnowledgeBases":           (*bedrockAgentStore).listKnowledgeBases,
		"CreateDataSource":             (*bedrockAgentStore).createDataSource,
		"GetDataSource":                (*bedrockAgentStore).getDataSource,
		"UpdateDataSource":             (*bedrockAgentStore).updateDataSource,
		"DeleteDataSource":             (*bedrockAgentStore).deleteDataSource,
		"ListDataSources":              (*bedrockAgentStore).listDataSources,
		"StartIngestionJob":            (*bedrockAgentStore).startIngestionJob,
		"GetIngestionJob":              (*bedrockAgentStore).getIngestionJob,
		"StopIngestionJob":             (*bedrockAgentStore).stopIngestionJob,
		"ListIngestionJobs":            (*bedrockAgentStore).listIngestionJobs,
		"IngestKnowledgeBaseDocuments": (*bedrockAgentStore).ingestDocuments,
		"GetKnowledgeBaseDocuments":    (*bedrockAgentStore).getDocuments,
		"DeleteKnowledgeBaseDocuments": (*bedrockAgentStore).deleteDocuments,
		"ListKnowledgeBaseDocuments":   (*bedrockAgentStore).listDocuments,
		"CreateVpcConfiguration":       (*bedrockAgentStore).createVpcConfiguration,
		"GetVpcConfiguration":          (*bedrockAgentStore).getVpcConfiguration,
		"DeleteVpcConfiguration":       (*bedrockAgentStore).deleteVpcConfiguration,
		"ListVpcConfigurations":        (*bedrockAgentStore).listVpcConfigurations,
	}
}

// --- knowledge bases ---

func (st *bedrockAgentStore) knowledgeBase(in map[string]any) (*baKB, *bedrockAgentError) {
	if err := baCheckID(in, "knowledgeBaseId"); err != nil {
		return nil, err
	}
	kb := st.kbs[baStr(in, "knowledgeBaseId")]
	if kb == nil {
		return nil, baNotFound("knowledge base", baStr(in, "knowledgeBaseId"))
	}
	return kb, nil
}

func (st *bedrockAgentStore) validateKnowledgeBase(in map[string]any, selfID string) *bedrockAgentError {
	if err := baCheckName(in, "name"); err != nil {
		return err
	}
	if err := baRequire(in, "roleArn", "knowledgeBaseConfiguration"); err != nil {
		return err
	}
	for id, kb := range st.kbs {
		if id != selfID && baStr(kb.kb, "name") == baStr(in, "name") {
			return baConflict("a knowledge base named " + baStr(in, "name") + " already exists")
		}
	}
	cfg, _ := in["knowledgeBaseConfiguration"].(map[string]any)
	if err := baRequire(cfg, "type"); err != nil {
		return baValidation("knowledgeBaseConfiguration.type is required")
	}
	if err := baCheckEnum(cfg, "type", "VECTOR", "KENDRA", "SQL", "MANAGED"); err != nil {
		return err
	}
	switch baStr(cfg, "type") {
	case "VECTOR":
		if baSubStr(cfg, "vectorKnowledgeBaseConfiguration", "embeddingModelArn") == "" {
			return baValidation("vectorKnowledgeBaseConfiguration.embeddingModelArn is required for a VECTOR knowledge base")
		}
		if in["storageConfiguration"] == nil {
			return baValidation("storageConfiguration is required for a VECTOR knowledge base")
		}
	case "KENDRA":
		if baSubStr(cfg, "kendraKnowledgeBaseConfiguration", "kendraIndexArn") == "" {
			return baValidation("kendraKnowledgeBaseConfiguration.kendraIndexArn is required for a KENDRA knowledge base")
		}
	}
	return nil
}

func (st *bedrockAgentStore) createKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	if err := st.validateKnowledgeBase(in, ""); err != nil {
		return nil, err
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["knowledgeBaseId"] = id
	rec["knowledgeBaseArn"] = baARN("knowledge-base/" + id)
	rec["status"] = "ACTIVE"
	rec["createdAt"], rec["updatedAt"] = now, now
	st.kbs[id] = &baKB{kb: rec, dataSources: map[string]*baDataSource{}, vpcs: map[string]map[string]any{}}
	st.setTags(rec["knowledgeBaseArn"].(string), in["tags"])
	return map[string]any{"knowledgeBase": baWithStatus(rec, "status", "CREATING")}, nil
}

func (st *bedrockAgentStore) getKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"knowledgeBase": kb.kb}, nil
}

func (st *bedrockAgentStore) updateKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, err
	}
	if err := st.validateKnowledgeBase(in, baStr(in, "knowledgeBaseId")); err != nil {
		return nil, err
	}
	baMerge(kb.kb, in)
	kb.kb["updatedAt"] = baNow()
	return map[string]any{"knowledgeBase": baWithStatus(kb.kb, "status", "UPDATING")}, nil
}

func (st *bedrockAgentStore) deleteKnowledgeBase(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, err
	}
	id := baStr(in, "knowledgeBaseId")
	st.forget(baStr(kb.kb, "knowledgeBaseArn"))
	delete(st.kbs, id)
	return map[string]any{"knowledgeBaseId": id, "status": "DELETING"}, nil
}

func (st *bedrockAgentStore) listKnowledgeBases(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	all := map[string]map[string]any{}
	for id, kb := range st.kbs {
		all[id] = kb.kb
	}
	return baListProjected(all, in, q, "knowledgeBaseSummaries", "knowledgeBaseId", "name", "description", "status", "updatedAt")
}

// --- data sources ---

func (st *bedrockAgentStore) dataSource(in map[string]any) (*baKB, *baDataSource, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, nil, err
	}
	if err := baCheckID(in, "dataSourceId"); err != nil {
		return nil, nil, err
	}
	ds := kb.dataSources[baStr(in, "dataSourceId")]
	if ds == nil {
		return nil, nil, baNotFound("data source", baStr(in, "dataSourceId"))
	}
	return kb, ds, nil
}

func validateDataSource(in map[string]any) *bedrockAgentError {
	if err := baCheckName(in, "name"); err != nil {
		return err
	}
	if err := baRequire(in, "dataSourceConfiguration"); err != nil {
		return err
	}
	cfg, _ := in["dataSourceConfiguration"].(map[string]any)
	if err := baRequire(cfg, "type"); err != nil {
		return baValidation("dataSourceConfiguration.type is required")
	}
	if err := baCheckEnum(cfg, "type", "S3", "WEB", "CONFLUENCE", "SALESFORCE", "SHAREPOINT", "CUSTOM", "REDSHIFT_METADATA", "MANAGED_KNOWLEDGE_BASE_CONNECTOR"); err != nil {
		return err
	}
	if baStr(cfg, "type") == "S3" && baSubStr(cfg, "s3Configuration", "bucketArn") == "" {
		return baValidation("s3Configuration.bucketArn is required for an S3 data source")
	}
	return baCheckEnum(in, "dataDeletionPolicy", "RETAIN", "DELETE")
}

func baDataSourceType(rec map[string]any) string {
	return baSubStr(rec, "dataSourceConfiguration", "type")
}

func (st *bedrockAgentStore) createDataSource(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, err
	}
	if err := validateDataSource(in); err != nil {
		return nil, err
	}
	for _, ds := range kb.dataSources {
		if baStr(ds.ds, "name") == baStr(in, "name") {
			return nil, baConflict("a data source named " + baStr(in, "name") + " already exists")
		}
	}
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["dataSourceId"] = id
	rec["status"] = "AVAILABLE"
	if rec["dataDeletionPolicy"] == nil {
		rec["dataDeletionPolicy"] = "DELETE"
	}
	rec["createdAt"], rec["updatedAt"] = now, now
	kb.dataSources[id] = &baDataSource{ds: rec, jobs: map[string]map[string]any{}, docs: map[string]map[string]any{}}
	return map[string]any{"dataSource": rec}, nil
}

func (st *bedrockAgentStore) getDataSource(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"dataSource": ds.ds}, nil
}

func (st *bedrockAgentStore) updateDataSource(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	if err := validateDataSource(in); err != nil {
		return nil, err
	}
	if baDataSourceType(in) != baDataSourceType(ds.ds) {
		return nil, baValidation("the data source type can't be changed")
	}
	for id, other := range kb.dataSources {
		if id != baStr(in, "dataSourceId") && baStr(other.ds, "name") == baStr(in, "name") {
			return nil, baConflict("a data source named " + baStr(in, "name") + " already exists")
		}
	}
	baMerge(ds.ds, in)
	ds.ds["updatedAt"] = baNow()
	return map[string]any{"dataSource": ds.ds}, nil
}

func (st *bedrockAgentStore) deleteDataSource(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, _, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	id := baStr(in, "dataSourceId")
	delete(kb.dataSources, id)
	return map[string]any{"knowledgeBaseId": baStr(in, "knowledgeBaseId"), "dataSourceId": id, "status": "DELETING"}, nil
}

func (st *bedrockAgentStore) listDataSources(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, err
	}
	all := map[string]map[string]any{}
	for id, ds := range kb.dataSources {
		all[id] = ds.ds
	}
	return baListProjected(all, in, q, "dataSourceSummaries", "knowledgeBaseId", "dataSourceId", "name", "status", "description", "updatedAt")
}

// --- ingestion jobs ---

// advance finishes running jobs: a job is IN_PROGRESS until it is next read,
// which gives StopIngestionJob a window and Get a terminal state.
func (ds *baDataSource) advance() {
	for _, job := range ds.jobs {
		if job["status"] == "IN_PROGRESS" {
			job["status"] = "COMPLETE"
			job["updatedAt"] = baNow()
			job["statistics"] = map[string]any{
				"numberOfDocumentsScanned": len(ds.docs), "numberOfNewDocumentsIndexed": len(ds.docs),
				"numberOfModifiedDocumentsIndexed": 0, "numberOfDocumentsDeleted": 0, "numberOfDocumentsFailed": 0,
				"numberOfMetadataDocumentsScanned": 0, "numberOfMetadataDocumentsModified": 0,
			}
		}
	}
}

func (st *bedrockAgentStore) startIngestionJob(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	ds.advance()
	id := baNewID(baIDAlphabet, 10)
	now := baNow()
	rec := baRecord(in)
	rec["ingestionJobId"] = id
	rec["status"] = "IN_PROGRESS"
	rec["startedAt"], rec["updatedAt"] = now, now
	ds.jobs[id] = rec
	return map[string]any{"ingestionJob": baWithStatus(rec, "status", "STARTING")}, nil
}

func (st *bedrockAgentStore) ingestionJob(in map[string]any) (*baDataSource, map[string]any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, nil, err
	}
	if err := baCheckID(in, "ingestionJobId"); err != nil {
		return nil, nil, err
	}
	job := ds.jobs[baStr(in, "ingestionJobId")]
	if job == nil {
		return nil, nil, baNotFound("ingestion job", baStr(in, "ingestionJobId"))
	}
	return ds, job, nil
}

func (st *bedrockAgentStore) getIngestionJob(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	ds, job, err := st.ingestionJob(in)
	if err != nil {
		return nil, err
	}
	ds.advance()
	return map[string]any{"ingestionJob": job}, nil
}

func (st *bedrockAgentStore) stopIngestionJob(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, job, err := st.ingestionJob(in)
	if err != nil {
		return nil, err
	}
	if job["status"] != "IN_PROGRESS" {
		return nil, baConflict("ingestion job is " + baStr(job, "status") + " and can't be stopped")
	}
	job["status"] = "STOPPED"
	job["updatedAt"] = baNow()
	return map[string]any{"ingestionJob": baWithStatus(job, "status", "STOPPING")}, nil
}

func (st *bedrockAgentStore) listIngestionJobs(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	ds.advance()
	statuses := map[string]bool{}
	filters, _ := in["filters"].([]any)
	for _, f := range filters {
		fm, _ := f.(map[string]any)
		if baStr(fm, "attribute") != "STATUS" || baStr(fm, "operator") != "EQ" {
			return nil, baValidation("filters support attribute STATUS with operator EQ")
		}
		values, _ := fm["values"].([]any)
		for _, v := range values {
			s, _ := v.(string)
			statuses[s] = true
		}
	}
	jobs := baSorted(ds.jobs)
	if len(statuses) > 0 {
		kept := jobs[:0:0]
		for _, j := range jobs {
			if statuses[baStr(j, "status")] {
				kept = append(kept, j)
			}
		}
		jobs = kept
	}
	if sortBy, ok := in["sortBy"].(map[string]any); ok {
		key := map[string]string{"STATUS": "status", "STARTED_AT": "startedAt"}[baStr(sortBy, "attribute")]
		if key == "" {
			return nil, baValidation("sortBy.attribute must be STATUS or STARTED_AT")
		}
		desc := baStr(sortBy, "order") == "DESCENDING"
		sort.SliceStable(jobs, func(i, j int) bool {
			if desc {
				return baStr(jobs[i], key) > baStr(jobs[j], key)
			}
			return baStr(jobs[i], key) < baStr(jobs[j], key)
		})
	}
	page, next, err := baPage(jobs, in, q)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(page))
	for i, j := range page {
		out[i] = baProject(j, "knowledgeBaseId", "dataSourceId", "ingestionJobId", "description", "status", "startedAt", "updatedAt", "statistics")
	}
	return baList("ingestionJobSummaries", out, next), nil
}

// --- documents ---

// baDocumentKey returns the identifier of a document (or document identifier)
// and its store key, or a validation error.
func baDocumentKey(identifier map[string]any) (map[string]any, string, *bedrockAgentError) {
	switch baStr(identifier, "dataSourceType") {
	case "CUSTOM":
		id := baSubStr(identifier, "custom", "id")
		if id == "" {
			return nil, "", baValidation("custom.id is required for a CUSTOM document")
		}
		return map[string]any{"dataSourceType": "CUSTOM", "custom": map[string]any{"id": id}}, "CUSTOM:" + id, nil
	case "S3":
		uri := baSubStr(identifier, "s3", "uri")
		if uri == "" {
			return nil, "", baValidation("s3.uri is required for an S3 document")
		}
		return map[string]any{"dataSourceType": "S3", "s3": map[string]any{"uri": uri}}, "S3:" + uri, nil
	}
	return nil, "", baValidation("dataSourceType must be CUSTOM or S3")
}

// baDocumentIdentifier derives the identifier of an ingested document from its content.
func baDocumentIdentifier(content map[string]any) (map[string]any, string, *bedrockAgentError) {
	switch baStr(content, "dataSourceType") {
	case "CUSTOM":
		custom, _ := content["custom"].(map[string]any)
		if err := baCheckEnum(custom, "sourceType", "IN_LINE", "S3_LOCATION"); err != nil || baStr(custom, "sourceType") == "" {
			return nil, "", baValidation("custom.sourceType must be IN_LINE or S3_LOCATION")
		}
		if baStr(custom, "sourceType") == "IN_LINE" && custom["inlineContent"] == nil {
			return nil, "", baValidation("custom.inlineContent is required for an IN_LINE document")
		}
		return baDocumentKey(map[string]any{"dataSourceType": "CUSTOM", "custom": map[string]any{"id": baSubStr(custom, "customDocumentIdentifier", "id")}})
	case "S3":
		s3, _ := content["s3"].(map[string]any)
		return baDocumentKey(map[string]any{"dataSourceType": "S3", "s3": map[string]any{"uri": baSubStr(s3, "s3Location", "uri")}})
	}
	return nil, "", baValidation("content.dataSourceType must be CUSTOM or S3")
}

func baDocumentDetail(in map[string]any, identifier map[string]any, status string) map[string]any {
	return map[string]any{"knowledgeBaseId": baStr(in, "knowledgeBaseId"), "dataSourceId": baStr(in, "dataSourceId"), "identifier": identifier, "status": status, "updatedAt": baNow()}
}

func (st *bedrockAgentStore) ingestDocuments(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	docs, _ := in["documents"].([]any)
	if len(docs) == 0 || len(docs) > 10 {
		return nil, baValidation("documents must contain between 1 and 10 documents")
	}
	type pending struct {
		key string
		rec map[string]any
	}
	batch := make([]pending, 0, len(docs))
	for _, d := range docs {
		dm, _ := d.(map[string]any)
		content, _ := dm["content"].(map[string]any)
		identifier, key, err := baDocumentIdentifier(content)
		if err != nil {
			return nil, err
		}
		if baStr(content, "dataSourceType") != baDataSourceType(ds.ds) {
			return nil, baValidation("a " + baStr(content, "dataSourceType") + " document can't be ingested into a " + baDataSourceType(ds.ds) + " data source")
		}
		rec := baDocumentDetail(in, identifier, "INDEXED")
		rec["content"], rec["metadata"] = content, dm["metadata"]
		batch = append(batch, pending{key, rec})
	}
	details := make([]map[string]any, len(batch))
	for i, p := range batch {
		ds.docs[p.key] = p.rec
		details[i] = baDocumentDetail(in, p.rec["identifier"].(map[string]any), "STARTING")
	}
	return map[string]any{"documentDetails": details}, nil
}

func (st *bedrockAgentStore) documentKeys(in map[string]any) ([]map[string]any, []string, *bedrockAgentError) {
	ids, _ := in["documentIdentifiers"].([]any)
	if len(ids) == 0 || len(ids) > 10 {
		return nil, nil, baValidation("documentIdentifiers must contain between 1 and 10 identifiers")
	}
	identifiers := make([]map[string]any, len(ids))
	keys := make([]string, len(ids))
	for i, raw := range ids {
		m, _ := raw.(map[string]any)
		identifier, key, err := baDocumentKey(m)
		if err != nil {
			return nil, nil, err
		}
		identifiers[i], keys[i] = identifier, key
	}
	return identifiers, keys, nil
}

func (st *bedrockAgentStore) getDocuments(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	identifiers, keys, err := st.documentKeys(in)
	if err != nil {
		return nil, err
	}
	details := make([]map[string]any, len(keys))
	for i, key := range keys {
		if doc := ds.docs[key]; doc != nil {
			details[i] = baProject(doc, "knowledgeBaseId", "dataSourceId", "identifier", "status", "updatedAt")
		} else {
			details[i] = baDocumentDetail(in, identifiers[i], "NOT_FOUND")
		}
	}
	return map[string]any{"documentDetails": details}, nil
}

func (st *bedrockAgentStore) deleteDocuments(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	identifiers, keys, err := st.documentKeys(in)
	if err != nil {
		return nil, err
	}
	details := make([]map[string]any, len(keys))
	for i, key := range keys {
		status := "DELETING"
		if ds.docs[key] == nil {
			status = "NOT_FOUND"
		}
		delete(ds.docs, key)
		details[i] = baDocumentDetail(in, identifiers[i], status)
	}
	return map[string]any{"documentDetails": details}, nil
}

func (st *bedrockAgentStore) listDocuments(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	_, ds, err := st.dataSource(in)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(ds.docs))
	for k := range ds.docs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	docs := make([]map[string]any, len(keys))
	for i, k := range keys {
		docs[i] = baProject(ds.docs[k], "knowledgeBaseId", "dataSourceId", "identifier", "status", "updatedAt")
	}
	page, next, err := baPage(docs, in, q)
	if err != nil {
		return nil, err
	}
	return baList("documentDetails", page, next), nil
}

// --- VPC configurations ---

func (st *bedrockAgentStore) createVpcConfiguration(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, err
	}
	if err := baRequire(in, "vpcId", "subnetIds", "resourceTarget", "port", "protocol", "resolutionMode"); err != nil {
		return nil, err
	}
	if port, ok := baInt(in["port"]); !ok || port < 1 || port > 65535 {
		return nil, baValidation("port must be between 1 and 65535")
	}
	if err := baCheckEnum(in, "protocol", "HTTP", "HTTPS"); err != nil {
		return nil, err
	}
	if err := baCheckEnum(in, "resolutionMode", "PUBLIC", "IN_VPC"); err != nil {
		return nil, err
	}
	id := baNewID("abcdefghijklmnopqrstuvwxyz0123456789", 32)
	now := baNow()
	rec := baRecord(in, "knowledgeBaseId")
	rec["vpcConfigurationId"] = id
	rec["status"] = "CREATED"
	rec["createdAt"], rec["updatedAt"] = now, now
	kb.vpcs[id] = rec
	return map[string]any{"vpcConfigurationId": id, "status": "CREATING"}, nil
}

func (st *bedrockAgentStore) vpcConfiguration(in map[string]any) (*baKB, map[string]any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, nil, err
	}
	id := baStr(in, "vpcConfigurationId")
	if !baVpcConfigIDPattern.MatchString(id) {
		return nil, nil, baValidation("vpcConfigurationId must be 32 lowercase alphanumeric characters")
	}
	rec := kb.vpcs[id]
	if rec == nil {
		return nil, nil, baNotFound("VPC configuration", id)
	}
	return kb, rec, nil
}

func (st *bedrockAgentStore) getVpcConfiguration(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	_, rec, err := st.vpcConfiguration(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"vpcConfiguration": rec}, nil
}

func (st *bedrockAgentStore) deleteVpcConfiguration(in map[string]any, _ url.Values) (any, *bedrockAgentError) {
	kb, _, err := st.vpcConfiguration(in)
	if err != nil {
		return nil, err
	}
	id := baStr(in, "vpcConfigurationId")
	delete(kb.vpcs, id)
	return map[string]any{"vpcConfigurationId": id, "status": "DELETING"}, nil
}

func (st *bedrockAgentStore) listVpcConfigurations(in map[string]any, q url.Values) (any, *bedrockAgentError) {
	kb, err := st.knowledgeBase(in)
	if err != nil {
		return nil, err
	}
	records := kb.vpcs
	if status := q.Get("status"); status != "" {
		records = map[string]map[string]any{}
		for id, r := range kb.vpcs {
			if r["status"] == status {
				records[id] = r
			}
		}
	}
	return baListProjected(records, in, q, "items", "vpcConfigurationId", "status", "vpcId", "resourceTarget", "port", "protocol", "resolutionMode", "name", "createdAt")
}
