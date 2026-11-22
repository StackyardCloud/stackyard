package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	backupRegion  = "us-east-1"
	backupAccount = "123456789012"
	// backupMaxItems bounds every caller-created collection.
	backupMaxItems = 1000
	backupMaxTags  = 50
)

var (
	bkVaultNamePattern = regexp.MustCompile(`^[a-zA-Z0-9\-_]{2,50}$`)
	bkRoleARNPattern   = regexp.MustCompile(`^arn:aws[a-z-]*:iam::[0-9]{12}:role/.+$`)
	bkTokenPattern     = regexp.MustCompile(`^[a-zA-Z0-9\-_.]{1,50}$`)
)

// backupStore keeps every AWS Backup resource. Records are the API shapes
// (map[string]any keyed by member name); responses are projections of them
// onto the generated backupShapeMembers lists, so internal state never leaks.
// ponytail: one mutex for the whole service; split per resource if contention shows.
type backupStore struct {
	mu             sync.Mutex
	vaults         map[string]*bkVault // by name
	plans          map[string]*bkPlan  // by id
	recoveryPoints map[string]*bkRecoveryPoint
	jobs           map[string]*bkJob // backup, copy, restore, scan and report jobs, by id
	protected      map[string]map[string]any
	frameworks     map[string]map[string]any
	reportPlans    map[string]map[string]any
	legalHolds     map[string]map[string]any
	testingPlans   map[string]*bkTestingPlan
	tiering        map[string]map[string]any
	accessPoints   map[string]map[string]any
	tags           map[string]map[string]string
	globalSettings map[string]any
	globalUpdated  float64
	regionSettings map[string]any
	idempotentJobs map[string]string // kind+token -> job id
}

func newBackupStore() *backupStore {
	optIn := map[string]any{}
	for _, t := range bkResourceTypes {
		optIn[t] = true
	}
	return &backupStore{
		vaults:         map[string]*bkVault{},
		plans:          map[string]*bkPlan{},
		recoveryPoints: map[string]*bkRecoveryPoint{},
		jobs:           map[string]*bkJob{},
		protected:      map[string]map[string]any{},
		frameworks:     map[string]map[string]any{},
		reportPlans:    map[string]map[string]any{},
		legalHolds:     map[string]map[string]any{},
		testingPlans:   map[string]*bkTestingPlan{},
		tiering:        map[string]map[string]any{},
		accessPoints:   map[string]map[string]any{},
		tags:           map[string]map[string]string{},
		globalSettings: map[string]any{"isCrossAccountBackupEnabled": "false", "isMpaEnabled": "false", "isDelegatedAdministratorEnabled": "false"},
		regionSettings: map[string]any{
			"ResourceTypeOptInPreference":      optIn,
			"ResourceTypeManagementPreference": map[string]any{"DynamoDB": false, "EFS": false},
			"ResourceTypeOptInIncremental":     map[string]any{},
		},
		idempotentJobs: map[string]string{},
	}
}

// bkResourceTypes are the types GetSupportedResourceTypes reports.
var bkResourceTypes = []string{"Aurora", "CloudFormation", "DocumentDB", "DynamoDB", "EBS", "EC2", "EFS", "FSx", "Neptune", "Redshift", "Redshift Serverless", "RDS", "S3", "SAP HANA on Amazon EC2", "Storage Gateway", "Timestream", "VirtualMachine"}

type bkHandler func(st *backupStore, s *Server, in map[string]any) (any, *bedrockAgentError)

var backupHandlers map[string]bkHandler

func init() {
	backupHandlers = map[string]bkHandler{
		"DescribeGlobalSettings":    (*backupStore).describeGlobalSettings,
		"UpdateGlobalSettings":      (*backupStore).updateGlobalSettings,
		"DescribeRegionSettings":    (*backupStore).describeRegionSettings,
		"UpdateRegionSettings":      (*backupStore).updateRegionSettings,
		"GetSupportedResourceTypes": (*backupStore).getSupportedResourceTypes,
		"TagResource":               (*backupStore).tagResource,
		"UntagResource":             (*backupStore).untagResource,
		"ListTags":                  (*backupStore).listTags,
	}
	for _, group := range []map[string]bkHandler{bkVaultHandlers(), bkPlanHandlers(), bkJobHandlers(), bkRecoveryPointHandlers(), bkAuditHandlers()} {
		for name, h := range group {
			backupHandlers[name] = h
		}
	}
}

// handle runs one operation and returns the encoded response. Encoding
// happens under the lock because responses share maps with stored records.
func (st *backupStore) handle(s *Server, op string, in map[string]any) ([]byte, *bedrockAgentError) {
	h, ok := backupHandlers[op]
	if !ok {
		return nil, &bedrockAgentError{status: 501, code: "NotImplementedException", msg: op + " is not implemented"}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, s, in)
	if err != nil {
		return nil, err
	}
	if rec, isRec := out.(map[string]any); isRec {
		if members, known := backupShapeMembers[op+"Output"]; known {
			out = bkProject(rec, members)
		}
	}
	if out == nil {
		out = map[string]any{}
	}
	body, mErr := json.Marshal(out)
	if mErr != nil {
		return nil, &bedrockAgentError{status: 500, code: "ServiceUnavailableException", msg: mErr.Error()}
	}
	return body, nil
}

// --- shared helpers ---

// bkNow is an epoch-seconds timestamp, the rest-json wire format.
func bkNow() float64 { return float64(time.Now().UnixMilli()) / 1000 }

func bkARN(kind, id string) string {
	return "arn:aws:backup:" + backupRegion + ":" + backupAccount + ":" + kind + ":" + id
}

func bkProject(rec map[string]any, members []string) map[string]any {
	out := make(map[string]any, len(members))
	for _, k := range members {
		v, ok := rec[k]
		switch t := v.(type) {
		case nil:
			ok = false
		case map[string]any:
			ok = ok && t != nil
		case []any:
			ok = ok && t != nil
		}
		if ok {
			out[k] = v
		}
	}
	return out
}

func bkShape(rec map[string]any, shape string) map[string]any {
	return bkProject(rec, backupShapeMembers[shape])
}

// bkRequire checks required members; AWS Backup reports them as
// MissingParameterValueException.
func bkRequire(in map[string]any, keys ...string) *bedrockAgentError {
	for _, k := range keys {
		v, ok := in[k]
		if !ok || v == nil {
			return bkMissing(k + " is required")
		}
		if s, isStr := v.(string); isStr && strings.TrimSpace(s) == "" {
			return bkMissing(k + " must not be empty")
		}
	}
	return nil
}

func bkStr(in map[string]any, key string) string { return baStr(in, key) }

func bkMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}

func bkList(in map[string]any, key string) []any {
	l, _ := in[key].([]any)
	return l
}

// bkInt reads an optional integer member at 32 bits (all Backup counts and
// day values fit). absent reports whether the member was missing.
func bkInt(in map[string]any, key string) (n int, absent bool, err *bedrockAgentError) {
	v, ok := in[key]
	if !ok || v == nil {
		return 0, true, nil
	}
	n, valid := baInt(v)
	if !valid {
		return 0, false, bkInvalid(key + " must be an integer")
	}
	return n, false, nil
}

func bkEnum(in map[string]any, key string, allowed ...string) *bedrockAgentError {
	v, ok := in[key]
	if !ok || v == nil {
		return nil
	}
	s, _ := v.(string)
	if containsString(allowed, s) {
		return nil
	}
	return bkInvalid(fmt.Sprintf("%s must be one of %s", key, strings.Join(allowed, ", ")))
}

func bkCheckRole(in map[string]any, key string) *bedrockAgentError {
	if v := bkStr(in, key); v != "" && !bkRoleARNPattern.MatchString(v) {
		return bkInvalid(key + " must be an IAM role ARN")
	}
	return nil
}

func bkCheckToken(in map[string]any, key string) *bedrockAgentError {
	if v := bkStr(in, key); v != "" && !bkTokenPattern.MatchString(v) {
		return bkInvalid(key + " must be 1-50 alphanumeric, '-', '_' or '.' characters")
	}
	return nil
}

// bkTime parses a timestamp filter: epoch seconds or ISO 8601.
func bkTime(in map[string]any, key string) (float64, bool, *bedrockAgentError) {
	v, ok := in[key]
	if !ok || v == nil {
		return 0, false, nil
	}
	s := fmt.Sprint(v)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return float64(t.UnixMilli()) / 1000, true, nil
		}
	}
	return 0, false, bkInvalid(key + " must be a timestamp")
}

func bkFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

// bkPage applies MaxResults/NextToken; the token is the offset of the next item.
func bkPage(items []map[string]any, in map[string]any, listKey string) (map[string]any, *bedrockAgentError) {
	limit := backupMaxItems
	if n, absent, err := bkInt(in, "MaxResults"); err != nil {
		return nil, err
	} else if !absent {
		if n < 1 || n > backupMaxItems {
			return nil, bkInvalid("MaxResults must be between 1 and 1000")
		}
		limit = n
	}
	start := 0
	if tok := bkStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, bkInvalid("invalid NextToken")
		}
		start = int(n)
	}
	end := min(len(items), start+limit)
	out := map[string]any{listKey: items[start:end]}
	if end < len(items) {
		out["NextToken"] = strconv.Itoa(end)
	}
	return out, nil
}

// bkSorted orders records by a timestamp member, then by key, so listings and
// page tokens are stable.
func bkSorted[T any](m map[string]T, rec func(T) map[string]any, timeKey string) []T {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ti, tj := bkFloat(rec(m[keys[i]])[timeKey]), bkFloat(rec(m[keys[j]])[timeKey])
		if ti != tj {
			return ti < tj
		}
		return keys[i] < keys[j]
	})
	out := make([]T, len(keys))
	for i, k := range keys {
		out[i] = m[k]
	}
	return out
}

func bkSelf(m map[string]any) map[string]any { return m }

func bkNewID() string { return uuid.NewString() }

// --- tags ---

// setTags validates and stores tags for a new or existing resource.
func (st *backupStore) setTags(arn string, tags map[string]any) *bedrockAgentError {
	if err := st.checkTags(arn, tags); err != nil || len(tags) == 0 {
		return err
	}
	cur := st.tags[arn]
	if cur == nil {
		cur = map[string]string{}
		st.tags[arn] = cur
	}
	for k, v := range tags {
		cur[k] = v.(string)
	}
	return nil
}

// checkTags validates tags without storing them, so callers can reject a
// request before creating anything.
func (st *backupStore) checkTags(arn string, tags map[string]any) *bedrockAgentError {
	total := len(st.tags[arn])
	for k, v := range tags {
		s, ok := v.(string)
		if !ok || k == "" || len(k) > 128 || len(s) > 256 {
			return bkInvalid("tag keys must be 1-128 characters and values strings of at most 256")
		}
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return bkInvalid("tag keys may not start with aws:")
		}
		if _, exists := st.tags[arn][k]; !exists {
			total++
		}
	}
	if total > backupMaxTags {
		return bkErr("LimitExceededException", "a resource can have at most 50 tags")
	}
	return nil
}

// taggable reports whether arn names a live resource that takes tags.
func (st *backupStore) taggable(arn string) bool {
	if _, ok := st.recoveryPoints[arn]; ok {
		return true
	}
	if _, ok := st.accessPoints[arn]; ok {
		return true
	}
	for _, v := range st.vaults {
		if v.rec["BackupVaultArn"] == arn {
			return true
		}
	}
	for _, p := range st.plans {
		if p.current()["BackupPlanArn"] == arn {
			return true
		}
	}
	for _, tp := range st.testingPlans {
		if tp.rec["RestoreTestingPlanArn"] == arn {
			return true
		}
	}
	for _, group := range []map[string]map[string]any{st.frameworks, st.reportPlans, st.legalHolds, st.tiering} {
		for _, rec := range group {
			for _, key := range []string{"FrameworkArn", "ReportPlanArn", "LegalHoldArn", "TieringConfigurationArn"} {
				if rec[key] == arn {
					return true
				}
			}
		}
	}
	return false
}

func (st *backupStore) tagResource(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "ResourceArn", "Tags"); err != nil {
		return nil, err
	}
	arn := bkStr(in, "ResourceArn")
	if !st.taggable(arn) {
		return nil, bkNotFound("resource", arn)
	}
	return nil, st.setTags(arn, bkMap(in, "Tags"))
}

func (st *backupStore) untagResource(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "ResourceArn", "TagKeyList"); err != nil {
		return nil, err
	}
	arn := bkStr(in, "ResourceArn")
	if !st.taggable(arn) {
		return nil, bkNotFound("resource", arn)
	}
	for _, k := range bkList(in, "TagKeyList") {
		key, _ := k.(string)
		delete(st.tags[arn], key)
	}
	return nil, nil
}

func (st *backupStore) listTags(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	arn := bkStr(in, "ResourceArn")
	if !st.taggable(arn) {
		return nil, bkNotFound("resource", arn)
	}
	tags := map[string]any{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return map[string]any{"Tags": tags}, nil
}

// --- settings ---

func (st *backupStore) describeGlobalSettings(_ *Server, _ map[string]any) (any, *bedrockAgentError) {
	out := map[string]any{"GlobalSettings": st.globalSettings}
	if st.globalUpdated > 0 {
		out["LastUpdateTime"] = st.globalUpdated
	}
	return out, nil
}

func (st *backupStore) updateGlobalSettings(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	for k, v := range bkMap(in, "GlobalSettings") {
		if _, known := st.globalSettings[k]; !known {
			return nil, bkInvalid("unknown global setting " + k)
		}
		if s, _ := v.(string); s != "true" && s != "false" {
			return nil, bkInvalid("global setting " + k + " must be \"true\" or \"false\"")
		}
		st.globalSettings[k] = v
	}
	st.globalUpdated = bkNow()
	return nil, nil
}

func (st *backupStore) describeRegionSettings(_ *Server, _ map[string]any) (any, *bedrockAgentError) {
	return st.regionSettings, nil
}

func (st *backupStore) updateRegionSettings(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	for _, key := range []string{"ResourceTypeOptInPreference", "ResourceTypeManagementPreference", "ResourceTypeOptInIncremental"} {
		cur := bkMap(st.regionSettings, key)
		for k, v := range bkMap(in, key) {
			if _, isBool := v.(bool); !isBool {
				return nil, bkInvalid(key + " values must be booleans")
			}
			if key == "ResourceTypeOptInPreference" && !containsString(bkResourceTypes, k) {
				return nil, bkInvalid("unsupported resource type " + k)
			}
			cur[k] = v
		}
	}
	return nil, nil
}

func (st *backupStore) getSupportedResourceTypes(_ *Server, _ map[string]any) (any, *bedrockAgentError) {
	return map[string]any{"ResourceTypes": bkResourceTypes}, nil
}
