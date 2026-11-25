package server

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS IoT Secure Tunneling: awsJson1_1, X-Amz-Target IoTSecuredTunneling.*,
// signed IoTSecuredTunneling. A tunnel opens with fresh source and
// destination access tokens that are returned once (OpenTunnel and
// RotateTunnelAccessToken) and never stored, and closes itself when its
// maximum lifetime runs out. No proxy runs, so both ends stay DISCONNECTED.
// budServe writes responses and errors with Content-Type
// application/x-amz-json-1.1.

const (
	istAccount = "123456789012"
	istRegion  = "us-east-1"
	// ponytail: AWS's default quota is 300 open tunnels; closed ones stay
	// listed until deleted, so stored tunnels are capped too.
	istMaxOpen     = 300
	istMaxTunnels  = 1000
	istMaxTags     = 50
	istMaxLifetime = 720 // minutes, also the default
	istARNPrefix   = "arn:aws:iot:" + istRegion + ":" + istAccount + ":tunnel/"
)

var istAPI = &budJSONAPI{
	prefix: "IoTSecuredTunneling.", signing: "IoTSecuredTunneling", contentType: "application/x-amz-json-1.1",
	invalidCode: "ValidationException", model: awsmodels.IoTSecureTunnelingModel,
}

func init() {
	istAPI.handlers = map[string]budHandler{
		"OpenTunnel":              (*Server).istOpenTunnel,
		"DescribeTunnel":          (*Server).istDescribeTunnel,
		"ListTunnels":             (*Server).istListTunnels,
		"CloseTunnel":             (*Server).istCloseTunnel,
		"RotateTunnelAccessToken": (*Server).istRotateToken,
		"TagResource":             (*Server).istTagResource,
		"UntagResource":           (*Server).istUntagResource,
		"ListTagsForResource":     (*Server).istListTags,
	}
}

func (s *Server) handleIoTSecureTunnelingJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, istAPI, iotSecureTunnelingOperationByName)
}

type istTunnel struct {
	id, arn, description, status string
	dest                         map[string]any // conformed destinationConfig
	lifetime                     int
	tags                         map[string]string
	created, updated             time.Time
	srcAt, dstAt                 time.Time // connection states last changed
}

type istStore struct {
	mu      sync.Mutex
	tunnels map[string]*istTunnel
	order   []string // tunnel ids, oldest first
}

func newIoTSecureTunnelingStore() *istStore {
	return &istStore{tunnels: map[string]*istTunnel{}}
}

func istNotFound(id string) *budErr {
	return budErrorf("ResourceNotFoundException", "Tunnel %s not found", id)
}

// settle closes an open tunnel whose lifetime has run out.
func (t *istTunnel) settle(now time.Time) {
	end := t.created.Add(time.Duration(t.lifetime) * time.Minute)
	if t.status == "OPEN" && !now.Before(end) {
		t.status, t.updated = "CLOSED", end
	}
}

// istToken returns a fresh client access token.
func istToken() string {
	b := make([]byte, 48)
	_, _ = rand.Read(b)
	return "AQGAAX" + base64.RawURLEncoding.EncodeToString(b)
}

func (st *istStore) get(id string, now time.Time) (*istTunnel, *budErr) {
	t := st.tunnels[id]
	if t == nil {
		return nil, istNotFound(id)
	}
	t.settle(now)
	return t, nil
}

func (s *Server) istOpenTunnel(in map[string]any) (map[string]any, *budErr) {
	tags, err := istTagsIn(budList(in, "tags"))
	if err != nil {
		return nil, err
	}
	lifetime := budInt(budMap(in, "timeoutConfig"), "maxLifetimeTimeoutMinutes", istMaxLifetime)
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	open := 0
	for _, t := range st.tunnels {
		t.settle(now)
		if t.status == "OPEN" {
			open++
		}
	}
	if open >= istMaxOpen {
		return nil, budErrorf("LimitExceededException", "You can have at most %d open tunnels", istMaxOpen)
	}
	if len(st.tunnels) >= istMaxTunnels {
		return nil, budErrorf("LimitExceededException", "At most %d tunnels can be stored; delete closed tunnels first", istMaxTunnels)
	}
	id := uuid.NewString()
	t := &istTunnel{
		id: id, arn: istARNPrefix + id,
		description: budStr(in, "description"), status: "OPEN", lifetime: lifetime, tags: tags,
		created: now, updated: now, srcAt: now, dstAt: now,
	}
	if d := budMap(in, "destinationConfig"); d != nil {
		t.dest = budCloneMap(d)
	}
	st.tunnels[id] = t
	st.order = append(st.order, id)
	return map[string]any{"tunnelId": id, "tunnelArn": t.arn, "sourceAccessToken": istToken(), "destinationAccessToken": istToken()}, nil
}

func (t *istTunnel) describe() map[string]any {
	state := func(at time.Time) map[string]any {
		return map[string]any{"status": "DISCONNECTED", "lastUpdatedAt": budEpoch(at)}
	}
	out := map[string]any{
		"tunnelId": t.id, "tunnelArn": t.arn, "status": t.status,
		"sourceConnectionState": state(t.srcAt), "destinationConnectionState": state(t.dstAt),
		"timeoutConfig": map[string]any{"maxLifetimeTimeoutMinutes": t.lifetime},
		"tags":          istTagsOut(t.tags), "createdAt": budEpoch(t.created), "lastUpdatedAt": budEpoch(t.updated),
	}
	if t.description != "" {
		out["description"] = t.description
	}
	if t.dest != nil {
		out["destinationConfig"] = budCloneMap(t.dest)
	}
	return out
}

func (s *Server) istDescribeTunnel(in map[string]any) (map[string]any, *budErr) {
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.get(budStr(in, "tunnelId"), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return map[string]any{"tunnel": t.describe()}, nil
}

func (s *Server) istListTunnels(in map[string]any) (map[string]any, *budErr) {
	thing := budStr(in, "thingName")
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	var match []*istTunnel
	for i := len(st.order) - 1; i >= 0; i-- { // newest first
		t := st.tunnels[st.order[i]]
		if thing != "" && budStr(t.dest, "thingName") != thing {
			continue
		}
		t.settle(now)
		match = append(match, t)
	}
	start, end, next, err := budPage(budStr(in, "nextToken"), len(match), budInt(in, "maxResults", 100), "ValidationException")
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, t := range match[start:end] {
		sum := map[string]any{"tunnelId": t.id, "tunnelArn": t.arn, "status": t.status,
			"createdAt": budEpoch(t.created), "lastUpdatedAt": budEpoch(t.updated)}
		if t.description != "" {
			sum["description"] = t.description
		}
		list = append(list, sum)
	}
	return map[string]any{"tunnelSummaries": list, "nextToken": next}, nil
}

// istCloseTunnel closes a tunnel, or with delete removes it altogether.
func (s *Server) istCloseTunnel(in map[string]any) (map[string]any, *budErr) {
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	t, err := st.get(budStr(in, "tunnelId"), now)
	if err != nil {
		return nil, err
	}
	if budBool(in, "delete") {
		delete(st.tunnels, t.id)
		for i, id := range st.order {
			if id == t.id {
				st.order = append(st.order[:i:i], st.order[i+1:]...)
				break
			}
		}
		return map[string]any{}, nil
	}
	if t.status == "OPEN" {
		t.status, t.updated = "CLOSED", now
	}
	return map[string]any{}, nil
}

// istRotateToken issues new tokens for the requested ends, which drops any
// connection those ends had.
func (s *Server) istRotateToken(in map[string]any) (map[string]any, *budErr) {
	mode := budStr(in, "clientMode")
	dest := budMap(in, "destinationConfig")
	if dest != nil && mode == "SOURCE" {
		return nil, budErrorf("ValidationException", "destinationConfig can only be set when clientMode is DESTINATION or ALL")
	}
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	t, err := st.get(budStr(in, "tunnelId"), now)
	if err != nil {
		return nil, err
	}
	if t.status != "OPEN" {
		return nil, budErrorf("ResourceNotFoundException", "Tunnel %s is closed", t.id)
	}
	out := map[string]any{"tunnelArn": t.arn}
	if mode == "SOURCE" || mode == "ALL" {
		out["sourceAccessToken"], t.srcAt = istToken(), now
	}
	if mode == "DESTINATION" || mode == "ALL" {
		out["destinationAccessToken"], t.dstAt = istToken(), now
		if dest != nil {
			t.dest = budCloneMap(dest)
		}
	}
	t.updated = now
	return out, nil
}

// ---- tags ----

func istTagsIn(list []any) (map[string]string, *budErr) {
	out := map[string]string{}
	for _, item := range list {
		m, _ := item.(map[string]any)
		k := budStr(m, "key")
		if len(k) >= 4 && (k[:4] == "aws:" || k[:4] == "AWS:") {
			return nil, budErrorf("ValidationException", "tag keys starting with aws: are reserved")
		}
		out[k] = budStr(m, "value")
	}
	if len(out) > istMaxTags {
		return nil, budErrorf("ValidationException", "a tunnel can have at most %d tags", istMaxTags)
	}
	return out, nil
}

func istTagsOut(tags map[string]string) []any {
	out := make([]any, 0, len(tags))
	for _, t := range budTagsOut(tags) {
		m := t.(map[string]any)
		out = append(out, map[string]any{"key": m["Key"], "value": m["Value"]})
	}
	return out
}

func (st *istStore) byARN(arn string) (*istTunnel, *budErr) {
	if id, ok := strings.CutPrefix(arn, istARNPrefix); ok && st.tunnels[id] != nil {
		return st.tunnels[id], nil
	}
	return nil, budErrorf("ResourceNotFoundException", "Resource %s not found", arn)
}

func (s *Server) istTagResource(in map[string]any) (map[string]any, *budErr) {
	add, err := istTagsIn(budList(in, "tags"))
	if err != nil {
		return nil, err
	}
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.byARN(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(t.tags, add, istMaxTags, "ValidationException")
	if err != nil {
		return nil, err
	}
	t.tags = merged
	return map[string]any{}, nil
}

func (s *Server) istUntagResource(in map[string]any) (map[string]any, *budErr) {
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.byARN(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	t.tags = budDropTags(t.tags, budStrings(in, "tagKeys"))
	return map[string]any{}, nil
}

func (s *Server) istListTags(in map[string]any) (map[string]any, *budErr) {
	st := s.iotSecureTunneling
	st.mu.Lock()
	defer st.mu.Unlock()
	t, err := st.byARN(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"tags": istTagsOut(t.tags)}, nil
}
