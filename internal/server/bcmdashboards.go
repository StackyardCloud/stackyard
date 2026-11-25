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

// AWS Billing and Cost Management Dashboards: awsJson1_0, X-Amz-Target
// AWSBCMDashboardsService.*, signed bcm-dashboards. Dashboards hold widgets
// whose Cost Explorer-style queries are checked (bcmdashboards_widgets.go)
// and stored, not run. Scheduled reports point at a dashboard; their health
// is derived on every read from whether the dashboard and widget still exist.

const (
	bcmdTargetPrefix  = "AWSBCMDashboardsService."
	bcmdContentType   = "application/x-amz-json-1.0"
	bcmdAccountID     = "123456789012"
	bcmdMaxDashboards = 200
	bcmdMaxReports    = 200
	bcmdMaxTags       = 200
	// ponytail: one byte budget for the whole store; every create, update
	// and tag change is charged against it.
	bcmdMaxStoreBytes = 16 << 20
)

var bcmdAPI = &budJSONAPI{
	prefix: bcmdTargetPrefix, signing: "bcm-dashboards", contentType: bcmdContentType,
	invalidCode: "ValidationException", model: awsmodels.BCMDashboardsModel,
	write: bcmdRespond, writeError: bcmdRespondError,
}

func init() {
	bcmdAPI.handlers = map[string]budHandler{
		"CreateDashboard":        (*Server).bcmdCreateDashboard,
		"GetDashboard":           (*Server).bcmdGetDashboard,
		"UpdateDashboard":        (*Server).bcmdUpdateDashboard,
		"DeleteDashboard":        (*Server).bcmdDeleteDashboard,
		"ListDashboards":         (*Server).bcmdListDashboards,
		"GetResourcePolicy":      (*Server).bcmdGetResourcePolicy,
		"CreateScheduledReport":  (*Server).bcmdCreateReport,
		"GetScheduledReport":     (*Server).bcmdGetReport,
		"UpdateScheduledReport":  (*Server).bcmdUpdateReport,
		"DeleteScheduledReport":  (*Server).bcmdDeleteReport,
		"ListScheduledReports":   (*Server).bcmdListReports,
		"ExecuteScheduledReport": (*Server).bcmdExecuteReport,
		"TagResource":            (*Server).bcmdTagResource,
		"UntagResource":          (*Server).bcmdUntagResource,
		"ListTagsForResource":    (*Server).bcmdListTags,
	}
}

func (s *Server) handleBCMDashboardsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, bcmdAPI, bcmdOperationByName)
}

func bcmdRespond(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", bcmdContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// bcmdRespondError writes an awsJson1_0 error; the model's error shapes
// carry a lowercase message.
func bcmdRespondError(w http.ResponseWriter, e *budErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "message": e.msg})
	w.Header().Set("Content-Type", bcmdContentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func bcmdErrorf(code, format string, args ...any) *budErr {
	status := map[string]int{
		"ResourceNotFoundException": http.StatusNotFound, "ConflictException": http.StatusConflict,
		"ServiceQuotaExceededException": http.StatusPaymentRequired, "AccessDeniedException": http.StatusForbidden,
	}[code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	return &budErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func bcmdInvalid(format string, args ...any) *budErr {
	return bcmdErrorf("ValidationException", format, args...)
}

type bcmdDashboard struct {
	arn, name, description string
	widgets                []any
	created, updated       time.Time
	tags                   map[string]string
	size                   int
}

type bcmdReport struct {
	arn              string
	input            map[string]any // ScheduledReportInput members
	created, updated time.Time
	lastExecution    time.Time
	tags             map[string]string
	token, tokenSum  string // clientToken and the request it created
	size             int
}

type bcmdStore struct {
	mu         sync.Mutex
	dashboards map[string]*bcmdDashboard
	reports    map[string]*bcmdReport
	bytes      int
}

func newBCMDashboardsStore() *bcmdStore {
	return &bcmdStore{dashboards: map[string]*bcmdDashboard{}, reports: map[string]*bcmdReport{}}
}

func bcmdSize(parts ...any) int {
	n := 0
	for _, p := range parts {
		raw, _ := json.Marshal(p)
		n += len(raw)
	}
	return n
}

// charge moves the store's byte total from old to new, refusing growth past
// the budget. Caller holds st.mu.
func (st *bcmdStore) charge(old, new int) *budErr {
	if st.bytes-old+new > bcmdMaxStoreBytes {
		return bcmdErrorf("ServiceQuotaExceededException", "dashboards and scheduled reports are limited to %d bytes in all", bcmdMaxStoreBytes)
	}
	st.bytes += new - old
	return nil
}

func bcmdArn(kind string) string {
	return "arn:aws:bcm-dashboards::" + bcmdAccountID + ":" + kind + "/" + budNewID()
}

// ---- tags ----

func bcmdTagsIn(list []any) (map[string]string, *budErr) {
	out := map[string]string{}
	for _, item := range list {
		m, _ := item.(map[string]any)
		k := budStr(m, "key")
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, bcmdInvalid("tag keys starting with aws: are reserved")
		}
		out[k] = budStr(m, "value")
	}
	if len(out) > bcmdMaxTags {
		return nil, bcmdInvalid("a resource can have at most %d tags", bcmdMaxTags)
	}
	return out, nil
}

func bcmdTagsOut(tags map[string]string) []any {
	out := []any{}
	for _, t := range budTagsOut(tags) {
		m := t.(map[string]any)
		out = append(out, map[string]any{"key": m["Key"], "value": m["Value"]})
	}
	return out
}

// tagsOf returns a pointer to the tags of the dashboard or report arn names,
// with the record's size field. Caller holds st.mu.
func (st *bcmdStore) tagsOf(arn string) (*map[string]string, *int, *budErr) {
	if d := st.dashboards[arn]; d != nil {
		return &d.tags, &d.size, nil
	}
	if r := st.reports[arn]; r != nil {
		return &r.tags, &r.size, nil
	}
	return nil, nil, bcmdErrorf("ResourceNotFoundException", "resource %s does not exist", arn)
}

func (s *Server) bcmdTagResource(in map[string]any) (map[string]any, *budErr) {
	add, err := bcmdTagsIn(budList(in, "resourceTags"))
	if err != nil {
		return nil, err
	}
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, size, err := st.tagsOf(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(*tags, add, bcmdMaxTags, "ValidationException")
	if err != nil {
		return nil, err
	}
	newSize := *size - bcmdSize(*tags) + bcmdSize(merged)
	if err := st.charge(*size, newSize); err != nil {
		return nil, err
	}
	*tags, *size = merged, newSize
	return map[string]any{}, nil
}

func (s *Server) bcmdUntagResource(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, size, err := st.tagsOf(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	kept := budDropTags(*tags, budStrings(in, "resourceTagKeys"))
	newSize := *size - bcmdSize(*tags) + bcmdSize(kept)
	_ = st.charge(*size, newSize) // shrinking never fails
	*tags, *size = kept, newSize
	return map[string]any{}, nil
}

func (s *Server) bcmdListTags(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, _, err := st.tagsOf(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"resourceTags": bcmdTagsOut(*tags)}, nil
}

// ---- dashboards ----

func (d *bcmdDashboard) out() map[string]any {
	return map[string]any{
		"arn": d.arn, "name": d.name, "description": d.description, "type": "CUSTOM",
		"widgets": budClone(d.widgets), "createdAt": budEpoch(d.created), "updatedAt": budEpoch(d.updated),
	}
}

func (st *bcmdStore) dashboard(arn string) (*bcmdDashboard, *budErr) {
	if d := st.dashboards[arn]; d != nil {
		return d, nil
	}
	return nil, bcmdErrorf("ResourceNotFoundException", "dashboard %s does not exist", arn)
}

func (s *Server) bcmdCreateDashboard(in map[string]any) (map[string]any, *budErr) {
	name := budStr(in, "name")
	if err := bcmdCheckName("name", name); err != nil {
		return nil, err
	}
	widgets, err := bcmdCheckWidgets(budList(in, "widgets"))
	if err != nil {
		return nil, err
	}
	tags, err := bcmdTagsIn(budList(in, "resourceTags"))
	if err != nil {
		return nil, err
	}
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.dashboards) >= bcmdMaxDashboards {
		return nil, bcmdErrorf("ServiceQuotaExceededException", "an account can have at most %d dashboards", bcmdMaxDashboards)
	}
	now := time.Now().UTC()
	d := &bcmdDashboard{arn: bcmdArn("dashboard"), name: name, description: budStr(in, "description"), widgets: widgets, created: now, updated: now, tags: tags}
	d.size = bcmdSize(d.name, d.description, d.widgets, d.tags)
	if err := st.charge(0, d.size); err != nil {
		return nil, err
	}
	st.dashboards[d.arn] = d
	return map[string]any{"arn": d.arn}, nil
}

func (s *Server) bcmdGetDashboard(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.dashboard(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	return d.out(), nil
}

func (s *Server) bcmdUpdateDashboard(in map[string]any) (map[string]any, *budErr) {
	name := budStr(in, "name")
	if err := bcmdCheckName("name", name); err != nil {
		return nil, err
	}
	_, replaceWidgets := in["widgets"]
	widgets, err := bcmdCheckWidgets(budList(in, "widgets"))
	if err != nil {
		return nil, err
	}
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.dashboard(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	next := *d
	next.name, next.updated = name, time.Now().UTC()
	if _, ok := in["description"]; ok {
		next.description = budStr(in, "description")
	}
	if replaceWidgets {
		next.widgets = widgets
	}
	next.size = bcmdSize(next.name, next.description, next.widgets, next.tags)
	if err := st.charge(d.size, next.size); err != nil {
		return nil, err
	}
	st.dashboards[d.arn] = &next
	return map[string]any{"arn": d.arn}, nil
}

func (s *Server) bcmdDeleteDashboard(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.dashboard(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	// Reports on it stay; they turn UNHEALTHY (DASHBOARD_NOT_FOUND).
	_ = st.charge(d.size, 0)
	delete(st.dashboards, d.arn)
	return map[string]any{"arn": d.arn}, nil
}

func (s *Server) bcmdListDashboards(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	all := make([]*bcmdDashboard, 0, len(st.dashboards))
	for _, d := range st.dashboards {
		all = append(all, d)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].name != all[j].name {
			return all[i].name < all[j].name
		}
		return all[i].arn < all[j].arn
	})
	start, end, next, err := budPage(budStr(in, "nextToken"), len(all), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, d := range all[start:end] {
		ref := d.out()
		delete(ref, "widgets")
		items = append(items, ref)
	}
	return map[string]any{"dashboards": items, "nextToken": next}, nil
}

// bcmdGetResourcePolicy returns the dashboard's resource-based policy: the
// owning account may read and manage it.
func (s *Server) bcmdGetResourcePolicy(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.dashboard(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	policy, _ := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []any{map[string]any{
			"Sid": "OwnerAccess", "Effect": "Allow",
			"Principal": map[string]any{"AWS": "arn:aws:iam::" + bcmdAccountID + ":root"},
			"Action":    []string{"bcm-dashboards:GetDashboard", "bcm-dashboards:UpdateDashboard", "bcm-dashboards:DeleteDashboard"},
			"Resource":  d.arn,
		}},
	})
	return map[string]any{"resourceArn": d.arn, "policyDocument": string(policy)}, nil
}
