package server

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	ccAccountID = "123456789012"
	ccRegion    = "us-east-1"
	// ponytail: per-kind caps well above AWS quotas (250 connections, 50
	// hosts) so repeated coverage runs against one server don't hit them.
	ccMaxResources      = 1000
	ccMaxTags           = 200
	ccMaxBlockersPerCfg = 20
	ccDefaultPageSize   = 100
)

type ccError struct {
	status    int
	code, msg string
}

func ccErr(status int, code, format string, args ...any) *ccError {
	return &ccError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func ccNotFound(format string, args ...any) *ccError {
	return ccErr(http.StatusBadRequest, "ResourceNotFoundException", format, args...)
}

func ccInvalid(format string, args ...any) *ccError {
	return ccErr(http.StatusBadRequest, "InvalidInputException", format, args...)
}

func ccValidation(format string, args ...any) *ccError {
	return ccErr(http.StatusBadRequest, "ValidationException", format, args...)
}

type ccTag struct{ Key, Value string }

type ccConnection struct {
	id, name, providerType, hostID, status string
	seq                                    int
}

type ccHost struct {
	id, name, providerType, endpoint, status string
	vpc                                      map[string]any
	seq                                      int
}

type ccLink struct {
	id, connectionID, ownerID, repo, providerType, kmsKey string
	seq                                                   int
}

type ccBlocker struct {
	id, reason, resolvedReason string
	created, resolved          time.Time
}

type ccSync struct {
	syncType, resourceName, linkID, branch, configFile, roleARN string
	publish, trigger, prComment                                 string
	updated                                                     time.Time
	blockers                                                    []*ccBlocker
	seq                                                         int
}

type codeConnectionsStore struct {
	mu          sync.Mutex
	seq         int
	connections map[string]*ccConnection
	hosts       map[string]*ccHost
	links       map[string]*ccLink
	syncs       map[string]*ccSync // syncType + "/" + resourceName
	tags        map[string][]ccTag // kind + "/" + id
}

func newCodeConnectionsStore() *codeConnectionsStore {
	return &codeConnectionsStore{
		connections: map[string]*ccConnection{},
		hosts:       map[string]*ccHost{},
		links:       map[string]*ccLink{},
		syncs:       map[string]*ccSync{},
		tags:        map[string][]ccTag{},
	}
}

func (st *codeConnectionsStore) nextSeq() int { st.seq++; return st.seq }

func (api *ccAPI) arn(kind, id string) string {
	return "arn:aws:" + api.service + ":" + ccRegion + ":" + ccAccountID + ":" + kind + "/" + id
}

// ccParseARN accepts either service prefix and returns kind and id.
func ccParseARN(arn string) (kind, id string, ok bool) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 || parts[0] != "arn" || (parts[2] != "codeconnections" && parts[2] != "codestar-connections") || parts[4] != ccAccountID {
		return "", "", false
	}
	kind, id, ok = strings.Cut(parts[5], "/")
	return kind, id, ok && id != "" && !strings.Contains(id, "/")
}

func ccStr(in map[string]any, key string) string {
	s, _ := in[key].(string)
	return s
}

func ccTagsIn(v any) []ccTag {
	items, _ := v.([]any)
	out := make([]ccTag, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		out = append(out, ccTag{ccStr(m, "Key"), ccStr(m, "Value")})
	}
	return out
}

func ccTagsOut(tags []ccTag) []any {
	out := make([]any, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]any{"Key": t.Key, "Value": t.Value})
	}
	return out
}

// mergeTagsLocked applies tags to a resource, replacing values of existing
// keys, within the per-resource cap.
func (st *codeConnectionsStore) mergeTagsLocked(key string, add []ccTag) *ccError {
	merged := append([]ccTag(nil), st.tags[key]...)
	for _, t := range add {
		if strings.HasPrefix(strings.ToLower(t.Key), "aws:") {
			return ccValidation("tag keys may not start with aws:")
		}
		replaced := false
		for i := range merged {
			if merged[i].Key == t.Key {
				merged[i].Value, replaced = t.Value, true
			}
		}
		if !replaced {
			merged = append(merged, t)
		}
	}
	if len(merged) > ccMaxTags {
		return ccErr(http.StatusBadRequest, "LimitExceededException", "a resource can have at most %d tags", ccMaxTags)
	}
	st.tags[key] = merged
	return nil
}

// ccPage slices sorted items for MaxResults/NextToken (an offset).
func ccPage[T any](items []T, in map[string]any) ([]T, string, *ccError) {
	size := ccDefaultPageSize
	if n, ok := in["MaxResults"].(float64); ok && n > 0 {
		size = int(n) // the model bounds MaxResults to 0..100
	}
	start := 0
	if tok := ccStr(in, "NextToken"); tok != "" {
		n, err := strconv.Atoi(tok)
		if err != nil || n < 0 || n > len(items) {
			return nil, "", ccInvalid("invalid NextToken")
		}
		start = n
	}
	end := min(start+size, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

func ccRandHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func ccEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

var ccSelfManaged = map[string]bool{"GitHubEnterpriseServer": true, "GitLabSelfManaged": true}

// handle runs op and shapes its result to the API's output shape while still
// holding the lock, so the response never aliases stored state.
func (st *codeConnectionsStore) handle(api *ccAPI, op string, in map[string]any) (any, *ccError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := st.dispatchLocked(api, op, in)
	if err != nil {
		return nil, err
	}
	if out["NextToken"] == "" {
		delete(out, "NextToken") // last page: AWS omits the token
	}
	return api.model().Conform(api.model().Operations[op].Output, out), nil
}

func (st *codeConnectionsStore) dispatchLocked(api *ccAPI, op string, in map[string]any) (map[string]any, *ccError) {
	switch op {
	case "CreateConnection":
		return st.createConnection(api, in)
	case "GetConnection", "DeleteConnection":
		c, err := st.connectionByARN(ccStr(in, "ConnectionArn"))
		if err != nil {
			return nil, err
		}
		if op == "DeleteConnection" {
			delete(st.connections, c.id)
			delete(st.tags, "connection/"+c.id)
			return map[string]any{}, nil
		}
		return map[string]any{"Connection": st.connectionOut(api, c)}, nil
	case "ListConnections":
		return st.listConnections(api, in)
	case "CreateHost":
		return st.createHost(api, in)
	case "GetHost":
		h, err := st.hostByARN(ccStr(in, "HostArn"))
		if err != nil {
			return nil, err
		}
		out := st.hostOut(api, h)
		delete(out, "HostArn")
		delete(out, "StatusMessage")
		return out, nil
	case "DeleteHost":
		h, err := st.hostByARN(ccStr(in, "HostArn"))
		if err != nil {
			return nil, err
		}
		for _, c := range st.connections {
			if c.hostID == h.id {
				return nil, ccErr(http.StatusBadRequest, "ResourceUnavailableException", "host %s still has connections; delete them first", h.id)
			}
		}
		delete(st.hosts, h.id)
		delete(st.tags, "host/"+h.id)
		return map[string]any{}, nil
	case "UpdateHost":
		h, err := st.hostByARN(ccStr(in, "HostArn"))
		if err != nil {
			return nil, err
		}
		if v := ccStr(in, "ProviderEndpoint"); v != "" {
			h.endpoint = v
		}
		if vpc, ok := in["VpcConfiguration"].(map[string]any); ok {
			h.vpc = vpc
		}
		return map[string]any{}, nil
	case "ListHosts":
		hosts := make([]*ccHost, 0, len(st.hosts))
		for _, h := range st.hosts {
			hosts = append(hosts, h)
		}
		sort.Slice(hosts, func(i, j int) bool { return hosts[i].seq < hosts[j].seq })
		page, next, err := ccPage(hosts, in)
		if err != nil {
			return nil, err
		}
		out := make([]any, 0, len(page))
		for _, h := range page {
			out = append(out, st.hostOut(api, h))
		}
		return map[string]any{"Hosts": out, "NextToken": next}, nil
	case "CreateRepositoryLink":
		return st.createRepositoryLink(api, in)
	case "GetRepositoryLink", "DeleteRepositoryLink", "UpdateRepositoryLink":
		return st.repositoryLinkOp(api, op, in)
	case "ListRepositoryLinks":
		links := st.sortedLinks()
		page, next, err := ccPage(links, in)
		if err != nil {
			return nil, err
		}
		out := make([]any, 0, len(page))
		for _, l := range page {
			out = append(out, st.linkOut(api, l))
		}
		return map[string]any{"RepositoryLinks": out, "NextToken": next}, nil
	case "CreateSyncConfiguration":
		return st.createSyncConfiguration(in)
	case "GetSyncConfiguration", "UpdateSyncConfiguration", "DeleteSyncConfiguration":
		return st.syncConfigurationOp(op, in)
	case "ListSyncConfigurations", "ListRepositorySyncDefinitions":
		return st.listSyncs(op, in)
	case "GetRepositorySyncStatus":
		return st.repositorySyncStatus(in)
	case "GetResourceSyncStatus":
		return st.resourceSyncStatus(in)
	case "GetSyncBlockerSummary", "UpdateSyncBlocker":
		return st.syncBlockerOp(op, in)
	case "TagResource", "UntagResource", "ListTagsForResource":
		return st.tagOp(op, in)
	}
	return nil, ccErr(http.StatusBadRequest, "UnknownOperationException", "unknown operation %s", op)
}

// --- connections and hosts ---

func (st *codeConnectionsStore) createConnection(api *ccAPI, in map[string]any) (map[string]any, *ccError) {
	provider, hostARN := ccStr(in, "ProviderType"), ccStr(in, "HostArn")
	c := &ccConnection{id: uuid.NewString(), name: ccStr(in, "ConnectionName"), providerType: provider, status: "PENDING"}
	switch {
	case hostARN != "":
		h, err := st.hostByARN(hostARN)
		if err != nil {
			return nil, err
		}
		if h.status != "AVAILABLE" {
			return nil, ccErr(http.StatusBadRequest, "ResourceUnavailableException", "host %s is %s, not AVAILABLE", h.id, h.status)
		}
		if provider != "" && provider != h.providerType {
			return nil, ccValidation("ProviderType %s does not match the host's %s", provider, h.providerType)
		}
		c.hostID, c.providerType = h.id, h.providerType
	case provider == "":
		return nil, ccValidation("one of ProviderType or HostArn is required")
	case ccSelfManaged[provider]:
		return nil, ccValidation("a %s connection needs a HostArn", provider)
	}
	if len(st.connections) >= ccMaxResources {
		return nil, ccErr(http.StatusBadRequest, "LimitExceededException", "connection limit of %d reached", ccMaxResources)
	}
	tags := ccTagsIn(in["Tags"])
	if err := st.mergeTagsLocked("connection/"+c.id, tags); err != nil {
		return nil, err
	}
	c.seq = st.nextSeq()
	st.connections[c.id] = c
	return map[string]any{"ConnectionArn": api.arn("connection", c.id), "Tags": ccTagsOut(st.tags["connection/"+c.id])}, nil
}

func (st *codeConnectionsStore) connectionByARN(arn string) (*ccConnection, *ccError) {
	kind, id, ok := ccParseARN(arn)
	c := st.connections[id]
	if !ok || kind != "connection" || c == nil {
		return nil, ccNotFound("connection %s not found", arn)
	}
	return c, nil
}

func (st *codeConnectionsStore) connectionOut(api *ccAPI, c *ccConnection) map[string]any {
	out := map[string]any{
		"ConnectionName": c.name, "ConnectionArn": api.arn("connection", c.id), "ProviderType": c.providerType,
		"OwnerAccountId": ccAccountID, "ConnectionStatus": c.status,
	}
	if c.hostID != "" {
		out["HostArn"] = api.arn("host", c.hostID)
	}
	return out
}

func (st *codeConnectionsStore) listConnections(api *ccAPI, in map[string]any) (map[string]any, *ccError) {
	hostFilter := ""
	if arn := ccStr(in, "HostArnFilter"); arn != "" {
		h, err := st.hostByARN(arn)
		if err != nil {
			return nil, err
		}
		hostFilter = h.id
	}
	provider := ccStr(in, "ProviderTypeFilter")
	conns := make([]*ccConnection, 0, len(st.connections))
	for _, c := range st.connections {
		if (provider == "" || c.providerType == provider) && (hostFilter == "" || c.hostID == hostFilter) {
			conns = append(conns, c)
		}
	}
	sort.Slice(conns, func(i, j int) bool { return conns[i].seq < conns[j].seq })
	page, next, err := ccPage(conns, in)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(page))
	for _, c := range page {
		out = append(out, st.connectionOut(api, c))
	}
	return map[string]any{"Connections": out, "NextToken": next}, nil
}

func (st *codeConnectionsStore) createHost(api *ccAPI, in map[string]any) (map[string]any, *ccError) {
	provider := ccStr(in, "ProviderType")
	if !ccSelfManaged[provider] {
		return nil, ccValidation("hosts are only for GitHubEnterpriseServer or GitLabSelfManaged, not %s", provider)
	}
	if len(st.hosts) >= ccMaxResources {
		return nil, ccErr(http.StatusBadRequest, "LimitExceededException", "host limit of %d reached", ccMaxResources)
	}
	h := &ccHost{id: ccHostIDPrefix(ccStr(in, "Name")) + "-" + ccRandHex(4), name: ccStr(in, "Name"), providerType: provider, endpoint: ccStr(in, "ProviderEndpoint"), status: "PENDING"}
	if vpc, ok := in["VpcConfiguration"].(map[string]any); ok {
		h.vpc = vpc
	}
	if err := st.mergeTagsLocked("host/"+h.id, ccTagsIn(in["Tags"])); err != nil {
		return nil, err
	}
	h.seq = st.nextSeq()
	st.hosts[h.id] = h
	return map[string]any{"HostArn": api.arn("host", h.id), "Tags": ccTagsOut(st.tags["host/"+h.id])}, nil
}

// ccHostIDPrefix keeps a host ID ARN-safe: the name may hold any character.
func ccHostIDPrefix(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)
}

func (st *codeConnectionsStore) hostByARN(arn string) (*ccHost, *ccError) {
	kind, id, ok := ccParseARN(arn)
	h := st.hosts[id]
	if !ok || kind != "host" || h == nil {
		return nil, ccNotFound("host %s not found", arn)
	}
	return h, nil
}

func (st *codeConnectionsStore) hostOut(api *ccAPI, h *ccHost) map[string]any {
	out := map[string]any{
		"Name": h.name, "HostArn": api.arn("host", h.id), "ProviderType": h.providerType,
		"ProviderEndpoint": h.endpoint, "Status": h.status,
	}
	if h.vpc != nil {
		out["VpcConfiguration"] = h.vpc // Conform copies it before encoding
	}
	return out
}

func (st *codeConnectionsStore) activateConnection(api *ccAPI, id string) (map[string]any, *ccError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, arnID, ok := ccParseARN(id); ok {
		id = arnID
	}
	c := st.connections[id]
	if c == nil {
		return nil, ccNotFound("connection %s not found", id)
	}
	c.status = "AVAILABLE"
	return st.connectionOut(api, c), nil
}

func (st *codeConnectionsStore) activateHost(api *ccAPI, id string) (map[string]any, *ccError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, arnID, ok := ccParseARN(id); ok {
		id = arnID
	}
	h := st.hosts[id]
	if h == nil {
		return nil, ccNotFound("host %s not found", id)
	}
	h.status = "AVAILABLE"
	out := st.hostOut(api, h)
	delete(out, "VpcConfiguration")
	return out, nil
}

// --- repository links ---

func (st *codeConnectionsStore) availableConnection(arn string) (*ccConnection, *ccError) {
	kind, id, ok := ccParseARN(arn)
	c := st.connections[id]
	if !ok || kind != "connection" || c == nil {
		return nil, ccInvalid("connection %s not found", arn)
	}
	if c.status != "AVAILABLE" {
		return nil, ccInvalid("connection %s is %s, not AVAILABLE", arn, c.status)
	}
	return c, nil
}

func (st *codeConnectionsStore) createRepositoryLink(api *ccAPI, in map[string]any) (map[string]any, *ccError) {
	c, err := st.availableConnection(ccStr(in, "ConnectionArn"))
	if err != nil {
		return nil, err
	}
	owner, repo := ccStr(in, "OwnerId"), ccStr(in, "RepositoryName")
	for _, l := range st.links {
		if l.connectionID == c.id && l.ownerID == owner && l.repo == repo {
			return nil, ccErr(http.StatusBadRequest, "ResourceAlreadyExistsException", "repository %s/%s is already linked through this connection", owner, repo)
		}
	}
	if len(st.links) >= ccMaxResources {
		return nil, ccErr(http.StatusBadRequest, "LimitExceededException", "repository link limit of %d reached", ccMaxResources)
	}
	l := &ccLink{id: uuid.NewString(), connectionID: c.id, ownerID: owner, repo: repo, providerType: c.providerType, kmsKey: ccStr(in, "EncryptionKeyArn")}
	if err := st.mergeTagsLocked("repository-link/"+l.id, ccTagsIn(in["Tags"])); err != nil {
		return nil, err
	}
	l.seq = st.nextSeq()
	st.links[l.id] = l
	return map[string]any{"RepositoryLinkInfo": st.linkOut(api, l)}, nil
}

func (st *codeConnectionsStore) linkOut(api *ccAPI, l *ccLink) map[string]any {
	out := map[string]any{
		"ConnectionArn": api.arn("connection", l.connectionID), "OwnerId": l.ownerID, "ProviderType": l.providerType,
		"RepositoryLinkArn": api.arn("repository-link", l.id), "RepositoryLinkId": l.id, "RepositoryName": l.repo,
	}
	if l.kmsKey != "" {
		out["EncryptionKeyArn"] = l.kmsKey
	}
	return out
}

func (st *codeConnectionsStore) sortedLinks() []*ccLink {
	links := make([]*ccLink, 0, len(st.links))
	for _, l := range st.links {
		links = append(links, l)
	}
	sort.Slice(links, func(i, j int) bool { return links[i].seq < links[j].seq })
	return links
}

func (st *codeConnectionsStore) linkByID(id string) (*ccLink, *ccError) {
	l := st.links[id]
	if l == nil {
		return nil, ccNotFound("repository link %s not found", id)
	}
	return l, nil
}

func (st *codeConnectionsStore) repositoryLinkOp(api *ccAPI, op string, in map[string]any) (map[string]any, *ccError) {
	l, err := st.linkByID(ccStr(in, "RepositoryLinkId"))
	if err != nil {
		return nil, err
	}
	switch op {
	case "DeleteRepositoryLink":
		for _, s := range st.syncs {
			if s.linkID == l.id {
				return nil, ccErr(http.StatusBadRequest, "SyncConfigurationStillExistsException", "repository link %s still has sync configuration %s", l.id, s.resourceName)
			}
		}
		delete(st.links, l.id)
		delete(st.tags, "repository-link/"+l.id)
		return map[string]any{}, nil
	case "UpdateRepositoryLink":
		if arn := ccStr(in, "ConnectionArn"); arn != "" {
			c, err := st.availableConnection(arn)
			if err != nil {
				return nil, err
			}
			if c.providerType != l.providerType {
				return nil, ccInvalid("connection %s is %s, the repository is %s", arn, c.providerType, l.providerType)
			}
			l.connectionID = c.id
		}
		if key := ccStr(in, "EncryptionKeyArn"); key != "" {
			l.kmsKey = key
		}
	}
	return map[string]any{"RepositoryLinkInfo": st.linkOut(api, l)}, nil
}

// --- sync configurations ---

func ccSyncKey(syncType, resource string) string { return syncType + "/" + resource }

func (st *codeConnectionsStore) syncByKey(syncType, resource string) (*ccSync, *ccError) {
	s := st.syncs[ccSyncKey(syncType, resource)]
	if s == nil {
		return nil, ccNotFound("no %s sync configuration for resource %s", syncType, resource)
	}
	return s, nil
}

func (st *codeConnectionsStore) syncOut(s *ccSync) map[string]any {
	l := st.links[s.linkID]
	out := map[string]any{
		"Branch": s.branch, "ConfigFile": s.configFile, "RepositoryLinkId": s.linkID, "ResourceName": s.resourceName,
		"RoleArn": s.roleARN, "SyncType": s.syncType,
	}
	if l != nil {
		out["OwnerId"], out["ProviderType"], out["RepositoryName"] = l.ownerID, l.providerType, l.repo
	}
	for key, v := range map[string]string{"PublishDeploymentStatus": s.publish, "TriggerResourceUpdateOn": s.trigger, "PullRequestComment": s.prComment} {
		if v != "" {
			out[key] = v
		}
	}
	return out
}

func (st *codeConnectionsStore) createSyncConfiguration(in map[string]any) (map[string]any, *ccError) {
	key := ccSyncKey(ccStr(in, "SyncType"), ccStr(in, "ResourceName"))
	if st.syncs[key] != nil {
		return nil, ccErr(http.StatusBadRequest, "ResourceAlreadyExistsException", "resource %s already has a %s sync configuration", ccStr(in, "ResourceName"), ccStr(in, "SyncType"))
	}
	if st.links[ccStr(in, "RepositoryLinkId")] == nil {
		return nil, ccInvalid("repository link %s not found", ccStr(in, "RepositoryLinkId"))
	}
	if len(st.syncs) >= ccMaxResources {
		return nil, ccErr(http.StatusBadRequest, "LimitExceededException", "sync configuration limit of %d reached", ccMaxResources)
	}
	s := &ccSync{
		syncType: ccStr(in, "SyncType"), resourceName: ccStr(in, "ResourceName"), linkID: ccStr(in, "RepositoryLinkId"),
		branch: ccStr(in, "Branch"), configFile: ccStr(in, "ConfigFile"), roleARN: ccStr(in, "RoleArn"),
		publish: ccStr(in, "PublishDeploymentStatus"), trigger: ccStr(in, "TriggerResourceUpdateOn"), prComment: ccStr(in, "PullRequestComment"),
		updated: time.Now().UTC(), seq: st.nextSeq(),
	}
	st.syncs[key] = s
	return map[string]any{"SyncConfiguration": st.syncOut(s)}, nil
}

func (st *codeConnectionsStore) syncConfigurationOp(op string, in map[string]any) (map[string]any, *ccError) {
	syncType, resource := ccStr(in, "SyncType"), ccStr(in, "ResourceName")
	s := st.syncs[ccSyncKey(syncType, resource)]
	if s == nil {
		if op == "DeleteSyncConfiguration" { // the model lists no ResourceNotFoundException here
			return nil, ccInvalid("no %s sync configuration for resource %s", syncType, resource)
		}
		return nil, ccNotFound("no %s sync configuration for resource %s", syncType, resource)
	}
	switch op {
	case "DeleteSyncConfiguration":
		delete(st.syncs, ccSyncKey(syncType, resource))
		return map[string]any{}, nil
	case "UpdateSyncConfiguration":
		if id := ccStr(in, "RepositoryLinkId"); id != "" {
			if st.links[id] == nil {
				return nil, ccNotFound("repository link %s not found", id)
			}
			s.linkID = id
		}
		for key, field := range map[string]*string{
			"Branch": &s.branch, "ConfigFile": &s.configFile, "RoleArn": &s.roleARN, "PublishDeploymentStatus": &s.publish,
			"TriggerResourceUpdateOn": &s.trigger, "PullRequestComment": &s.prComment,
		} {
			if v := ccStr(in, key); v != "" {
				*field = v
			}
		}
		s.updated = time.Now().UTC()
	}
	return map[string]any{"SyncConfiguration": st.syncOut(s)}, nil
}

func (st *codeConnectionsStore) listSyncs(op string, in map[string]any) (map[string]any, *ccError) {
	l, err := st.linkByID(ccStr(in, "RepositoryLinkId"))
	if err != nil {
		return nil, err
	}
	syncs := []*ccSync{}
	for _, s := range st.syncs {
		if s.linkID == l.id && s.syncType == ccStr(in, "SyncType") {
			syncs = append(syncs, s)
		}
	}
	sort.Slice(syncs, func(i, j int) bool { return syncs[i].seq < syncs[j].seq })
	page, next, perr := ccPage(syncs, in)
	if perr != nil {
		return nil, perr
	}
	out := make([]any, 0, len(page))
	for _, s := range page {
		if op == "ListRepositorySyncDefinitions" {
			// ponytail: one definition per sync configuration, read from the
			// config rather than from the repository's deployment files.
			out = append(out, map[string]any{"Branch": s.branch, "Directory": path.Dir(s.configFile), "Parent": s.resourceName, "Target": s.resourceName})
		} else {
			out = append(out, st.syncOut(s))
		}
	}
	if op == "ListRepositorySyncDefinitions" {
		return map[string]any{"RepositorySyncDefinitions": out, "NextToken": next}, nil
	}
	return map[string]any{"SyncConfigurations": out, "NextToken": next}, nil
}

// --- sync status and blockers ---

// ccSha is a stable stand-in commit for a sync configuration's current state.
func ccSha(s *ccSync) string {
	sum := sha1.Sum([]byte(s.linkID + "\x00" + s.branch + "\x00" + s.configFile + "\x00" + s.updated.String()))
	return hex.EncodeToString(sum[:])
}

func ccSyncEvents(s *ccSync) []any {
	t := ccEpoch(s.updated)
	return []any{
		map[string]any{"Event": "Sync initiated for branch " + s.branch, "Time": t, "Type": "INFO"},
		map[string]any{"Event": "Sync succeeded", "Time": t, "Type": "INFO"},
	}
}

func (st *codeConnectionsStore) repositorySyncStatus(in map[string]any) (map[string]any, *ccError) {
	l, err := st.linkByID(ccStr(in, "RepositoryLinkId"))
	if err != nil {
		return nil, err
	}
	var latest *ccSync
	for _, s := range st.syncs {
		if s.linkID == l.id && s.branch == ccStr(in, "Branch") && s.syncType == ccStr(in, "SyncType") && (latest == nil || s.updated.After(latest.updated)) {
			latest = s
		}
	}
	if latest == nil {
		return nil, ccNotFound("no sync has run for branch %s of repository link %s", ccStr(in, "Branch"), l.id)
	}
	return map[string]any{"LatestSync": map[string]any{"StartedAt": ccEpoch(latest.updated), "Status": "SUCCEEDED", "Events": ccSyncEvents(latest)}}, nil
}

func (st *codeConnectionsStore) resourceSyncStatus(in map[string]any) (map[string]any, *ccError) {
	s, err := st.syncByKey(ccStr(in, "SyncType"), ccStr(in, "ResourceName"))
	if err != nil {
		return nil, err
	}
	l := st.links[s.linkID]
	revision := map[string]any{"Branch": s.branch, "Directory": s.configFile, "Sha": ccSha(s)}
	if l != nil {
		revision["OwnerId"], revision["RepositoryName"], revision["ProviderType"] = l.ownerID, l.repo, l.providerType
	}
	attempt := func() map[string]any {
		return map[string]any{
			"Events": ccSyncEvents(s), "InitialRevision": revision, "StartedAt": ccEpoch(s.updated),
			"Status": "SUCCEEDED", "TargetRevision": revision, "Target": s.resourceName,
		}
	}
	return map[string]any{"DesiredState": revision, "LatestSync": attempt(), "LatestSuccessfulSync": attempt()}, nil
}

func ccBlockerOut(b *ccBlocker) map[string]any {
	out := map[string]any{"Id": b.id, "Type": "AUTOMATED", "Status": "ACTIVE", "CreatedReason": b.reason, "CreatedAt": ccEpoch(b.created)}
	if !b.resolved.IsZero() {
		out["Status"], out["ResolvedReason"], out["ResolvedAt"] = "RESOLVED", b.resolvedReason, ccEpoch(b.resolved)
	}
	return out
}

func (st *codeConnectionsStore) syncBlockerOp(op string, in map[string]any) (map[string]any, *ccError) {
	s, err := st.syncByKey(ccStr(in, "SyncType"), ccStr(in, "ResourceName"))
	if err != nil {
		return nil, err
	}
	if op == "GetSyncBlockerSummary" {
		blockers := make([]any, 0, len(s.blockers))
		for _, b := range s.blockers {
			blockers = append(blockers, ccBlockerOut(b))
		}
		return map[string]any{"SyncBlockerSummary": map[string]any{"ResourceName": s.resourceName, "LatestBlockers": blockers}}, nil
	}
	for _, b := range s.blockers {
		if b.id != ccStr(in, "Id") {
			continue
		}
		if !b.resolved.IsZero() {
			return nil, ccInvalid("sync blocker %s is already resolved", b.id)
		}
		b.resolved, b.resolvedReason = time.Now().UTC(), ccStr(in, "ResolvedReason")
		return map[string]any{"ResourceName": s.resourceName, "SyncBlocker": ccBlockerOut(b)}, nil
	}
	return nil, ccErr(http.StatusBadRequest, "SyncBlockerDoesNotExistException", "sync blocker %s does not exist", ccStr(in, "Id"))
}

func (st *codeConnectionsStore) addSyncBlocker(syncType, resource, reason string) (map[string]any, *ccError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, err := st.syncByKey(syncType, resource)
	if err != nil {
		return nil, err
	}
	if len(s.blockers) >= ccMaxBlockersPerCfg {
		return nil, ccErr(http.StatusBadRequest, "LimitExceededException", "a sync configuration can have at most %d blockers", ccMaxBlockersPerCfg)
	}
	if reason == "" {
		reason = "Stackyard: sync failed"
	}
	if len(reason) > 1024 {
		return nil, ccValidation("CreatedReason must be at most 1024 characters")
	}
	b := &ccBlocker{id: ccRandHex(8), reason: reason, created: time.Now().UTC()}
	s.blockers = append(s.blockers, b)
	return ccBlockerOut(b), nil
}

// --- tagging ---

func (st *codeConnectionsStore) tagOp(op string, in map[string]any) (map[string]any, *ccError) {
	arn := ccStr(in, "ResourceArn")
	kind, id, ok := ccParseARN(arn)
	exists := false
	switch kind {
	case "connection":
		exists = st.connections[id] != nil
	case "host":
		exists = st.hosts[id] != nil
	case "repository-link":
		exists = st.links[id] != nil
	}
	if !ok || !exists {
		return nil, ccNotFound("resource %s not found", arn)
	}
	key := kind + "/" + id
	switch op {
	case "TagResource":
		if err := st.mergeTagsLocked(key, ccTagsIn(in["Tags"])); err != nil {
			return nil, err
		}
	case "UntagResource":
		drop := map[string]bool{}
		for _, k := range ccListStrings(in["TagKeys"]) {
			drop[k] = true
		}
		kept := []ccTag{}
		for _, t := range st.tags[key] {
			if !drop[t.Key] {
				kept = append(kept, t)
			}
		}
		st.tags[key] = kept
	case "ListTagsForResource":
		return map[string]any{"Tags": ccTagsOut(st.tags[key])}, nil
	}
	return map[string]any{}, nil
}

func ccListStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
