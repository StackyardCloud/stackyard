package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"time"
)

// Scheduled reports for BCM Dashboards. Health is not stored: it is derived
// on every read from whether the dashboard (and the chosen widget) still
// exist.

var bcmdSchedule = regexp.MustCompile(`^(cron\([^()]{1,200}\)|rate\([1-9][0-9]{0,4} (minute|minutes|hour|hours|day|days)\))$`)

// bcmdCheckReport validates a full ScheduledReportInput and returns a copy
// with schedule defaults. Caller holds st.mu (the dashboard must exist).
func (st *bcmdStore) checkReport(in map[string]any) (map[string]any, *budErr) {
	r := budCloneMap(in)
	if err := bcmdCheckName("name", budStr(r, "name")); err != nil {
		return nil, err
	}
	cfg := budMap(r, "scheduleConfig")
	if !bcmdSchedule.MatchString(budStr(cfg, "scheduleExpression")) {
		return nil, bcmdInvalid("scheduleConfig.scheduleExpression must be a cron(...) or rate(N unit) expression")
	}
	if budStr(cfg, "scheduleExpressionTimeZone") == "" {
		cfg["scheduleExpressionTimeZone"] = "UTC"
	}
	if budStr(cfg, "state") == "" {
		cfg["state"] = "ENABLED"
	}
	if p := budMap(cfg, "schedulePeriod"); p != nil {
		start, ok1 := budTime(p["startTime"])
		end, ok2 := budTime(p["endTime"])
		if ok1 && ok2 && !end.After(start) {
			return nil, bcmdInvalid("schedulePeriod endTime must be after startTime")
		}
	}
	if err := bcmdCheckRange(budMap(r, "widgetDateRangeOverride")); err != nil {
		return nil, err
	}
	d := st.dashboards[budStr(r, "dashboardArn")]
	if d == nil {
		return nil, bcmdInvalid("dashboard %s does not exist", budStr(r, "dashboardArn"))
	}
	for _, id := range budStrings(r, "widgetIds") {
		if !bcmdHasWidget(d, id) {
			return nil, bcmdInvalid("widget %s is not on dashboard %s", id, d.arn)
		}
	}
	return r, nil
}

func bcmdHasWidget(d *bcmdDashboard, id string) bool {
	for _, w := range d.widgets {
		if budStr(w.(map[string]any), "id") == id {
			return true
		}
	}
	return false
}

// health derives a report's health now. Caller holds st.mu.
func (st *bcmdStore) health(r *bcmdReport, now time.Time) map[string]any {
	reasons := []any{}
	if d := st.dashboards[budStr(r.input, "dashboardArn")]; d == nil {
		reasons = append(reasons, "DASHBOARD_NOT_FOUND")
	} else {
		for _, id := range budStrings(r.input, "widgetIds") {
			if !bcmdHasWidget(d, id) {
				reasons = append(reasons, "WIDGET_ID_NOT_FOUND")
			}
		}
	}
	code := "HEALTHY"
	if len(reasons) > 0 {
		code = "UNHEALTHY"
	}
	return map[string]any{"statusCode": code, "lastRefreshedAt": budEpoch(now), "statusReasons": reasons}
}

func (st *bcmdStore) report(arn string) (*bcmdReport, *budErr) {
	if r := st.reports[arn]; r != nil {
		return r, nil
	}
	return nil, bcmdErrorf("ResourceNotFoundException", "scheduled report %s does not exist", arn)
}

func (st *bcmdStore) nameTaken(name, except string) bool {
	for _, r := range st.reports {
		if r.arn != except && budStr(r.input, "name") == name {
			return true
		}
	}
	return false
}

func bcmdSum(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Server) bcmdCreateReport(in map[string]any) (map[string]any, *budErr) {
	tags, err := bcmdTagsIn(budList(in, "resourceTags"))
	if err != nil {
		return nil, err
	}
	token, sum := budStr(in, "clientToken"), bcmdSum(map[string]any{"r": in["scheduledReport"], "t": in["resourceTags"]})
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	if token != "" {
		for _, r := range st.reports {
			if r.token != token {
				continue
			}
			if r.tokenSum != sum {
				return nil, bcmdErrorf("ConflictException", "clientToken %s was used with a different request", token)
			}
			return map[string]any{"arn": r.arn}, nil
		}
	}
	input, err := st.checkReport(budMap(in, "scheduledReport"))
	if err != nil {
		return nil, err
	}
	if st.nameTaken(budStr(input, "name"), "") {
		return nil, bcmdErrorf("ConflictException", "a scheduled report named %s already exists", budStr(input, "name"))
	}
	if len(st.reports) >= bcmdMaxReports {
		return nil, bcmdErrorf("ServiceQuotaExceededException", "an account can have at most %d scheduled reports", bcmdMaxReports)
	}
	now := time.Now().UTC()
	r := &bcmdReport{arn: bcmdArn("scheduled-report"), input: input, created: now, updated: now, tags: tags, token: token, tokenSum: sum}
	r.size = bcmdSize(r.input, r.tags, r.token)
	if err := st.charge(0, r.size); err != nil {
		return nil, err
	}
	st.reports[r.arn] = r
	return map[string]any{"arn": r.arn}, nil
}

func (st *bcmdStore) reportOut(r *bcmdReport, now time.Time) map[string]any {
	out := budCloneMap(r.input)
	out["arn"], out["createdAt"], out["updatedAt"] = r.arn, budEpoch(r.created), budEpoch(r.updated)
	out["healthStatus"] = st.health(r, now)
	if !r.lastExecution.IsZero() {
		out["lastExecutionAt"] = budEpoch(r.lastExecution)
	}
	return out
}

func (s *Server) bcmdGetReport(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.report(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"scheduledReport": st.reportOut(r, time.Now().UTC())}, nil
}

func (s *Server) bcmdUpdateReport(in map[string]any) (map[string]any, *budErr) {
	if budBool(in, "clearWidgetIds") && in["widgetIds"] != nil {
		return nil, bcmdInvalid("set widgetIds or clearWidgetIds, not both")
	}
	if budBool(in, "clearWidgetDateRangeOverride") && in["widgetDateRangeOverride"] != nil {
		return nil, bcmdInvalid("set widgetDateRangeOverride or clearWidgetDateRangeOverride, not both")
	}
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.report(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	merged := budCloneMap(r.input)
	for _, k := range []string{"name", "description", "dashboardArn", "scheduledReportExecutionRoleArn", "widgetIds", "widgetDateRangeOverride"} {
		if v, ok := in[k]; ok {
			merged[k] = budClone(v)
		}
	}
	if cfg := budMap(in, "scheduleConfig"); cfg != nil {
		next := budMap(merged, "scheduleConfig")
		for k, v := range cfg {
			next[k] = budClone(v)
		}
	}
	if budBool(in, "clearWidgetIds") {
		delete(merged, "widgetIds")
	}
	if budBool(in, "clearWidgetDateRangeOverride") {
		delete(merged, "widgetDateRangeOverride")
	}
	input, err := st.checkReport(merged)
	if err != nil {
		return nil, err
	}
	if st.nameTaken(budStr(input, "name"), r.arn) {
		return nil, bcmdErrorf("ConflictException", "a scheduled report named %s already exists", budStr(input, "name"))
	}
	next := *r
	next.input, next.updated = input, time.Now().UTC()
	next.size = bcmdSize(next.input, next.tags, next.token)
	if err := st.charge(r.size, next.size); err != nil {
		return nil, err
	}
	st.reports[r.arn] = &next
	return map[string]any{"arn": r.arn}, nil
}

func (s *Server) bcmdDeleteReport(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.report(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	_ = st.charge(r.size, 0)
	delete(st.reports, r.arn)
	return map[string]any{"arn": r.arn}, nil
}

func (s *Server) bcmdListReports(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	all := make([]*bcmdReport, 0, len(st.reports))
	for _, r := range st.reports {
		all = append(all, r)
	}
	sort.Slice(all, func(i, j int) bool { return budStr(all[i].input, "name") < budStr(all[j].input, "name") })
	start, end, next, err := budPage(budStr(in, "nextToken"), len(all), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	items := make([]any, 0, end-start)
	for _, r := range all[start:end] {
		cfg := budMap(r.input, "scheduleConfig")
		item := map[string]any{
			"arn": r.arn, "name": r.input["name"], "dashboardArn": r.input["dashboardArn"],
			"scheduleExpression": cfg["scheduleExpression"], "scheduleExpressionTimeZone": cfg["scheduleExpressionTimeZone"],
			"state": cfg["state"], "healthStatus": st.health(r, now),
		}
		if ids, ok := r.input["widgetIds"]; ok {
			item["widgetIds"] = budClone(ids)
		}
		items = append(items, item)
	}
	return map[string]any{"scheduledReports": items, "nextToken": next}, nil
}

// bcmdExecuteReport runs a report now. A dry run only reports health; an
// unhealthy report is not triggered.
func (s *Server) bcmdExecuteReport(in map[string]any) (map[string]any, *budErr) {
	st := s.bcmDashboards
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.report(budStr(in, "arn"))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	health := st.health(r, now)
	triggered := health["statusCode"] == "HEALTHY" && !budBool(in, "dryRun")
	if triggered {
		next := *r
		next.lastExecution = now
		st.reports[r.arn] = &next
	}
	return map[string]any{"healthStatus": health, "executionTriggered": triggered}, nil
}
