package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	schedulerRegion       = "us-east-1"
	schedulerAccountID    = "123456789012"
	schedulerDefaultGroup = "default"
	// AWS allows 500 groups and 1,000,000 schedules per Region.
	// ponytail: schedules capped far lower to bound emulator memory.
	schedulerMaxGroups    = 500
	schedulerMaxSchedules = 10000
	schedulerMaxTags      = 200
)

type schedulerError struct {
	status int
	code   string
	msg    string
}

func schedulerErr(status int, code, format string, args ...any) *schedulerError {
	return &schedulerError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func schedulerValidation(format string, args ...any) *schedulerError {
	return schedulerErr(http.StatusBadRequest, "ValidationException", format, args...)
}

func schedulerNotFound(format string, args ...any) *schedulerError {
	return schedulerErr(http.StatusNotFound, "ResourceNotFoundException", format, args...)
}

func schedulerConflict(format string, args ...any) *schedulerError {
	return schedulerErr(http.StatusConflict, "ConflictException", format, args...)
}

type schedulerTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type schedulerGroup struct {
	name        string
	state       string // ACTIVE or DELETING
	created     time.Time
	modified    time.Time
	clientToken string
	tags        []schedulerTag
}

type schedulerSchedule struct {
	group, name   string
	expression    string
	timezone      string
	description   *string
	start, end    *time.Time
	state         string
	kmsKeyArn     *string
	target        []byte // canonical JSON of the validated Target, decoded fresh per response
	targetArn     string
	window        []byte // canonical JSON of FlexibleTimeWindow
	afterComplete string
	clientToken   string
	created       time.Time
	modified      time.Time
}

// schedulerScheduleInput is the body of CreateSchedule and UpdateSchedule.
type schedulerScheduleInput struct {
	GroupName                  *string                  `json:"GroupName"`
	ScheduleExpression         *string                  `json:"ScheduleExpression"`
	StartDate                  *schedulerTime           `json:"StartDate"`
	EndDate                    *schedulerTime           `json:"EndDate"`
	Description                *string                  `json:"Description"`
	ScheduleExpressionTimezone *string                  `json:"ScheduleExpressionTimezone"`
	State                      *string                  `json:"State"`
	KmsKeyArn                  *string                  `json:"KmsKeyArn"`
	Target                     *schedulerTarget         `json:"Target"`
	FlexibleTimeWindow         *schedulerFlexibleWindow `json:"FlexibleTimeWindow"`
	ClientToken                *string                  `json:"ClientToken"`
	ActionAfterCompletion      *string                  `json:"ActionAfterCompletion"`
}

// schedulerTime accepts the epoch seconds botocore sends for rest-json
// timestamps, or an ISO 8601 string.
type schedulerTime struct{ time.Time }

func (t *schedulerTime) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err == nil {
		sec := int64(f)
		t.Time = time.Unix(sec, int64((f-float64(sec))*1e9)).UTC()
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed.UTC()
	return nil
}

func schedulerEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

type schedulerStore struct {
	mu        sync.Mutex
	groups    map[string]*schedulerGroup
	schedules map[string]*schedulerSchedule // group + "/" + name
	now       func() time.Time
}

func newSchedulerStore() *schedulerStore {
	now := time.Now().UTC()
	return &schedulerStore{
		groups:    map[string]*schedulerGroup{schedulerDefaultGroup: {name: schedulerDefaultGroup, state: "ACTIVE", created: now, modified: now}},
		schedules: map[string]*schedulerSchedule{},
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func schedulerGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:scheduler:%s:%s:schedule-group/%s", schedulerRegion, schedulerAccountID, name)
}

func schedulerScheduleARN(group, name string) string {
	return fmt.Sprintf("arn:aws:scheduler:%s:%s:schedule/%s/%s", schedulerRegion, schedulerAccountID, group, name)
}

// activeGroupLocked returns the group a schedule call names. A group being
// deleted can't take schedules.
func (st *schedulerStore) activeGroupLocked(name string) (*schedulerGroup, *schedulerError) {
	g, ok := st.groups[name]
	if !ok {
		return nil, schedulerNotFound("Schedule group %s does not exist.", name)
	}
	if g.state != "ACTIVE" {
		return nil, schedulerConflict("Schedule group %s is being deleted.", name)
	}
	return g, nil
}

// settleGroupLocked finishes a pending group deletion: a DELETING group is
// reported once, then removed (deletion is asynchronous in AWS).
func (st *schedulerStore) settleGroupLocked(g *schedulerGroup) {
	if g.state == "DELETING" {
		delete(st.groups, g.name)
	}
}

func groupNameOr(v *string) string {
	if v == nil || *v == "" {
		return schedulerDefaultGroup
	}
	return *v
}

// buildSchedule validates a Create/UpdateSchedule body into a schedule.
func (st *schedulerStore) buildSchedule(name string, in schedulerScheduleInput, now time.Time) (*schedulerSchedule, *schedulerError) {
	if err := schedulerValidName("Name", name); err != nil {
		return nil, err
	}
	group := groupNameOr(in.GroupName)
	if err := schedulerValidName("GroupName", group); err != nil {
		return nil, err
	}
	if in.ScheduleExpression == nil {
		return nil, schedulerValidation("ScheduleExpression is required")
	}
	tz := ""
	if in.ScheduleExpressionTimezone != nil {
		tz = *in.ScheduleExpressionTimezone
		if tz == "" {
			return nil, schedulerValidation("ScheduleExpressionTimezone must not be empty")
		}
	}
	if err := schedulerValidateExpression(*in.ScheduleExpression, tz); err != nil {
		return nil, err
	}
	if err := schedulerValidateWindow(in.FlexibleTimeWindow); err != nil {
		return nil, err
	}
	if err := schedulerValidateTarget(in.Target); err != nil {
		return nil, err
	}
	if in.Description != nil && len(*in.Description) > 512 {
		return nil, schedulerValidation("Description must be at most 512 characters")
	}
	if in.KmsKeyArn != nil && (!schedulerKMSPattern.MatchString(*in.KmsKeyArn) || len(*in.KmsKeyArn) > 2048) {
		return nil, schedulerValidation("KmsKeyArn must be a KMS key or alias ARN")
	}
	state := "ENABLED"
	if in.State != nil {
		if *in.State != "ENABLED" && *in.State != "DISABLED" {
			return nil, schedulerValidation("State must be ENABLED or DISABLED")
		}
		state = *in.State
	}
	after := "NONE"
	if in.ActionAfterCompletion != nil {
		if *in.ActionAfterCompletion != "NONE" && *in.ActionAfterCompletion != "DELETE" {
			return nil, schedulerValidation("ActionAfterCompletion must be NONE or DELETE")
		}
		after = *in.ActionAfterCompletion
	}
	token := ""
	if in.ClientToken != nil {
		if !schedulerTokenPattern.MatchString(*in.ClientToken) {
			return nil, schedulerValidation("ClientToken must be 1-64 characters from [a-zA-Z0-9-_]")
		}
		token = *in.ClientToken
	}
	sc := &schedulerSchedule{
		group: group, name: name, expression: *in.ScheduleExpression, timezone: tz,
		description: in.Description, state: state, kmsKeyArn: in.KmsKeyArn,
		targetArn: *in.Target.Arn, afterComplete: after, clientToken: token,
	}
	if in.StartDate != nil {
		if in.StartDate.Before(now.Add(-5 * time.Minute)) {
			return nil, schedulerValidation("The StartDate must not be earlier than 5 minutes ago.")
		}
		start := in.StartDate.Time
		sc.start = &start
	}
	if in.EndDate != nil {
		if !in.EndDate.After(now) {
			return nil, schedulerValidation("The EndDate must be in the future.")
		}
		if sc.start != nil && !in.EndDate.After(*sc.start) {
			return nil, schedulerValidation("The EndDate must be later than the StartDate.")
		}
		end := in.EndDate.Time
		sc.end = &end
	}
	target := *in.Target
	if target.RetryPolicy == nil {
		target.RetryPolicy = &schedulerRetryPolicy{}
	}
	// AWS fills the retry policy with its defaults (24 hours, 185 attempts).
	if target.RetryPolicy.MaximumEventAgeInSeconds == nil {
		v := int64(86400)
		target.RetryPolicy.MaximumEventAgeInSeconds = &v
	}
	if target.RetryPolicy.MaximumRetryAttempts == nil {
		v := int64(185)
		target.RetryPolicy.MaximumRetryAttempts = &v
	}
	sc.target, _ = json.Marshal(target)
	sc.window, _ = json.Marshal(in.FlexibleTimeWindow)
	return sc, nil
}

func (st *schedulerStore) createSchedule(name string, in schedulerScheduleInput) (map[string]any, *schedulerError) {
	now := st.now()
	sc, err := st.buildSchedule(name, in, now)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.activeGroupLocked(sc.group); err != nil {
		return nil, err
	}
	key := sc.group + "/" + name
	if existing, ok := st.schedules[key]; ok {
		// A retried create with the same client token is idempotent.
		if sc.clientToken != "" && existing.clientToken == sc.clientToken {
			return map[string]any{"ScheduleArn": schedulerScheduleARN(sc.group, name)}, nil
		}
		return nil, schedulerConflict("Schedule %s already exists.", name)
	}
	if len(st.schedules) >= schedulerMaxSchedules {
		return nil, schedulerErr(http.StatusPaymentRequired, "ServiceQuotaExceededException", "The number of schedules reached the limit of %d.", schedulerMaxSchedules)
	}
	sc.created, sc.modified = now, now
	st.schedules[key] = sc
	return map[string]any{"ScheduleArn": schedulerScheduleARN(sc.group, name)}, nil
}

func (st *schedulerStore) updateSchedule(name string, in schedulerScheduleInput) (map[string]any, *schedulerError) {
	now := st.now()
	sc, err := st.buildSchedule(name, in, now)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.activeGroupLocked(sc.group); err != nil {
		return nil, err
	}
	key := sc.group + "/" + name
	existing, ok := st.schedules[key]
	if !ok {
		return nil, schedulerNotFound("Schedule %s does not exist.", name)
	}
	// UpdateSchedule replaces every setting; only identity and creation stay.
	sc.created, sc.modified = existing.created, now
	st.schedules[key] = sc
	return map[string]any{"ScheduleArn": schedulerScheduleARN(sc.group, name)}, nil
}

func (st *schedulerStore) getSchedule(group, name string) (map[string]any, *schedulerError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	sc, ok := st.schedules[group+"/"+name]
	if !ok {
		return nil, schedulerNotFound("Schedule %s does not exist.", name)
	}
	return sc.view(), nil
}

// view builds a fresh response map; it never aliases stored state.
func (sc *schedulerSchedule) view() map[string]any {
	var target, window map[string]any
	_ = json.Unmarshal(sc.target, &target)
	_ = json.Unmarshal(sc.window, &window)
	out := map[string]any{
		"Arn": schedulerScheduleARN(sc.group, sc.name), "GroupName": sc.group, "Name": sc.name,
		"ScheduleExpression": sc.expression, "State": sc.state, "Target": target,
		"FlexibleTimeWindow": window, "ActionAfterCompletion": sc.afterComplete,
		"CreationDate": schedulerEpoch(sc.created), "LastModificationDate": schedulerEpoch(sc.modified),
	}
	if sc.timezone != "" {
		out["ScheduleExpressionTimezone"] = sc.timezone
	}
	if sc.description != nil {
		out["Description"] = *sc.description
	}
	if sc.kmsKeyArn != nil {
		out["KmsKeyArn"] = *sc.kmsKeyArn
	}
	if sc.start != nil {
		out["StartDate"] = schedulerEpoch(*sc.start)
	}
	if sc.end != nil {
		out["EndDate"] = schedulerEpoch(*sc.end)
	}
	return out
}

func (st *schedulerStore) deleteSchedule(group, name, token string) *schedulerError {
	if token != "" && !schedulerTokenPattern.MatchString(token) {
		return schedulerValidation("ClientToken must be 1-64 characters from [a-zA-Z0-9-_]")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	key := group + "/" + name
	if _, ok := st.schedules[key]; !ok {
		return schedulerNotFound("Schedule %s does not exist.", name)
	}
	delete(st.schedules, key)
	return nil
}

// schedulerPage pages a sorted key list with an opaque offset token.
func schedulerPage(keys []string, token string, max int) ([]string, string, *schedulerError) {
	start := 0
	if token != "" {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		n, convErr := strconv.ParseInt(string(raw), 10, 32)
		if err != nil || convErr != nil || n < 0 || int(n) > len(keys) {
			return nil, "", schedulerValidation("Invalid NextToken")
		}
		start = int(n)
	}
	end := min(start+max, len(keys))
	next := ""
	if end < len(keys) {
		next = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	return keys[start:end], next, nil
}

func (st *schedulerStore) listSchedules(group, prefix, state, token string, max int) (map[string]any, *schedulerError) {
	if state != "" && state != "ENABLED" && state != "DISABLED" {
		return nil, schedulerValidation("State must be ENABLED or DISABLED")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if group != "" {
		if _, ok := st.groups[group]; !ok {
			return nil, schedulerNotFound("Schedule group %s does not exist.", group)
		}
	}
	var keys []string
	for key, sc := range st.schedules {
		if (group == "" || sc.group == group) && strings.HasPrefix(sc.name, prefix) && (state == "" || sc.state == state) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	page, next, err := schedulerPage(keys, token, max)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, key := range page {
		sc := st.schedules[key]
		items = append(items, map[string]any{
			"Arn": schedulerScheduleARN(sc.group, sc.name), "Name": sc.name, "GroupName": sc.group, "State": sc.state,
			"CreationDate": schedulerEpoch(sc.created), "LastModificationDate": schedulerEpoch(sc.modified),
			"Target": map[string]any{"Arn": sc.targetArn},
		})
	}
	out := map[string]any{"Schedules": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (st *schedulerStore) createGroup(name, token string, tags []schedulerTag) (map[string]any, *schedulerError) {
	if err := schedulerValidName("Name", name); err != nil {
		return nil, err
	}
	if token != "" && !schedulerTokenPattern.MatchString(token) {
		return nil, schedulerValidation("ClientToken must be 1-64 characters from [a-zA-Z0-9-_]")
	}
	if err := schedulerValidateTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if g, ok := st.groups[name]; ok {
		if token != "" && g.clientToken == token && g.state == "ACTIVE" {
			return map[string]any{"ScheduleGroupArn": schedulerGroupARN(name)}, nil
		}
		return nil, schedulerConflict("Schedule group %s already exists.", name)
	}
	if len(st.groups) >= schedulerMaxGroups {
		return nil, schedulerErr(http.StatusPaymentRequired, "ServiceQuotaExceededException", "The number of schedule groups reached the limit of %d.", schedulerMaxGroups)
	}
	now := st.now()
	st.groups[name] = &schedulerGroup{name: name, state: "ACTIVE", created: now, modified: now, clientToken: token, tags: schedulerMergeTags(nil, tags)}
	return map[string]any{"ScheduleGroupArn": schedulerGroupARN(name)}, nil
}

func (g *schedulerGroup) view() map[string]any {
	return map[string]any{
		"Arn": schedulerGroupARN(g.name), "Name": g.name, "State": g.state,
		"CreationDate": schedulerEpoch(g.created), "LastModificationDate": schedulerEpoch(g.modified),
	}
}

func (st *schedulerStore) getGroup(name string) (map[string]any, *schedulerError) {
	if err := schedulerValidName("Name", name); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, ok := st.groups[name]
	if !ok {
		return nil, schedulerNotFound("Schedule group %s does not exist.", name)
	}
	out := g.view()
	st.settleGroupLocked(g)
	return out, nil
}

// deleteGroup deletes the group's schedules and marks it DELETING; it is
// removed after it is next read, as AWS finishes deletion asynchronously.
func (st *schedulerStore) deleteGroup(name, token string) *schedulerError {
	if err := schedulerValidName("Name", name); err != nil {
		return err
	}
	if token != "" && !schedulerTokenPattern.MatchString(token) {
		return schedulerValidation("ClientToken must be 1-64 characters from [a-zA-Z0-9-_]")
	}
	if name == schedulerDefaultGroup {
		return schedulerValidation("You can't delete the default schedule group.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, ok := st.groups[name]
	if !ok {
		return schedulerNotFound("Schedule group %s does not exist.", name)
	}
	if g.state == "DELETING" {
		return schedulerConflict("Schedule group %s is already being deleted.", name)
	}
	for key, sc := range st.schedules {
		if sc.group == name {
			delete(st.schedules, key)
		}
	}
	g.state, g.modified = "DELETING", st.now()
	return nil
}

func (st *schedulerStore) listGroups(prefix, token string, max int) (map[string]any, *schedulerError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var names []string
	for name := range st.groups {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	page, next, err := schedulerPage(names, token, max)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, name := range page {
		g := st.groups[name]
		items = append(items, g.view())
		st.settleGroupLocked(g)
	}
	out := map[string]any{"ScheduleGroups": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func schedulerValidateTags(tags []schedulerTag) *schedulerError {
	if len(tags) > schedulerMaxTags {
		return schedulerValidation("At most %d tags are allowed.", schedulerMaxTags)
	}
	for _, t := range tags {
		if t.Key == "" || len(t.Key) > 128 || t.Value == "" || len(t.Value) > 256 {
			return schedulerValidation("Tag keys must be 1-128 and values 1-256 characters")
		}
		if strings.HasPrefix(strings.ToLower(t.Key), "aws:") {
			return schedulerValidation("Tag keys must not start with aws:")
		}
	}
	return nil
}

func schedulerMergeTags(existing, add []schedulerTag) []schedulerTag {
	out := append([]schedulerTag(nil), existing...)
	for _, t := range add {
		replaced := false
		for i := range out {
			if out[i].Key == t.Key {
				out[i].Value, replaced = t.Value, true
			}
		}
		if !replaced {
			out = append(out, t)
		}
	}
	return out
}

// taggedGroupLocked resolves a tagging ARN. Only schedule groups take tags.
func (st *schedulerStore) taggedGroupLocked(arn string) (*schedulerGroup, *schedulerError) {
	prefix := fmt.Sprintf("arn:aws:scheduler:%s:%s:schedule-group/", schedulerRegion, schedulerAccountID)
	name, ok := strings.CutPrefix(arn, prefix)
	if !ok || len(arn) > 1011 || schedulerValidName("ResourceArn", name) != nil {
		return nil, schedulerValidation("ResourceArn must be a schedule group ARN")
	}
	g, found := st.groups[name]
	if !found || g.state != "ACTIVE" {
		return nil, schedulerNotFound("Schedule group %s does not exist.", name)
	}
	return g, nil
}

func (st *schedulerStore) tagResource(arn string, tags []schedulerTag) *schedulerError {
	if err := schedulerValidateTags(tags); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.taggedGroupLocked(arn)
	if err != nil {
		return err
	}
	merged := schedulerMergeTags(g.tags, tags)
	if len(merged) > schedulerMaxTags {
		return schedulerValidation("At most %d tags are allowed.", schedulerMaxTags)
	}
	g.tags = merged
	return nil
}

func (st *schedulerStore) untagResource(arn string, keys []string) *schedulerError {
	if len(keys) > schedulerMaxTags {
		return schedulerValidation("At most %d tag keys are allowed.", schedulerMaxTags)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.taggedGroupLocked(arn)
	if err != nil {
		return err
	}
	drop := map[string]bool{}
	for _, k := range keys {
		drop[k] = true
	}
	kept := g.tags[:0:0]
	for _, t := range g.tags {
		if !drop[t.Key] {
			kept = append(kept, t)
		}
	}
	g.tags = kept
	return nil
}

func (st *schedulerStore) listTags(arn string) (map[string]any, *schedulerError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.taggedGroupLocked(arn)
	if err != nil {
		return nil, err
	}
	tags := make([]any, 0, len(g.tags))
	for _, t := range g.tags {
		tags = append(tags, map[string]any{"Key": t.Key, "Value": t.Value})
	}
	return map[string]any{"Tags": tags}, nil
}

// schedulerFireTarget is what a fire needs, copied out under the lock.
type schedulerFireTarget struct {
	arn    string
	target schedulerTarget
}

// takeForFire returns the schedule's target and, for a one-time at()
// schedule set to DELETE after completion, deletes it.
func (st *schedulerStore) takeForFire(group, name string) (schedulerFireTarget, *schedulerError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	key := group + "/" + name
	sc, ok := st.schedules[key]
	if !ok {
		return schedulerFireTarget{}, schedulerNotFound("Schedule %s does not exist.", name)
	}
	if sc.state != "ENABLED" {
		return schedulerFireTarget{}, schedulerConflict("Schedule %s is DISABLED.", name)
	}
	var t schedulerTarget
	_ = json.Unmarshal(sc.target, &t)
	if sc.afterComplete == "DELETE" && strings.HasPrefix(sc.expression, "at(") {
		delete(st.schedules, key)
	}
	return schedulerFireTarget{arn: sc.targetArn, target: t}, nil
}
