package server

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	txAccountID          = "123456789012"
	txRegion             = "us-east-1"
	txMaxAdapters        = 100
	txMaxVersions        = 100
	txMaxTags            = 50
	txDefaultAdapterPage = 100
)

type txAdapterVersion struct {
	version     string
	created     time.Time
	status      string
	dataset     map[string]any
	output      map[string]any
	kmsKeyID    string
	tags        map[string]string
	token, hash string
}

type txAdapter struct {
	id, name, description, autoUpdate string
	features                          []any
	created                           time.Time
	tags                              map[string]string
	versions                          map[string]*txAdapterVersion
	nextVersion                       int
}

type txTokenRef struct{ id, hash string }

type textractStore struct {
	mu            sync.Mutex
	jobs          map[string]*txJob
	jobOrder      []string
	jobTokens     map[string]txTokenRef // api/token -> job
	adapters      map[string]*txAdapter
	adapterTokens map[string]txTokenRef // token -> adapter id
	storedBlocks  int                   // sum of job result costs
}

func newTextractStore() *textractStore {
	return &textractStore{
		jobs: map[string]*txJob{}, jobTokens: map[string]txTokenRef{},
		adapters: map[string]*txAdapter{}, adapterTokens: map[string]txTokenRef{},
	}
}

// ---- jobs ----

// jobForToken returns the job an earlier Start* with this token created.
func (st *textractStore) jobForToken(api, token, hash string) (string, *txErr) {
	if token == "" {
		return "", nil
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	ref, ok := st.jobTokens[api+"/"+token]
	if !ok || st.jobs[ref.id] == nil {
		return "", nil
	}
	if ref.hash != hash {
		return "", txError("IdempotentParameterMismatchException", "ClientRequestToken was reused with different parameters")
	}
	return ref.id, nil
}

func (st *textractStore) addJob(job *txJob, result *txResult, token, hash string) string {
	st.mu.Lock()
	defer st.mu.Unlock()
	if ref, ok := st.jobTokens[job.api+"/"+token]; token != "" && ok && st.jobs[ref.id] != nil {
		return ref.id // a concurrent Start with the same token won
	}
	// Jobs finish on read, so the oldest can go to make room.
	for len(st.jobOrder) > 0 && (len(st.jobOrder) >= txMaxJobs || st.storedBlocks+result.cost > txMaxStoredBlocks) {
		old := st.jobOrder[0]
		st.jobOrder = st.jobOrder[1:]
		st.storedBlocks -= st.jobs[old].result.cost
		delete(st.jobs, old)
		for k, ref := range st.jobTokens {
			if ref.id == old {
				delete(st.jobTokens, k)
			}
		}
	}
	job.id, job.created, job.result = txJobID(), time.Now().UTC(), result
	st.storedBlocks += result.cost
	st.jobs[job.id] = job
	st.jobOrder = append(st.jobOrder, job.id)
	if token != "" {
		st.jobTokens[job.api+"/"+token] = txTokenRef{id: job.id, hash: hash}
	}
	return job.id
}

// readJob advances a job one step per read: IN_PROGRESS, then SUCCEEDED.
func (st *textractStore) readJob(api, id string) (*txJob, string, string, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	job := st.jobs[id]
	if job == nil || job.api != api {
		return nil, "", "", txError("InvalidJobIdException", "Request has invalid Job Id")
	}
	job.reads++
	if job.reads == 1 {
		return job, "IN_PROGRESS", "", nil
	}
	if job.failure != "" {
		return job, "FAILED", job.failure, nil
	}
	return job, "SUCCEEDED", "", nil
}

// failJob records why job failed and returns the reason.
func (st *textractStore) failJob(job *txJob, reason string) string {
	st.mu.Lock()
	defer st.mu.Unlock()
	job.failure = reason
	return reason
}

// markNotified reports whether the caller should run job's completion side
// effects (exactly once).
func (st *textractStore) markNotified(job *txJob) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	if job.notified {
		return false
	}
	job.notified = true
	return true
}

// ---- adapters ----

func txAdapterARN(id string) string {
	return "arn:aws:textract:" + txRegion + ":" + txAccountID + ":/adapters/" + id
}

var txARNRe = regexp.MustCompile(`^arn:aws[a-z-]*:textract:[a-z0-9-]+:\d{12}:/adapters/([^/]+)(?:/versions/([^/]+))?$`)

func txTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		return time.UnixMilli(int64(t * 1000)).UTC(), true
	case string:
		if p, err := time.Parse(time.RFC3339, t); err == nil {
			return p, true
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return time.UnixMilli(int64(f * 1000)).UTC(), true
		}
	}
	return time.Time{}, false
}

func txEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func txTagsIn(raw map[string]any, into map[string]string) *txErr {
	for k, v := range raw {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return txError("ValidationException", "Tag keys can't start with aws:")
		}
		s, _ := v.(string)
		into[k] = s
	}
	if len(into) > txMaxTags {
		return txError("ServiceQuotaExceededException", "A resource can have at most "+strconv.Itoa(txMaxTags)+" tags")
	}
	return nil
}

func txTagsOut(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func (s *Server) txCreateAdapter(in map[string]any) (map[string]any, *txErr) {
	st := s.textract
	st.mu.Lock()
	defer st.mu.Unlock()
	token, hash := txStr(in, "ClientRequestToken"), txParamHash(in)
	if ref, ok := st.adapterTokens[token]; token != "" && ok && st.adapters[ref.id] != nil {
		if ref.hash != hash {
			return nil, txError("IdempotentParameterMismatchException", "ClientRequestToken was reused with different parameters")
		}
		return map[string]any{"AdapterId": ref.id}, nil
	}
	if len(st.adapters) >= txMaxAdapters {
		return nil, txError("ServiceQuotaExceededException", "An account can have at most "+strconv.Itoa(txMaxAdapters)+" adapters")
	}
	features, _ := in["FeatureTypes"].([]any)
	if len(features) == 0 {
		return nil, txInvalid("FeatureTypes must name at least one feature")
	}
	ad := &txAdapter{
		id: txJobID()[:12], name: txStr(in, "AdapterName"), description: txStr(in, "Description"),
		autoUpdate: txStr(in, "AutoUpdate"), features: features, created: time.Now().UTC(),
		tags: map[string]string{}, versions: map[string]*txAdapterVersion{},
	}
	if ad.autoUpdate == "" {
		ad.autoUpdate = "DISABLED"
	}
	if err := txTagsIn(txMap(in, "Tags"), ad.tags); err != nil {
		return nil, err
	}
	st.adapters[ad.id] = ad
	if token != "" {
		st.adapterTokens[token] = txTokenRef{id: ad.id, hash: hash}
	}
	return map[string]any{"AdapterId": ad.id}, nil
}

func (st *textractStore) adapterLocked(id string) (*txAdapter, *txErr) {
	ad := st.adapters[id]
	if ad == nil {
		return nil, txError("ResourceNotFoundException", "Adapter "+id+" does not exist")
	}
	return ad, nil
}

func (ad *txAdapter) out() map[string]any {
	return map[string]any{
		"AdapterId": ad.id, "AdapterName": ad.name, "CreationTime": txEpoch(ad.created), "Description": ad.description,
		"FeatureTypes": ad.features, "AutoUpdate": ad.autoUpdate, "Tags": txTagsOut(ad.tags),
	}
}

func (st *textractStore) getAdapter(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ad, err := st.adapterLocked(txStr(in, "AdapterId"))
	if err != nil {
		return nil, err
	}
	return ad.out(), nil
}

func (st *textractStore) updateAdapter(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ad, err := st.adapterLocked(txStr(in, "AdapterId"))
	if err != nil {
		return nil, err
	}
	if v, ok := in["AdapterName"].(string); ok {
		ad.name = v
	}
	if v, ok := in["Description"].(string); ok {
		ad.description = v
	}
	if v, ok := in["AutoUpdate"].(string); ok {
		ad.autoUpdate = v
	}
	out := ad.out()
	delete(out, "Tags")
	return out, nil
}

func (st *textractStore) deleteAdapter(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ad, err := st.adapterLocked(txStr(in, "AdapterId"))
	if err != nil {
		return nil, err
	}
	for _, v := range ad.versions {
		if v.status == "CREATION_IN_PROGRESS" {
			return nil, txError("ConflictException", "Adapter "+ad.id+" has a version that is still being created")
		}
	}
	delete(st.adapters, ad.id)
	for k, ref := range st.adapterTokens {
		if ref.id == ad.id {
			delete(st.adapterTokens, k)
		}
	}
	return map[string]any{}, nil
}

// txWindow applies After/BeforeCreationTime filters.
func txWindow(in map[string]any, created time.Time) bool {
	if after, ok := txTime(in["AfterCreationTime"]); ok && !created.After(after) {
		return false
	}
	if before, ok := txTime(in["BeforeCreationTime"]); ok && !created.Before(before) {
		return false
	}
	return true
}

func txPage(in map[string]any, items []any, key string) (map[string]any, *txErr) {
	start, err := txPageToken(txStr(in, "NextToken"), len(items))
	if err != nil {
		return nil, err
	}
	size := txDefaultAdapterPage
	if n, ok := in["MaxResults"].(float64); ok && n >= 1 && n < 1000 {
		size = int(n)
	}
	end := min(start+size, len(items))
	out := map[string]any{key: items[start:end]}
	if end < len(items) {
		out["NextToken"] = "tx-" + strconv.Itoa(end)
	}
	return out, nil
}

func (st *textractStore) listAdapters(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var ads []*txAdapter
	for _, ad := range st.adapters {
		if txWindow(in, ad.created) {
			ads = append(ads, ad)
		}
	}
	sort.Slice(ads, func(i, j int) bool {
		if !ads[i].created.Equal(ads[j].created) {
			return ads[i].created.Before(ads[j].created)
		}
		return ads[i].id < ads[j].id
	})
	items := make([]any, 0, len(ads))
	for _, ad := range ads {
		items = append(items, map[string]any{"AdapterId": ad.id, "AdapterName": ad.name, "CreationTime": txEpoch(ad.created), "FeatureTypes": ad.features})
	}
	return txPage(in, items, "Adapters")
}

// ---- adapter versions ----

func (s *Server) txCreateAdapterVersion(in map[string]any) (map[string]any, *txErr) {
	dataset := txMap(in, "DatasetConfig")
	if manifest := txMap(dataset, "ManifestS3Object"); manifest != nil {
		if _, err := s.txS3Bytes(s3CallerIn(in), manifest, txAsyncMaxBytes); err != nil {
			return nil, err
		}
	} else {
		return nil, txInvalid("DatasetConfig needs a ManifestS3Object")
	}
	output := txMap(in, "OutputConfig")
	if _, err := s.s3.GetBucket(txStr(output, "S3Bucket")); err != nil {
		return nil, txError("InvalidS3ObjectException", "OutputConfig bucket "+txStr(output, "S3Bucket")+" does not exist")
	}
	st := s.textract
	st.mu.Lock()
	defer st.mu.Unlock()
	ad, err := st.adapterLocked(txStr(in, "AdapterId"))
	if err != nil {
		return nil, err
	}
	token, hash := txStr(in, "ClientRequestToken"), txParamHash(in)
	if token != "" {
		for _, v := range ad.versions {
			if v.token == token {
				if v.hash != hash {
					return nil, txError("IdempotentParameterMismatchException", "ClientRequestToken was reused with different parameters")
				}
				return map[string]any{"AdapterId": ad.id, "AdapterVersion": v.version}, nil
			}
		}
	}
	if len(ad.versions) >= txMaxVersions {
		return nil, txError("ServiceQuotaExceededException", "An adapter can have at most "+strconv.Itoa(txMaxVersions)+" versions")
	}
	ad.nextVersion++
	v := &txAdapterVersion{
		version: strconv.Itoa(ad.nextVersion), created: time.Now().UTC(), status: "CREATION_IN_PROGRESS",
		dataset: dataset, output: output, kmsKeyID: txStr(in, "KMSKeyId"), tags: map[string]string{}, token: token, hash: hash,
	}
	if err := txTagsIn(txMap(in, "Tags"), v.tags); err != nil {
		return nil, err
	}
	ad.versions[v.version] = v
	return map[string]any{"AdapterId": ad.id, "AdapterVersion": v.version}, nil
}

// settle moves a version out of CREATION_IN_PROGRESS on read.
func (v *txAdapterVersion) settle() {
	if v.status == "CREATION_IN_PROGRESS" {
		v.status = "ACTIVE"
	}
}

func (st *textractStore) versionLocked(in map[string]any) (*txAdapter, *txAdapterVersion, *txErr) {
	ad, err := st.adapterLocked(txStr(in, "AdapterId"))
	if err != nil {
		return nil, nil, err
	}
	v := ad.versions[txStr(in, "AdapterVersion")]
	if v == nil {
		return nil, nil, txError("ResourceNotFoundException", "Adapter "+ad.id+" has no version "+txStr(in, "AdapterVersion"))
	}
	return ad, v, nil
}

func (st *textractStore) getAdapterVersion(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ad, v, err := st.versionLocked(in)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"AdapterId": ad.id, "AdapterVersion": v.version, "CreationTime": txEpoch(v.created), "FeatureTypes": ad.features,
		"Status": v.status, "DatasetConfig": v.dataset, "OutputConfig": v.output, "Tags": txTagsOut(v.tags),
	}
	if v.kmsKeyID != "" {
		out["KMSKeyId"] = v.kmsKeyID
	}
	if v.status == "ACTIVE" {
		// ponytail: fixed evaluation scores; no training runs here.
		var metrics []any
		for _, f := range ad.features {
			metrics = append(metrics, map[string]any{
				"FeatureType":    f,
				"Baseline":       map[string]any{"F1Score": 0.8, "Precision": 0.8, "Recall": 0.8},
				"AdapterVersion": map[string]any{"F1Score": 0.9, "Precision": 0.9, "Recall": 0.9},
			})
		}
		out["EvaluationMetrics"] = metrics
	}
	v.settle()
	return out, nil
}

func (st *textractStore) deleteAdapterVersion(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ad, v, err := st.versionLocked(in)
	if err != nil {
		return nil, err
	}
	delete(ad.versions, v.version)
	return map[string]any{}, nil
}

func (st *textractStore) listAdapterVersions(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var ads []*txAdapter
	if id := txStr(in, "AdapterId"); id != "" {
		ad, err := st.adapterLocked(id)
		if err != nil {
			return nil, err
		}
		ads = append(ads, ad)
	} else {
		for _, ad := range st.adapters {
			ads = append(ads, ad)
		}
	}
	type entry struct {
		ad *txAdapter
		v  *txAdapterVersion
	}
	var all []entry
	for _, ad := range ads {
		for _, v := range ad.versions {
			if txWindow(in, v.created) {
				all = append(all, entry{ad, v})
			}
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].ad.id != all[j].ad.id {
			return all[i].ad.id < all[j].ad.id
		}
		a, _ := strconv.Atoi(all[i].v.version)
		b, _ := strconv.Atoi(all[j].v.version)
		return a < b
	})
	items := make([]any, 0, len(all))
	for _, e := range all {
		items = append(items, map[string]any{
			"AdapterId": e.ad.id, "AdapterVersion": e.v.version, "CreationTime": txEpoch(e.v.created),
			"FeatureTypes": e.ad.features, "Status": e.v.status,
		})
		e.v.settle()
	}
	return txPage(in, items, "AdapterVersions")
}

// checkAdapters makes sure every adapter version an analysis names is ACTIVE.
func (st *textractStore) checkAdapters(in map[string]any) *txErr {
	list, _ := txMap(in, "AdaptersConfig")["Adapters"].([]any)
	if len(list) == 0 {
		return nil
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, raw := range list {
		a, _ := raw.(map[string]any)
		ad := st.adapters[txStr(a, "AdapterId")]
		if ad == nil {
			return txInvalid("Adapter %s does not exist", txStr(a, "AdapterId"))
		}
		v := ad.versions[txStr(a, "Version")]
		if v == nil || v.status != "ACTIVE" {
			return txInvalid("Adapter %s version %s is not ACTIVE", ad.id, txStr(a, "Version"))
		}
	}
	return nil
}

// ---- tags ----

// tagsLocked returns the tag map for an adapter or adapter version ARN.
func (st *textractStore) tagsLocked(arn string) (map[string]string, *txErr) {
	m := txARNRe.FindStringSubmatch(arn)
	if m == nil {
		return nil, txError("ResourceNotFoundException", "Resource "+arn+" does not exist")
	}
	ad := st.adapters[m[1]]
	if ad == nil {
		return nil, txError("ResourceNotFoundException", "Resource "+arn+" does not exist")
	}
	if m[2] == "" {
		return ad.tags, nil
	}
	v := ad.versions[m[2]]
	if v == nil {
		return nil, txError("ResourceNotFoundException", "Resource "+arn+" does not exist")
	}
	return v.tags, nil
}

func (st *textractStore) tagResource(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagsLocked(txStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	next := make(map[string]string, len(tags))
	for k, v := range tags {
		next[k] = v
	}
	if err := txTagsIn(txMap(in, "Tags"), next); err != nil {
		return nil, err
	}
	for k, v := range next {
		tags[k] = v
	}
	return map[string]any{}, nil
}

func (st *textractStore) untagResource(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagsLocked(txStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	keys, _ := in["TagKeys"].([]any)
	for _, k := range keys {
		if s, ok := k.(string); ok {
			delete(tags, s)
		}
	}
	return map[string]any{}, nil
}

func (st *textractStore) listTags(in map[string]any) (map[string]any, *txErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagsLocked(txStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Tags": txTagsOut(tags)}, nil
}
