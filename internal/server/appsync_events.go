package server

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AppSync Events APIs (the /v2/apis surface): Api and ChannelNamespace
// resources. The GraphQL-side operations stay on the tags-only stub in
// appsync_store.go.

const (
	appSyncEventsAccount      = "123456789012"
	appSyncEventsRegion       = "us-east-1"
	appSyncMaxEventAPIs       = 100
	appSyncMaxNamespaces      = 50
	appSyncMaxListPage        = 25
	appSyncMaxCodeBytes       = 32768
	appSyncMaxAuthEntries     = 25
	appSyncMaxOwnerContactLen = 250
	appSyncMaxTags            = 50
)

var (
	appSyncAPINamePattern      = regexp.MustCompile(`^[A-Za-z0-9_\- ]{1,50}$`)
	appSyncNamespacePattern    = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9\-]{0,48}[A-Za-z0-9])?$`)
	appSyncOwnerContactPattern = regexp.MustCompile(`^[A-Za-z0-9_\- .]*$`)
	appSyncAuthTypes           = map[string]bool{"API_KEY": true, "AWS_IAM": true, "AMAZON_COGNITO_USER_POOLS": true, "OPENID_CONNECT": true, "AWS_LAMBDA": true}
	appSyncLogLevels           = map[string]bool{"NONE": true, "ERROR": true, "ALL": true, "INFO": true, "DEBUG": true}
	appSyncHandlerBehaviors    = map[string]bool{"CODE": true, "DIRECT": true}
	appSyncInvokeTypes         = map[string]bool{"REQUEST_RESPONSE": true, "EVENT": true}
)

var appSyncEventsActions = map[string]bool{
	"CreateApi": true, "GetApi": true, "UpdateApi": true, "DeleteApi": true, "ListApis": true,
	"CreateChannelNamespace": true, "GetChannelNamespace": true, "UpdateChannelNamespace": true,
	"DeleteChannelNamespace": true, "ListChannelNamespaces": true,
}

type appSyncEventsError struct {
	status int
	code   string
	msg    string
}

func appSyncBadRequest(msg string) *appSyncEventsError {
	return &appSyncEventsError{http.StatusBadRequest, "BadRequestException", msg}
}

func appSyncNotFound(msg string) *appSyncEventsError {
	return &appSyncEventsError{http.StatusNotFound, "NotFoundException", msg}
}

// Stored values (eventConfig, auth modes, handler configs) are rebuilt from
// known members on write and replaced wholesale on update, never mutated in
// place, so responses may share them after the lock is released.
type appSyncEventAPI struct {
	id, name, owner, arn string
	created              float64
	eventConfig          map[string]any
}

type appSyncNamespace struct {
	name, arn         string
	created, modified float64
	subscribe         []any
	publish           []any
	code              string
	handlers          map[string]any
}

func (s *appSyncStore) handleEvents(action string, payload map[string]any, params map[string]string, query url.Values) (map[string]any, *appSyncEventsError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.eventAPIs == nil {
		s.eventAPIs = map[string]*appSyncEventAPI{}
		s.namespaces = map[string]map[string]*appSyncNamespace{}
	}
	apiID := params["apiId"]
	switch action {
	case "CreateApi":
		return s.createEventAPI(payload)
	case "ListApis":
		return s.listEventAPIs(query)
	}
	api := s.eventAPIs[apiID]
	if api == nil {
		return nil, appSyncNotFound("API " + apiID + " not found")
	}
	switch action {
	case "GetApi":
		return map[string]any{"api": s.eventAPIShape(api)}, nil
	case "UpdateApi":
		return s.updateEventAPI(api, payload)
	case "DeleteApi":
		for _, ns := range s.namespaces[api.id] {
			delete(s.tags, ns.arn)
		}
		delete(s.namespaces, api.id)
		delete(s.eventAPIs, api.id)
		delete(s.tags, api.arn)
		return map[string]any{}, nil
	case "CreateChannelNamespace":
		return s.createNamespace(api, payload)
	case "ListChannelNamespaces":
		return s.listNamespaces(api, query)
	}
	name := params["name"]
	ns := s.namespaces[api.id][name]
	if ns == nil {
		return nil, appSyncNotFound("channel namespace " + name + " not found")
	}
	switch action {
	case "GetChannelNamespace":
		return map[string]any{"channelNamespace": s.namespaceShape(api, ns)}, nil
	case "UpdateChannelNamespace":
		if err := s.applyNamespace(ns, payload); err != nil {
			return nil, err
		}
		ns.modified = appSyncNow()
		return map[string]any{"channelNamespace": s.namespaceShape(api, ns)}, nil
	case "DeleteChannelNamespace":
		delete(s.namespaces[api.id], name)
		delete(s.tags, ns.arn)
		return map[string]any{}, nil
	}
	return nil, appSyncBadRequest("unknown action")
}

func (s *appSyncStore) createEventAPI(payload map[string]any) (map[string]any, *appSyncEventsError) {
	name, owner, cfg, err := appSyncAPIFields(payload)
	if err != nil {
		return nil, err
	}
	tags, err := appSyncValidTags(payload["tags"])
	if err != nil {
		return nil, err
	}
	if len(s.eventAPIs) >= appSyncMaxEventAPIs {
		return nil, &appSyncEventsError{http.StatusBadRequest, "ServiceQuotaExceededException", "too many Event APIs"}
	}
	id := appSyncNewID()
	api := &appSyncEventAPI{
		id: id, name: name, owner: owner, created: appSyncNow(), eventConfig: cfg,
		arn: fmt.Sprintf("arn:aws:appsync:%s:%s:apis/%s", appSyncEventsRegion, appSyncEventsAccount, id),
	}
	s.eventAPIs[id] = api
	s.namespaces[id] = map[string]*appSyncNamespace{}
	s.tags[api.arn] = tags
	return map[string]any{"api": s.eventAPIShape(api)}, nil
}

func (s *appSyncStore) updateEventAPI(api *appSyncEventAPI, payload map[string]any) (map[string]any, *appSyncEventsError) {
	name, owner, cfg, err := appSyncAPIFields(payload)
	if err != nil {
		return nil, err
	}
	api.name, api.owner, api.eventConfig = name, owner, cfg
	return map[string]any{"api": s.eventAPIShape(api)}, nil
}

func appSyncAPIFields(payload map[string]any) (string, string, map[string]any, *appSyncEventsError) {
	name, _ := payload["name"].(string)
	if !appSyncAPINamePattern.MatchString(name) {
		return "", "", nil, appSyncBadRequest("name must be 1-50 letters, numbers, spaces, hyphens or underscores")
	}
	owner, _ := payload["ownerContact"].(string)
	if len(owner) > appSyncMaxOwnerContactLen || !appSyncOwnerContactPattern.MatchString(owner) {
		return "", "", nil, appSyncBadRequest("ownerContact is invalid")
	}
	raw, ok := payload["eventConfig"].(map[string]any)
	if !ok {
		return "", "", nil, appSyncBadRequest("eventConfig is required")
	}
	cfg := map[string]any{}
	providers, ok := raw["authProviders"].([]any)
	if !ok || len(providers) == 0 || len(providers) > appSyncMaxAuthEntries {
		return "", "", nil, appSyncBadRequest("eventConfig.authProviders must list 1-25 providers")
	}
	clean := make([]any, 0, len(providers))
	for _, p := range providers {
		pm, _ := p.(map[string]any)
		authType, _ := pm["authType"].(string)
		if !appSyncAuthTypes[authType] {
			return "", "", nil, appSyncBadRequest("authProviders[].authType is invalid")
		}
		out := map[string]any{"authType": authType}
		for _, k := range []string{"cognitoConfig", "openIDConnectConfig", "lambdaAuthorizerConfig"} {
			if m, ok := pm[k].(map[string]any); ok {
				out[k] = m
			}
		}
		clean = append(clean, out)
	}
	cfg["authProviders"] = clean
	for _, k := range []string{"connectionAuthModes", "defaultPublishAuthModes", "defaultSubscribeAuthModes"} {
		modes, err := appSyncAuthModes(raw[k], k, true)
		if err != nil {
			return "", "", nil, err
		}
		cfg[k] = modes
	}
	if lc, ok := raw["logConfig"].(map[string]any); ok {
		level, _ := lc["logLevel"].(string)
		role, _ := lc["cloudWatchLogsRoleArn"].(string)
		if !appSyncLogLevels[level] || role == "" {
			return "", "", nil, appSyncBadRequest("eventConfig.logConfig needs a valid logLevel and cloudWatchLogsRoleArn")
		}
		cfg["logConfig"] = map[string]any{"logLevel": level, "cloudWatchLogsRoleArn": role}
	}
	return name, owner, cfg, nil
}

func appSyncAuthModes(v any, field string, required bool) ([]any, *appSyncEventsError) {
	if v == nil && !required {
		return nil, nil
	}
	list, ok := v.([]any)
	if !ok || (required && len(list) == 0) || len(list) > appSyncMaxAuthEntries {
		return nil, appSyncBadRequest(field + " must list 1-25 auth modes")
	}
	out := make([]any, 0, len(list))
	for _, m := range list {
		mm, _ := m.(map[string]any)
		authType, _ := mm["authType"].(string)
		if !appSyncAuthTypes[authType] {
			return nil, appSyncBadRequest(field + "[].authType is invalid")
		}
		out = append(out, map[string]any{"authType": authType})
	}
	return out, nil
}

func (s *appSyncStore) createNamespace(api *appSyncEventAPI, payload map[string]any) (map[string]any, *appSyncEventsError) {
	name, _ := payload["name"].(string)
	if !appSyncNamespacePattern.MatchString(name) {
		return nil, appSyncBadRequest("name must be 1-50 letters, numbers or inner hyphens")
	}
	if s.namespaces[api.id][name] != nil {
		return nil, &appSyncEventsError{http.StatusConflict, "ConflictException", "channel namespace " + name + " already exists"}
	}
	if len(s.namespaces[api.id]) >= appSyncMaxNamespaces {
		return nil, &appSyncEventsError{http.StatusBadRequest, "ServiceQuotaExceededException", "too many channel namespaces"}
	}
	tags, err := appSyncValidTags(payload["tags"])
	if err != nil {
		return nil, err
	}
	now := appSyncNow()
	ns := &appSyncNamespace{name: name, created: now, modified: now, arn: api.arn + "/channelNamespace/" + name}
	if err := s.applyNamespace(ns, payload); err != nil {
		return nil, err
	}
	s.namespaces[api.id][name] = ns
	s.tags[ns.arn] = tags
	return map[string]any{"channelNamespace": s.namespaceShape(api, ns)}, nil
}

// applyNamespace validates every optional member before changing ns.
func (s *appSyncStore) applyNamespace(ns *appSyncNamespace, payload map[string]any) *appSyncEventsError {
	sub, err := appSyncAuthModes(payload["subscribeAuthModes"], "subscribeAuthModes", false)
	if err != nil {
		return err
	}
	pub, err := appSyncAuthModes(payload["publishAuthModes"], "publishAuthModes", false)
	if err != nil {
		return err
	}
	code, _ := payload["codeHandlers"].(string)
	if _, present := payload["codeHandlers"]; present && (code == "" || len(code) > appSyncMaxCodeBytes) {
		return appSyncBadRequest("codeHandlers must be 1-32768 characters")
	}
	var handlers map[string]any
	if raw, ok := payload["handlerConfigs"].(map[string]any); ok {
		handlers = map[string]any{}
		for _, k := range []string{"onPublish", "onSubscribe"} {
			hc, ok := raw[k].(map[string]any)
			if !ok {
				continue
			}
			behavior, _ := hc["behavior"].(string)
			integ, _ := hc["integration"].(map[string]any)
			ds, _ := integ["dataSourceName"].(string)
			if !appSyncHandlerBehaviors[behavior] || ds == "" {
				return appSyncBadRequest("handlerConfigs." + k + " needs a valid behavior and integration.dataSourceName")
			}
			outInteg := map[string]any{"dataSourceName": ds}
			if lc, ok := integ["lambdaConfig"].(map[string]any); ok {
				it, _ := lc["invokeType"].(string)
				if it != "" && !appSyncInvokeTypes[it] {
					return appSyncBadRequest("lambdaConfig.invokeType is invalid")
				}
				outInteg["lambdaConfig"] = map[string]any{"invokeType": it}
			}
			handlers[k] = map[string]any{"behavior": behavior, "integration": outInteg}
		}
	}
	ns.subscribe, ns.publish, ns.code, ns.handlers = sub, pub, code, handlers
	return nil
}

func appSyncValidTags(v any) (map[string]string, *appSyncEventsError) {
	raw, _ := v.(map[string]any)
	if len(raw) > appSyncMaxTags {
		return nil, appSyncBadRequest("at most 50 tags")
	}
	out := make(map[string]string, len(raw))
	for k, val := range raw {
		text, ok := val.(string)
		if !ok || k == "" || len(k) > 128 || len(text) > 256 || strings.HasPrefix(k, "aws:") {
			return nil, appSyncBadRequest("invalid tag " + k)
		}
		out[k] = text
	}
	return out, nil
}

func (s *appSyncStore) eventAPIShape(api *appSyncEventAPI) map[string]any {
	out := map[string]any{
		"apiId": api.id, "name": api.name, "apiArn": api.arn, "created": api.created,
		"xrayEnabled": false, "eventConfig": api.eventConfig, "tags": appSyncCloneTags(s.tags[api.arn]),
		"dns": map[string]any{
			"HTTP":     api.id + ".appsync-api." + appSyncEventsRegion + ".amazonaws.com",
			"REALTIME": api.id + ".appsync-realtime-api." + appSyncEventsRegion + ".amazonaws.com",
		},
	}
	if api.owner != "" {
		out["ownerContact"] = api.owner
	}
	return out
}

func (s *appSyncStore) namespaceShape(api *appSyncEventAPI, ns *appSyncNamespace) map[string]any {
	out := map[string]any{
		"apiId": api.id, "name": ns.name, "channelNamespaceArn": ns.arn, "created": ns.created,
		"lastModified": ns.modified, "tags": appSyncCloneTags(s.tags[ns.arn]),
	}
	if ns.subscribe != nil {
		out["subscribeAuthModes"] = ns.subscribe
	}
	if ns.publish != nil {
		out["publishAuthModes"] = ns.publish
	}
	if ns.code != "" {
		out["codeHandlers"] = ns.code
	}
	if ns.handlers != nil {
		out["handlerConfigs"] = ns.handlers
	}
	return out
}

func (s *appSyncStore) listEventAPIs(query url.Values) (map[string]any, *appSyncEventsError) {
	ids := make([]string, 0, len(s.eventAPIs))
	for id := range s.eventAPIs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	page, next, err := appSyncPage(ids, query)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, id := range page {
		items = append(items, s.eventAPIShape(s.eventAPIs[id]))
	}
	return appSyncWithToken(map[string]any{"apis": items}, next), nil
}

func (s *appSyncStore) listNamespaces(api *appSyncEventAPI, query url.Values) (map[string]any, *appSyncEventsError) {
	names := make([]string, 0, len(s.namespaces[api.id]))
	for n := range s.namespaces[api.id] {
		names = append(names, n)
	}
	sort.Strings(names)
	page, next, err := appSyncPage(names, query)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, n := range page {
		items = append(items, s.namespaceShape(api, s.namespaces[api.id][n]))
	}
	return appSyncWithToken(map[string]any{"channelNamespaces": items}, next), nil
}

// appSyncPage pages sorted keys; the token is the decimal offset.
func appSyncPage(keys []string, query url.Values) ([]string, string, *appSyncEventsError) {
	limit := appSyncMaxListPage
	if raw := query.Get("maxResults"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 0 || n > appSyncMaxListPage {
			return nil, "", appSyncBadRequest("maxResults must be between 0 and 25")
		}
		if n > 0 {
			limit = int(n)
		}
	}
	start := 0
	if raw := query.Get("nextToken"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 0 || int(n) > len(keys) {
			return nil, "", appSyncBadRequest("invalid nextToken")
		}
		start = int(n)
	}
	end := min(start+limit, len(keys))
	next := ""
	if end < len(keys) {
		next = strconv.Itoa(end)
	}
	return keys[start:end], next, nil
}

func appSyncWithToken(out map[string]any, next string) map[string]any {
	if next != "" {
		out["nextToken"] = next
	}
	return out
}

func appSyncNow() float64 {
	return float64(time.Now().UnixMilli()) / 1000
}

func appSyncNewID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 26)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}
