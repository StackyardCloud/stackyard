package server

import (
	"strings"

	"github.com/google/uuid"
)

// Media insights pipeline configurations, Kinesis video stream pools and
// speaker search / voice tone analysis tasks.

// cmplCheckTyped requires each item of list to carry the configuration
// member its Type names (Type "S3RecordingSink" needs
// "S3RecordingSinkConfiguration").
func cmplCheckTyped(list []any, what string) *chimeSDKErr {
	for _, raw := range list {
		item := chimeSDKMapOf(raw)
		typ := mmgrStr(item, "Type")
		if mmgrMap(item, typ+"Configuration") == nil {
			return chimeSDKBadRequest("%s of type %s needs a %sConfiguration.", what, typ, typ)
		}
	}
	return nil
}

func cmplCheckConfig(in map[string]any) *chimeSDKErr {
	if !strings.HasPrefix(mmgrStr(in, "ResourceAccessRoleArn"), "arn:aws:iam::") {
		return chimeSDKBadRequest("ResourceAccessRoleArn must be an IAM role ARN.")
	}
	elements := mmgrList(in, "Elements")
	if len(elements) == 0 {
		return chimeSDKBadRequest("A media insights pipeline configuration needs at least one element.")
	}
	if cerr := cmplCheckTyped(elements, "Element"); cerr != nil {
		return cerr
	}
	return cmplCheckTyped(mmgrList(mmgrMap(in, "RealTimeAlertConfiguration"), "Rules"), "Rule")
}

func (c *cmplConfig) hasElement(typ string) bool {
	for _, raw := range mmgrList(c.doc, "Elements") {
		if mmgrStr(chimeSDKMapOf(raw), "Type") == typ {
			return true
		}
	}
	return false
}

func (c *cmplConfig) view() map[string]any {
	out := chimeSDKCopy(c.doc)
	out["MediaInsightsPipelineConfigurationArn"], out["MediaInsightsPipelineConfigurationId"] = c.arn, c.id
	out["CreatedTimestamp"], out["UpdatedTimestamp"] = chimeSDKTime(c.created), chimeSDKTime(c.updated)
	return out
}

// configByIdentifier resolves a configuration name, ID or ARN.
func (st *cmplStore) configByIdentifier(ident string) *cmplConfig {
	if strings.HasPrefix(ident, "arn:") {
		c := st.configs[ident[strings.LastIndex(ident, "/")+1:]]
		if c != nil && c.arn == ident {
			return c
		}
		return nil
	}
	if c := st.configs[ident]; c != nil {
		return c
	}
	return st.configs[st.configIDs[ident]]
}

func (st *cmplStore) config(in map[string]any) (*cmplConfig, *chimeSDKErr) {
	c := st.configByIdentifier(mmgrStr(in, "Identifier"))
	if c == nil {
		return nil, chimeSDKNotFound("The media insights pipeline configuration %s doesn't exist.", mmgrStr(in, "Identifier"))
	}
	return c, nil
}

func cmplConfigDoc(in map[string]any) map[string]any {
	doc := map[string]any{"ResourceAccessRoleArn": in["ResourceAccessRoleArn"], "Elements": in["Elements"]}
	if v := mmgrMap(in, "RealTimeAlertConfiguration"); v != nil {
		doc["RealTimeAlertConfiguration"] = v
	}
	return doc
}

func (st *cmplStore) createConfig(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	name := mmgrStr(in, "MediaInsightsPipelineConfigurationName")
	tokenKey := "config|" + mmgrStr(in, "ClientRequestToken")
	if c := st.configs[name]; c != nil {
		if mmgrStr(in, "ClientRequestToken") != "" && st.tokens[tokenKey] == name {
			return map[string]any{"MediaInsightsPipelineConfiguration": c.view()}, nil
		}
		return nil, chimeSDKConflict("The media insights pipeline configuration %s already exists.", name)
	}
	if cerr := cmplCheckConfig(in); cerr != nil {
		return nil, cerr
	}
	doc := cmplConfigDoc(in)
	doc["MediaInsightsPipelineConfigurationName"] = name
	size := cmplDocSize(doc)
	if len(st.configs) >= cmplMaxConfigs || st.bytes+size > cmplMaxBytes {
		return nil, chimeSDKLimit("The account can have at most %d media insights pipeline configurations.", cmplMaxConfigs)
	}
	tags, cerr := chimeSDKMergeTags(nil, mmgrList(in, "Tags"), cmplMaxTags, cmplTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	now := st.now()
	c := &cmplConfig{
		id: uuid.NewString(), name: name, arn: cmplARNBase + "media-insights-pipeline-configuration/" + name,
		doc: doc, created: now, updated: now, tags: tags, size: size, pipelines: map[string]struct{}{},
	}
	st.configs[name] = c
	st.configIDs[c.id] = name
	st.bytes += size
	if mmgrStr(in, "ClientRequestToken") != "" {
		st.tokens[tokenKey] = name
		c.tokenKey = tokenKey
	}
	return map[string]any{"MediaInsightsPipelineConfiguration": c.view()}, nil
}

func (st *cmplStore) getConfig(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	c, cerr := st.config(in)
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"MediaInsightsPipelineConfiguration": c.view()}, nil
}

func (st *cmplStore) updateConfig(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	c, cerr := st.config(in)
	if cerr != nil {
		return nil, cerr
	}
	if cerr := cmplCheckConfig(in); cerr != nil {
		return nil, cerr
	}
	doc := cmplConfigDoc(in)
	doc["MediaInsightsPipelineConfigurationName"] = c.name
	size := cmplDocSize(doc)
	if st.bytes-c.size+size > cmplMaxBytes {
		return nil, chimeSDKLimit("Stored media pipeline configurations are limited to %d bytes.", cmplMaxBytes)
	}
	st.bytes += size - c.size
	c.doc, c.size, c.updated = doc, size, st.now()
	return map[string]any{"MediaInsightsPipelineConfiguration": c.view()}, nil
}

// inUse reports whether a running media insights pipeline uses c.
func (st *cmplStore) inUse(c *cmplConfig) bool {
	for id := range c.pipelines {
		p := st.pipelines[id]
		if p == nil { // defensive: drop a stale reference rather than panic under the lock
			delete(c.pipelines, id)
			continue
		}
		st.refresh(p, nil)
		if p.status != "Stopped" && p.status != "Failed" {
			return true
		}
	}
	return false
}

func (st *cmplStore) deleteConfig(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	c, cerr := st.config(in)
	if cerr != nil {
		return nil, cerr
	}
	if st.inUse(c) {
		return nil, chimeSDKConflict("Media insights pipeline configuration %s is in use.", c.name)
	}
	delete(st.configs, c.name)
	delete(st.configIDs, c.id)
	delete(st.tokens, c.tokenKey)
	st.bytes -= c.size
	return nil, nil
}

func (st *cmplStore) listConfigs(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	names := make([]string, 0, len(st.configs))
	for name := range st.configs {
		names = append(names, name)
	}
	page, next := chimeSDKPage(names, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 100)
	out := make([]any, 0, len(page))
	for _, name := range page {
		c := st.configs[name]
		out = append(out, map[string]any{"MediaInsightsPipelineConfigurationName": c.name, "MediaInsightsPipelineConfigurationId": c.id, "MediaInsightsPipelineConfigurationArn": c.arn})
	}
	resp := map[string]any{"MediaInsightsPipelineConfigurations": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

// ---- Kinesis video stream pools ----

// poolStatus settles CREATING and UPDATING pools to ACTIVE.
func (st *cmplStore) poolStatus(p *cmplPool) string {
	if (p.status == "CREATING" || p.status == "UPDATING") && !st.now().Before(p.updated.Add(st.settle)) {
		p.status = "ACTIVE"
	}
	return p.status
}

func (st *cmplStore) poolView(p *cmplPool) map[string]any {
	size := 0
	for _, n := range p.refs {
		size += n
	}
	return map[string]any{"KinesisVideoStreamPoolConfiguration": map[string]any{
		"PoolArn": p.arn, "PoolName": p.name, "PoolId": p.id, "PoolStatus": st.poolStatus(p), "PoolSize": size,
		"StreamConfiguration": chimeSDKCopy(p.streamConfig), "CreatedTimestamp": chimeSDKTime(p.created), "UpdatedTimestamp": chimeSDKTime(p.updated),
	}}
}

func (st *cmplStore) pool(in map[string]any) (*cmplPool, *chimeSDKErr) {
	p := st.poolByIdentifier(mmgrStr(in, "Identifier"))
	if p == nil {
		return nil, chimeSDKNotFound("The Kinesis video stream pool %s doesn't exist.", mmgrStr(in, "Identifier"))
	}
	return p, nil
}

func (st *cmplStore) createPool(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	name := mmgrStr(in, "PoolName")
	if p := st.pools[name]; p != nil {
		if token := mmgrStr(in, "ClientRequestToken"); token != "" && st.tokens["pool|"+token] == name {
			return st.poolView(p), nil
		}
		return nil, chimeSDKConflict("The Kinesis video stream pool %s already exists.", name)
	}
	if len(st.pools) >= cmplMaxPools {
		return nil, chimeSDKLimit("The account can have at most %d Kinesis video stream pools.", cmplMaxPools)
	}
	tags, cerr := chimeSDKMergeTags(nil, mmgrList(in, "Tags"), cmplMaxTags, cmplTooManyTags)
	if cerr != nil {
		return nil, cerr
	}
	now := st.now()
	p := &cmplPool{
		id: "ChimeMediaPipelines-" + name + "-" + uuid.NewString()[:8], name: name,
		arn: cmplARNBase + "media-pipeline-kinesis-video-stream-pool/" + name, status: "CREATING",
		streamConfig: mmgrMap(in, "StreamConfiguration"), created: now, updated: now, tags: tags, refs: map[string]int{},
	}
	st.pools[name] = p
	st.poolIDs[p.id] = name
	if token := mmgrStr(in, "ClientRequestToken"); token != "" {
		p.tokenKey = "pool|" + token
		st.tokens[p.tokenKey] = name
	}
	return st.poolView(p), nil
}

func (st *cmplStore) getPool(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	p, cerr := st.pool(in)
	if cerr != nil {
		return nil, cerr
	}
	return st.poolView(p), nil
}

func (st *cmplStore) updatePool(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	p, cerr := st.pool(in)
	if cerr != nil {
		return nil, cerr
	}
	if st.poolStatus(p) != "ACTIVE" && p.status != "UPDATING" {
		return nil, chimeSDKConflict("A %s Kinesis video stream pool can't be updated.", p.status)
	}
	if update := mmgrMap(in, "StreamConfiguration"); update != nil {
		cfg := chimeSDKCopy(p.streamConfig)
		cfg["DataRetentionInHours"] = update["DataRetentionInHours"]
		p.streamConfig = cfg
	}
	p.status, p.updated = "UPDATING", st.now()
	return st.poolView(p), nil
}

func (st *cmplStore) deletePool(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	p, cerr := st.pool(in)
	if cerr != nil {
		return nil, cerr
	}
	if len(p.refs) > 0 {
		return nil, chimeSDKConflict("Kinesis video stream pool %s is used by %d media stream pipelines.", p.name, len(p.refs))
	}
	delete(st.pools, p.name)
	delete(st.poolIDs, p.id)
	delete(st.tokens, p.tokenKey)
	return nil, nil
}

func (st *cmplStore) listPools(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	names := make([]string, 0, len(st.pools))
	for name := range st.pools {
		names = append(names, name)
	}
	page, next := chimeSDKPage(names, mmgrStr(in, "NextToken"), mmgrInt(in, "MaxResults", 0), 100)
	out := make([]any, 0, len(page))
	for _, name := range page {
		p := st.pools[name]
		out = append(out, map[string]any{"PoolName": p.name, "PoolId": p.id, "PoolArn": p.arn})
	}
	resp := map[string]any{"KinesisVideoStreamPools": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

// ---- speaker search and voice tone analysis tasks ----

// taskStatus settles a task: Initializing -> InProgress, Stopping -> Stopped.
func (st *cmplStore) taskStatus(t *cmplTask) string {
	if !st.now().Before(t.updated.Add(st.settle)) {
		switch t.status {
		case "Initializing":
			t.status, t.updated = "InProgress", t.updated.Add(st.settle)
		case "Stopping":
			t.status, t.updated = "Stopped", t.updated.Add(st.settle)
		}
	}
	return t.status
}

func (st *cmplStore) taskView(t *cmplTask, idKey, statusKey string) map[string]any {
	return map[string]any{idKey: t.id, statusKey: st.taskStatus(t), "CreatedTimestamp": chimeSDKTime(t.created), "UpdatedTimestamp": chimeSDKTime(t.updated)}
}

// startTask starts a voice analytics task of kind on a running media
// insights pipeline whose configuration has a VoiceAnalyticsProcessor.
func (st *cmplStore) startTask(in map[string]any, kind string) (*cmplTask, *chimeSDKErr) {
	p, cerr := st.insights(mmgrStr(in, "Identifier"), nil)
	if cerr != nil {
		return nil, cerr
	}
	tokenKey := ""
	if token := mmgrStr(in, "ClientRequestToken"); token != "" {
		tokenKey = kind + "|" + p.id + "|" + token
		if t := p.tasks[st.tokens[tokenKey]]; t != nil {
			return t, nil
		}
	}
	if c := st.configs[p.configName]; c == nil || !c.hasElement("VoiceAnalyticsProcessor") {
		return nil, chimeSDKBadRequest("Media insights pipeline %s doesn't have voice analytics enabled.", p.id)
	}
	if p.status == "Stopped" || p.status == "Failed" {
		return nil, chimeSDKConflict("Media insights pipeline %s is %s.", p.id, p.status)
	}
	if len(p.tasks) >= cmplMaxTasks {
		return nil, chimeSDKLimit("A media insights pipeline can have at most %d tasks.", cmplMaxTasks)
	}
	now := st.now()
	t := &cmplTask{id: uuid.NewString(), kind: kind, status: "Initializing", tokenKey: tokenKey, created: now, updated: now}
	p.tasks[t.id] = t
	if tokenKey != "" {
		st.tokens[tokenKey] = t.id
	}
	return t, nil
}

func (st *cmplStore) task(in map[string]any, kind, idKey string) (*cmplTask, *chimeSDKErr) {
	p, cerr := st.insights(mmgrStr(in, "Identifier"), nil)
	if cerr != nil {
		return nil, cerr
	}
	t := p.tasks[mmgrStr(in, idKey)]
	if t == nil || t.kind != kind {
		return nil, chimeSDKNotFound("The task %s doesn't exist.", mmgrStr(in, idKey))
	}
	return t, nil
}

func (st *cmplStore) stopTask(in map[string]any, kind, idKey string) *chimeSDKErr {
	t, cerr := st.task(in, kind, idKey)
	if cerr != nil {
		return cerr
	}
	if s := st.taskStatus(t); s == "Stopping" || s == "Stopped" || s == "Failed" {
		return chimeSDKConflict("Task %s is already %s.", t.id, s)
	}
	t.status, t.updated = "Stopping", st.now()
	return nil
}

func (st *cmplStore) startSpeakerSearch(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	if !strings.HasPrefix(mmgrStr(in, "VoiceProfileDomainArn"), "arn:aws:chime:") {
		return nil, chimeSDKBadRequest("VoiceProfileDomainArn must be a Chime SDK voice profile domain ARN.")
	}
	t, cerr := st.startTask(in, "speaker")
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"SpeakerSearchTask": st.taskView(t, "SpeakerSearchTaskId", "SpeakerSearchTaskStatus")}, nil
}

func (st *cmplStore) getSpeakerSearch(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	t, cerr := st.task(in, "speaker", "SpeakerSearchTaskId")
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"SpeakerSearchTask": st.taskView(t, "SpeakerSearchTaskId", "SpeakerSearchTaskStatus")}, nil
}

func (st *cmplStore) stopSpeakerSearch(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	return nil, st.stopTask(in, "speaker", "SpeakerSearchTaskId")
}

func (st *cmplStore) startVoiceTone(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	t, cerr := st.startTask(in, "voice")
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"VoiceToneAnalysisTask": st.taskView(t, "VoiceToneAnalysisTaskId", "VoiceToneAnalysisTaskStatus")}, nil
}

func (st *cmplStore) getVoiceTone(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	t, cerr := st.task(in, "voice", "VoiceToneAnalysisTaskId")
	if cerr != nil {
		return nil, cerr
	}
	return map[string]any{"VoiceToneAnalysisTask": st.taskView(t, "VoiceToneAnalysisTaskId", "VoiceToneAnalysisTaskStatus")}, nil
}

func (st *cmplStore) stopVoiceTone(in map[string]any, _ map[string]bool) (map[string]any, *chimeSDKErr) {
	return nil, st.stopTask(in, "voice", "VoiceToneAnalysisTaskId")
}
