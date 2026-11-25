package server

import (
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Cost Optimization Hub: awsJson1_0, X-Amz-Target
// CostOptimizationHubService.*, signed cost-optimization-hub. Enrollment and
// preferences are stored; recommendations come from a fixed catalog
// (costoptimizationhub_catalog.go) and are served only while enrolled.
// budServe writes responses with Content-Type application/x-amz-json-1.0.

const (
	cohTargetPrefix = "CostOptimizationHubService."
	cohAccountID    = "123456789012"
	cohMaxPeriods   = 400 // efficiency-metric periods per request
)

var cohAPI = &budJSONAPI{
	prefix: cohTargetPrefix, signing: "cost-optimization-hub", contentType: "application/x-amz-json-1.0",
	invalidCode: "ValidationException", model: awsmodels.CostOptimizationHubModel,
}

func init() {
	cohAPI.handlers = map[string]budHandler{
		"GetPreferences":              (*Server).cohGetPreferences,
		"UpdatePreferences":           (*Server).cohUpdatePreferences,
		"ListEnrollmentStatuses":      (*Server).cohListEnrollmentStatuses,
		"UpdateEnrollmentStatus":      (*Server).cohUpdateEnrollmentStatus,
		"GetRecommendation":           (*Server).cohGetRecommendation,
		"ListRecommendations":         (*Server).cohListRecommendations,
		"ListRecommendationSummaries": (*Server).cohListRecommendationSummaries,
		"ListEfficiencyMetrics":       (*Server).cohListEfficiencyMetrics,
	}
}

func (s *Server) handleCostOptimizationHubJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, cohAPI, cohOperationByName)
}

func cohInvalid(format string, args ...any) *budErr {
	return budErrorf("ValidationException", format, args...)
}

type cohStore struct {
	mu                    sync.Mutex
	created, updated      time.Time
	status                string
	includeMemberAccounts bool
	savingsEstimationMode string
	memberDiscountVisible string
	term, paymentOption   string
}

func newCostOptimizationHubStore() *cohStore {
	now := time.Now().UTC()
	return &cohStore{
		created: now, updated: now, status: "Inactive",
		savingsEstimationMode: "BeforeDiscounts", memberDiscountVisible: "All", term: "OneYear", paymentOption: "NoUpfront",
	}
}

// ---- enrollment and preferences ----

func (st *cohStore) preferences() map[string]any {
	return map[string]any{
		"savingsEstimationMode": st.savingsEstimationMode, "memberAccountDiscountVisibility": st.memberDiscountVisible,
		"preferredCommitment": map[string]any{"term": st.term, "paymentOption": st.paymentOption},
	}
}

func (s *Server) cohGetPreferences(map[string]any) (map[string]any, *budErr) {
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.preferences(), nil
}

func (s *Server) cohUpdatePreferences(in map[string]any) (map[string]any, *budErr) {
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	if v := budStr(in, "savingsEstimationMode"); v != "" {
		st.savingsEstimationMode = v
	}
	if v := budStr(in, "memberAccountDiscountVisibility"); v != "" {
		st.memberDiscountVisible = v
	}
	pc := budMap(in, "preferredCommitment")
	if v := budStr(pc, "term"); v != "" {
		st.term = v
	}
	if v := budStr(pc, "paymentOption"); v != "" {
		st.paymentOption = v
	}
	return st.preferences(), nil
}

func (s *Server) cohUpdateEnrollmentStatus(in map[string]any) (map[string]any, *budErr) {
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	st.status, st.updated = budStr(in, "status"), time.Now().UTC()
	st.includeMemberAccounts = budBool(in, "includeMemberAccounts")
	return map[string]any{"status": st.status}, nil
}

func (s *Server) cohListEnrollmentStatuses(in map[string]any) (map[string]any, *budErr) {
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	var items []any
	if acct := budStr(in, "accountId"); acct == "" || acct == cohAccountID {
		items = append(items, map[string]any{
			"accountId": cohAccountID, "status": st.status,
			"createdTimestamp": budEpoch(st.created), "lastUpdatedTimestamp": budEpoch(st.updated),
		})
	}
	start, end, next, err := budPage(budStr(in, "nextToken"), len(items), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items[start:end], "includeMemberAccounts": st.includeMemberAccounts, "nextToken": next}, nil
}

// ---- recommendations ----

// recommendations returns the catalog when enrolled; callers hold st.mu.
func (st *cohStore) recommendations(includeAll bool) ([]map[string]any, *budErr) {
	if st.status != "Active" {
		return nil, budErrorf("AccessDeniedException", "The account isn't enrolled in Cost Optimization Hub. Call UpdateEnrollmentStatus with status Active first.")
	}
	refreshed := st.updated.Truncate(24 * time.Hour)
	out := make([]map[string]any, 0, len(cohSeeds))
	for _, sd := range cohSeeds {
		if includeAll || !sd.superseded {
			out = append(out, cohRecommendation(sd, cohAccountID, refreshed))
		}
	}
	return out, nil
}

func (s *Server) cohGetRecommendation(in map[string]any) (map[string]any, *budErr) {
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	recs, err := st.recommendations(true)
	if err != nil {
		return nil, err
	}
	id := budStr(in, "recommendationId")
	for _, r := range recs {
		if r["recommendationId"] == id {
			return r, nil
		}
	}
	return nil, budErrorf("ResourceNotFoundException", "Recommendation %s doesn't exist.", id)
}

func cohIn(list []string, v any) bool {
	if len(list) == 0 {
		return true
	}
	str, _ := v.(string)
	for _, x := range list {
		if x == str {
			return true
		}
	}
	return false
}

func cohMatch(rec, filter map[string]any) bool {
	for _, k := range []string{"restartNeeded", "rollbackPossible"} {
		if want, ok := filter[k].(bool); ok && rec[k] != want {
			return false
		}
	}
	lists := map[string]string{
		"implementationEfforts": "implementationEffort", "accountIds": "accountId", "regions": "region",
		"resourceTypes": "currentResourceType", "actionTypes": "actionType", "resourceIds": "resourceId",
		"resourceArns": "resourceArn", "recommendationIds": "recommendationId",
	}
	for fk, rk := range lists {
		if !cohIn(budStrings(filter, fk), rec[rk]) {
			return false
		}
	}
	if want := budList(filter, "tags"); len(want) > 0 {
		for _, w := range want {
			wm, _ := w.(map[string]any)
			for _, t := range budList(rec, "tags") {
				tm, _ := t.(map[string]any)
				if tm["key"] == wm["key"] && tm["value"] == wm["value"] {
					return true
				}
			}
		}
		return false
	}
	return true
}

// cohDimensions maps orderBy / groupBy dimensions to recommendation fields.
var cohDimensions = map[string]string{
	"AccountId": "accountId", "ActionType": "actionType", "CurrentResourceType": "currentResourceType",
	"ResourceType": "currentResourceType", "RecommendedResourceType": "recommendedResourceType",
	"EstimatedMonthlySavings": "estimatedMonthlySavings", "EstimatedSavingsPercentage": "estimatedSavingsPercentage",
	"EstimatedMonthlyCost": "estimatedMonthlyCost", "ImplementationEffort": "implementationEffort", "Region": "region",
	"ResourceArn": "resourceArn", "ResourceId": "resourceId", "RestartNeeded": "restartNeeded",
	"RollbackPossible": "rollbackPossible", "Source": "source", "RecommendationId": "recommendationId",
}

var cohEffortRank = map[string]int{"VeryLow": 0, "Low": 1, "Medium": 2, "High": 3, "VeryHigh": 4}

func cohLess(a, b any, field string) bool {
	switch x := a.(type) {
	case float64:
		y, _ := b.(float64)
		return x < y
	case bool:
		y, _ := b.(bool)
		return !x && y
	case string:
		y, _ := b.(string)
		if field == "implementationEffort" {
			return cohEffortRank[x] < cohEffortRank[y]
		}
		return x < y
	}
	return false
}

func cohFiltered(st *cohStore, in map[string]any, includeAll bool) ([]map[string]any, *budErr) {
	recs, err := st.recommendations(includeAll)
	if err != nil {
		return nil, err
	}
	filter := budMap(in, "filter")
	var out []map[string]any
	for _, r := range recs {
		if cohMatch(r, filter) {
			out = append(out, r)
		}
	}
	return out, nil
}

var cohListFields = []string{
	"recommendationId", "accountId", "region", "resourceId", "resourceArn", "currentResourceType", "recommendedResourceType",
	"estimatedMonthlySavings", "estimatedSavingsPercentage", "estimatedMonthlyCost", "currencyCode", "implementationEffort",
	"restartNeeded", "actionType", "rollbackPossible", "currentResourceSummary", "recommendedResourceSummary",
	"lastRefreshTimestamp", "recommendationLookbackPeriodInDays", "source", "tags",
}

func (s *Server) cohListRecommendations(in map[string]any) (map[string]any, *budErr) {
	field, desc := "estimatedMonthlySavings", true
	if ob := budMap(in, "orderBy"); ob != nil {
		if d := budStr(ob, "dimension"); d != "" {
			if field = cohDimensions[d]; field == "" {
				return nil, cohInvalid("orderBy dimension %s isn't supported.", d)
			}
		}
		desc = budStr(ob, "order") != "Asc"
	}
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	recs, err := cohFiltered(st, in, budBool(in, "includeAllRecommendations"))
	if err != nil {
		return nil, err
	}
	sort.SliceStable(recs, func(i, j int) bool {
		a, b := recs[i][field], recs[j][field]
		if cohLess(a, b, field) == cohLess(b, a, field) { // equal: stable by id
			return recs[i]["recommendationId"].(string) < recs[j]["recommendationId"].(string)
		}
		return cohLess(a, b, field) != desc
	})
	start, end, next, err := budPage(budStr(in, "nextToken"), len(recs), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, r := range recs[start:end] {
		item := map[string]any{}
		for _, f := range cohListFields {
			item[f] = r[f]
		}
		items = append(items, item)
	}
	return map[string]any{"items": items, "nextToken": next}, nil
}

var cohGroupBy = map[string]bool{
	"AccountId": true, "Region": true, "ActionType": true, "ResourceType": true,
	"RestartNeeded": true, "RollbackPossible": true, "ImplementationEffort": true,
}

func cohGroupValue(v any) string {
	if b, ok := v.(bool); ok {
		return strconv.FormatBool(b)
	}
	s, _ := v.(string)
	return s
}

func (s *Server) cohListRecommendationSummaries(in map[string]any) (map[string]any, *budErr) {
	groupBy := budStr(in, "groupBy")
	if !cohGroupBy[groupBy] {
		return nil, cohInvalid("groupBy must be one of AccountId, Region, ActionType, ResourceType, RestartNeeded, RollbackPossible or ImplementationEffort.")
	}
	field := cohDimensions[groupBy]
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	recs, err := cohFiltered(st, in, false)
	if err != nil {
		return nil, err
	}
	type group struct {
		savings float64
		count   int
	}
	groups := map[string]*group{}
	var total, cost float64
	for _, r := range recs {
		key := cohGroupValue(r[field])
		if groups[key] == nil {
			groups[key] = &group{}
		}
		sv, _ := r["estimatedMonthlySavings"].(float64)
		c, _ := r["estimatedMonthlyCost"].(float64)
		groups[key].savings += sv
		groups[key].count++
		total += sv
		cost += c
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if groups[keys[i]].savings != groups[keys[j]].savings {
			return groups[keys[i]].savings > groups[keys[j]].savings
		}
		return keys[i] < keys[j]
	})
	start, end, next, err := budPage(budStr(in, "nextToken"), len(keys), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, k := range keys[start:end] {
		items = append(items, map[string]any{"group": k, "estimatedMonthlySavings": cohRound(groups[k].savings), "recommendationCount": float64(groups[k].count)})
	}
	out := map[string]any{
		"estimatedTotalDedupedSavings": cohRound(total), "items": items, "groupBy": groupBy, "currencyCode": "USD", "nextToken": next,
	}
	for _, m := range budStrings(in, "metrics") {
		if m == "SavingsPercentage" && cost > 0 {
			out["metrics"] = map[string]any{"savingsPercentage": strconv.FormatFloat(cohRound(total/cost*100), 'f', 2, 64)}
		}
	}
	return out, nil
}

func cohParseDay(v string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02", "2006-01", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func (s *Server) cohListEfficiencyMetrics(in map[string]any) (map[string]any, *budErr) {
	monthly := budStr(in, "granularity") == "Monthly"
	period := budMap(in, "timePeriod")
	from, ok1 := cohParseDay(budStr(period, "start"))
	to, ok2 := cohParseDay(budStr(period, "end"))
	if !ok1 || !ok2 || !from.Before(to) {
		return nil, cohInvalid("timePeriod needs start before end, as YYYY-MM-DD.")
	}
	groupBy := budStr(in, "groupBy")
	field := ""
	if groupBy != "" {
		if field = cohDimensions[groupBy]; field == "" || !cohGroupBy[groupBy] {
			return nil, cohInvalid("groupBy %s isn't supported.", groupBy)
		}
	}
	var stamps []time.Time
	layout := "2006-01-02"
	if monthly {
		layout = "2006-01"
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	for p := from; p.Before(to); {
		if len(stamps) >= cohMaxPeriods {
			return nil, cohInvalid("timePeriod spans more than %d periods.", cohMaxPeriods)
		}
		stamps = append(stamps, p)
		if monthly {
			p = p.AddDate(0, 1, 0)
		} else {
			p = p.AddDate(0, 0, 1)
		}
	}
	st := s.costOptimizationHub
	st.mu.Lock()
	defer st.mu.Unlock()
	recs, err := st.recommendations(false)
	if err != nil {
		return nil, err
	}
	type totals struct{ spend, savings float64 }
	groups := map[string]*totals{}
	for _, r := range recs {
		key := ""
		if field != "" {
			key = cohGroupValue(r[field])
		}
		if groups[key] == nil {
			groups[key] = &totals{}
		}
		c, _ := r["estimatedMonthlyCost"].(float64)
		sv, _ := r["estimatedMonthlySavings"].(float64)
		groups[key].spend += c
		groups[key].savings += sv
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if ob := budMap(in, "orderBy"); budStr(ob, "order") == "Desc" {
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	}
	start, end, next, err := budPage(budStr(in, "nextToken"), len(keys), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	scale := 1.0
	if !monthly {
		scale = 1.0 / 30
	}
	items := make([]any, 0, end-start)
	for _, k := range keys[start:end] {
		g := groups[k]
		score := 100.0
		if g.spend > 0 {
			score = cohRound((1 - g.savings/g.spend) * 100)
		}
		points := make([]any, 0, len(stamps))
		for _, t := range stamps {
			points = append(points, map[string]any{
				"timestamp": t.Format(layout), "spend": cohRound(g.spend * scale), "savings": cohRound(g.savings * scale), "score": score,
			})
		}
		item := map[string]any{"metricsByTime": points}
		if field != "" {
			item["group"] = k
		}
		items = append(items, item)
	}
	return map[string]any{"efficiencyMetricsByGroup": items, "nextToken": next}, nil
}
