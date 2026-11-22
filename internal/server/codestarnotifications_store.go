package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	csnRegion    = "us-east-1"
	csnAccount   = "123456789012"
	csnCreatedBy = "arn:aws:iam::123456789012:root"
	// AWS quotas: 1,000 rules per account and 10 targets per rule.
	csnMaxRules   = 1000
	csnMaxTargets = 10
	// ponytail: AWS has no published target quota; cap the registry so it
	// can't grow without bound.
	csnMaxTargetRegistry = 5000
	csnMaxTags           = 50
	csnMaxBody           = 1 << 20
)

type csnEventType struct {
	EventTypeId   string `json:"EventTypeId"`
	ServiceName   string `json:"ServiceName"`
	EventTypeName string `json:"EventTypeName"`
	ResourceType  string `json:"ResourceType"`
}

// csnEventTypes is the event catalog AWS returns from ListEventTypes.
var csnEventTypes = func() []csnEventType {
	add := func(out []csnEventType, service, resource, prefix string, names map[string]string) []csnEventType {
		for suffix, name := range names {
			out = append(out, csnEventType{prefix + suffix, service, name, resource})
		}
		return out
	}
	var out []csnEventType
	out = add(out, "CodeCommit", "Repository", "codecommit-repository-", map[string]string{
		"comments-on-commits":         "Comments - On commits",
		"comments-on-pull-requests":   "Comments - On pull requests",
		"approvals-status-changed":    "Approvals - Status changed",
		"approvals-rule-override":     "Approvals - Rule override",
		"pull-request-created":        "Pull request - Created",
		"pull-request-source-updated": "Pull request - Source updated",
		"pull-request-status-changed": "Pull request - Status changed",
		"pull-request-merged":         "Pull request - Merged",
		"branches-and-tags-created":   "Branches and tags - Created",
		"branches-and-tags-deleted":   "Branches and tags - Deleted",
		"branches-and-tags-updated":   "Branches and tags - Updated",
	})
	out = add(out, "CodeBuild", "Project", "codebuild-project-", map[string]string{
		"build-state-failed":      "Build state - Failed",
		"build-state-succeeded":   "Build state - Succeeded",
		"build-state-in-progress": "Build state - In-progress",
		"build-state-stopped":     "Build state - Stopped",
		"build-phase-failure":     "Build phase - Failure",
		"build-phase-success":     "Build phase - Success",
	})
	out = add(out, "CodeDeploy", "Application", "codedeploy-application-", map[string]string{
		"deployment-failed":    "Deployment - Failed",
		"deployment-succeeded": "Deployment - Succeeded",
		"deployment-started":   "Deployment - Started",
	})
	out = add(out, "CodePipeline", "Pipeline", "codepipeline-pipeline-", map[string]string{
		"action-execution-succeeded":    "Action execution - Succeeded",
		"action-execution-failed":       "Action execution - Failed",
		"action-execution-canceled":     "Action execution - Canceled",
		"action-execution-started":      "Action execution - Started",
		"stage-execution-started":       "Stage execution - Started",
		"stage-execution-succeeded":     "Stage execution - Succeeded",
		"stage-execution-resumed":       "Stage execution - Resumed",
		"stage-execution-canceled":      "Stage execution - Canceled",
		"stage-execution-failed":        "Stage execution - Failed",
		"pipeline-execution-failed":     "Pipeline execution - Failed",
		"pipeline-execution-canceled":   "Pipeline execution - Canceled",
		"pipeline-execution-started":    "Pipeline execution - Started",
		"pipeline-execution-resumed":    "Pipeline execution - Resumed",
		"pipeline-execution-succeeded":  "Pipeline execution - Succeeded",
		"pipeline-execution-superseded": "Pipeline execution - Superseded",
		"manual-approval-failed":        "Manual approval - Failed",
		"manual-approval-needed":        "Manual approval - Needed",
		"manual-approval-succeeded":     "Manual approval - Succeeded",
	})
	sort.Slice(out, func(i, j int) bool { return out[i].EventTypeId < out[j].EventTypeId })
	return out
}()

var csnEventTypeByID = func() map[string]csnEventType {
	out := make(map[string]csnEventType, len(csnEventTypes))
	for _, e := range csnEventTypes {
		out[e.EventTypeId] = e
	}
	return out
}()

var (
	csnRuleARNPattern    = regexp.MustCompile(`^arn:aws[^:\s]*:codestar-notifications:[^:\s]+:\d{12}:notificationrule/(.*\S)?$`)
	csnResourcePattern   = regexp.MustCompile(`^arn:aws[^:\s]*:[^:\s]*:[^:\s]*:[0-9]{12}:[^\s]+$`)
	csnRuleNamePattern   = regexp.MustCompile(`^[A-Za-z0-9\-_ ]+$`)
	csnTokenPattern      = regexp.MustCompile(`^[\w:/-]+$`)
	csnTargetTypePattern = regexp.MustCompile(`^[A-Za-z]+$`)
	csnTagPattern        = regexp.MustCompile(`^([\p{L}\p{Z}\p{N}_.:/=+\-@]*)$`)
	csnChatbotPattern    = regexp.MustCompile(`^arn:aws[^:\s]*:chatbot::\d{12}:chat-configuration/(slack|microsoft-teams)-channel/\S+$`)
	csnNextTokenPattern  = regexp.MustCompile(`^[\w/+=]+$`)
)

type csnError struct {
	status int
	code   string
	msg    string
}

func csnErr(status int, code, format string, args ...any) *csnError {
	return &csnError{status, code, fmt.Sprintf(format, args...)}
}

func csnValidation(format string, args ...any) *csnError {
	return csnErr(http.StatusBadRequest, "ValidationException", format, args...)
}

func csnNotFound(format string, args ...any) *csnError {
	return csnErr(http.StatusNotFound, "ResourceNotFoundException", format, args...)
}

type csnTarget struct {
	TargetType    string `json:"TargetType"`
	TargetAddress string `json:"TargetAddress"`
}

type csnRule struct {
	arn        string
	name       string
	eventTypes []string
	resource   string
	targets    []string // target addresses, in subscription order
	detailType string
	status     string
	token      string
	created    time.Time
	modified   time.Time
	tags       map[string]string
}

type codeStarNotificationsStore struct {
	mu      sync.Mutex
	rules   map[string]*csnRule // by ARN
	order   []string            // rule ARNs in creation order
	targets map[string]string   // target address -> target type
	tOrder  []string            // target addresses in registration order
}

func newCodeStarNotificationsStore() *codeStarNotificationsStore {
	return &codeStarNotificationsStore{rules: map[string]*csnRule{}, targets: map[string]string{}}
}

// csnStatusFunc reports a target's status; the server checks SNS topics.
type csnStatusFunc func(targetType, address string) string

// csnPage slices items by a decimal offset token.
func csnPage[T any](items []T, token string, max int) ([]T, string, *csnError) {
	start := 0
	if token != "" {
		n, err := strconv.ParseInt(token, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", csnErr(http.StatusBadRequest, "InvalidNextTokenException", "The specified NextToken is not valid")
		}
		start = int(n)
	}
	end := min(start+max, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

func csnValidTags(tags map[string]string) *csnError {
	if len(tags) > csnMaxTags {
		return csnErr(http.StatusBadRequest, "LimitExceededException", "A notification rule can have at most %d tags", csnMaxTags)
	}
	for k, v := range tags {
		if len(k) < 1 || len(k) > 128 || !csnTagPattern.MatchString(k) {
			return csnValidation("Tag key %q is not valid", k)
		}
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return csnValidation("Tag keys can't start with aws:")
		}
		if len(v) > 256 || !csnTagPattern.MatchString(v) {
			return csnValidation("Tag value for %q is not valid", k)
		}
	}
	return nil
}

func csnValidRuleName(name string) *csnError {
	if len(name) < 1 || len(name) > 64 || !csnRuleNamePattern.MatchString(name) {
		return csnValidation("Name must be 1-64 letters, numbers, spaces, hyphens or underscores")
	}
	return nil
}

func csnValidRuleARN(arn string) *csnError {
	if !csnRuleARNPattern.MatchString(arn) {
		return csnValidation("%q is not a notification rule ARN", arn)
	}
	return nil
}

// csnEventTypesFor checks that every id exists and belongs to service.
func csnEventTypesFor(ids []string, service string) ([]string, *csnError) {
	if len(ids) == 0 {
		return nil, csnValidation("EventTypeIds must contain at least one event type")
	}
	if len(ids) > len(csnEventTypes) {
		return nil, csnValidation("EventTypeIds has more entries than there are event types")
	}
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		e, ok := csnEventTypeByID[id]
		if !ok {
			return nil, csnValidation("Event type %q does not exist", id)
		}
		if e.ServiceName != service {
			return nil, csnValidation("Event type %s is a %s event and can't be used with a %s resource", id, e.ServiceName, service)
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

// registerTargetsLocked records targets in the registry, rejecting an
// address already registered with another type.
func (st *codeStarNotificationsStore) registerTargetsLocked(targets []csnTarget) *csnError {
	added := 0
	for _, t := range targets {
		if have, ok := st.targets[t.TargetAddress]; ok {
			if have != t.TargetType {
				return csnValidation("Target %s is already registered with type %s", t.TargetAddress, have)
			}
		} else {
			added++
		}
	}
	if len(st.targets)+added > csnMaxTargetRegistry {
		return csnErr(http.StatusBadRequest, "LimitExceededException", "Too many notification targets")
	}
	for _, t := range targets {
		if _, ok := st.targets[t.TargetAddress]; !ok {
			st.targets[t.TargetAddress] = t.TargetType
			st.tOrder = append(st.tOrder, t.TargetAddress)
		}
	}
	return nil
}

func csnAddresses(targets []csnTarget) []string {
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		if !containsString(out, t.TargetAddress) {
			out = append(out, t.TargetAddress)
		}
	}
	return out
}

type csnCreateInput struct {
	Name               string            `json:"Name"`
	EventTypeIds       []string          `json:"EventTypeIds"`
	Resource           string            `json:"Resource"`
	Targets            []csnTarget       `json:"Targets"`
	DetailType         string            `json:"DetailType"`
	ClientRequestToken string            `json:"ClientRequestToken"`
	Tags               map[string]string `json:"Tags"`
	Status             string            `json:"Status"`
}

// create stores a rule whose resource and targets the server already checked.
func (st *codeStarNotificationsStore) create(in csnCreateInput, eventTypes []string) (map[string]any, *csnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, arn := range st.order {
		r := st.rules[arn]
		if r.name != in.Name {
			continue
		}
		if in.ClientRequestToken != "" && r.token == in.ClientRequestToken {
			return map[string]any{"Arn": r.arn}, nil // idempotent retry
		}
		return nil, csnErr(http.StatusConflict, "ResourceAlreadyExistsException", "A notification rule named %s already exists", in.Name)
	}
	if len(st.rules) >= csnMaxRules {
		return nil, csnErr(http.StatusBadRequest, "LimitExceededException", "An account can have at most %d notification rules", csnMaxRules)
	}
	if err := st.registerTargetsLocked(in.Targets); err != nil {
		return nil, err
	}
	id := make([]byte, 20)
	_, _ = rand.Read(id)
	now := time.Now().UTC()
	status := in.Status
	if status == "" {
		status = "ENABLED"
	}
	tags := map[string]string{}
	for k, v := range in.Tags {
		tags[k] = v
	}
	r := &csnRule{
		arn:  fmt.Sprintf("arn:aws:codestar-notifications:%s:%s:notificationrule/%s", csnRegion, csnAccount, hex.EncodeToString(id)),
		name: in.Name, eventTypes: eventTypes, resource: in.Resource, targets: csnAddresses(in.Targets),
		detailType: in.DetailType, status: status, token: in.ClientRequestToken, created: now, modified: now, tags: tags,
	}
	st.rules[r.arn] = r
	st.order = append(st.order, r.arn)
	return map[string]any{"Arn": r.arn}, nil
}

// ruleResource returns a rule's resource ARN, so the server can check event
// types and targets against it before an update.
func (st *codeStarNotificationsStore) ruleResource(arn string) (string, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[arn]
	if !ok {
		return "", false
	}
	return r.resource, true
}

type csnUpdateInput struct {
	Arn          string      `json:"Arn"`
	Name         *string     `json:"Name"`
	Status       *string     `json:"Status"`
	EventTypeIds []string    `json:"EventTypeIds"`
	Targets      []csnTarget `json:"Targets"`
	DetailType   *string     `json:"DetailType"`
}

func (st *codeStarNotificationsStore) update(in csnUpdateInput, eventTypes []string) *csnError {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[in.Arn]
	if !ok {
		return csnNotFound("Notification rule %s does not exist", in.Arn)
	}
	if in.Name != nil && *in.Name != r.name {
		for _, other := range st.rules {
			if other.name == *in.Name {
				return csnValidation("A notification rule named %s already exists", *in.Name)
			}
		}
	}
	if in.Targets != nil {
		if err := st.registerTargetsLocked(in.Targets); err != nil {
			return err
		}
		r.targets = csnAddresses(in.Targets)
	}
	if in.Name != nil {
		r.name = *in.Name
	}
	if in.Status != nil {
		r.status = *in.Status
	}
	if eventTypes != nil {
		r.eventTypes = eventTypes
	}
	if in.DetailType != nil {
		r.detailType = *in.DetailType
	}
	r.modified = time.Now().UTC()
	return nil
}

func (st *codeStarNotificationsStore) delete(arn string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.rules[arn]; !ok {
		return // AWS treats deleting a missing rule as success
	}
	delete(st.rules, arn)
	st.order = csnWithout(st.order, arn)
}

func (st *codeStarNotificationsStore) describe(arn string, status csnStatusFunc) (map[string]any, *csnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[arn]
	if !ok {
		return nil, csnNotFound("Notification rule %s does not exist", arn)
	}
	events := make([]csnEventType, 0, len(r.eventTypes))
	for _, id := range r.eventTypes {
		events = append(events, csnEventTypeByID[id])
	}
	tags := map[string]string{}
	for k, v := range r.tags {
		tags[k] = v
	}
	return map[string]any{
		"Arn": r.arn, "Name": r.name, "EventTypes": events, "Resource": r.resource,
		"Targets": st.targetSummariesLocked(r.targets, status), "DetailType": r.detailType,
		"CreatedBy": csnCreatedBy, "Status": r.status, "Tags": tags,
		"CreatedTimestamp": float64(r.created.UnixMilli()) / 1000, "LastModifiedTimestamp": float64(r.modified.UnixMilli()) / 1000,
	}, nil
}

func (st *codeStarNotificationsStore) targetSummariesLocked(addresses []string, status csnStatusFunc) []map[string]any {
	out := make([]map[string]any, 0, len(addresses))
	for _, addr := range addresses {
		typ := st.targets[addr]
		out = append(out, map[string]any{"TargetAddress": addr, "TargetType": typ, "TargetStatus": status(typ, addr)})
	}
	return out
}

type csnFilter struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

func (st *codeStarNotificationsStore) listRules(filters []csnFilter, token string, max int) (map[string]any, *csnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var matched []map[string]any
	for _, arn := range st.order {
		r := st.rules[arn]
		keep := true
		for _, f := range filters {
			switch f.Name {
			case "EVENT_TYPE_ID":
				keep = keep && containsString(r.eventTypes, f.Value)
			case "CREATED_BY":
				keep = keep && f.Value == csnCreatedBy
			case "RESOURCE":
				keep = keep && r.resource == f.Value
			case "TARGET_ADDRESS":
				keep = keep && containsString(r.targets, f.Value)
			}
		}
		if keep {
			matched = append(matched, map[string]any{"Id": r.arn[strings.LastIndex(r.arn, "/")+1:], "Arn": r.arn})
		}
	}
	page, next, err := csnPage(matched, token, max)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"NotificationRules": append([]map[string]any{}, page...)}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (st *codeStarNotificationsStore) listTargets(filters []csnFilter, token string, max int, status csnStatusFunc) (map[string]any, *csnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var matched []map[string]any
	for _, s := range st.targetSummariesLocked(st.tOrder, status) {
		keep := true
		for _, f := range filters {
			switch f.Name {
			case "TARGET_TYPE":
				keep = keep && s["TargetType"] == f.Value
			case "TARGET_ADDRESS":
				keep = keep && s["TargetAddress"] == f.Value
			case "TARGET_STATUS":
				keep = keep && s["TargetStatus"] == f.Value
			}
		}
		if keep {
			matched = append(matched, s)
		}
	}
	page, next, err := csnPage(matched, token, max)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"Targets": append([]map[string]any{}, page...)}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (st *codeStarNotificationsStore) subscribe(arn string, t csnTarget) *csnError {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[arn]
	if !ok {
		return csnNotFound("Notification rule %s does not exist", arn)
	}
	if containsString(r.targets, t.TargetAddress) {
		return nil
	}
	if len(r.targets) >= csnMaxTargets {
		return csnValidation("A notification rule can have at most %d targets", csnMaxTargets)
	}
	if err := st.registerTargetsLocked([]csnTarget{t}); err != nil {
		return err
	}
	r.targets = append(r.targets, t.TargetAddress)
	r.modified = time.Now().UTC()
	return nil
}

func (st *codeStarNotificationsStore) unsubscribe(arn, address string) *csnError {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[arn]
	if !ok {
		return csnValidation("Notification rule %s does not exist", arn)
	}
	if containsString(r.targets, address) {
		r.targets = csnWithout(r.targets, address)
		r.modified = time.Now().UTC()
	}
	return nil
}

func (st *codeStarNotificationsStore) deleteTarget(address string, force bool) *csnError {
	st.mu.Lock()
	defer st.mu.Unlock()
	var using []*csnRule
	for _, arn := range st.order {
		if r := st.rules[arn]; containsString(r.targets, address) {
			using = append(using, r)
		}
	}
	if len(using) > 0 && !force {
		return csnValidation("Target %s is still subscribed to %d notification rule(s); set ForceUnsubscribeAll to remove it", address, len(using))
	}
	for _, r := range using {
		r.targets = csnWithout(r.targets, address)
	}
	if _, ok := st.targets[address]; ok {
		delete(st.targets, address)
		st.tOrder = csnWithout(st.tOrder, address)
	}
	return nil
}

func (st *codeStarNotificationsStore) tag(arn string, tags map[string]string) (map[string]any, *csnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[arn]
	if !ok {
		return nil, csnNotFound("Notification rule %s does not exist", arn)
	}
	merged := map[string]string{}
	for k, v := range r.tags {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	if err := csnValidTags(merged); err != nil {
		return nil, err
	}
	r.tags = merged
	out := map[string]string{}
	for k, v := range merged {
		out[k] = v
	}
	return map[string]any{"Tags": out}, nil
}

func (st *codeStarNotificationsStore) untag(arn string, keys []string) *csnError {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[arn]
	if !ok {
		return csnNotFound("Notification rule %s does not exist", arn)
	}
	for _, k := range keys {
		delete(r.tags, k)
	}
	return nil
}

func (st *codeStarNotificationsStore) listTags(arn string) (map[string]any, *csnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, ok := st.rules[arn]
	if !ok {
		return nil, csnNotFound("Notification rule %s does not exist", arn)
	}
	out := map[string]string{}
	for k, v := range r.tags {
		out[k] = v
	}
	return map[string]any{"Tags": out}, nil
}

// csnWithout returns items without any entry equal to v, as a new slice.
func csnWithout(items []string, v string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it != v {
			out = append(out, it)
		}
	}
	return out
}
