package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/s3tables"
)

// s3TablesMigrationPaths are the top-level s3tables paths that share no prefix with the
// bucket/namespace/table routes, so request routing must recognise them explicitly.
var s3TablesMigrationPaths = map[string]struct{}{
	"/start-table-migration":          {},
	"/sync-table-migration":           {},
	"/get-table-migration":            {},
	"/update-table-migration":         {},
	"/complete-table-migration":       {},
	"/list-table-migrations":          {},
	"/table-replication/disassociate": {},
}

// handleS3TablesMigrationRoutes serves table migrations, bucket logging and replication
// disassociation. It reports whether it handled the request.
func (s *Server) handleS3TablesMigrationRoutes(w http.ResponseWriter, r *http.Request, path string) bool {
	type route struct{ method, path string }
	switch (route{r.Method, path}) {
	case route{http.MethodPost, "/start-table-migration"}:
		s.handleS3TablesStartTableMigration(w, r)
	case route{http.MethodPost, "/sync-table-migration"}:
		s.handleS3TablesSyncTableMigration(w, r)
	case route{http.MethodGet, "/get-table-migration"}:
		s.handleS3TablesGetTableMigration(w, r)
	case route{http.MethodPost, "/update-table-migration"}:
		s.handleS3TablesUpdateTableMigration(w, r)
	case route{http.MethodPost, "/complete-table-migration"}:
		s.handleS3TablesCompleteTableMigration(w, r)
	case route{http.MethodGet, "/list-table-migrations"}:
		s.handleS3TablesListTableMigrations(w, r)
	case route{http.MethodPut, "/table-replication/disassociate"}:
		arn := strings.TrimSpace(r.URL.Query().Get("tableArn"))
		if arn == "" {
			respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "tableArn is required")
		} else if !s.s3tables.DisassociateTableReplication(arn) {
			respondS3TablesErrorJSON(w, http.StatusNotFound, "NotFoundException", "table not found")
		} else {
			respondS3TablesJSON(w, http.StatusOK, struct{}{})
		}
	default:
		if !strings.HasPrefix(path, "/buckets/") || !strings.HasSuffix(path, "/logging") {
			return false
		}
		arn, ok := decodeS3TablesARNSuffix(path, "/buckets/", "/logging")
		if !ok {
			respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "TableBucketARN is required")
			return true
		}
		switch r.Method {
		case http.MethodPut:
			s.handleS3TablesPutTableBucketLogging(w, r, arn)
		case http.MethodGet:
			s.handleS3TablesGetTableBucketLogging(w, arn)
		default:
			respondS3TablesErrorJSON(w, http.StatusMethodNotAllowed, "MethodNotAllowedException", "unsupported method")
		}
	}
	return true
}

type s3tLoggingRequest struct {
	Status *struct {
		LoggingEnabled *struct {
			TargetBucket          string          `json:"targetBucket"`
			TargetPrefix          string          `json:"targetPrefix"`
			TargetObjectKeyFormat json.RawMessage `json:"targetObjectKeyFormat"`
		} `json:"loggingEnabled,omitempty"`
	} `json:"tableBucketLoggingStatus"`
}

func (s *Server) handleS3TablesPutTableBucketLogging(w http.ResponseWriter, r *http.Request, arn string) {
	var req s3tLoggingRequest
	if !s3tDecode(w, r, &req) {
		return
	}
	if req.Status == nil {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "tableBucketLoggingStatus is required")
		return
	}
	if le := req.Status.LoggingEnabled; le != nil && (strings.TrimSpace(le.TargetBucket) == "" || len(le.TargetObjectKeyFormat) == 0) {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "targetBucket, targetPrefix and targetObjectKeyFormat are required")
		return
	}
	status, err := json.Marshal(req.Status)
	if err != nil {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "tableBucketLoggingStatus is invalid")
		return
	}
	if !s.s3tables.PutTableBucketLogging(arn, status) {
		respondS3TablesErrorJSON(w, http.StatusNotFound, "NotFoundException", "table bucket not found")
		return
	}
	respondS3TablesJSON(w, http.StatusOK, struct{}{})
}

func (s *Server) handleS3TablesGetTableBucketLogging(w http.ResponseWriter, arn string) {
	status, ok := s.s3tables.GetTableBucketLogging(arn)
	if !ok {
		respondS3TablesErrorJSON(w, http.StatusNotFound, "NotFoundException", "table bucket not found")
		return
	}
	if len(status) == 0 {
		status = json.RawMessage(`{}`)
	}
	respondS3TablesJSON(w, http.StatusOK, struct {
		Status json.RawMessage `json:"tableBucketLoggingStatus"`
	}{status})
}

type s3tMigrationOutputRequest struct {
	Bucket   string `json:"bucket"`
	Prefixes *struct {
		TableCompatibilityReport  string `json:"tableCompatibilityReport,omitempty"`
		ObjectCompatibilityReport string `json:"objectCompatibilityReport,omitempty"`
		ObjectMigrationReport     string `json:"objectMigrationReport,omitempty"`
	} `json:"prefixes,omitempty"`
}

type s3tStartMigrationRequest struct {
	RoleARN          string `json:"roleARN"`
	DestinationTable struct {
		TableBucketARN            string          `json:"tableBucketARN"`
		Namespace                 string          `json:"namespace"`
		Name                      string          `json:"name"`
		EncryptionConfiguration   json.RawMessage `json:"encryptionConfiguration,omitempty"`
		StorageClassConfiguration json.RawMessage `json:"storageClassConfiguration,omitempty"`
		Tags                      json.RawMessage `json:"tags,omitempty"`
		Properties                json.RawMessage `json:"properties,omitempty"`
		Schema                    json.RawMessage `json:"schema,omitempty"`
		IcebergSortOrder          json.RawMessage `json:"icebergSortOrder,omitempty"`
	} `json:"destinationTable"`
	SourceType string `json:"sourceType"`
	Source     struct {
		GlueTable *struct {
			CatalogID    string `json:"catalogId"`
			DatabaseName string `json:"databaseName"`
			TableName    string `json:"tableName"`
		} `json:"glueTable,omitempty"`
	} `json:"source"`
	DataMigration *struct {
		MaintenanceSettings json.RawMessage `json:"maintenanceSettings,omitempty"`
		MigrationPolicies   json.RawMessage `json:"migrationPolicies,omitempty"`
	} `json:"dataMigration,omitempty"`
	ReviewMigrationPlan bool                       `json:"reviewMigrationPlan"`
	MigrationOutput     *s3tMigrationOutputRequest `json:"migrationOutput"`
}

func (s *Server) handleS3TablesStartTableMigration(w http.ResponseWriter, r *http.Request) {
	var req s3tStartMigrationRequest
	if !s3tDecode(w, r, &req) {
		return
	}
	dst, gt := req.DestinationTable, req.Source.GlueTable
	if strings.TrimSpace(req.RoleARN) == "" || dst.TableBucketARN == "" || dst.Namespace == "" || dst.Name == "" ||
		req.SourceType != "glueTable" || gt == nil || gt.CatalogID == "" || gt.DatabaseName == "" || gt.TableName == "" {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "roleARN, destinationTable, sourceType=glueTable and source.glueTable are required")
		return
	}
	output, ok := s3tMigrationOutput(w, req.MigrationOutput)
	if !ok {
		return
	}
	source, _ := json.Marshal(req.Source)
	data := json.RawMessage(`{"maintenanceSettings":{},"migrationPolicies":{}}`)
	if dm := req.DataMigration; dm != nil {
		data, _ = json.Marshal(map[string]json.RawMessage{
			"maintenanceSettings": s3tOrEmptyObject(dm.MaintenanceSettings),
			"migrationPolicies":   s3tOrEmptyObject(dm.MigrationPolicies),
		})
	}
	m, code := s.s3tables.StartTableMigration(s3tables.TableMigration{
		TableBucketARN: dst.TableBucketARN, Namespace: dst.Namespace, Name: dst.Name,
		RoleARN: req.RoleARN, ReviewMigrationPlan: req.ReviewMigrationPlan,
		SourceType: req.SourceType, Source: source, DataMigration: data, MigrationOutput: output,
	})
	if code != "" {
		respondS3TablesMigrationError(w, code)
		return
	}
	respondS3TablesJSON(w, http.StatusOK, map[string]string{"migrationId": m.MigrationID, "state": m.State})
}

func (s *Server) handleS3TablesSyncTableMigration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoleARN             string                     `json:"roleARN"`
		TableARN            string                     `json:"tableARN"`
		ReviewMigrationPlan bool                       `json:"reviewMigrationPlan"`
		MigrationOutput     *s3tMigrationOutputRequest `json:"migrationOutput"`
	}
	if !s3tDecode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.RoleARN) == "" || strings.TrimSpace(req.TableARN) == "" {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "roleARN and tableARN are required")
		return
	}
	output, ok := s3tMigrationOutput(w, req.MigrationOutput)
	if !ok {
		return
	}
	m, code := s.s3tables.SyncTableMigration(req.TableARN, req.RoleARN, req.ReviewMigrationPlan, output)
	if code != "" {
		respondS3TablesMigrationError(w, code)
		return
	}
	respondS3TablesJSON(w, http.StatusOK, map[string]string{"migrationId": m.MigrationID, "state": m.State})
}

func (s *Server) handleS3TablesGetTableMigration(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id, tableARN := strings.TrimSpace(q.Get("migrationId")), strings.TrimSpace(q.Get("tableARN"))
	if id == "" && tableARN == "" {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "migrationId or tableARN is required")
		return
	}
	m, ok := s.s3tables.GetTableMigration(id, tableARN)
	if !ok {
		respondS3TablesMigrationError(w, s3tables.MigrationNotFound)
		return
	}
	respondS3TablesJSON(w, http.StatusOK, s3tMigrationDetail(m))
}

func (s *Server) handleS3TablesUpdateTableMigration(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id, tableARN := strings.TrimSpace(q.Get("migrationId")), strings.TrimSpace(q.Get("tableARN"))
	var req struct {
		RequestedState    string `json:"requestedState"`
		UpdateDescription string `json:"updateDescription"`
	}
	if !s3tDecode(w, r, &req) {
		return
	}
	if (id == "" && tableARN == "") || (req.RequestedState != "queued" && req.RequestedState != "cancelled") {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "migrationId or tableARN, and requestedState queued|cancelled are required")
		return
	}
	m, code := s.s3tables.UpdateTableMigration(id, tableARN, req.RequestedState)
	if code != "" {
		respondS3TablesMigrationError(w, code)
		return
	}
	resp := map[string]string{"migrationId": m.MigrationID, "state": m.State}
	if m.TableARN != "" {
		resp["tableARN"] = m.TableARN
	}
	respondS3TablesJSON(w, http.StatusOK, resp)
}

func (s *Server) handleS3TablesCompleteTableMigration(w http.ResponseWriter, r *http.Request) {
	tableARN := strings.TrimSpace(r.URL.Query().Get("tableARN"))
	if tableARN == "" {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "tableARN is required")
		return
	}
	if code := s.s3tables.CompleteTableMigration(tableARN); code != "" {
		respondS3TablesMigrationError(w, code)
		return
	}
	respondS3TablesJSON(w, http.StatusOK, struct{}{})
}

const s3tMaxMigrationPage = 1000

func (s *Server) handleS3TablesListTableMigrations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, offset := s3tMaxMigrationPage, 0
	if v := q.Get("maxResults"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 1 || n > s3tMaxMigrationPage {
			respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "maxResults must be between 1 and 1000")
			return
		}
		limit = int(n)
	}
	if v := q.Get("nextToken"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 0 {
			respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "nextToken is invalid")
			return
		}
		offset = int(n)
	}
	all := s.s3tables.ListTableMigrations(s3tables.MigrationFilter{
		TableBucketARN: q.Get("tableBucketARN"), Namespace: q.Get("namespace"), Name: q.Get("name"),
		TableARN: q.Get("tableARN"), State: q.Get("state"),
	})
	if offset > len(all) {
		offset = len(all)
	}
	end := min(offset+limit, len(all))
	items := make([]map[string]any, 0, end-offset)
	for _, m := range all[offset:end] {
		items = append(items, s3tMigrationSummary(m))
	}
	resp := map[string]any{"migrations": items}
	if end < len(all) {
		resp["nextToken"] = strconv.Itoa(end)
	}
	respondS3TablesJSON(w, http.StatusOK, resp)
}

func s3tMigrationSummary(m s3tables.TableMigration) map[string]any {
	out := map[string]any{
		"migrationId":    m.MigrationID,
		"tableBucketARN": m.TableBucketARN,
		"namespace":      m.Namespace,
		"name":           m.Name,
		"type":           m.Type,
		"state":          m.State,
		"hasValidPlan":   true,
		"createdAt":      m.CreatedAt.Format(time.RFC3339),
		"stateUpdatedAt": m.StateUpdatedAt.Format(time.RFC3339),
	}
	if m.TableARN != "" {
		out["tableARN"] = m.TableARN
	}
	return out
}

func s3tMigrationDetail(m s3tables.TableMigration) map[string]any {
	out := s3tMigrationSummary(m)
	out["roleARN"] = m.RoleARN
	out["reviewMigrationPlan"] = m.ReviewMigrationPlan
	out["sourceType"] = m.SourceType
	out["source"] = m.Source
	out["dataMigration"] = m.DataMigration
	out["migrationOutput"] = m.MigrationOutput
	return out
}

// s3tMigrationOutput validates the request's report destination and expands it into the
// response shape (one report location per report kind).
func s3tMigrationOutput(w http.ResponseWriter, in *s3tMigrationOutputRequest) (json.RawMessage, bool) {
	if in == nil || strings.TrimSpace(in.Bucket) == "" {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "migrationOutput.bucket is required")
		return nil, false
	}
	prefix := func(get func() string, def string) map[string]string {
		if in.Prefixes != nil && get() != "" {
			return map[string]string{"prefix": get()}
		}
		return map[string]string{"prefix": def}
	}
	out, err := json.Marshal(map[string]any{
		"bucket":                    in.Bucket,
		"tableCompatibilityReport":  prefix(func() string { return in.Prefixes.TableCompatibilityReport }, "table-compatibility/"),
		"objectCompatibilityReport": prefix(func() string { return in.Prefixes.ObjectCompatibilityReport }, "object-compatibility/"),
		"objectMigrationReport":     prefix(func() string { return in.Prefixes.ObjectMigrationReport }, "object-migration/"),
	})
	if err != nil {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "migrationOutput is invalid")
		return nil, false
	}
	return out, true
}

func s3tOrEmptyObject(v json.RawMessage) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(`{}`)
	}
	return v
}

func s3tDecode(w http.ResponseWriter, r *http.Request, target any) bool {
	body, err := readBodyBytes(r)
	if err != nil {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "invalid request body")
		return false
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	if err := decodeS3TablesJSON(body, target); err != nil {
		respondS3TablesErrorJSON(w, http.StatusBadRequest, "BadRequestException", "invalid JSON")
		return false
	}
	return true
}

func respondS3TablesMigrationError(w http.ResponseWriter, code string) {
	switch code {
	case s3tables.MigrationNotFound:
		respondS3TablesErrorJSON(w, http.StatusNotFound, code, "migration, table or table bucket not found")
	case s3tables.MigrationConflict:
		respondS3TablesErrorJSON(w, http.StatusConflict, code, "migration state does not allow this request")
	default:
		respondS3TablesErrorJSON(w, http.StatusTooManyRequests, s3tables.MigrationLimit, "too many table migrations")
	}
}
