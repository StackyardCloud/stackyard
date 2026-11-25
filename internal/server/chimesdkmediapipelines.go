package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Chime SDK Media Pipelines: REST JSON signed chime, served from
// media-pipelines-chime.<region>.amazonaws.com. Capture, live connector
// and stream pipelines read meetings created through Chime SDK Meetings
// (the meeting must exist; a pipeline stops once its meeting is deleted),
// concatenation pipelines read capture pipelines, and media insights
// pipelines run a media insights pipeline configuration. Pipelines,
// Kinesis video stream pools and speaker search / voice tone analysis
// tasks settle asynchronously (Initializing -> InProgress, CREATING ->
// ACTIVE, Stopping -> Stopped) when read. Artifacts aren't written to S3.

const (
	cmplMaxPipelines = 500
	cmplMaxConfigs   = 100
	cmplMaxPools     = 100
	cmplMaxTasks     = 20
	cmplMaxSources   = 10
	cmplMaxMetadata  = 20
	cmplMaxTags      = 200
	cmplMaxBytes     = 8 << 20
	cmplSettle       = 2 * time.Second
	cmplARNBase      = "arn:aws:chime:" + chimeSDKRegion + ":" + chimeSDKAccount + ":"
)

// cmplKinds maps a pipeline kind to its response member.
var cmplKinds = map[string]string{
	"capture": "MediaCapturePipeline", "concatenation": "MediaConcatenationPipeline", "live": "MediaLiveConnectorPipeline",
	"insights": "MediaInsightsPipeline", "stream": "MediaStreamPipeline",
}

type cmplTask struct {
	id, kind, status, tokenKey string
	created, updated           time.Time
}

type cmplPipeline struct {
	id, arn, kind, status, tokenKey, configName string
	doc                                         map[string]any
	meetings                                    []string
	pools                                       map[string]int
	created, updated                            time.Time
	tags                                        map[string]string
	tasks                                       map[string]*cmplTask
	size                                        int
}

type cmplConfig struct {
	id, name, arn, tokenKey string
	pipelines               map[string]struct{}
	doc                     map[string]any
	created, updated        time.Time
	tags                    map[string]string
	size                    int
}

type cmplPool struct {
	id, name, arn, status, tokenKey string
	streamConfig                    map[string]any
	created, updated                time.Time
	tags                            map[string]string
	refs                            map[string]int
}

type cmplStore struct {
	mu        sync.Mutex
	pipelines map[string]*cmplPipeline
	configs   map[string]*cmplConfig
	configIDs map[string]string
	pools     map[string]*cmplPool
	poolIDs   map[string]string
	tokens    map[string]string
	bytes     int
	settle    time.Duration
	now       func() time.Time
}

func newChimeSDKMediaPipelinesStore() *cmplStore {
	return &cmplStore{
		pipelines: map[string]*cmplPipeline{}, configs: map[string]*cmplConfig{}, configIDs: map[string]string{},
		pools: map[string]*cmplPool{}, poolIDs: map[string]string{}, tokens: map[string]string{}, settle: cmplSettle, now: time.Now,
	}
}

// cmplHandler runs with the store lock held. alive reports, for the
// meetings the request or its pipeline reference, whether they exist.
type cmplHandler func(st *cmplStore, in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr)

func (s *Server) handleChimeSDKMediaPipelinesRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	op, params, ok := chimeSDKMatch(r, "media-pipelines-chime.", chimeSDKMediaPipelinesOperations)
	if !ok || (chimeSDKIsTagOp(op) && chimeSDKTagOwner(r) != "pipelines") {
		return false
	}
	model := awsmodels.ChimeSDKMediaPipelinesModel()
	in, ok := s.chimeSDKDecode(w, r, model, op, params)
	if !ok {
		return true
	}
	st := s.chimeSDKPipelines
	// Look meetings up before taking the pipelines lock: this store never
	// holds its own lock while calling into the meetings store.
	refs := cmplMeetingRefs(in)
	if id := mmgrStr(in, "MediaPipelineId"); id != "" {
		st.mu.Lock()
		if p := st.pipelines[id]; p != nil {
			refs = append(refs, p.meetings...)
		}
		st.mu.Unlock()
	}
	alive := make(map[string]bool, len(refs))
	for _, arn := range refs {
		alive[arn] = s.chimeSDKMeetings.meetingExists(arn)
	}
	st.mu.Lock()
	out, cerr := cmplHandlers[op.Name](st, in, alive)
	var body []byte
	if cerr == nil && op.Code != http.StatusNoContent {
		if out == nil {
			out = map[string]any{}
		}
		body, _ = json.Marshal(model.Conform(model.Operations[op.Name].Output, out))
	}
	st.mu.Unlock()
	if cerr != nil {
		chimeSDKWriteError(w, cerr)
		return true
	}
	status := op.Code
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	return true
}

var cmplHandlers = map[string]cmplHandler{
	"CreateMediaCapturePipeline":                (*cmplStore).createCapture,
	"CreateMediaConcatenationPipeline":          (*cmplStore).createConcatenation,
	"CreateMediaLiveConnectorPipeline":          (*cmplStore).createLiveConnector,
	"CreateMediaStreamPipeline":                 (*cmplStore).createStream,
	"CreateMediaInsightsPipeline":               (*cmplStore).createInsights,
	"GetMediaCapturePipeline":                   (*cmplStore).getCapture,
	"GetMediaPipeline":                          (*cmplStore).getPipeline,
	"DeleteMediaCapturePipeline":                (*cmplStore).deleteCapture,
	"DeleteMediaPipeline":                       (*cmplStore).deletePipeline,
	"ListMediaCapturePipelines":                 (*cmplStore).listCapture,
	"ListMediaPipelines":                        (*cmplStore).listPipelines,
	"UpdateMediaInsightsPipelineStatus":         (*cmplStore).updateInsightsStatus,
	"CreateMediaInsightsPipelineConfiguration":  (*cmplStore).createConfig,
	"GetMediaInsightsPipelineConfiguration":     (*cmplStore).getConfig,
	"UpdateMediaInsightsPipelineConfiguration":  (*cmplStore).updateConfig,
	"DeleteMediaInsightsPipelineConfiguration":  (*cmplStore).deleteConfig,
	"ListMediaInsightsPipelineConfigurations":   (*cmplStore).listConfigs,
	"CreateMediaPipelineKinesisVideoStreamPool": (*cmplStore).createPool,
	"GetMediaPipelineKinesisVideoStreamPool":    (*cmplStore).getPool,
	"UpdateMediaPipelineKinesisVideoStreamPool": (*cmplStore).updatePool,
	"DeleteMediaPipelineKinesisVideoStreamPool": (*cmplStore).deletePool,
	"ListMediaPipelineKinesisVideoStreamPools":  (*cmplStore).listPools,
	"StartSpeakerSearchTask":                    (*cmplStore).startSpeakerSearch,
	"GetSpeakerSearchTask":                      (*cmplStore).getSpeakerSearch,
	"StopSpeakerSearchTask":                     (*cmplStore).stopSpeakerSearch,
	"StartVoiceToneAnalysisTask":                (*cmplStore).startVoiceTone,
	"GetVoiceToneAnalysisTask":                  (*cmplStore).getVoiceTone,
	"StopVoiceToneAnalysisTask":                 (*cmplStore).stopVoiceTone,
	"TagResource":                               (*cmplStore).tagResource,
	"UntagResource":                             (*cmplStore).untagResource,
	"ListTagsForResource":                       (*cmplStore).listTags,
}

// cmplMeetingRefs returns the meeting ARNs a create request reads.
func cmplMeetingRefs(in map[string]any) []string {
	var out []string
	if arn := mmgrStr(in, "SourceArn"); arn != "" {
		out = append(out, arn)
	}
	for i, raw := range mmgrList(in, "Sources") {
		if i >= cmplMaxSources {
			break
		}
		src := chimeSDKMapOf(raw)
		for _, arn := range []string{mmgrStr(src, "SourceArn"), mmgrStr(mmgrMap(src, "ChimeSdkMeetingLiveConnectorConfiguration"), "Arn")} {
			if arn != "" {
				out = append(out, arn)
			}
		}
	}
	return out
}

// ---- pipelines ----

// refresh settles p: Initializing becomes InProgress once settled, and a
// running pipeline whose meeting is gone stops.
func (st *cmplStore) refresh(p *cmplPipeline, alive map[string]bool) {
	now := st.now()
	if p.status == "Initializing" && !now.Before(p.created.Add(st.settle)) {
		p.status, p.updated = "InProgress", p.created.Add(st.settle)
	}
	for _, m := range p.meetings {
		if ok, known := alive[m]; known && !ok && p.status != "Stopped" && p.status != "Failed" {
			p.status, p.updated = "Stopped", now
		}
	}
}

func (st *cmplStore) view(p *cmplPipeline) map[string]any {
	out := chimeSDKCopy(p.doc)
	out["MediaPipelineId"], out["MediaPipelineArn"], out["Status"] = p.id, p.arn, p.status
	out["CreatedTimestamp"], out["UpdatedTimestamp"] = chimeSDKTime(p.created), chimeSDKTime(p.updated)
	if p.kind == "insights" {
		var statuses []any
		if c := st.configs[p.configName]; c != nil {
			for _, raw := range mmgrList(c.doc, "Elements") {
				statuses = append(statuses, map[string]any{"Type": mmgrStr(chimeSDKMapOf(raw), "Type"), "Status": p.status})
			}
		}
		out["ElementStatuses"] = statuses
	}
	return out
}

func cmplDocSize(doc map[string]any) int {
	raw, _ := json.Marshal(doc)
	return len(raw)
}

// newPipeline stores a pipeline of kind built from the request members it
// echoes, honouring ClientRequestToken idempotency and the store caps.
func (st *cmplStore) newPipeline(kind string, in map[string]any, meetings []string, alive map[string]bool) (*cmplPipeline, bool, *chimeSDKErr) {
	tokenKey := ""
	if token := mmgrStr(in, "ClientRequestToken"); token != "" {
		tokenKey = kind + "|" + token
		if p := st.pipelines[st.tokens[tokenKey]]; p != nil {
			return p, true, nil
		}
	}
	for _, m := range meetings {
		if !alive[m] {
			return nil, false, chimeSDKNotFound("The meeting %s doesn't exist.", m)
		}
	}
	doc := chimeSDKCopy(in)
	delete(doc, "Tags")
	delete(doc, "ClientRequestToken")
	size := cmplDocSize(doc)
	if len(st.pipelines) >= cmplMaxPipelines || st.bytes+size > cmplMaxBytes {
		return nil, false, chimeSDKLimit("The account can have at most %d media pipelines.", cmplMaxPipelines)
	}
	tags, cerr := chimeSDKMergeTags(nil, mmgrList(in, "Tags"), cmplMaxTags, cmplTooManyTags)
	if cerr != nil {
		return nil, false, cerr
	}
	now, id := st.now(), uuid.NewString()
	p := &cmplPipeline{
		id: id, arn: cmplARNBase + "media-pipeline/" + id, kind: kind, status: "Initializing", tokenKey: tokenKey,
		doc: doc, meetings: meetings, pools: map[string]int{}, created: now, updated: now, tags: tags,
		tasks: map[string]*cmplTask{}, size: size,
	}
	st.pipelines[id] = p
	st.bytes += size
	if tokenKey != "" {
		st.tokens[tokenKey] = id
	}
	return p, false, nil
}

func (st *cmplStore) created(kind string, p *cmplPipeline) map[string]any {
	return map[string]any{cmplKinds[kind]: st.view(p)}
}

func cmplS3(arn string) *chimeSDKErr {
	if !strings.HasPrefix(arn, "arn:aws:s3:::") || len(arn) == len("arn:aws:s3:::") {
		return chimeSDKBadRequest("%s is an invalid S3 bucket ARN.", arn)
	}
	return nil
}

func (st *cmplStore) createCapture(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	if cerr := cmplS3(mmgrStr(in, "SinkArn")); cerr != nil {
		return nil, cerr
	}
	if mmgrMap(in, "SseAwsKeyManagementParams") != nil && mmgrStr(in, "SinkIamRoleArn") == "" {
		return nil, chimeSDKBadRequest("SseAwsKeyManagementParams needs a SinkIamRoleArn.")
	}
	p, _, cerr := st.newPipeline("capture", in, []string{mmgrStr(in, "SourceArn")}, alive)
	if cerr != nil {
		return nil, cerr
	}
	return st.created("capture", p), nil
}

func (st *cmplStore) createConcatenation(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	for _, raw := range mmgrList(in, "Sources") {
		arn := mmgrStr(mmgrMap(chimeSDKMapOf(raw), "MediaCapturePipelineSourceConfiguration"), "MediaPipelineArn")
		src := st.pipelines[arn[strings.LastIndex(arn, "/")+1:]]
		if src == nil || src.arn != arn || src.kind != "capture" {
			return nil, chimeSDKNotFound("The media capture pipeline %s doesn't exist.", arn)
		}
	}
	for _, raw := range mmgrList(in, "Sinks") {
		if cerr := cmplS3(mmgrStr(mmgrMap(chimeSDKMapOf(raw), "S3BucketSinkConfiguration"), "Destination")); cerr != nil {
			return nil, cerr
		}
	}
	p, _, cerr := st.newPipeline("concatenation", in, nil, alive)
	if cerr != nil {
		return nil, cerr
	}
	return st.created("concatenation", p), nil
}

func (st *cmplStore) createLiveConnector(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	var meetings []string
	for _, raw := range mmgrList(in, "Sources") {
		cfg := mmgrMap(chimeSDKMapOf(raw), "ChimeSdkMeetingLiveConnectorConfiguration")
		if mmgrStr(cfg, "MuxType") == "AudioWithCompositedVideo" && mmgrMap(cfg, "CompositedVideo") == nil {
			return nil, chimeSDKBadRequest("MuxType AudioWithCompositedVideo needs a CompositedVideo configuration.")
		}
		meetings = append(meetings, mmgrStr(cfg, "Arn"))
	}
	for _, raw := range mmgrList(in, "Sinks") {
		url := mmgrStr(mmgrMap(chimeSDKMapOf(raw), "RTMPConfiguration"), "Url")
		if !strings.HasPrefix(url, "rtmp://") && !strings.HasPrefix(url, "rtmps://") {
			return nil, chimeSDKBadRequest("RTMP sink URL %s must use rtmp:// or rtmps://.", url)
		}
	}
	p, _, cerr := st.newPipeline("live", in, meetings, alive)
	if cerr != nil {
		return nil, cerr
	}
	return st.created("live", p), nil
}

// poolByIdentifier resolves a pool name, ID or ARN.
func (st *cmplStore) poolByIdentifier(ident string) *cmplPool {
	if strings.HasPrefix(ident, "arn:") {
		p := st.pools[ident[strings.LastIndex(ident, "/")+1:]]
		if p != nil && p.arn == ident {
			return p
		}
		return nil
	}
	if p := st.pools[ident]; p != nil {
		return p
	}
	return st.pools[st.poolIDs[ident]]
}

func (st *cmplStore) createStream(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	sources := mmgrList(in, "Sources")
	if len(sources) > cmplMaxSources {
		return nil, chimeSDKBadRequest("A media stream pipeline can have at most %d sources.", cmplMaxSources)
	}
	var meetings []string
	for _, raw := range sources {
		meetings = append(meetings, mmgrStr(chimeSDKMapOf(raw), "SourceArn"))
	}
	reserve := map[string]int{}
	for _, raw := range mmgrList(in, "Sinks") {
		sink := chimeSDKMapOf(raw)
		pool := st.poolByIdentifier(mmgrStr(sink, "SinkArn"))
		if pool == nil || !strings.HasPrefix(mmgrStr(sink, "SinkArn"), "arn:") {
			return nil, chimeSDKNotFound("The Kinesis video stream pool %s doesn't exist.", mmgrStr(sink, "SinkArn"))
		}
		if st.poolStatus(pool) == "DELETING" || st.poolStatus(pool) == "FAILED" {
			return nil, chimeSDKConflict("The Kinesis video stream pool %s isn't available.", pool.name)
		}
		reserve[pool.name] += mmgrInt(sink, "ReservedStreamCapacity", 1)
	}
	p, existing, cerr := st.newPipeline("stream", in, meetings, alive)
	if cerr != nil {
		return nil, cerr
	}
	if !existing {
		for name, n := range reserve {
			st.pools[name].refs[p.id] = n
			p.pools[name] = n
		}
	}
	return st.created("stream", p), nil
}

func (st *cmplStore) createInsights(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	arn := mmgrStr(in, "MediaInsightsPipelineConfigurationArn")
	c := st.configByIdentifier(arn)
	if c == nil || c.arn != arn {
		return nil, chimeSDKNotFound("The media insights pipeline configuration %s doesn't exist.", arn)
	}
	live, recording := mmgrMap(in, "KinesisVideoStreamSourceRuntimeConfiguration"), mmgrMap(in, "KinesisVideoStreamRecordingSourceRuntimeConfiguration")
	if (live == nil) == (recording == nil) {
		return nil, chimeSDKBadRequest("Specify exactly one of KinesisVideoStreamSourceRuntimeConfiguration or KinesisVideoStreamRecordingSourceRuntimeConfiguration.")
	}
	if recording != nil && mmgrMap(in, "S3RecordingSinkRuntimeConfiguration") == nil && !c.hasElement("S3RecordingSink") {
		return nil, chimeSDKBadRequest("Recording sources need an S3 recording sink.")
	}
	if len(mmgrMap(in, "MediaInsightsRuntimeMetadata")) > cmplMaxMetadata {
		return nil, chimeSDKBadRequest("MediaInsightsRuntimeMetadata can have at most %d entries.", cmplMaxMetadata)
	}
	p, _, cerr := st.newPipeline("insights", in, nil, alive)
	if cerr != nil {
		return nil, cerr
	}
	p.configName = c.name
	c.pipelines[p.id] = struct{}{}
	return st.created("insights", p), nil
}

func (st *cmplStore) pipeline(id string, alive map[string]bool) (*cmplPipeline, *chimeSDKErr) {
	p := st.pipelines[id]
	if p == nil {
		return nil, chimeSDKNotFound("The media pipeline %s doesn't exist.", id)
	}
	st.refresh(p, alive)
	return p, nil
}

func (st *cmplStore) getCapture(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	p, cerr := st.pipeline(mmgrStr(in, "MediaPipelineId"), alive)
	if cerr != nil || p.kind != "capture" {
		return nil, chimeSDKNotFound("The media capture pipeline %s doesn't exist.", mmgrStr(in, "MediaPipelineId"))
	}
	return map[string]any{"MediaCapturePipeline": st.view(p)}, nil
}

func (st *cmplStore) getPipeline(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	p, cerr := st.pipeline(mmgrStr(in, "MediaPipelineId"), alive)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"MediaPipeline": st.created(p.kind, p)}, nil
}

func (st *cmplStore) removePipeline(p *cmplPipeline) {
	for name := range p.pools {
		if pool := st.pools[name]; pool != nil {
			delete(pool.refs, p.id)
		}
	}
	for _, t := range p.tasks {
		delete(st.tokens, t.tokenKey)
	}
	if c := st.configs[p.configName]; c != nil {
		delete(c.pipelines, p.id)
	}
	delete(st.tokens, p.tokenKey)
	delete(st.pipelines, p.id)
	st.bytes -= p.size
}

func (st *cmplStore) deleteCapture(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	p := st.pipelines[mmgrStr(in, "MediaPipelineId")]
	if p == nil || p.kind != "capture" {
		return nil, chimeSDKNotFound("The media capture pipeline %s doesn't exist.", mmgrStr(in, "MediaPipelineId"))
	}
	st.removePipeline(p)
	return nil, nil
}

func (st *cmplStore) deletePipeline(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	p, cerr := st.pipeline(mmgrStr(in, "MediaPipelineId"), alive)
	if cerr != nil {
		return nil, cerr
	}
	st.removePipeline(p)
	return nil, nil
}

func (st *cmplStore) listKind(in map[string]any, kind, key string) map[string]any {
	var ids []string
	for id, p := range st.pipelines {
		if kind == "" || p.kind == kind {
			ids = append(ids, id)
		}
	}
	page, next := chimeSDKPage(ids, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 100)
	out := make([]any, 0, len(page))
	for _, id := range page {
		out = append(out, map[string]any{"MediaPipelineId": id, "MediaPipelineArn": st.pipelines[id].arn})
	}
	resp := map[string]any{key: out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp
}

func (st *cmplStore) listCapture(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	return st.listKind(in, "capture", "MediaCapturePipelines"), nil
}

func (st *cmplStore) listPipelines(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	return st.listKind(in, "", "MediaPipelines"), nil
}

// insights resolves a media insights pipeline by ID or ARN.
func (st *cmplStore) insights(ident string, alive map[string]bool) (*cmplPipeline, *chimeSDKErr) {
	p := st.pipelines[ident[strings.LastIndex(ident, "/")+1:]]
	if p == nil || p.kind != "insights" || (strings.HasPrefix(ident, "arn:") && p.arn != ident) {
		return nil, chimeSDKNotFound("The media insights pipeline %s doesn't exist.", ident)
	}
	st.refresh(p, alive)
	return p, nil
}

func (st *cmplStore) updateInsightsStatus(in map[string]any, alive map[string]bool) (map[string]any, *chimeSDKErr) {
	p, cerr := st.insights(mmgrStr(in, "Identifier"), alive)
	if cerr != nil {
		return nil, cerr
	}
	switch update := mmgrStr(in, "UpdateStatus"); {
	case update == "Pause" && (p.status == "Initializing" || p.status == "InProgress"):
		p.status = "Paused"
	case update == "Resume" && p.status == "Paused":
		p.status = "InProgress"
	default:
		return nil, chimeSDKConflict("A %s media insights pipeline can't %s.", p.status, strings.ToLower(update))
	}
	p.updated = st.now()
	return nil, nil
}

// ---- tags ----

func cmplTooManyTags() *chimeSDKErr {
	return chimeSDKLimit("A resource can have at most %d tags.", cmplMaxTags)
}

func (st *cmplStore) tagged(in map[string]any) (*map[string]string, *chimeSDKErr) {
	arn := mmgrStr(in, "ResourceARN")
	name := arn[strings.LastIndex(arn, "/")+1:]
	switch {
	case strings.Contains(arn, ":media-pipeline/"):
		if p := st.pipelines[name]; p != nil && p.arn == arn {
			return &p.tags, nil
		}
	case strings.Contains(arn, ":media-insights-pipeline-configuration/"):
		if c := st.configs[name]; c != nil && c.arn == arn {
			return &c.tags, nil
		}
	case strings.Contains(arn, ":media-pipeline-kinesis-video-stream-pool/"):
		if p := st.pools[name]; p != nil && p.arn == arn {
			return &p.tags, nil
		}
	}
	return nil, chimeSDKNotFound("The resource %s doesn't exist.", arn)
}

func (st *cmplStore) tagResource(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	tags, cerr := st.tagged(in)
	if cerr != nil {
		return nil, cerr
	}
	merged, cerr := chimeSDKMergeTags(*tags, mmgrList(in, "Tags"), cmplMaxTags, cmplTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	*tags = merged
	return nil, nil
}

func (st *cmplStore) untagResource(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	tags, cerr := st.tagged(in)
	if cerr != nil {
		return nil, cerr
	}
	*tags = chimeSDKUntag(*tags, mmgrList(in, "TagKeys"))
	return nil, nil
}

func (st *cmplStore) listTags(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	tags, cerr := st.tagged(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"Tags": chimeSDKTagList(*tags)}, nil
}
