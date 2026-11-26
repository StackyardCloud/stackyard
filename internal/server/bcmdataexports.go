package server

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Billing and Cost Management Data Exports: awsJson1_1, X-Amz-Target
// AWSBillingAndCostManagementDataExports.*, signed bcm-data-exports.
//
// Every create and update runs one execution synchronously: it writes the
// manifest (and, for CSV, a header-only data file) to the export's bucket in
// Stackyard's S3. A missing bucket leaves the export UNHEALTHY with
// INSUFFICIENT_PERMISSION and the execution DELIVERY_FAILURE.

const (
	bcmdeTargetPrefix  = "AWSBillingAndCostManagementDataExports."
	bcmdeAccountID     = "123456789012"
	bcmdeRegion        = "us-east-1"
	bcmdeMaxExports    = 300
	bcmdeMaxExecutions = 100 // per export; oldest drop first
	bcmdeMaxTags       = 200
)

const bcmdeServicePrincipal = "bcm-data-exports.amazonaws.com"

var bcmdeAPI = &budJSONAPI{
	prefix: bcmdeTargetPrefix, signing: "bcm-data-exports", contentType: "application/x-amz-json-1.1",
	invalidCode: "ValidationException", model: awsmodels.BCMDataExportsModel, s3Caller: true,
}

func init() {
	bcmdeAPI.handlers = map[string]budHandler{
		"CreateExport":        (*Server).bcmdeCreateExport,
		"GetExport":           (*Server).bcmdeGetExport,
		"ListExports":         (*Server).bcmdeListExports,
		"UpdateExport":        (*Server).bcmdeUpdateExport,
		"DeleteExport":        (*Server).bcmdeDeleteExport,
		"GetExecution":        (*Server).bcmdeGetExecution,
		"ListExecutions":      (*Server).bcmdeListExecutions,
		"GetTable":            (*Server).bcmdeGetTable,
		"ListTables":          (*Server).bcmdeListTables,
		"TagResource":         (*Server).bcmdeTagResource,
		"UntagResource":       (*Server).bcmdeUntagResource,
		"ListTagsForResource": (*Server).bcmdeListTags,
	}
}

func (s *Server) handleBCMDataExportsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, bcmdeAPI, bcmdeOperationByName)
}

func bcmdeInvalid(format string, args ...any) *budErr {
	return budErrorf("ValidationException", format, args...)
}

func bcmdeNotFound(arn string) *budErr {
	return budErrorf("ResourceNotFoundException", "Export %s doesn't exist.", arn)
}

type bcmdeExport struct {
	arn                         string
	export                      map[string]any
	created, updated, refreshed time.Time
	healthy                     bool
	executions                  []map[string]any // oldest first
	tags                        map[string]string
}

type bcmdeStore struct {
	mu      sync.Mutex
	exports map[string]*bcmdeExport
}

func newBCMDataExportsStore() *bcmdeStore { return &bcmdeStore{exports: map[string]*bcmdeExport{}} }

func (st *bcmdeStore) byName(name string) *bcmdeExport {
	for _, e := range st.exports {
		if budStr(e.export, "Name") == name {
			return e
		}
	}
	return nil
}

func (st *bcmdeStore) get(arn string) (*bcmdeExport, *budErr) {
	if e := st.exports[arn]; e != nil {
		return e, nil
	}
	return nil, bcmdeNotFound(arn)
}

// bcmdePlan is a checked export plus what its executions deliver.
type bcmdePlan struct {
	export  map[string]any
	columns []bcmdeColumn
}

func bcmdeCheck(raw map[string]any) (*bcmdePlan, *budErr) {
	export := budCloneMap(raw)
	query := budMap(export, "DataQuery")
	_, cols, err := bcmdeParseQuery(budStr(query, "QueryStatement"), budMap(query, "TableConfigurations"))
	if err != nil {
		return nil, err
	}
	dest := budMap(budMap(export, "DestinationConfigurations"), "S3Destination")
	if budStr(dest, "S3Bucket") == "" || budStr(dest, "S3Region") == "" {
		return nil, bcmdeInvalid("S3Destination needs S3Bucket and S3Region.")
	}
	out := budMap(dest, "S3OutputConfigurations")
	format, compression := budStr(out, "Format"), budStr(out, "Compression")
	if (format == "PARQUET") != (compression == "PARQUET") {
		return nil, bcmdeInvalid("PARQUET format needs PARQUET compression, and TEXT_OR_CSV needs GZIP or ZIP.")
	}
	return &bcmdePlan{export: export, columns: cols}, nil
}

// bcmdeDeliver writes one execution's files to S3 and reports whether the
// bucket accepted them. It takes the S3 store's lock, never ours.
func (s *Server) bcmdeDeliver(c s3Caller, plan *bcmdePlan, executionID string, now time.Time) bool {
	dest := budMap(budMap(plan.export, "DestinationConfigurations"), "S3Destination")
	out := budMap(dest, "S3OutputConfigurations")
	bucket, name := budStr(dest, "S3Bucket"), budStr(plan.export, "Name")
	if _, err := s.s3.GetBucket(bucket); err != nil {
		return false
	}
	base := strings.Trim(budStr(dest, "S3Prefix"), "/")
	if base != "" {
		base += "/"
	}
	base += name
	period := "BILLING_PERIOD=" + now.Format("2006-01")
	dataDir := base + "/data/" + period
	if budStr(out, "Overwrite") == "CREATE_NEW_REPORT" {
		dataDir += "/" + executionID
	}
	var dataFiles []string
	if budStr(out, "Format") == "TEXT_OR_CSV" {
		names := make([]string, 0, len(plan.columns))
		for _, c := range plan.columns {
			names = append(names, c.name)
		}
		key, body, err := bcmdeCSV(dataDir+"/"+name+"-00001.csv", strings.Join(names, ",")+"\n", budStr(out, "Compression"))
		if err != nil {
			return false
		}
		if !s.authorizeServiceS3As(c, bcmdeServicePrincipal, bucket, key, "s3:PutObject") {
			return false
		}
		if _, err := s.s3.PutObject(bucket, key, "application/octet-stream", body, nil, "", ""); err != nil {
			return false
		}
		dataFiles = append(dataFiles, key)
	}
	// ponytail: PARQUET exports deliver only the manifest; there are no rows to encode.
	cols := make([]any, 0, len(plan.columns))
	for _, c := range plan.columns {
		cols = append(cols, map[string]any{"name": c.name, "type": c.typ})
	}
	manifest, _ := json.Marshal(map[string]any{
		"exportArn": plan.export["ExportArn"], "exportName": name, "executionId": executionID,
		"billingPeriod": now.Format("2006-01"), "columns": cols, "dataFiles": dataFiles, "deliveredAt": now.Format(time.RFC3339),
	})
	key := base + "/metadata/" + period + "/" + name + "-Manifest.json"
	if !s.authorizeServiceS3As(c, bcmdeServicePrincipal, bucket, key, "s3:PutObject") {
		return false
	}
	_, err := s.s3.PutObject(bucket, key, "application/json", manifest, nil, "", "")
	return err == nil
}

func bcmdeCSV(key, content, compression string) (string, []byte, error) {
	var buf bytes.Buffer
	if compression == "ZIP" {
		zw := zip.NewWriter(&buf)
		f, err := zw.Create(key[strings.LastIndex(key, "/")+1:])
		if err == nil {
			_, err = f.Write([]byte(content))
		}
		if cerr := zw.Close(); err == nil {
			err = cerr
		}
		return key + ".zip", buf.Bytes(), err
	}
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(content))
	if cerr := gz.Close(); err == nil {
		err = cerr
	}
	return key + ".gz", buf.Bytes(), err
}

// startExecution starts an execution on e (caller holds st.mu) and returns its id.
func (e *bcmdeExport) startExecution(now time.Time) string {
	id := budNewID()
	e.executions = append(e.executions, map[string]any{
		"ExecutionId":     id,
		"ExecutionStatus": map[string]any{"StatusCode": "DELIVERY_IN_PROCESS", "CreatedAt": budEpoch(now), "LastUpdatedAt": budEpoch(now)},
	})
	if len(e.executions) > bcmdeMaxExecutions {
		e.executions = e.executions[len(e.executions)-bcmdeMaxExecutions:]
	}
	return id
}

// finishExecution records a delivery result; caller holds st.mu.
func (e *bcmdeExport) finishExecution(id string, ok bool, now time.Time) {
	for i, x := range e.executions {
		if x["ExecutionId"] != id {
			continue
		}
		status := map[string]any{"StatusCode": "DELIVERY_SUCCESS", "CreatedAt": budMap(x, "ExecutionStatus")["CreatedAt"], "CompletedAt": budEpoch(now), "LastUpdatedAt": budEpoch(now)}
		if !ok {
			status["StatusCode"], status["StatusReason"] = "DELIVERY_FAILURE", "INSUFFICIENT_PERMISSION"
		}
		e.executions[i] = map[string]any{"ExecutionId": id, "ExecutionStatus": status}
	}
	e.healthy, e.refreshed = ok, now
}

// bcmdeRunDelivery delivers outside the store lock, then records the result
// if the export still exists.
// A denied write fails the delivery with INSUFFICIENT_PERMISSION.
func (s *Server) bcmdeRunDelivery(c s3Caller, arn, execID string, plan *bcmdePlan) {
	now := time.Now().UTC()
	ok := s.bcmdeDeliver(c, plan, execID, now)
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	if e := st.exports[arn]; e != nil {
		e.finishExecution(execID, ok, now)
	}
}

func (s *Server) bcmdeCreateExport(in map[string]any) (map[string]any, *budErr) {
	plan, err := bcmdeCheck(budMap(in, "Export"))
	if err != nil {
		return nil, err
	}
	tags, err := budTagsIn(budList(in, "ResourceTags"), "ValidationException")
	if err != nil {
		return nil, err
	}
	if len(tags) > bcmdeMaxTags {
		return nil, bcmdeInvalid("An export can have at most %d tags.", bcmdeMaxTags)
	}
	name, now := budStr(plan.export, "Name"), time.Now().UTC()
	arn := fmt.Sprintf("arn:aws:bcm-data-exports:%s:%s:export/%s-%s", bcmdeRegion, bcmdeAccountID, name, budNewID())
	plan.export["ExportArn"] = arn
	st := s.bcmDataExports
	st.mu.Lock()
	if st.byName(name) != nil {
		st.mu.Unlock()
		return nil, bcmdeInvalid("An export named %s already exists.", name)
	}
	if len(st.exports) >= bcmdeMaxExports {
		st.mu.Unlock()
		return nil, budErrorf("ServiceQuotaExceededException", "The account already has %d exports.", bcmdeMaxExports)
	}
	e := &bcmdeExport{arn: arn, export: plan.export, created: now, updated: now, tags: tags}
	execID := e.startExecution(now)
	st.exports[arn] = e
	st.mu.Unlock()
	s.bcmdeRunDelivery(s3CallerIn(in), arn, execID, plan)
	return map[string]any{"ExportArn": arn}, nil
}

func (s *Server) bcmdeUpdateExport(in map[string]any) (map[string]any, *budErr) {
	arn := budStr(in, "ExportArn")
	raw := budMap(in, "Export")
	if given := budStr(raw, "ExportArn"); given != "" && given != arn {
		return nil, bcmdeInvalid("Export.ExportArn doesn't match ExportArn.")
	}
	plan, err := bcmdeCheck(raw)
	if err != nil {
		return nil, err
	}
	plan.export["ExportArn"] = arn
	now := time.Now().UTC()
	st := s.bcmDataExports
	st.mu.Lock()
	e, err := st.get(arn)
	if err == nil {
		if other := st.byName(budStr(plan.export, "Name")); other != nil && other != e {
			err = bcmdeInvalid("An export named %s already exists.", budStr(plan.export, "Name"))
		}
	}
	if err != nil {
		st.mu.Unlock()
		return nil, err
	}
	e.export, e.updated = plan.export, now
	execID := e.startExecution(now)
	st.mu.Unlock()
	s.bcmdeRunDelivery(s3CallerIn(in), arn, execID, plan)
	return map[string]any{"ExportArn": arn}, nil
}

func (e *bcmdeExport) status() map[string]any {
	out := map[string]any{
		"StatusCode": "HEALTHY", "CreatedAt": budEpoch(e.created), "LastUpdatedAt": budEpoch(e.updated),
	}
	if !e.refreshed.IsZero() {
		out["LastRefreshedAt"] = budEpoch(e.refreshed)
	}
	if !e.healthy {
		out["StatusCode"], out["StatusReason"] = "UNHEALTHY", "INSUFFICIENT_PERMISSION"
	}
	return out
}

func (s *Server) bcmdeGetExport(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.get(budStr(in, "ExportArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Export": budCloneMap(e.export), "ExportStatus": e.status()}, nil
}

func (s *Server) bcmdeListExports(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	all := make([]*bcmdeExport, 0, len(st.exports))
	for _, e := range st.exports {
		all = append(all, e)
	}
	sort.Slice(all, func(i, j int) bool { return budStr(all[i].export, "Name") < budStr(all[j].export, "Name") })
	start, end, next, err := budPage(budStr(in, "NextToken"), len(all), budInt(in, "MaxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, e := range all[start:end] {
		items = append(items, map[string]any{"ExportArn": e.arn, "ExportName": e.export["Name"], "ExportStatus": e.status()})
	}
	return map[string]any{"Exports": items, "NextToken": next}, nil
}

func (s *Server) bcmdeDeleteExport(in map[string]any) (map[string]any, *budErr) {
	arn := budStr(in, "ExportArn")
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.get(arn); err != nil {
		return nil, err
	}
	delete(st.exports, arn)
	return map[string]any{"ExportArn": arn}, nil
}

func (s *Server) bcmdeGetExecution(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.get(budStr(in, "ExportArn"))
	if err != nil {
		return nil, err
	}
	id := budStr(in, "ExecutionId")
	for _, x := range e.executions {
		if x["ExecutionId"] == id {
			return map[string]any{"ExecutionId": id, "Export": budCloneMap(e.export), "ExecutionStatus": budClone(x["ExecutionStatus"])}, nil
		}
	}
	return nil, budErrorf("ResourceNotFoundException", "Execution %s doesn't exist.", id)
}

func (s *Server) bcmdeListExecutions(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.get(budStr(in, "ExportArn"))
	if err != nil {
		return nil, err
	}
	start, end, next, err := budPage(budStr(in, "NextToken"), len(e.executions), budInt(in, "MaxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for i := start; i < end; i++ { // newest first
		items = append(items, budClone(e.executions[len(e.executions)-1-i]))
	}
	return map[string]any{"Executions": items, "NextToken": next}, nil
}

// ---- tables ----

func (s *Server) bcmdeGetTable(in map[string]any) (map[string]any, *budErr) {
	name := budStr(in, "TableName")
	t := bcmdeTableByName(name)
	if t == nil {
		return nil, bcmdeInvalid("Table %s doesn't exist.", name)
	}
	props, err := t.resolveProps(budMap(in, "TableProperties"))
	if err != nil {
		return nil, err
	}
	schema := make([]any, 0, len(t.columns))
	for _, c := range t.schema(props) {
		schema = append(schema, map[string]any{"Name": c.name, "Type": c.typ, "Description": c.desc})
	}
	propsOut := map[string]any{}
	for k, v := range props {
		propsOut[k] = v
	}
	return map[string]any{"TableName": t.name, "Description": t.desc, "TableProperties": propsOut, "Schema": schema}, nil
}

func (s *Server) bcmdeListTables(in map[string]any) (map[string]any, *budErr) {
	start, end, next, err := budPage(budStr(in, "NextToken"), len(bcmdeTables), budInt(in, "MaxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, t := range bcmdeTables[start:end] {
		props := make([]any, 0, len(t.props))
		for _, p := range t.props {
			valid := make([]any, 0, len(p.valid))
			for _, v := range p.valid {
				valid = append(valid, v)
			}
			props = append(props, map[string]any{"Name": p.name, "ValidValues": valid, "DefaultValue": p.def, "Description": p.desc})
		}
		items = append(items, map[string]any{"TableName": t.name, "Description": t.desc, "TableProperties": props})
	}
	return map[string]any{"Tables": items, "NextToken": next}, nil
}

// ---- tags ----

func (s *Server) bcmdeTagResource(in map[string]any) (map[string]any, *budErr) {
	add, err := budTagsIn(budList(in, "ResourceTags"), "ValidationException")
	if err != nil {
		return nil, err
	}
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.get(budStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(e.tags, add, bcmdeMaxTags, "ValidationException")
	if err != nil {
		return nil, err
	}
	e.tags = merged
	return map[string]any{}, nil
}

func (s *Server) bcmdeUntagResource(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.get(budStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	e.tags = budDropTags(e.tags, budStrings(in, "ResourceTagKeys"))
	return map[string]any{}, nil
}

func (s *Server) bcmdeListTags(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDataExports
	st.mu.Lock()
	defer st.mu.Unlock()
	e, err := st.get(budStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	tags := budTagsOut(e.tags)
	start, end, next, err := budPage(budStr(in, "NextToken"), len(tags), budInt(in, "MaxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	return map[string]any{"ResourceTags": tags[start:end], "NextToken": next}, nil
}
