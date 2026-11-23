package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	osisAccountID    = "123456789012"
	osisRegion       = "us-east-1"
	osisMaxUnits     = 384
	osisMaxPipelines = 1000 // ponytail: far above the AWS quota so repeated test runs don't hit it
	osisMaxEndpoints = 1000
	osisMaxTags      = 200
	osisMaxBody      = 1 << 20
	osisMaxChanges   = 20 // change-progress records kept per pipeline
)

var (
	osisNamePattern = regexp.MustCompile(`^[a-z][a-z0-9\-]+$`)
	osisPipelineARN = regexp.MustCompile(`^arn:aws:osis:[a-z0-9-]+:\d{12}:pipeline/[a-z][a-z0-9\-]{2,27}$`)
	osisEndpointID  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_]{2,511}$`)
)

// osisError is an OpenSearch Ingestion API error; status follows the botocore model.
type osisError struct {
	status int
	code   string
	msg    string
}

func osisValidation(format string, args ...any) *osisError {
	return &osisError{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func osisNotFound(format string, args ...any) *osisError {
	return &osisError{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf(format, args...)}
}

func osisConflict(format string, args ...any) *osisError {
	return &osisError{http.StatusConflict, "ConflictException", fmt.Sprintf(format, args...)}
}

func osisLimit(format string, args ...any) *osisError {
	return &osisError{http.StatusConflict, "LimitExceededException", fmt.Sprintf(format, args...)}
}

type osisTag struct{ key, value string }

type osisChange struct {
	start time.Time
	kind  string // CREATE or UPDATE
}

type osisPipeline struct {
	name, arn, body, roleArn string
	minUnits, maxUnits       int
	status                   string // current status; transitional ones settle on the next read
	created, updated         time.Time
	ingestURL                string
	destinations             []osisDestination
	logOptions, buffer       map[string]any
	encryption, vpcOptions   map[string]any
	vpcID, vpcEndpointID     string
	tags                     []osisTag
	changes                  []osisChange
	policy                   string
}

type osisEndpoint struct {
	id, pipelineArn, vpcID, status string
	subnets, groups                []string
	ingestURL                      string
}

// osisStore holds Amazon OpenSearch Ingestion state.
type osisStore struct {
	mu        sync.Mutex
	pipelines map[string]*osisPipeline // by name
	endpoints map[string]*osisEndpoint // by endpoint id
	order     []string                 // endpoint ids in creation order
}

func newOSISStore() *osisStore {
	return &osisStore{pipelines: map[string]*osisPipeline{}, endpoints: map[string]*osisEndpoint{}}
}

func osisARN(name string) string {
	return fmt.Sprintf("arn:aws:osis:%s:%s:pipeline/%s", osisRegion, osisAccountID, name)
}

func osisRandID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// osisSpec is a validated CreatePipeline / UpdatePipeline request.
type osisSpec struct {
	minUnits, maxUnits       *int
	body, roleArn            string
	destinations             []osisDestination
	logOptions, buffer       map[string]any
	encryption, vpcOptions   map[string]any
	vpcID                    string
	tags                     []osisTag
	hasLog, hasBuffer, hasEn bool
}

// settle moves a transitional pipeline to its resting state. Reads call it, so
// a change reports its transitional state once and then completes.
// ponytail: settle-on-read instead of timers; there's no background work.
func (p *osisPipeline) settle() {
	switch p.status {
	case "CREATING", "UPDATING", "STARTING":
		p.status = "ACTIVE"
	case "STOPPING":
		p.status = "STOPPED"
	}
}

func (st *osisStore) pipeline(name string) (*osisPipeline, *osisError) {
	p := st.pipelines[name]
	if p == nil {
		return nil, osisNotFound("Pipeline %s does not exist.", name)
	}
	return p, nil
}

func (st *osisStore) create(name, host string, spec osisSpec) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.pipelines[name] != nil {
		return nil, &osisError{http.StatusConflict, "ResourceAlreadyExistsException", fmt.Sprintf("Pipeline %s already exists.", name)}
	}
	if len(st.pipelines) >= osisMaxPipelines {
		return nil, osisLimit("This account can have at most %d pipelines.", osisMaxPipelines)
	}
	now := time.Now().UTC()
	p := &osisPipeline{
		name: name, arn: osisARN(name), body: spec.body, roleArn: spec.roleArn,
		minUnits: *spec.minUnits, maxUnits: *spec.maxUnits, status: "CREATING", created: now, updated: now,
		ingestURL:    fmt.Sprintf("%s/_stackyard/osis/%s-%s/ingest", host, name, osisRandID(6)),
		destinations: spec.destinations, logOptions: spec.logOptions, buffer: spec.buffer,
		encryption: spec.encryption, vpcOptions: spec.vpcOptions, vpcID: spec.vpcID, tags: spec.tags,
		changes: []osisChange{{start: now, kind: "CREATE"}},
	}
	if spec.vpcOptions != nil && osisStr(spec.vpcOptions, "VpcEndpointManagement") != "CUSTOMER" {
		p.vpcEndpointID = "vpce-" + osisRandID(8)
	}
	st.pipelines[name] = p
	return map[string]any{"Pipeline": p.view()}, nil
}

func (st *osisStore) update(name string, spec osisSpec) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeline(name)
	if err != nil {
		return nil, err
	}
	if p.status != "ACTIVE" && p.status != "STOPPED" {
		return nil, osisConflict("Pipeline %s can't be updated while it is %s.", name, p.status)
	}
	minU, maxU := p.minUnits, p.maxUnits
	if spec.minUnits != nil {
		minU = *spec.minUnits
	}
	if spec.maxUnits != nil {
		maxU = *spec.maxUnits
	}
	if minU > maxU {
		return nil, osisValidation("MinUnits (%d) can't be greater than MaxUnits (%d).", minU, maxU)
	}
	buffered := p.buffer
	if spec.hasBuffer {
		buffered = spec.buffer
	}
	if osisBool(buffered, "PersistentBufferEnabled") && minU < 2 {
		return nil, osisValidation("A pipeline with persistent buffering needs MinUnits of at least 2.")
	}
	next := *p // validated in full before anything is applied
	next.minUnits, next.maxUnits = minU, maxU
	if spec.body != "" {
		next.body, next.destinations = spec.body, spec.destinations
	}
	if spec.roleArn != "" {
		next.roleArn = spec.roleArn
	}
	if spec.hasLog {
		next.logOptions = spec.logOptions
	}
	if spec.hasBuffer {
		next.buffer = spec.buffer
	}
	if spec.hasEn {
		next.encryption = spec.encryption
	}
	if p.status == "ACTIVE" {
		next.status = "UPDATING"
	}
	next.updated = time.Now().UTC()
	next.changes = append(append([]osisChange(nil), p.changes...), osisChange{start: next.updated, kind: "UPDATE"})
	if len(next.changes) > osisMaxChanges {
		next.changes = next.changes[len(next.changes)-osisMaxChanges:]
	}
	*p = next
	return map[string]any{"Pipeline": p.view()}, nil
}

// setRunning starts or stops a pipeline.
func (st *osisStore) setRunning(name string, start bool) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeline(name)
	if err != nil {
		return nil, err
	}
	switch {
	case start && p.status == "STOPPED":
		p.status = "STARTING"
	case !start && p.status == "ACTIVE":
		p.status = "STOPPING"
	default:
		verb, want := "stopped", "ACTIVE"
		if start {
			verb, want = "started", "STOPPED"
		}
		return nil, osisConflict("Pipeline %s can only be %s when it is %s; it is %s.", name, verb, want, p.status)
	}
	p.updated = time.Now().UTC()
	return map[string]any{"Pipeline": p.view()}, nil
}

func (st *osisStore) get(name string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeline(name)
	if err != nil {
		return nil, err
	}
	p.settle()
	return map[string]any{"Pipeline": p.view()}, nil
}

// remove deletes a pipeline and its endpoints at once.
// ponytail: DELETING isn't observable; the pipeline is gone immediately.
func (st *osisStore) remove(name string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeline(name)
	if err != nil {
		return nil, err
	}
	if p.status == "CREATING" || p.status == "UPDATING" {
		return nil, osisConflict("Pipeline %s can't be deleted while it is %s.", name, p.status)
	}
	delete(st.pipelines, name)
	for id, e := range st.endpoints {
		if e.pipelineArn == p.arn {
			delete(st.endpoints, id)
		}
	}
	return map[string]any{}, nil
}

func (st *osisStore) list(token string, max int) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	names := make([]string, 0, len(st.pipelines))
	for name := range st.pipelines {
		names = append(names, name)
	}
	sort.Strings(names)
	start, end, next, err := osisPage(len(names), token, max)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, name := range names[start:end] {
		p := st.pipelines[name]
		p.settle()
		out = append(out, p.summary())
	}
	resp := map[string]any{"Pipelines": out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (st *osisStore) changeProgress(name string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipeline(name)
	if err != nil {
		return nil, err
	}
	// The newest change is in progress until the pipeline settles; report it
	// before settling, as the change completes on this read.
	busy := p.status == "CREATING" || p.status == "UPDATING"
	p.settle()
	out := make([]any, 0, len(p.changes))
	for i := len(p.changes) - 1; i >= 0; i-- {
		c := p.changes[i]
		inProgress := busy && i == len(p.changes)-1
		status := "COMPLETED"
		if inProgress {
			status = "IN_PROGRESS"
		}
		stages := []any{}
		for j, n := range []struct{ name, desc string }{
			{"VALIDATION", "Validating the pipeline configuration"},
			{"CREATE_ENV", "Creating the pipeline environment"},
			{"PROVISION_COMPUTE", "Provisioning Ingestion OCUs"},
			{"DEPLOY_PIPELINE", "Deploying the pipeline configuration"},
		} {
			s := "COMPLETED"
			if inProgress && j == 1 {
				s = "IN_PROGRESS"
			} else if inProgress && j > 1 {
				s = "PENDING"
			}
			stages = append(stages, map[string]any{"Name": n.name, "Status": s, "Description": n.desc, "LastUpdatedAt": osisTime(c.start)})
		}
		out = append(out, map[string]any{"StartTime": osisTime(c.start), "Status": status, "TotalNumberOfStages": 4, "ChangeProgressStages": stages})
	}
	return map[string]any{"ChangeProgressStatuses": out}, nil
}

func (st *osisStore) pipelineByARN(arn string) (*osisPipeline, *osisError) {
	name, ok := strings.CutPrefix(arn, "arn:aws:osis:"+osisRegion+":"+osisAccountID+":pipeline/")
	if !ok {
		return nil, osisNotFound("Pipeline %s does not exist.", arn)
	}
	return st.pipeline(name)
}

func (st *osisStore) listTags(arn string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(arn)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Tags": osisTagList(p.tags)}, nil
}

func (st *osisStore) tag(arn string, tags []osisTag) *osisError {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(arn)
	if err != nil {
		return err
	}
	merged := osisMergeTags(p.tags, tags)
	if len(merged) > osisMaxTags {
		return osisLimit("A pipeline can have at most %d tags.", osisMaxTags)
	}
	p.tags = merged
	return nil
}

func (st *osisStore) untag(arn string, keys []string) *osisError {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(arn)
	if err != nil {
		return err
	}
	drop := map[string]bool{}
	for _, k := range keys {
		drop[k] = true
	}
	kept := []osisTag{}
	for _, t := range p.tags {
		if !drop[t.key] {
			kept = append(kept, t)
		}
	}
	p.tags = kept
	return nil
}

func (st *osisStore) putPolicy(arn, policy string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(arn)
	if err != nil {
		return nil, err
	}
	p.policy = policy
	return map[string]any{"ResourceArn": p.arn, "Policy": policy}, nil
}

func (st *osisStore) getPolicy(arn string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(arn)
	if err != nil {
		return nil, err
	}
	if p.policy == "" {
		return nil, osisNotFound("Pipeline %s has no resource policy.", arn)
	}
	return map[string]any{"ResourceArn": p.arn, "Policy": p.policy}, nil
}

func (st *osisStore) deletePolicy(arn string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(arn)
	if err != nil {
		return nil, err
	}
	if p.policy == "" {
		return nil, osisNotFound("Pipeline %s has no resource policy.", arn)
	}
	p.policy = ""
	return map[string]any{}, nil
}

func (st *osisStore) createEndpoint(pipelineArn, vpcID, host string, subnets, groups []string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(pipelineArn)
	if err != nil {
		return nil, err
	}
	if len(st.endpoints) >= osisMaxEndpoints {
		return nil, osisLimit("This account can have at most %d pipeline endpoints.", osisMaxEndpoints)
	}
	e := &osisEndpoint{
		id: "pe-" + osisRandID(8), pipelineArn: p.arn, vpcID: vpcID, status: "CREATING",
		subnets: subnets, groups: groups, ingestURL: fmt.Sprintf("%s/_stackyard/osis/%s/ingest", host, p.name),
	}
	st.endpoints[e.id] = e
	st.order = append(st.order, e.id)
	return map[string]any{"PipelineArn": e.pipelineArn, "EndpointId": e.id, "Status": e.status, "VpcId": e.vpcID}, nil
}

func (st *osisStore) deleteEndpoint(id string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.endpoints[id] == nil {
		return nil, osisNotFound("Pipeline endpoint %s does not exist.", id)
	}
	delete(st.endpoints, id)
	return map[string]any{}, nil
}

// endpointIDs lists live endpoint ids in creation order, dropping deleted ones.
func (st *osisStore) endpointIDs() []string {
	live := st.order[:0]
	for _, id := range st.order {
		if st.endpoints[id] != nil {
			live = append(live, id)
		}
	}
	st.order = live
	return append([]string(nil), live...)
}

func (st *osisStore) listEndpoints(token string, max int, connections bool) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ids := st.endpointIDs()
	start, end, next, err := osisPage(len(ids), token, max)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, id := range ids[start:end] {
		e := st.endpoints[id]
		if e.status == "CREATING" {
			e.status = "ACTIVE"
		}
		if connections {
			out = append(out, map[string]any{"PipelineArn": e.pipelineArn, "EndpointId": e.id, "Status": e.status, "VpcEndpointOwner": osisAccountID})
			continue
		}
		out = append(out, map[string]any{
			"PipelineArn": e.pipelineArn, "EndpointId": e.id, "Status": e.status, "VpcId": e.vpcID,
			"VpcOptions": map[string]any{"SubnetIds": osisAnyList(e.subnets), "SecurityGroupIds": osisAnyList(e.groups)}, "IngestEndpointUrl": e.ingestURL,
		})
	}
	key := "PipelineEndpoints"
	if connections {
		key = "PipelineEndpointConnections"
	}
	resp := map[string]any{key: out}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (st *osisStore) revoke(pipelineArn string, ids []string) (map[string]any, *osisError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.pipelineByARN(pipelineArn)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		e := st.endpoints[id]
		if e == nil || e.pipelineArn != p.arn {
			return nil, osisValidation("Pipeline endpoint %s isn't connected to pipeline %s.", id, p.name)
		}
	}
	for _, id := range ids {
		st.endpoints[id].status = "REVOKED"
	}
	return map[string]any{"PipelineArn": p.arn}, nil
}

// view is the Pipeline output shape, built fresh under the store lock.
func (p *osisPipeline) view() map[string]any {
	out := map[string]any{
		"PipelineName": p.name, "PipelineArn": p.arn, "MinUnits": p.minUnits, "MaxUnits": p.maxUnits,
		"Status": p.status, "StatusReason": map[string]any{"Description": osisStatusText(p.status)},
		"PipelineConfigurationBody": p.body, "CreatedAt": osisTime(p.created), "LastUpdatedAt": osisTime(p.updated),
		"IngestEndpointUrls": []any{p.ingestURL}, "Destinations": osisDestinationList(p.destinations), "Tags": osisTagList(p.tags),
		"ServiceVpcEndpoints": []any{},
	}
	if p.roleArn != "" {
		out["PipelineRoleArn"] = p.roleArn
	}
	for key, v := range map[string]map[string]any{"LogPublishingOptions": p.logOptions, "BufferOptions": p.buffer, "EncryptionAtRestOptions": p.encryption} {
		if v != nil {
			out[key] = osisCopy(v)
		}
	}
	if p.vpcOptions != nil {
		ep := map[string]any{"VpcId": p.vpcID, "VpcOptions": osisCopy(p.vpcOptions)}
		if p.vpcEndpointID != "" {
			ep["VpcEndpointId"] = p.vpcEndpointID
		}
		out["VpcEndpoints"] = []any{ep}
		out["VpcEndpointService"] = "com.amazonaws.osis." + osisRegion + "." + p.name
	}
	return out
}

func (p *osisPipeline) summary() map[string]any {
	return map[string]any{
		"PipelineName": p.name, "PipelineArn": p.arn, "MinUnits": p.minUnits, "MaxUnits": p.maxUnits,
		"Status": p.status, "StatusReason": map[string]any{"Description": osisStatusText(p.status)},
		"CreatedAt": osisTime(p.created), "LastUpdatedAt": osisTime(p.updated),
		"Destinations": osisDestinationList(p.destinations), "Tags": osisTagList(p.tags),
	}
}

func osisStatusText(status string) string {
	switch status {
	case "ACTIVE":
		return "The pipeline is ready to ingest data."
	case "STOPPED":
		return "The pipeline is stopped."
	}
	return "The pipeline is " + strings.ToLower(status) + "."
}

func osisTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func osisTagList(tags []osisTag) []any {
	out := make([]any, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]any{"Key": t.key, "Value": t.value})
	}
	return out
}

func osisMergeTags(have, add []osisTag) []osisTag {
	out := append([]osisTag(nil), have...)
	for _, t := range add {
		replaced := false
		for i := range out {
			if out[i].key == t.key {
				out[i].value, replaced = t.value, true
			}
		}
		if !replaced {
			out = append(out, t)
		}
	}
	return out
}

func osisDestinationList(ds []osisDestination) []any {
	out := make([]any, 0, len(ds))
	for _, d := range ds {
		out = append(out, map[string]any{"ServiceName": d.service, "Endpoint": d.endpoint})
	}
	return out
}

// osisCopy deep-copies a decoded JSON object so responses never share stored maps.
func osisCopy(m map[string]any) map[string]any {
	raw, _ := json.Marshal(m)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func osisAnyList(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}

// osisPage slices n items by an offset token.
func osisPage(n int, token string, max int) (int, int, string, *osisError) {
	start := 0
	if token != "" {
		v, err := strconv.ParseInt(token, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", &osisError{http.StatusBadRequest, "InvalidPaginationTokenException", "The pagination token is invalid."}
		}
		start = int(v)
	}
	end := min(n, start+max)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func osisStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func osisBool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}
