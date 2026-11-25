package server

import (
	"encoding/base64"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Personalize state shared by the control plane, Personalize Events and
// Personalize Runtime. Async resources report "CREATE IN_PROGRESS" (or the
// matching transitional status) on their first Describe and settle on the
// next one; anything that depends on them requires ACTIVE.

const (
	pszAccount      = "123456789012"
	pszRegion       = "us-east-1"
	pszARNBase      = "arn:aws:personalize:" + pszRegion + ":" + pszAccount + ":"
	pszMaxResources = 5000  // every kind together
	pszMaxTags      = 50    // per resource
	pszMaxRecords   = 20000 // items, users or actions per dataset
	pszMaxEvents    = 20000 // interactions kept per dataset; the oldest are dropped
	pszMaxFilters   = 10    // per dataset group
	// pszMaxDatasets bounds stored records overall: each dataset keeps at
	// most pszMaxRecords entities and pszMaxEvents interactions.
	pszMaxDatasets   = 200
	pszDefaultPage   = 100
	pszStatusActive  = "ACTIVE"
	pszStatusCreate  = "CREATE IN_PROGRESS"
	pszStatusUpdate  = "UPDATE IN_PROGRESS"
	pszStatusFailed  = "CREATE FAILED"
	pszErrInvalid    = "InvalidInputException"
	pszErrNotFound   = "ResourceNotFoundException"
	pszErrInUse      = "ResourceInUseException"
	pszErrExists     = "ResourceAlreadyExistsException"
	pszErrLimit      = "LimitExceededException"
	pszErrToken      = "InvalidNextTokenException"
	pszErrTooManyTag = "TooManyTagsException"
)

type pszErr struct{ code, msg string }

func pszError(code, format string, args ...any) *pszErr {
	return &pszErr{code: code, msg: fmt.Sprintf(format, args...)}
}

func pszInvalid(format string, args ...any) *pszErr { return pszError(pszErrInvalid, format, args...) }

func pszMissing(arn string) *pszErr {
	return pszError(pszErrNotFound, "Could not find resource %s.", arn)
}

// pszKind describes one resource type: its ARN segment and model shapes.
type pszKind struct {
	id, arnPart, shape, summary, descKey, listKey, arnKey, nameKey string
}

// pszKinds is a package variable (not built in init) because other files'
// init functions read it, and package variables are initialized first.
var pszKinds = func() map[string]*pszKind {
	out := map[string]*pszKind{}
	for _, k := range []pszKind{
		{"datasetGroup", "dataset-group", "DatasetGroup", "DatasetGroupSummary", "datasetGroup", "datasetGroups", "datasetGroupArn", "name"},
		{"schema", "schema", "DatasetSchema", "DatasetSchemaSummary", "schema", "schemas", "schemaArn", "name"},
		{"dataset", "dataset", "Dataset", "DatasetSummary", "dataset", "datasets", "datasetArn", "name"},
		{"importJob", "dataset-import-job", "DatasetImportJob", "DatasetImportJobSummary", "datasetImportJob", "datasetImportJobs", "datasetImportJobArn", "jobName"},
		{"exportJob", "dataset-export-job", "DatasetExportJob", "DatasetExportJobSummary", "datasetExportJob", "datasetExportJobs", "datasetExportJobArn", "jobName"},
		{"deletionJob", "data-deletion-job", "DataDeletionJob", "DataDeletionJobSummary", "dataDeletionJob", "dataDeletionJobs", "dataDeletionJobArn", "jobName"},
		{"eventTracker", "event-tracker", "EventTracker", "EventTrackerSummary", "eventTracker", "eventTrackers", "eventTrackerArn", "name"},
		{"filter", "filter", "Filter", "FilterSummary", "filter", "Filters", "filterArn", "name"},
		{"solution", "solution", "Solution", "SolutionSummary", "solution", "solutions", "solutionArn", "name"},
		{"solutionVersion", "solution", "SolutionVersion", "SolutionVersionSummary", "solutionVersion", "solutionVersions", "solutionVersionArn", "name"},
		{"campaign", "campaign", "Campaign", "CampaignSummary", "campaign", "campaigns", "campaignArn", "name"},
		{"recommender", "recommender", "Recommender", "RecommenderSummary", "recommender", "recommenders", "recommenderArn", "name"},
		{"metricAttribution", "metric-attribution", "MetricAttribution", "MetricAttributionSummary", "metricAttribution", "metricAttributions", "metricAttributionArn", "name"},
		{"batchInferenceJob", "batch-inference-job", "BatchInferenceJob", "BatchInferenceJobSummary", "batchInferenceJob", "batchInferenceJobs", "batchInferenceJobArn", "jobName"},
		{"batchSegmentJob", "batch-segment-job", "BatchSegmentJob", "BatchSegmentJobSummary", "batchSegmentJob", "batchSegmentJobs", "batchSegmentJobArn", "jobName"},
	} {
		k := k
		out[k.id] = &k
	}
	return out
}()

// pszUpdate is a resource's latest*Update summary.
type pszUpdate struct {
	key              string
	fields           map[string]any
	status           string
	next             []string
	created, updated time.Time
}

type pszRes struct {
	kind             *pszKind
	arn              string
	seq              int64
	fields           map[string]any    // model members owned by the store
	rel              map[string]string // related ARNs, keyed by the List filter member
	status           string
	next             []string // statuses later Describes move through
	failure          string
	created, updated time.Time
	upd              *pszUpdate
	// per kind
	schemaFields []string   // schema: field names
	data         *pszData   // dataset
	filter       *pszFilter // filter
	recipe       *pszRecipe // solution, solutionVersion, campaign, recommender
	metrics      []any      // metric attribution
	numDeleted   int        // data deletion job
}

type personalizeStore struct {
	mu       sync.Mutex
	model    *awsmodels.Model
	res      map[string]*pszRes
	seq      int64
	tags     map[string]map[string]string
	trackers map[string]string // trackingId -> event tracker ARN
}

func newPersonalizeStore() *personalizeStore {
	return &personalizeStore{
		model: awsmodels.PersonalizeModel(), res: map[string]*pszRes{},
		tags: map[string]map[string]string{}, trackers: map[string]string{},
	}
}

// add registers a new resource; callers hold the lock and have checked the ARN is free.
func (st *personalizeStore) add(kind, arn string, fields map[string]any, rel map[string]string, tags [][2]string, now time.Time) (*pszRes, *pszErr) {
	if len(st.res) >= pszMaxResources {
		return nil, pszError(pszErrLimit, "Stackyard keeps at most %d Personalize resources.", pszMaxResources)
	}
	if len(tags) > pszMaxTags {
		return nil, pszError(pszErrTooManyTag, "A resource can have at most %d tags.", pszMaxTags)
	}
	if err := pszCheckTagKeys(tags); err != nil {
		return nil, err
	}
	st.seq++
	if rel == nil {
		rel = map[string]string{}
	}
	r := &pszRes{kind: pszKinds[kind], arn: arn, seq: st.seq, fields: fields, rel: rel,
		status: pszStatusCreate, next: []string{pszStatusActive}, created: now, updated: now}
	st.res[arn] = r
	if len(tags) > 0 {
		m := map[string]string{}
		for _, t := range tags {
			m[t[0]] = t[1]
		}
		st.tags[arn] = m
	}
	return r, nil
}

func pszCheckTagKeys(tags [][2]string) *pszErr {
	for _, t := range tags {
		if strings.HasPrefix(strings.ToLower(t[0]), "aws:") {
			return pszInvalid("Tag keys can't start with aws:.")
		}
	}
	return nil
}

// get returns the resource of kind at arn, or ResourceNotFoundException.
func (st *personalizeStore) get(kind, arn string) (*pszRes, *pszErr) {
	if r := st.res[arn]; r != nil && r.kind.id == kind {
		return r, nil
	}
	return nil, pszMissing(arn)
}

// active is get plus a check that the resource has settled to ACTIVE.
func (st *personalizeStore) active(kind, arn string) (*pszRes, *pszErr) {
	r, err := st.get(kind, arn)
	if err != nil {
		return nil, err
	}
	if r.status != pszStatusActive {
		return nil, pszError(pszErrInUse, "%s is in status %s, not ACTIVE.", arn, r.status)
	}
	return r, nil
}

func (st *personalizeStore) exists(arn string) *pszErr {
	if st.res[arn] != nil {
		return pszError(pszErrExists, "%s already exists.", arn)
	}
	return nil
}

func (r *pszRes) settle(now time.Time) {
	if len(r.next) > 0 {
		r.status, r.next, r.updated = r.next[0], r.next[1:], now
	}
	if u := r.upd; u != nil && len(u.next) > 0 {
		u.status, u.next, u.updated = u.next[0], u.next[1:], now
	}
}

func (r *pszRes) updating() bool { return r.upd != nil && len(r.upd.next) > 0 }

// startUpdate records a latest*Update summary that settles on a later read.
func (r *pszRes) startUpdate(key string, fields map[string]any, now time.Time) {
	r.upd = &pszUpdate{key: key, fields: fields, status: pszStatusUpdate, next: []string{pszStatusActive}, created: now, updated: now}
	r.updated = now
}

// view returns r as its full model shape. Callers hold the lock and conform
// the result before releasing it.
func (st *personalizeStore) view(r *pszRes) map[string]any {
	out := make(map[string]any, len(r.fields)+8)
	maps.Copy(out, r.fields)
	out[r.kind.arnKey] = r.arn
	out["status"] = r.status
	out["creationDateTime"] = pszEpoch(r.created)
	out["lastUpdatedDateTime"] = pszEpoch(r.updated)
	if strings.HasSuffix(r.status, "FAILED") && r.failure != "" {
		out["failureReason"] = r.failure
	}
	if u := r.upd; u != nil {
		v := make(map[string]any, len(u.fields)+3)
		maps.Copy(v, u.fields)
		v["status"], v["creationDateTime"], v["lastUpdatedDateTime"] = u.status, pszEpoch(u.created), pszEpoch(u.updated)
		out[u.key] = v
	}
	switch r.kind.id {
	case "dataset":
		if t := st.trackerFor(r.rel["datasetGroupArn"]); t != nil && strings.Contains(r.fields["datasetType"].(string), "Interactions") {
			out["trackingId"] = t.fields["trackingId"]
		}
	case "solution":
		var latest *pszRes
		for _, v := range st.res {
			if v.kind.id == "solutionVersion" && v.rel["solutionArn"] == r.arn && (latest == nil || v.seq > latest.seq) {
				latest = v
			}
		}
		if latest != nil {
			out["latestSolutionVersion"] = st.model.Conform("SolutionVersionSummary", st.view(latest))
		}
	case "recommender":
		if r.status == pszStatusActive {
			out["modelMetrics"] = pszModelMetrics(r.arn)
		}
	case "deletionJob":
		out["numDeleted"] = r.numDeleted
	}
	return out
}

func (st *personalizeStore) trackerFor(groupARN string) *pszRes {
	for _, r := range st.res {
		if r.kind.id == "eventTracker" && r.rel["datasetGroupArn"] == groupARN {
			return r
		}
	}
	return nil
}

// ---- generic operations ----

func (st *personalizeStore) describe(k *pszKind, arn string) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get(k.id, arn)
	if err != nil {
		return nil, err
	}
	out := map[string]any{k.descKey: st.model.Conform(k.shape, st.view(r))}
	r.settle(time.Now().UTC())
	return out, nil
}

func (st *personalizeStore) list(k *pszKind, filterKey, filterValue, token string, max int) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if filterKey == "solutionArn" && filterValue != "" && k.id == "solutionVersion" {
		if _, err := st.get("solution", filterValue); err != nil {
			return nil, err
		}
	}
	var matches []*pszRes
	for _, r := range st.res {
		if r.kind == k && (filterValue == "" || r.rel[filterKey] == filterValue) {
			matches = append(matches, r)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].seq < matches[j].seq })
	page, next, err := pszPage(len(matches), token, max)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, page[1]-page[0])
	for _, r := range matches[page[0]:page[1]] {
		items = append(items, st.model.Conform(k.summary, st.view(r)))
	}
	out := map[string]any{k.listKey: items}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// pszPage turns a token and page size into a [start, end) window over n items.
func pszPage(n int, token string, max int) ([2]int, string, *pszErr) {
	start := 0
	if token != "" {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		v, perr := strconv.ParseInt(strings.TrimPrefix(string(raw), "psz:"), 10, 32)
		if err != nil || perr != nil || !strings.HasPrefix(string(raw), "psz:") || v < 0 || int(v) > n {
			return [2]int{}, "", pszError(pszErrToken, "The nextToken is invalid.")
		}
		start = int(v)
	}
	if max < 1 || max > pszDefaultPage {
		max = pszDefaultPage
	}
	end := min(n, start+max)
	next := ""
	if end < n {
		next = base64.RawURLEncoding.EncodeToString([]byte("psz:" + strconv.Itoa(end)))
	}
	return [2]int{start, end}, next, nil
}

func (st *personalizeStore) delete(k *pszKind, arn string) *pszErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get(k.id, arn)
	if err != nil {
		return err
	}
	if blocker := st.dependent(r); blocker != "" {
		return pszError(pszErrInUse, "%s can't be deleted while %s uses it.", arn, blocker)
	}
	st.remove(r)
	if k.id == "solution" { // a solution's versions go with it
		for _, v := range st.res {
			if v.kind.id == "solutionVersion" && v.rel["solutionArn"] == arn {
				st.remove(v)
			}
		}
	}
	return nil
}

func (st *personalizeStore) remove(r *pszRes) {
	delete(st.res, r.arn)
	delete(st.tags, r.arn)
	if r.kind.id == "eventTracker" {
		delete(st.trackers, r.fields["trackingId"].(string))
	}
}

// dependent names a resource that keeps r from being deleted, or "".
func (st *personalizeStore) dependent(r *pszRes) string {
	for _, v := range st.res {
		switch {
		case r.kind.id == "datasetGroup" && v.rel["datasetGroupArn"] == r.arn && !strings.HasSuffix(v.kind.id, "Job") && v.kind.id != "solutionVersion" && v.kind.id != "campaign":
			return v.arn
		case r.kind.id == "schema" && v.kind.id == "dataset" && v.fields["schemaArn"] == r.arn:
			return v.arn
		case r.kind.id == "solution" && v.kind.id == "campaign" && v.rel["solutionArn"] == r.arn:
			return v.arn
		}
	}
	return ""
}

// ---- tags ----

func (st *personalizeStore) listTags(arn string) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.res[arn] == nil {
		return nil, pszMissing(arn)
	}
	keys := make([]string, 0, len(st.tags[arn]))
	for k := range st.tags[arn] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tags := make([]any, 0, len(keys))
	for _, k := range keys {
		tags = append(tags, map[string]any{"tagKey": k, "tagValue": st.tags[arn][k]})
	}
	return map[string]any{"tags": tags}, nil
}

func (st *personalizeStore) tagResource(arn string, tags [][2]string) *pszErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.res[arn] == nil {
		return pszMissing(arn)
	}
	if err := pszCheckTagKeys(tags); err != nil {
		return err
	}
	next := maps.Clone(st.tags[arn])
	if next == nil {
		next = map[string]string{}
	}
	for _, t := range tags {
		next[t[0]] = t[1]
	}
	if len(next) > pszMaxTags {
		return pszError(pszErrTooManyTag, "A resource can have at most %d tags.", pszMaxTags)
	}
	st.tags[arn] = next
	return nil
}

func (st *personalizeStore) untagResource(arn string, keys []string) *pszErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.res[arn] == nil {
		return pszMissing(arn)
	}
	next := maps.Clone(st.tags[arn])
	for _, k := range keys {
		delete(next, k)
	}
	st.tags[arn] = next
	return nil
}
