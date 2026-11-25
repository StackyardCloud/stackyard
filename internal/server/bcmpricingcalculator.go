package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Billing and Cost Management Pricing Calculator: awsJson1_0,
// X-Amz-Target AWSBCMPricingCalculator.*, signed bcm-pricing-calculator.
//
// Bill scenarios collect usage and commitment modifications; a bill estimate
// prices a scenario's snapshot against the built-in catalog
// (bcmpricingcalculator_catalog.go). Workload estimates price usage lines
// directly. Async work takes one request to finish: a new estimate is
// IN_PROGRESS (its scenario LOCKED) and a changed workload estimate is
// UPDATING until the next Pricing Calculator request, which completes them.

const (
	bcmpcTargetPrefix = "AWSBCMPricingCalculator."
	bcmpcContentType  = "application/x-amz-json-1.0"
	bcmpcAccountID    = "123456789012"
	bcmpcMaxScenarios = 100
	bcmpcMaxEstimates = 200
	bcmpcMaxWorkloads = 100
	bcmpcMaxUsage     = 1000 // usage lines per scenario or workload estimate
	bcmpcMaxCommits   = 100  // commitment modifications per scenario
	bcmpcMaxTags      = 200
	bcmpcLifetime     = 30 * 24 * time.Hour // default expiry of every resource
	// ponytail: one byte budget for the whole store, charged on every write.
	bcmpcMaxStoreBytes = 32 << 20
)

var bcmpcAPI = &budJSONAPI{
	prefix: bcmpcTargetPrefix, signing: "bcm-pricing-calculator", contentType: bcmpcContentType,
	invalidCode: "ValidationException", model: awsmodels.BCMPricingCalculatorModel,
	write: bcmpcRespond, writeError: bcmpcRespondError,
}

func init() {
	bcmpcAPI.handlers = map[string]budHandler{
		"GetPreferences":                                (*Server).bcmpcGetPreferences,
		"UpdatePreferences":                             (*Server).bcmpcUpdatePreferences,
		"TagResource":                                   (*Server).bcmpcTagResource,
		"UntagResource":                                 (*Server).bcmpcUntagResource,
		"ListTagsForResource":                           (*Server).bcmpcListTags,
		"CreateBillScenario":                            (*Server).bcmpcCreateScenario,
		"GetBillScenario":                               (*Server).bcmpcGetScenario,
		"UpdateBillScenario":                            (*Server).bcmpcUpdateScenario,
		"DeleteBillScenario":                            (*Server).bcmpcDeleteScenario,
		"ListBillScenarios":                             (*Server).bcmpcListScenarios,
		"BatchCreateBillScenarioUsageModification":      (*Server).bcmpcCreateUsageMods,
		"BatchUpdateBillScenarioUsageModification":      (*Server).bcmpcUpdateUsageMods,
		"BatchDeleteBillScenarioUsageModification":      (*Server).bcmpcDeleteUsageMods,
		"ListBillScenarioUsageModifications":            (*Server).bcmpcListUsageMods,
		"BatchCreateBillScenarioCommitmentModification": (*Server).bcmpcCreateCommitMods,
		"BatchUpdateBillScenarioCommitmentModification": (*Server).bcmpcUpdateCommitMods,
		"BatchDeleteBillScenarioCommitmentModification": (*Server).bcmpcDeleteCommitMods,
		"ListBillScenarioCommitmentModifications":       (*Server).bcmpcListCommitMods,
		"CreateBillEstimate":                            (*Server).bcmpcCreateEstimate,
		"GetBillEstimate":                               (*Server).bcmpcGetEstimate,
		"UpdateBillEstimate":                            (*Server).bcmpcUpdateEstimate,
		"DeleteBillEstimate":                            (*Server).bcmpcDeleteEstimate,
		"ListBillEstimates":                             (*Server).bcmpcListEstimates,
		"ListBillEstimateLineItems":                     (*Server).bcmpcListLineItems,
		"ListBillEstimateCommitments":                   (*Server).bcmpcListEstimateCommitments,
		"ListBillEstimateInputUsageModifications":       (*Server).bcmpcListEstimateInputUsage,
		"ListBillEstimateInputCommitmentModifications":  (*Server).bcmpcListEstimateInputCommits,
		"CreateWorkloadEstimate":                        (*Server).bcmpcCreateWorkload,
		"GetWorkloadEstimate":                           (*Server).bcmpcGetWorkload,
		"UpdateWorkloadEstimate":                        (*Server).bcmpcUpdateWorkload,
		"DeleteWorkloadEstimate":                        (*Server).bcmpcDeleteWorkload,
		"ListWorkloadEstimates":                         (*Server).bcmpcListWorkloads,
		"BatchCreateWorkloadEstimateUsage":              (*Server).bcmpcCreateWorkloadUsage,
		"BatchUpdateWorkloadEstimateUsage":              (*Server).bcmpcUpdateWorkloadUsage,
		"BatchDeleteWorkloadEstimateUsage":              (*Server).bcmpcDeleteWorkloadUsage,
		"ListWorkloadEstimateUsage":                     (*Server).bcmpcListWorkloadUsage,
		"GetUsageLinesEstimatedCost":                    (*Server).bcmpcUsageLinesCost,
	}
}

func (s *Server) handleBCMPricingCalculatorJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, bcmpcAPI, bcmpcOperationByName)
}

func bcmpcRespond(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", bcmpcContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// bcmpcRespondError writes an awsJson1_0 error with the members the model's
// error shapes require: resourceId and resourceType on not-found, conflict
// and quota errors, a reason on validation errors.
func bcmpcRespondError(w http.ResponseWriter, e *budErr) {
	body := map[string]string{"__type": e.code, "message": e.msg}
	for k, v := range e.extra {
		body[k] = v
	}
	if e.code == "ValidationException" && body["reason"] == "" {
		body["reason"] = "fieldValidationFailed"
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", bcmpcContentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

func bcmpcInvalid(format string, args ...any) *budErr {
	return &budErr{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

// bcmpcResourceErr builds a not-found, conflict or quota error naming the
// resource it is about.
func bcmpcResourceErr(code, id, kind, format string, args ...any) *budErr {
	status := map[string]int{
		"ResourceNotFoundException": http.StatusNotFound, "ConflictException": http.StatusConflict,
		"ServiceQuotaExceededException": http.StatusPaymentRequired,
	}[code]
	return &budErr{status: status, code: code, msg: fmt.Sprintf(format, args...),
		extra: map[string]string{"resourceId": id, "resourceType": kind}}
}

func bcmpcNotFound(id, kind string) *budErr {
	return bcmpcResourceErr("ResourceNotFoundException", id, kind, "%s %s does not exist", kind, id)
}

type bcmpcStore struct {
	mu    sync.Mutex
	prefs map[string][]string // management, member, standalone rate types

	scenarios map[string]*bcmpcScenario
	estimates map[string]*bcmpcEstimate
	workloads map[string]*bcmpcWorkload
	// pending work finishes on the next request (see advance).
	pendingEstimates, pendingWorkloads []string
	bytes                              int
}

func newBCMPricingCalculatorStore() *bcmpcStore {
	all := []string{"BEFORE_DISCOUNTS", "AFTER_DISCOUNTS", "AFTER_DISCOUNTS_AND_COMMITMENTS"}
	return &bcmpcStore{
		prefs: map[string][]string{
			"managementAccountRateTypeSelections": all, "memberAccountRateTypeSelections": all,
			"standaloneAccountRateTypeSelections": all,
		},
		scenarios: map[string]*bcmpcScenario{}, estimates: map[string]*bcmpcEstimate{}, workloads: map[string]*bcmpcWorkload{},
	}
}

// lock takes the store lock and finishes the work earlier requests started.
func (st *bcmpcStore) lock() {
	st.mu.Lock()
	st.advance()
}

func (st *bcmpcStore) advance() {
	for _, id := range st.pendingEstimates {
		if e := st.estimates[id]; e != nil && e.status == "IN_PROGRESS" {
			next := *e
			next.status = "COMPLETE"
			st.estimates[id] = &next
		}
	}
	for _, id := range st.pendingWorkloads {
		if w := st.workloads[id]; w != nil && w.status == "UPDATING" {
			next := *w
			next.status = "VALID"
			st.workloads[id] = &next
		}
	}
	st.pendingEstimates, st.pendingWorkloads = nil, nil
}

func bcmpcSize(parts ...any) int {
	n := 0
	for _, p := range parts {
		raw, _ := json.Marshal(p)
		n += len(raw)
	}
	return n
}

// charge moves the byte total from old to new; growth past the budget is
// refused. Caller holds st.mu.
func (st *bcmpcStore) charge(old, new int, id, kind string) *budErr {
	if st.bytes-old+new > bcmpcMaxStoreBytes {
		return bcmpcResourceErr("ServiceQuotaExceededException", id, kind, "Pricing Calculator data is limited to %d bytes", bcmpcMaxStoreBytes)
	}
	st.bytes += new - old
	return nil
}

// ---- preferences ----

func (st *bcmpcStore) prefsOut() map[string]any {
	out := map[string]any{}
	for k, v := range st.prefs {
		list := make([]any, 0, len(v))
		for _, r := range v {
			list = append(list, r)
		}
		out[k] = list
	}
	return out
}

func (s *Server) bcmpcGetPreferences(map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	return st.prefsOut(), nil
}

func (s *Server) bcmpcUpdatePreferences(in map[string]any) (map[string]any, *budErr) {
	if len(in) == 0 {
		return nil, bcmpcInvalid("set at least one rate type selection")
	}
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	next := map[string][]string{}
	for k, v := range st.prefs {
		next[k] = v
	}
	for k := range st.prefs {
		if _, ok := in[k]; !ok {
			continue
		}
		seen := map[string]bool{}
		var list []string
		for _, r := range budStrings(in, k) {
			if !seen[r] {
				seen[r] = true
				list = append(list, r)
			}
		}
		next[k] = list
	}
	st.prefs = next
	return st.prefsOut(), nil
}

// allowRate refuses rate types the account's preferences exclude
// (Stackyard's account is a standalone account). Caller holds st.mu.
func (st *bcmpcStore) allowRate(rate string) *budErr {
	for _, r := range st.prefs["standaloneAccountRateTypeSelections"] {
		if r == rate {
			return nil
		}
	}
	return &budErr{status: http.StatusBadRequest, code: "ValidationException",
		msg: "rate type " + rate + " is not enabled in the account's preferences", extra: map[string]string{"reason": "disallowedRate"}}
}

// ---- tags ----

func bcmpcArn(kind, id string) string {
	return "arn:aws:bcm-pricing-calculator::" + bcmpcAccountID + ":" + kind + "/" + id
}

// tagsOf finds the tags (and the owning record's size) for an ARN. Caller
// holds st.mu.
func (st *bcmpcStore) tagsOf(arn string) (*map[string]string, *int, *budErr) {
	rest, ok := strings.CutPrefix(arn, "arn:aws:bcm-pricing-calculator::"+bcmpcAccountID+":")
	kind, id, _ := strings.Cut(rest, "/")
	if ok {
		switch kind {
		case "bill-scenario":
			if r := st.scenarios[id]; r != nil {
				return &r.tags, &r.size, nil
			}
		case "bill-estimate":
			if r := st.estimates[id]; r != nil {
				return &r.tags, &r.size, nil
			}
		case "workload-estimate":
			if r := st.workloads[id]; r != nil {
				return &r.tags, &r.size, nil
			}
		}
	}
	return nil, nil, bcmpcNotFound(arn, "resource")
}

func bcmpcTagsIn(in map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range budMap(in, "tags") {
		out[k], _ = v.(string)
	}
	return out
}

func bcmpcTagsOut(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func (s *Server) bcmpcTagResource(in map[string]any) (map[string]any, *budErr) {
	arn := budStr(in, "arn")
	add := bcmpcTagsIn(in)
	for k := range add {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, bcmpcInvalid("tag keys starting with aws: are reserved")
		}
	}
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	tags, size, err := st.tagsOf(arn)
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(*tags, add, bcmpcMaxTags, "ValidationException")
	if err != nil {
		return nil, bcmpcResourceErr("ServiceQuotaExceededException", arn, "resource", "a resource can have at most %d tags", bcmpcMaxTags)
	}
	newSize := *size - bcmpcSize(*tags) + bcmpcSize(merged)
	if err := st.charge(*size, newSize, arn, "resource"); err != nil {
		return nil, err
	}
	*tags, *size = merged, newSize
	return map[string]any{}, nil
}

func (s *Server) bcmpcUntagResource(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	tags, size, err := st.tagsOf(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	kept := budDropTags(*tags, budStrings(in, "tagKeys"))
	newSize := *size - bcmpcSize(*tags) + bcmpcSize(kept)
	st.bytes += newSize - *size // shrinking never exceeds the budget
	*tags, *size = kept, newSize
	return map[string]any{}, nil
}

func (s *Server) bcmpcListTags(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmPricingCalculator
	st.lock()
	defer st.mu.Unlock()
	tags, _, err := st.tagsOf(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"tags": bcmpcTagsOut(*tags)}, nil
}

// ---- shared list helpers ----

// bcmpcMatch applies one list filter (values OR'ed) to a field value.
func bcmpcMatch(value string, f map[string]any) bool {
	for _, want := range budStrings(f, "values") {
		switch budStr(f, "matchOption") {
		case "STARTS_WITH":
			if strings.HasPrefix(value, want) {
				return true
			}
		case "CONTAINS":
			if strings.Contains(value, want) {
				return true
			}
		default:
			if value == want {
				return true
			}
		}
	}
	return false
}

// bcmpcFiltered keeps the items whose fields match every filter; field
// maps a filter name to the item's value.
func bcmpcFiltered(items []map[string]any, filters []any, field func(item map[string]any, name string) string) []map[string]any {
	out := make([]map[string]any, 0, len(items))
next:
	for _, item := range items {
		for _, f := range filters {
			fm, _ := f.(map[string]any)
			if !bcmpcMatch(field(item, budStr(fm, "name")), fm) {
				continue next
			}
		}
		out = append(out, item)
	}
	return out
}

// bcmpcInWindow applies a createdAtFilter/expiresAtFilter.
func bcmpcInWindow(t time.Time, f map[string]any) bool {
	if after, ok := budTime(f["afterTimestamp"]); ok && !t.After(after) {
		return false
	}
	if before, ok := budTime(f["beforeTimestamp"]); ok && !t.Before(before) {
		return false
	}
	return true
}

// bcmpcPage returns one page of items (already shaped) and the next token.
func bcmpcPage(in map[string]any, items []map[string]any, def int) (map[string]any, *budErr) {
	start, end, next, err := budPage(budStr(in, "nextToken"), len(items), budInt(in, "maxResults", def), "ValidationException")
	if err != nil {
		return nil, bcmpcInvalid("%s", err.msg)
	}
	out := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, budCloneMap(item))
	}
	return map[string]any{"items": out, "nextToken": next}, nil
}

// bcmpcUsageField reads the fields ListUsageFilter names from a usage item.
func bcmpcUsageField(item map[string]any, name string) string {
	hist := budMap(item, "historicalUsage")
	switch name {
	case "USAGE_ACCOUNT_ID":
		return budStr(item, "usageAccountId")
	case "SERVICE_CODE":
		return budStr(item, "serviceCode")
	case "USAGE_TYPE":
		return budStr(item, "usageType")
	case "OPERATION":
		return budStr(item, "operation")
	case "LOCATION":
		return budStr(item, "location")
	case "USAGE_GROUP":
		return budStr(item, "group")
	case "LINE_ITEM_TYPE":
		return budStr(item, "lineItemType")
	case "HISTORICAL_USAGE_ACCOUNT_ID":
		return budStr(hist, "usageAccountId")
	case "HISTORICAL_SERVICE_CODE":
		return budStr(hist, "serviceCode")
	case "HISTORICAL_USAGE_TYPE":
		return budStr(hist, "usageType")
	case "HISTORICAL_OPERATION":
		return budStr(hist, "operation")
	case "HISTORICAL_LOCATION":
		return budStr(hist, "location")
	}
	return ""
}

// bcmpcSortByCreated orders resources newest first, then by id.
func bcmpcSortByCreated(items []map[string]any) {
	sort.Slice(items, func(i, j int) bool {
		a, _ := items[i]["createdAt"].(float64)
		b, _ := items[j]["createdAt"].(float64)
		if a != b {
			return a > b
		}
		return budStr(items[i], "id") < budStr(items[j], "id")
	})
}

// bcmpcExpiry reads an expiresAt update, which must be in the future.
func bcmpcExpiry(in map[string]any, now time.Time) (time.Time, bool, *budErr) {
	v, ok := in["expiresAt"]
	if !ok {
		return time.Time{}, false, nil
	}
	t, ok := budTime(v)
	if !ok || !t.After(now) {
		return time.Time{}, false, bcmpcInvalid("expiresAt must be a time in the future")
	}
	return t, true, nil
}

// bcmpcInterval is the bill interval new scenarios cover: the current month.
func bcmpcInterval(now time.Time) (time.Time, time.Time) {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

func bcmpcIntervalOut(start, end time.Time) map[string]any {
	return map[string]any{"start": budEpoch(start), "end": budEpoch(end)}
}
