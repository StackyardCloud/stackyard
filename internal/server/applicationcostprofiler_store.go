package server

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	acpMaxDefinitions = 50      // ServiceQuotaExceededException beyond this
	acpMaxUsageBytes  = 1 << 20 // largest usage file an import reads
	acpMaxUsageRows   = 10000
)

// acpDefinition is one stored report definition; records are replaced, never
// mutated, so a snapshot can be read after the lock is released.
type acpDefinition struct {
	id, description, frequency, format string
	bucket, prefix                     string
	created, updated                   time.Time
}

type acpStore struct {
	mu   sync.Mutex
	defs map[string]*acpDefinition
}

func newApplicationCostProfilerStore() *acpStore {
	return &acpStore{defs: map[string]*acpDefinition{}}
}

func (s *Server) acpDispatch(op string, in map[string]any) (map[string]any, *budErr) {
	switch op {
	case "PutReportDefinition":
		return s.acpPut(in)
	case "UpdateReportDefinition":
		return s.acpUpdate(in)
	case "GetReportDefinition":
		return s.applicationCostProfiler.get(budStr(in, "reportId"))
	case "ListReportDefinitions":
		return s.applicationCostProfiler.list(budStr(in, "nextToken"), budInt(in, "maxResults", 100))
	case "DeleteReportDefinition":
		return s.applicationCostProfiler.delete(budStr(in, "reportId"))
	case "ImportApplicationUsage":
		return s.acpImport(in)
	}
	return nil, acpErrorf("UnknownOperationException", "unknown operation %s", op)
}

// acpDefinitionIn reads the definition members and checks that the
// destination bucket exists in Stackyard's S3 (outside our lock).
func (s *Server) acpDefinitionIn(id string, in map[string]any) (*acpDefinition, *budErr) {
	dest := budMap(in, "destinationS3Location")
	d := &acpDefinition{
		id: id, description: budStr(in, "reportDescription"), frequency: budStr(in, "reportFrequency"),
		format: budStr(in, "format"), bucket: budStr(dest, "bucket"), prefix: strings.Trim(budStr(dest, "prefix"), "/"),
	}
	if d.prefix == "" {
		return nil, acpInvalid("destinationS3Location.prefix must name a folder")
	}
	if _, err := s.s3.GetBucket(d.bucket); err != nil {
		return nil, acpInvalid("destination bucket %s does not exist or is not accessible", d.bucket)
	}
	return d, nil
}

func (s *Server) acpPut(in map[string]any) (map[string]any, *budErr) {
	d, err := s.acpDefinitionIn(budStr(in, "reportId"), in)
	if err != nil {
		return nil, err
	}
	st := s.applicationCostProfiler
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.defs[d.id] != nil {
		return nil, acpInvalid("report definition %s already exists", d.id)
	}
	if len(st.defs) >= acpMaxDefinitions {
		return nil, acpErrorf("ServiceQuotaExceededException", "an account can have at most %d report definitions", acpMaxDefinitions)
	}
	now := time.Now().UTC()
	d.created, d.updated = now, now
	st.defs[d.id] = d
	return map[string]any{"reportId": d.id}, nil
}

func (s *Server) acpUpdate(in map[string]any) (map[string]any, *budErr) {
	d, err := s.acpDefinitionIn(budStr(in, "reportId"), in)
	if err != nil {
		return nil, err
	}
	st := s.applicationCostProfiler
	st.mu.Lock()
	defer st.mu.Unlock()
	old := st.defs[d.id]
	if old == nil {
		return nil, acpInvalid("report definition %s does not exist", d.id)
	}
	d.created, d.updated = old.created, time.Now().UTC()
	st.defs[d.id] = d
	return map[string]any{"reportId": d.id}, nil
}

func (d *acpDefinition) out(updatedKey string) map[string]any {
	return map[string]any{
		"reportId": d.id, "reportDescription": d.description, "reportFrequency": d.frequency, "format": d.format,
		"destinationS3Location": map[string]any{"bucket": d.bucket, "prefix": d.prefix},
		"createdAt":             budEpoch(d.created), updatedKey: budEpoch(d.updated),
	}
}

func (st *acpStore) get(id string) (map[string]any, *budErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d := st.defs[id]
	if d == nil {
		return nil, acpInvalid("report definition %s does not exist", id)
	}
	return d.out("lastUpdated"), nil
}

func (st *acpStore) delete(id string) (map[string]any, *budErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.defs[id] == nil {
		return nil, acpInvalid("report definition %s does not exist", id)
	}
	delete(st.defs, id)
	return map[string]any{"reportId": id}, nil
}

// list pages definitions by id. Tokens are standard base64 (the model's
// Token pattern) of the offset.
func (st *acpStore) list(token string, size int) (map[string]any, *budErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ids := make([]string, 0, len(st.defs))
	for id := range st.defs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	start := 0
	if token != "" {
		raw, err := base64.StdEncoding.DecodeString(token)
		n, perr := strconv.ParseInt(strings.TrimPrefix(string(raw), "acp:"), 10, 32)
		if err != nil || perr != nil || !strings.HasPrefix(string(raw), "acp:") || n < 0 || int(n) > len(ids) {
			return nil, acpInvalid("the nextToken is not valid")
		}
		start = int(n)
	}
	end := min(len(ids), start+size)
	items := make([]any, 0, end-start)
	for _, id := range ids[start:end] {
		items = append(items, st.defs[id].out("lastUpdatedAt"))
	}
	out := map[string]any{"reportDefinitions": items}
	if end < len(ids) {
		out["nextToken"] = base64.StdEncoding.EncodeToString([]byte("acp:" + strconv.Itoa(end)))
	}
	return out, nil
}

// ---- usage import and report delivery ----

// acpUsage is one parsed usage record: a tenant's use of a resource.
type acpUsage struct {
	app, tenant, desc, resource string
	hours                       float64
}

// acpImport reads the usage file from S3, then writes each definition's
// report. Only S3's lock is taken while reading and writing; ours only to
// snapshot the definitions.
func (s *Server) acpImport(in map[string]any) (map[string]any, *budErr) {
	src := budMap(in, "sourceS3Location")
	bucket, key := budStr(src, "bucket"), budStr(src, "key")
	obj, err := s.s3.GetObject(bucket, key)
	if err != nil {
		return nil, acpInvalid("unable to read s3://%s/%s", bucket, key)
	}
	if len(obj.Body) > acpMaxUsageBytes {
		return nil, acpInvalid("usage files are limited to %d bytes", acpMaxUsageBytes)
	}
	usage, perr := acpParseUsage(obj.Body)
	if perr != nil {
		return nil, acpInvalid("s3://%s/%s: %v", bucket, key, perr)
	}
	st := s.applicationCostProfiler
	st.mu.Lock()
	defs := make([]*acpDefinition, 0, len(st.defs))
	for _, d := range st.defs {
		defs = append(defs, d)
	}
	st.mu.Unlock()

	importID := budNewID()
	now := time.Now().UTC()
	report := acpReport(usage)
	for _, d := range defs {
		// A bucket deleted since the definition was saved just misses this delivery.
		_ = s.acpDeliver(d, importID, report, now)
	}
	return map[string]any{"importId": importID}, nil
}

// acpParseUsage reads a CSV usage file whose header names at least
// ApplicationId, TenantId, ResourceId, StartTime and EndTime (TenantDesc is
// optional). Times are epoch seconds or RFC 3339.
func acpParseUsage(body []byte) ([]acpUsage, error) {
	r := csv.NewReader(bytes.NewReader(body))
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return nil, errors.New("usage file needs a CSV header row")
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	for _, need := range []string{"ApplicationId", "TenantId", "ResourceId", "StartTime", "EndTime"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("header is missing column %s", need)
		}
	}
	field := func(rec []string, name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	var out []acpUsage
	for line := 2; ; line++ {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if len(out) >= acpMaxUsageRows {
			return nil, fmt.Errorf("usage files are limited to %d rows", acpMaxUsageRows)
		}
		u := acpUsage{app: field(rec, "ApplicationId"), tenant: field(rec, "TenantId"), desc: field(rec, "TenantDesc"), resource: field(rec, "ResourceId")}
		start, ok1 := budTime(acpTimeValue(field(rec, "StartTime")))
		end, ok2 := budTime(acpTimeValue(field(rec, "EndTime")))
		if u.app == "" || u.tenant == "" || u.resource == "" || !ok1 || !ok2 || !end.After(start) {
			return nil, fmt.Errorf("line %d: needs ApplicationId, TenantId, ResourceId and StartTime before EndTime", line)
		}
		u.hours = end.Sub(start).Hours()
		out = append(out, u)
	}
	return out, nil
}

// acpTimeValue lets budTime read epoch seconds written as text.
func acpTimeValue(v string) any {
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return v
}

// acpReport aggregates usage per application, tenant and resource and gives
// each tenant its share of the resource's total hours, as CSV.
func acpReport(usage []acpUsage) []byte {
	type key struct{ app, tenant, resource string }
	hours := map[key]float64{}
	descs := map[key]string{}
	perResource := map[string]float64{}
	for _, u := range usage {
		k := key{u.app, u.tenant, u.resource}
		hours[k] += u.hours
		if u.desc != "" {
			descs[k] = u.desc
		}
		perResource[u.resource] += u.hours
	}
	keys := make([]key, 0, len(hours))
	for k := range hours {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.resource != b.resource {
			return a.resource < b.resource
		}
		if a.app != b.app {
			return a.app < b.app
		}
		return a.tenant < b.tenant
	})
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"ApplicationId", "TenantId", "TenantDesc", "ResourceId", "UsageHours", "UsagePercentage"})
	for _, k := range keys {
		share := hours[k] / perResource[k.resource] * 100
		_ = w.Write([]string{k.app, k.tenant, descs[k], k.resource,
			strconv.FormatFloat(hours[k], 'f', 4, 64), strconv.FormatFloat(share, 'f', 4, 64)})
	}
	w.Flush()
	return buf.Bytes()
}

// acpReportPrefix is where a definition's reports for now are written:
// prefix/reportId/YYYY/MM[/DD] (no day for MONTHLY).
func acpReportPrefix(d *acpDefinition, now time.Time) string {
	layout := "2006/01/02"
	if d.frequency == "MONTHLY" {
		layout = "2006/01"
	}
	return d.prefix + "/" + d.id + "/" + now.Format(layout)
}

// acpDeliver writes a manifest and, for CSV definitions, the report itself.
// ponytail: PARQUET definitions get only the manifest; encode Parquet when
// a consumer needs the rows.
func (s *Server) acpDeliver(d *acpDefinition, importID string, report []byte, now time.Time) error {
	base := acpReportPrefix(d, now)
	var files []string
	if d.format == "CSV" {
		key := base + "/" + importID + ".csv"
		if _, err := s.s3.PutObject(d.bucket, key, "text/csv", report, nil, "", ""); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		files = append(files, key)
	}
	manifest, _ := json.Marshal(map[string]any{
		"reportId": d.id, "importId": importID, "format": d.format, "reportFrequency": d.frequency,
		"generatedAt": now.Format(time.RFC3339), "reportFiles": files,
	})
	if _, err := s.s3.PutObject(d.bucket, base+"/"+importID+"-manifest.json", "application/json", manifest, nil, "", ""); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
