package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Performance analysis reports and their tags. Reports run on read: the first
// GetPerformanceAnalysisReport reports RUNNING and later reads SUCCEEDED.

const (
	piMaxReportsPerResource = 100
	piMaxReports            = 2000
	piMaxTags               = 50
	piMinReportRange        = 5 * time.Minute
	piMaxReportRange        = 6 * 24 * time.Hour
)

var piReportARNRe = regexp.MustCompile(`^arn:aws:pi:[a-z0-9-]+:\d{12}:perf-reports/(RDS|DOCDB)/(db-[A-Za-z0-9]+)/(report-[0-9a-f]{17})$`)

type piReport struct {
	id, serviceType, identifier string
	created, start, end         time.Time
	reads                       int
	tags                        map[string]string
}

func (r *piReport) arn() string {
	return fmt.Sprintf("arn:aws:pi:%s:%s:perf-reports/%s/%s/%s", piRegion, piAccountID, r.serviceType, r.identifier, r.id)
}

func (r *piReport) status() string {
	if r.reads == 0 {
		return "RUNNING"
	}
	return "SUCCEEDED"
}

func (r *piReport) tagList() []any {
	keys := make([]string, 0, len(r.tags))
	for k := range r.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []any{}
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": r.tags[k]})
	}
	return out
}

type piStore struct {
	mu      sync.Mutex
	reports map[string]*piReport // by report id
}

func newPIStore() *piStore { return &piStore{reports: map[string]*piReport{}} }

func piReportID() string {
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	return "report-" + hex.EncodeToString(b)[:17]
}

// piTags validates a Tags list and returns it as a map.
func piTags(in map[string]any) (map[string]string, *piErr) {
	out := map[string]string{}
	for _, t := range piAnyList(in["Tags"]) {
		tag, _ := t.(map[string]any)
		k := piStr(tag, "Key")
		if strings.HasPrefix(k, "aws:") || strings.HasPrefix(k, "rds:") {
			return nil, piInvalid("Tag keys can't start with aws: or rds:.")
		}
		out[k] = piStr(tag, "Value")
	}
	return out, nil
}

func (s *Server) piCreateReport(in map[string]any) (map[string]any, *piErr) {
	serviceType, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	start, _, err := piTime(in, "StartTime")
	if err != nil {
		return nil, err
	}
	end, hasEnd, err := piTime(in, "EndTime")
	if err != nil {
		return nil, err
	}
	if !hasEnd {
		end = start.Add(time.Hour)
	}
	if d := end.Sub(start); d < piMinReportRange || d > piMaxReportRange {
		return nil, piInvalid("The analysis period must be between 5 minutes and 6 days.")
	}
	tags, err := piTags(in)
	if err != nil {
		return nil, err
	}
	if len(tags) > piMaxTags {
		return nil, piInvalid("A report can have at most %d tags.", piMaxTags)
	}
	st := s.pi
	st.mu.Lock()
	defer st.mu.Unlock()
	perResource := 0
	for _, r := range st.reports {
		if r.identifier == id {
			perResource++
		}
	}
	if perResource >= piMaxReportsPerResource || len(st.reports) >= piMaxReports {
		return nil, piInvalid("The analysis report quota for %s has been reached.", id)
	}
	r := &piReport{id: piReportID(), serviceType: serviceType, identifier: id, created: time.Now().UTC(), start: start, end: end, tags: tags}
	st.reports[r.id] = r
	return map[string]any{"AnalysisReportId": r.id}, nil
}

// piReportFor returns a report that belongs to the identifier. Callers hold mu.
func (st *piStore) reportFor(id, reportID string) (*piReport, *piErr) {
	r := st.reports[reportID]
	if r == nil || r.identifier != id {
		return nil, piInvalid("Analysis report %s doesn't exist for %s.", reportID, id)
	}
	return r, nil
}

func (s *Server) piGetReport(in map[string]any) (map[string]any, *piErr) {
	_, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	st := s.pi
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.reportFor(id, piStr(in, "AnalysisReportId"))
	if err != nil {
		return nil, err
	}
	report := map[string]any{
		"AnalysisReportId": r.id, "Identifier": r.identifier, "ServiceType": r.serviceType,
		"CreateTime": piEpoch(r.created), "StartTime": piEpoch(r.start), "EndTime": piEpoch(r.end), "Status": r.status(),
	}
	if r.status() == "SUCCEEDED" {
		report["Insights"] = piInsights(r, piStr(in, "TextFormat") == "MARKDOWN")
	}
	r.reads++
	return map[string]any{"AnalysisReport": report}, nil
}

func (s *Server) piListReports(in map[string]any) (map[string]any, *piErr) {
	_, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	withTags, _ := in["ListTags"].(bool)
	st := s.pi
	st.mu.Lock()
	defer st.mu.Unlock()
	var mine []*piReport
	for _, r := range st.reports {
		if r.identifier == id {
			mine = append(mine, r)
		}
	}
	sort.Slice(mine, func(i, j int) bool {
		if !mine[i].created.Equal(mine[j].created) {
			return mine[i].created.Before(mine[j].created)
		}
		return mine[i].id < mine[j].id
	})
	start, end, next, err := piPage(in, len(mine), 25)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, r := range mine[start:end] {
		sum := map[string]any{"AnalysisReportId": r.id, "CreateTime": piEpoch(r.created), "StartTime": piEpoch(r.start), "EndTime": piEpoch(r.end), "Status": r.status()}
		if withTags {
			sum["Tags"] = r.tagList()
		}
		out = append(out, sum)
	}
	resp := map[string]any{"AnalysisReports": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (s *Server) piListRecommendations(in map[string]any) (map[string]any, *piErr) {
	_, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	st := s.pi
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.reportFor(id, piStr(in, "AnalysisReportId"))
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, rid := range piList(in, "RecommendationIds") {
		want[rid] = true
	}
	var all []any
	for _, rec := range piRecommendations(r, true) {
		if len(want) == 0 || want[rec["RecommendationId"].(string)] {
			all = append(all, rec)
		}
	}
	start, end, next, err := piPage(in, len(all), 25)
	if err != nil {
		return nil, err
	}
	resp := map[string]any{"Recommendations": append([]any{}, all[start:end]...)}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (s *Server) piDeleteReport(in map[string]any) (map[string]any, *piErr) {
	_, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	st := s.pi
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.reportFor(id, piStr(in, "AnalysisReportId"))
	if err != nil {
		return nil, err
	}
	delete(st.reports, r.id)
	return map[string]any{}, nil
}

// ---- insights ----

func piRecommendations(r *piReport, markdown bool) []map[string]any {
	code := func(s string) string {
		if markdown {
			return "`" + s + "`"
		}
		return s
	}
	return []map[string]any{
		{"RecommendationId": r.id + "-rec-1", "RecommendationDescription": "Investigate the top SQL statement driving the load.",
			"RecommendationDetails": "The statement " + code("SELECT * FROM orders WHERE customer_id = ?") + " accounts for most of the database load. Review its execution plan and add an index on " + code("customer_id") + " if one is missing."},
		{"RecommendationId": r.id + "-rec-2", "RecommendationDescription": "Reduce lock contention.",
			"RecommendationDetails": "Sessions spent time waiting on " + code("Lock:transactionid") + ". Keep transactions short and update rows in a consistent order."},
	}
}

// piInsights builds the report's insights. They're fixed per report, so
// repeated reads agree.
// ponytail: a fixed "high load" insight rather than analysis of the series;
// derive severity from piValue over the window if callers need it to vary.
func piInsights(r *piReport, markdown bool) []any {
	heading := func(s string) string {
		if markdown {
			return "**" + s + "**"
		}
		return s
	}
	recs := []any{}
	for _, rec := range piRecommendations(r, markdown) {
		recs = append(recs, rec)
	}
	window := map[string]any{"StartTime": piEpoch(r.start), "EndTime": piEpoch(r.end)}
	support := map[string]any{
		"InsightId": r.id + "-insight-2", "InsightType": "TopSQL", "Context": "CONTEXTUAL", "Severity": "MEDIUM",
		"StartTime": window["StartTime"], "EndTime": window["EndTime"],
		"Description": heading("Top SQL") + ": one statement accounts for most of the database load.",
		"InsightData": []any{map[string]any{"PerformanceInsightsMetric": map[string]any{"Metric": "db.load.avg", "DisplayName": "Database load", "Dimensions": map[string]any{"db.sql_tokenized.id": "9F2E1D0C8B7A6954"}}}},
	}
	return []any{map[string]any{
		"InsightId": r.id + "-insight-1", "InsightType": "HighDBLoad", "Context": "CAUSAL", "Severity": "HIGH",
		"StartTime": window["StartTime"], "EndTime": window["EndTime"],
		"Description":        heading("High database load") + ": the database load exceeded the number of vCPUs during the analysis period.",
		"Recommendations":    recs,
		"SupportingInsights": []any{support},
		"InsightData":        []any{map[string]any{"PerformanceInsightsMetric": map[string]any{"Metric": "db.load.avg", "DisplayName": "Database load"}}},
		"BaselineData":       []any{map[string]any{"PerformanceInsightsMetric": map[string]any{"Metric": "db.load.avg", "DisplayName": "Database load (baseline)"}}},
	}}
}

// ---- tags ----

// reportByARN resolves a report ARN, checking its ServiceType. Callers hold mu.
func (st *piStore) reportByARN(serviceType, arn string) (*piReport, *piErr) {
	m := piReportARNRe.FindStringSubmatch(arn)
	if m == nil {
		return nil, piInvalid("%s is not a Performance Insights report ARN.", arn)
	}
	r := st.reports[m[3]]
	if r == nil || r.identifier != m[2] || r.serviceType != m[1] || m[1] != serviceType {
		return nil, piInvalid("The resource %s doesn't exist.", arn)
	}
	return r, nil
}

func (st *piStore) tagResource(in map[string]any) (map[string]any, *piErr) {
	tags, err := piTags(in)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.reportByARN(piStr(in, "ServiceType"), piStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	merged := map[string]string{}
	for k, v := range r.tags {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	if len(merged) > piMaxTags {
		return nil, piInvalid("A report can have at most %d tags.", piMaxTags)
	}
	r.tags = merged
	return map[string]any{}, nil
}

func (st *piStore) untagResource(in map[string]any) (map[string]any, *piErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.reportByARN(piStr(in, "ServiceType"), piStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	for _, k := range piList(in, "TagKeys") {
		delete(r.tags, k)
	}
	return map[string]any{}, nil
}

func (st *piStore) listTags(in map[string]any) (map[string]any, *piErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.reportByARN(piStr(in, "ServiceType"), piStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Tags": r.tagList()}, nil
}
