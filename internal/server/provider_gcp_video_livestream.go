package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const gcpVideoLivestreamGRPCPathPrefix = "/gcp/google.cloud.video.livestream.v1.LivestreamService/"

var (
	gcpVideoLivestreamReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	gcpVideoLivestreamProjectPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]{2,62}$`)
	gcpVideoLivestreamLocationPattern    = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)
	gcpVideoLivestreamIDPattern          = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{0,61}[a-z0-9])?$`)
	gcpVideoLivestreamOperationIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
)

type gcpVideoLivestreamRouteContext struct {
	Parent string
	Name   string
	Query  url.Values
}

func (s *Server) handleGCPVideoLivestreamRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_video_livestream(w, r) {
		return true
	}

	path := normalizeGCPVideoLivestreamPath(rawRequestPath(r))
	if isGCPVideoLivestreamLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVideoLivestreamListLocations(w, r, path) {
			return true
		}
		if handleGCPVideoLivestreamGetLocation(w, path) {
			return true
		}
		return false
	}

	if strings.HasPrefix(path, gcpVideoLivestreamGRPCPathPrefix) {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPVideoLivestreamJSONBody(w, r, path)
		if !ok {
			return true
		}
		method := strings.TrimSpace(strings.TrimPrefix(path, gcpVideoLivestreamGRPCPathPrefix))
		if method == "" {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		if handleGCPVideoLivestreamRPCMethod(w, path, method, body, gcpVideoLivestreamRouteContext{Query: r.URL.Query()}) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if !isGCPVideoLivestreamPath(path, hasGCPVideoLivestreamHint(r)) {
		return false
	}

	method, ctx, needsBody, ok := mapGCPVideoLivestreamRESTToMethod(r, path)
	if !ok {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	body := map[string]any{}
	if needsBody {
		var valid bool
		body, valid = decodeGCPVideoLivestreamJSONBody(w, r, path)
		if !valid {
			return true
		}
	}

	if handleGCPVideoLivestreamRPCMethod(w, path, method, body, ctx) {
		return true
	}
	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func normalizeGCPVideoLivestreamPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVideoLivestreamHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "video-livestream",
		"video_livestream",
		"video-livestream-apiv1",
		"video_livestream_apiv1",
		"livestream",
		"livestream-apiv1",
		"gcp-video-livestream",
		"gcp-livestream":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-video-livestream-apiv1") || strings.Contains(ua, "cloud.google.com/go/video/livestream/apiv1")
}

func isGCPVideoLivestreamLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVideoLivestreamHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPVideoLivestreamPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpVideoLivestreamGRPCPathPrefix) {
		return true
	}
	if !includeHint {
		return false
	}
	if _, _, _, ok := parseGCPProjectLocationPath(path); ok {
		return true
	}
	_, _, tail, ok := parseGCPVideoLivestreamLocationTail(path)
	if !ok {
		return strings.HasPrefix(path, "/gcp/v1/projects/")
	}
	parts := gcpVideoLivestreamTailParts(tail)
	if len(parts) == 0 {
		return true
	}
	switch parts[0] {
	case "channels", "inputs", "assets", "pools":
		return true
	case "operations":
		return true
	default:
		return true
	}
}

func mapGCPVideoLivestreamRESTToMethod(r *http.Request, path string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	ctx := gcpVideoLivestreamRouteContext{Query: r.URL.Query()}

	project, location, list, ok := parseGCPProjectLocationPath(path)
	if ok {
		if list && r.Method == http.MethodGet {
			ctx.Name = "projects/" + project
			return "ListLocations", ctx, false, true
		}
		if !list && r.Method == http.MethodGet {
			ctx.Name = fmt.Sprintf("projects/%s/locations/%s", project, location)
			return "GetLocation", ctx, false, true
		}
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}

	project, location, tail, ok := parseGCPVideoLivestreamLocationTail(path)
	if !ok {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Parent = fmt.Sprintf("projects/%s/locations/%s", project, location)

	parts := gcpVideoLivestreamTailParts(tail)
	if len(parts) == 0 {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}

	switch parts[0] {
	case "channels":
		return mapGCPVideoLivestreamRESTChannelMethod(r, ctx, parts)
	case "inputs":
		return mapGCPVideoLivestreamRESTInputMethod(r, ctx, parts)
	case "assets":
		return mapGCPVideoLivestreamRESTAssetMethod(r, ctx, parts)
	case "pools":
		return mapGCPVideoLivestreamRESTPoolMethod(r, ctx, parts)
	case "operations":
		return mapGCPVideoLivestreamRESTOperationMethod(r, ctx, parts)
	default:
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
}

func mapGCPVideoLivestreamRESTChannelMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return "ListChannels", ctx, false, true
		case http.MethodPost:
			return "CreateChannel", ctx, true, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}

	channelID, channelAction, channelHasAction := gcpVideoLivestreamSplitIDAction(parts[1])
	if strings.TrimSpace(channelID) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	channelName := ctx.Parent + "/channels/" + channelID

	if len(parts) == 2 {
		if channelHasAction {
			ctx.Name = channelName
			switch {
			case r.Method == http.MethodPost && channelAction == "start":
				return "StartChannel", ctx, true, true
			case r.Method == http.MethodPost && channelAction == "stop":
				return "StopChannel", ctx, true, true
			default:
				return "", gcpVideoLivestreamRouteContext{}, false, false
			}
		}
		ctx.Name = channelName
		switch r.Method {
		case http.MethodGet:
			return "GetChannel", ctx, false, true
		case http.MethodPatch:
			return "UpdateChannel", ctx, true, true
		case http.MethodDelete:
			return "DeleteChannel", ctx, false, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}

	if len(parts) == 4 && parts[2] == "distributions" {
		distributionID, action, hasAction := gcpVideoLivestreamSplitIDAction(parts[3])
		if !hasAction || strings.TrimSpace(distributionID) == "" {
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
		ctx.Name = channelName + "/distributions/" + distributionID
		if r.Method == http.MethodPost && action == "start" {
			return "StartDistribution", ctx, true, true
		}
		if r.Method == http.MethodPost && action == "stop" {
			return "StopDistribution", ctx, true, true
		}
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}

	if len(parts) >= 3 && parts[2] == "events" {
		return mapGCPVideoLivestreamRESTEventMethod(r, ctx, channelName, parts[3:])
	}
	if len(parts) >= 3 && parts[2] == "clips" {
		return mapGCPVideoLivestreamRESTClipMethod(r, ctx, channelName, parts[3:])
	}
	if len(parts) >= 3 && parts[2] == "dvrSessions" {
		return mapGCPVideoLivestreamRESTDvrSessionMethod(r, ctx, channelName, parts[3:])
	}

	return "", gcpVideoLivestreamRouteContext{}, false, false
}

func mapGCPVideoLivestreamRESTEventMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, channelName string, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) == 0 {
		ctx.Parent = channelName
		switch r.Method {
		case http.MethodGet:
			return "ListEvents", ctx, false, true
		case http.MethodPost:
			return "CreateEvent", ctx, true, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}
	if len(parts) != 1 || strings.TrimSpace(parts[0]) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Name = channelName + "/events/" + strings.TrimSpace(parts[0])
	switch r.Method {
	case http.MethodGet:
		return "GetEvent", ctx, false, true
	case http.MethodDelete:
		return "DeleteEvent", ctx, false, true
	default:
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
}

func mapGCPVideoLivestreamRESTClipMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, channelName string, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) == 0 {
		ctx.Parent = channelName
		switch r.Method {
		case http.MethodGet:
			return "ListClips", ctx, false, true
		case http.MethodPost:
			return "CreateClip", ctx, true, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}
	if len(parts) != 1 || strings.TrimSpace(parts[0]) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Name = channelName + "/clips/" + strings.TrimSpace(parts[0])
	switch r.Method {
	case http.MethodGet:
		return "GetClip", ctx, false, true
	case http.MethodDelete:
		return "DeleteClip", ctx, false, true
	default:
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
}

func mapGCPVideoLivestreamRESTDvrSessionMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, channelName string, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) == 0 {
		ctx.Parent = channelName
		switch r.Method {
		case http.MethodGet:
			return "ListDvrSessions", ctx, false, true
		case http.MethodPost:
			return "CreateDvrSession", ctx, true, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}
	if len(parts) != 1 || strings.TrimSpace(parts[0]) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Name = channelName + "/dvrSessions/" + strings.TrimSpace(parts[0])
	switch r.Method {
	case http.MethodGet:
		return "GetDvrSession", ctx, false, true
	case http.MethodPatch:
		return "UpdateDvrSession", ctx, true, true
	case http.MethodDelete:
		return "DeleteDvrSession", ctx, false, true
	default:
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
}

func mapGCPVideoLivestreamRESTInputMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return "ListInputs", ctx, false, true
		case http.MethodPost:
			return "CreateInput", ctx, true, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}
	inputID, action, hasAction := gcpVideoLivestreamSplitIDAction(parts[1])
	if strings.TrimSpace(inputID) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Name = ctx.Parent + "/inputs/" + inputID
	if len(parts) == 2 {
		if hasAction {
			if r.Method == http.MethodPost && action == "preview" {
				return "PreviewInput", ctx, true, true
			}
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
		switch r.Method {
		case http.MethodGet:
			return "GetInput", ctx, false, true
		case http.MethodPatch:
			return "UpdateInput", ctx, true, true
		case http.MethodDelete:
			return "DeleteInput", ctx, false, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}
	return "", gcpVideoLivestreamRouteContext{}, false, false
}

func mapGCPVideoLivestreamRESTAssetMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return "ListAssets", ctx, false, true
		case http.MethodPost:
			return "CreateAsset", ctx, true, true
		default:
			return "", gcpVideoLivestreamRouteContext{}, false, false
		}
	}
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Name = ctx.Parent + "/assets/" + strings.TrimSpace(parts[1])
	switch r.Method {
	case http.MethodGet:
		return "GetAsset", ctx, false, true
	case http.MethodDelete:
		return "DeleteAsset", ctx, false, true
	default:
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
}

func mapGCPVideoLivestreamRESTPoolMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Name = ctx.Parent + "/pools/" + strings.TrimSpace(parts[1])
	switch r.Method {
	case http.MethodGet:
		return "GetPool", ctx, false, true
	case http.MethodPatch:
		return "UpdatePool", ctx, true, true
	default:
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
}

func mapGCPVideoLivestreamRESTOperationMethod(r *http.Request, ctx gcpVideoLivestreamRouteContext, parts []string) (string, gcpVideoLivestreamRouteContext, bool, bool) {
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			ctx.Name = ctx.Parent
			return "ListOperations", ctx, false, true
		}
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	if len(parts) != 2 {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	operationID, action, hasAction := gcpVideoLivestreamSplitIDAction(parts[1])
	if strings.TrimSpace(operationID) == "" {
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	ctx.Name = ctx.Parent + "/operations/" + operationID
	if hasAction {
		if r.Method == http.MethodPost && action == "cancel" {
			return "CancelOperation", ctx, true, true
		}
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
	switch r.Method {
	case http.MethodGet:
		return "GetOperation", ctx, false, true
	case http.MethodDelete:
		return "DeleteOperation", ctx, false, true
	default:
		return "", gcpVideoLivestreamRouteContext{}, false, false
	}
}

func handleGCPVideoLivestreamRPCMethod(w http.ResponseWriter, path, method string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	switch method {
	case "ListLocations":
		return handleGCPVideoLivestreamListLocationsByMethod(w, path, body, ctx)
	case "GetLocation":
		return handleGCPVideoLivestreamGetLocationByMethod(w, path, body, ctx)
	case "CreateChannel":
		return handleGCPVideoLivestreamCreateChannel(w, path, body, ctx)
	case "ListChannels":
		return handleGCPVideoLivestreamListChannels(w, path, body, ctx)
	case "GetChannel":
		return handleGCPVideoLivestreamGetChannel(w, path, body, ctx)
	case "DeleteChannel":
		return handleGCPVideoLivestreamDeleteChannel(w, path, body, ctx)
	case "UpdateChannel":
		return handleGCPVideoLivestreamUpdateChannel(w, path, body, ctx)
	case "StartChannel":
		return handleGCPVideoLivestreamStartChannel(w, path, body, ctx)
	case "StopChannel":
		return handleGCPVideoLivestreamStopChannel(w, path, body, ctx)
	case "StartDistribution":
		return handleGCPVideoLivestreamStartDistribution(w, path, body, ctx)
	case "StopDistribution":
		return handleGCPVideoLivestreamStopDistribution(w, path, body, ctx)
	case "CreateInput":
		return handleGCPVideoLivestreamCreateInput(w, path, body, ctx)
	case "ListInputs":
		return handleGCPVideoLivestreamListInputs(w, path, body, ctx)
	case "GetInput":
		return handleGCPVideoLivestreamGetInput(w, path, body, ctx)
	case "DeleteInput":
		return handleGCPVideoLivestreamDeleteInput(w, path, body, ctx)
	case "UpdateInput":
		return handleGCPVideoLivestreamUpdateInput(w, path, body, ctx)
	case "PreviewInput":
		return handleGCPVideoLivestreamPreviewInput(w, path, body, ctx)
	case "CreateEvent":
		return handleGCPVideoLivestreamCreateEvent(w, path, body, ctx)
	case "ListEvents":
		return handleGCPVideoLivestreamListEvents(w, path, body, ctx)
	case "GetEvent":
		return handleGCPVideoLivestreamGetEvent(w, path, body, ctx)
	case "DeleteEvent":
		return handleGCPVideoLivestreamDeleteEvent(w, path, body, ctx)
	case "ListClips":
		return handleGCPVideoLivestreamListClips(w, path, body, ctx)
	case "GetClip":
		return handleGCPVideoLivestreamGetClip(w, path, body, ctx)
	case "CreateClip":
		return handleGCPVideoLivestreamCreateClip(w, path, body, ctx)
	case "DeleteClip":
		return handleGCPVideoLivestreamDeleteClip(w, path, body, ctx)
	case "CreateDvrSession":
		return handleGCPVideoLivestreamCreateDvrSession(w, path, body, ctx)
	case "ListDvrSessions":
		return handleGCPVideoLivestreamListDvrSessions(w, path, body, ctx)
	case "GetDvrSession":
		return handleGCPVideoLivestreamGetDvrSession(w, path, body, ctx)
	case "DeleteDvrSession":
		return handleGCPVideoLivestreamDeleteDvrSession(w, path, body, ctx)
	case "UpdateDvrSession":
		return handleGCPVideoLivestreamUpdateDvrSession(w, path, body, ctx)
	case "CreateAsset":
		return handleGCPVideoLivestreamCreateAsset(w, path, body, ctx)
	case "DeleteAsset":
		return handleGCPVideoLivestreamDeleteAsset(w, path, body, ctx)
	case "GetAsset":
		return handleGCPVideoLivestreamGetAsset(w, path, body, ctx)
	case "ListAssets":
		return handleGCPVideoLivestreamListAssets(w, path, body, ctx)
	case "GetPool":
		return handleGCPVideoLivestreamGetPool(w, path, body, ctx)
	case "UpdatePool":
		return handleGCPVideoLivestreamUpdatePool(w, path, body, ctx)
	case "GetOperation":
		return handleGCPVideoLivestreamGetOperation(w, path, body, ctx)
	case "ListOperations":
		return handleGCPVideoLivestreamListOperations(w, path, body, ctx)
	case "CancelOperation":
		return handleGCPVideoLivestreamCancelOperation(w, path, body, ctx)
	case "DeleteOperation":
		return handleGCPVideoLivestreamDeleteOperation(w, path, body, ctx)
	default:
		return false
	}
}

func handleGCPVideoLivestreamListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, nil, r.URL.Query())
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamLocationFixture(project, "us-central1"),
		gcpVideoLivestreamLocationFixture(project, "global"),
	}
	return respondGCPVideoLivestreamList(w, "locations", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamLocationFixture(project, location))
	return true
}

func handleGCPVideoLivestreamListLocationsByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := strings.TrimSpace(gcpVideoLivestreamString(body, "name", "parent"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project, ok := gcpVideoLivestreamProjectFromName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamLocationFixture(project, "us-central1"),
		gcpVideoLivestreamLocationFixture(project, "global"),
	}
	return respondGCPVideoLivestreamList(w, "locations", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetLocationByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := strings.TrimSpace(gcpVideoLivestreamString(body, "name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamLocationFixture(project, location))
	return true
}

func handleGCPVideoLivestreamCreateChannel(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := gcpVideoLivestreamResolveParent(body, ctx)
	if _, _, ok := gcpVideoLivestreamProjectLocationFromParent(parent); !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	channel := gcpVideoLivestreamBodyMap(body, "channel")
	if len(channel) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "channel is required")
		return true
	}
	channelID := strings.TrimSpace(gcpVideoLivestreamString(body, "channelId", "channel_id"))
	if channelID == "" && ctx.Query != nil {
		channelID = strings.TrimSpace(ctx.Query.Get("channelId"))
		if channelID == "" {
			channelID = strings.TrimSpace(ctx.Query.Get("channel_id"))
		}
	}
	if channelID == "" {
		if name := strings.TrimSpace(gcpVideoLivestreamString(channel, "name")); name != "" {
			_, parsedID, ok := gcpVideoLivestreamParseChannelName(name)
			if !ok {
				respondGCPVideoLivestreamInvalidArgument(w, path, "channel.name is invalid")
				return true
			}
			channelID = parsedID
		}
	}
	if channelID == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "channel_id is required")
		return true
	}
	if !gcpVideoLivestreamIDPattern.MatchString(channelID) {
		respondGCPVideoLivestreamInvalidArgument(w, path, "channel_id is invalid")
		return true
	}
	expectedName := parent + "/channels/" + channelID
	if name := strings.TrimSpace(gcpVideoLivestreamString(channel, "name")); name != "" && name != expectedName {
		respondGCPVideoLivestreamInvalidArgument(w, path, "channel.name must match parent and channel_id")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "createChannel."+channelID, expectedName, "create", true))
	return true
}

func handleGCPVideoLivestreamListChannels(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := gcpVideoLivestreamResolveParent(body, ctx)
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamChannelFixture(project, location, "channel-1"),
		gcpVideoLivestreamChannelFixture(project, location, "channel-2"),
	}
	return respondGCPVideoLivestreamList(w, "channels", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetChannel(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		respondGCPVideoLivestreamNotFound(w, path, "channel not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamChannelFixture(project, location, channelID))
	return true
}

func handleGCPVideoLivestreamDeleteChannel(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		respondGCPVideoLivestreamNotFound(w, path, "channel not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "deleteChannel."+channelID, name, "delete", true))
	return true
}

func handleGCPVideoLivestreamUpdateChannel(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	channel := gcpVideoLivestreamBodyMap(body, "channel")
	if len(channel) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "channel is required")
		return true
	}
	if strings.TrimSpace(gcpVideoLivestreamString(body, "updateMask", "update_mask")) == "" && gcpVideoLivestreamUpdateMaskFromQuery(ctx.Query) == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "update_mask is required")
		return true
	}
	name := strings.TrimSpace(gcpVideoLivestreamString(channel, "name"))
	if name == "" {
		name = gcpVideoLivestreamResolveName(body, ctx)
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "channel.name is required")
		return true
	}
	if ctx.Name != "" && name != ctx.Name {
		respondGCPVideoLivestreamInvalidArgument(w, path, "channel.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "updateChannel."+channelID, name, "update", true))
	return true
}

func handleGCPVideoLivestreamStartChannel(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		respondGCPVideoLivestreamNotFound(w, path, "channel not found")
		return true
	}
	if strings.Contains(strings.ToLower(channelID), "running") {
		respondGCPVideoLivestreamFailedPrecondition(w, path, "channel already running")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "startChannel."+channelID, name, "start", true))
	return true
}

func handleGCPVideoLivestreamStopChannel(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		respondGCPVideoLivestreamNotFound(w, path, "channel not found")
		return true
	}
	if strings.Contains(strings.ToLower(channelID), "stopped") {
		respondGCPVideoLivestreamFailedPrecondition(w, path, "channel already stopped")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "stopChannel."+channelID, name, "stop", true))
	return true
}

func handleGCPVideoLivestreamStartDistribution(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, distributionID, ok := gcpVideoLivestreamParseDistributionName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(channelID) || isGCPVideoLivestreamMissingID(distributionID) {
		respondGCPVideoLivestreamNotFound(w, path, "distribution not found")
		return true
	}
	opID := "startDistribution." + channelID + "." + distributionID
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, opID, name, "start", true))
	return true
}

func handleGCPVideoLivestreamStopDistribution(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, distributionID, ok := gcpVideoLivestreamParseDistributionName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(channelID) || isGCPVideoLivestreamMissingID(distributionID) {
		respondGCPVideoLivestreamNotFound(w, path, "distribution not found")
		return true
	}
	opID := "stopDistribution." + channelID + "." + distributionID
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, opID, name, "stop", true))
	return true
}

func handleGCPVideoLivestreamCreateInput(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := gcpVideoLivestreamResolveParent(body, ctx)
	if _, _, ok := gcpVideoLivestreamProjectLocationFromParent(parent); !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	input := gcpVideoLivestreamBodyMap(body, "input")
	if len(input) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "input is required")
		return true
	}
	inputID := strings.TrimSpace(gcpVideoLivestreamString(body, "inputId", "input_id"))
	if inputID == "" && ctx.Query != nil {
		inputID = strings.TrimSpace(ctx.Query.Get("inputId"))
		if inputID == "" {
			inputID = strings.TrimSpace(ctx.Query.Get("input_id"))
		}
	}
	if inputID == "" {
		if name := strings.TrimSpace(gcpVideoLivestreamString(input, "name")); name != "" {
			_, parsedID, ok := gcpVideoLivestreamParseInputName(name)
			if !ok {
				respondGCPVideoLivestreamInvalidArgument(w, path, "input.name is invalid")
				return true
			}
			inputID = parsedID
		}
	}
	if inputID == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "input_id is required")
		return true
	}
	if !gcpVideoLivestreamIDPattern.MatchString(inputID) {
		respondGCPVideoLivestreamInvalidArgument(w, path, "input_id is invalid")
		return true
	}
	expectedName := parent + "/inputs/" + inputID
	if name := strings.TrimSpace(gcpVideoLivestreamString(input, "name")); name != "" && name != expectedName {
		respondGCPVideoLivestreamInvalidArgument(w, path, "input.name must match parent and input_id")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "createInput."+inputID, expectedName, "create", true))
	return true
}

func handleGCPVideoLivestreamListInputs(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := gcpVideoLivestreamResolveParent(body, ctx)
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamInputFixture(project, location, "input-1"),
		gcpVideoLivestreamInputFixture(project, location, "input-2"),
	}
	return respondGCPVideoLivestreamList(w, "inputs", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetInput(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, inputID, ok := gcpVideoLivestreamParseInputName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(inputID) {
		respondGCPVideoLivestreamNotFound(w, path, "input not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamInputFixture(project, location, inputID))
	return true
}

func handleGCPVideoLivestreamDeleteInput(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, inputID, ok := gcpVideoLivestreamParseInputName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(inputID) {
		respondGCPVideoLivestreamNotFound(w, path, "input not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "deleteInput."+inputID, name, "delete", true))
	return true
}

func handleGCPVideoLivestreamUpdateInput(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	input := gcpVideoLivestreamBodyMap(body, "input")
	if len(input) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "input is required")
		return true
	}
	if strings.TrimSpace(gcpVideoLivestreamString(body, "updateMask", "update_mask")) == "" && gcpVideoLivestreamUpdateMaskFromQuery(ctx.Query) == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "update_mask is required")
		return true
	}
	name := strings.TrimSpace(gcpVideoLivestreamString(input, "name"))
	if name == "" {
		name = gcpVideoLivestreamResolveName(body, ctx)
	}
	parent, inputID, ok := gcpVideoLivestreamParseInputName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "input.name is required")
		return true
	}
	if ctx.Name != "" && name != ctx.Name {
		respondGCPVideoLivestreamInvalidArgument(w, path, "input.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "updateInput."+inputID, name, "update", true))
	return true
}

func handleGCPVideoLivestreamPreviewInput(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, inputID, ok := gcpVideoLivestreamParseInputName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(inputID) {
		respondGCPVideoLivestreamNotFound(w, path, "input not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamPreviewInputFixture(project, location, inputID))
	return true
}

func handleGCPVideoLivestreamCreateEvent(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := strings.TrimSpace(gcpVideoLivestreamString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	parentLocation, channelID, ok := gcpVideoLivestreamParseChannelName(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	channelName := parentLocation + "/channels/" + channelID
	event := gcpVideoLivestreamBodyMap(body, "event")
	if len(event) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "event is required")
		return true
	}
	eventID := strings.TrimSpace(gcpVideoLivestreamString(body, "eventId", "event_id"))
	if eventID == "" && ctx.Query != nil {
		eventID = strings.TrimSpace(ctx.Query.Get("eventId"))
		if eventID == "" {
			eventID = strings.TrimSpace(ctx.Query.Get("event_id"))
		}
	}
	if eventID == "" {
		if name := strings.TrimSpace(gcpVideoLivestreamString(event, "name")); name != "" {
			_, _, parsedID, ok := gcpVideoLivestreamParseEventName(name)
			if !ok {
				respondGCPVideoLivestreamInvalidArgument(w, path, "event.name is invalid")
				return true
			}
			eventID = parsedID
		}
	}
	if eventID == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "event_id is required")
		return true
	}
	if !gcpVideoLivestreamIDPattern.MatchString(eventID) {
		respondGCPVideoLivestreamInvalidArgument(w, path, "event_id is invalid")
		return true
	}
	expectedName := channelName + "/events/" + eventID
	if name := strings.TrimSpace(gcpVideoLivestreamString(event, "name")); name != "" && name != expectedName {
		respondGCPVideoLivestreamInvalidArgument(w, path, "event.name must match parent and event_id")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parentLocation)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamEventFixture(project, location, channelID, eventID))
	return true
}

func handleGCPVideoLivestreamListEvents(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := strings.TrimSpace(gcpVideoLivestreamString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	_, channelID, ok := gcpVideoLivestreamParseChannelName(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	channelParent := strings.TrimSuffix(parent, "/channels/"+channelID)
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(channelParent)
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamEventFixture(project, location, channelID, "event-1"),
		gcpVideoLivestreamEventFixture(project, location, channelID, "event-2"),
	}
	return respondGCPVideoLivestreamList(w, "events", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetEvent(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, eventID, ok := gcpVideoLivestreamParseEventName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(eventID) {
		respondGCPVideoLivestreamNotFound(w, path, "event not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamEventFixture(project, location, channelID, eventID))
	return true
}

func handleGCPVideoLivestreamDeleteEvent(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	_, _, eventID, ok := gcpVideoLivestreamParseEventName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(eventID) {
		respondGCPVideoLivestreamNotFound(w, path, "event not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVideoLivestreamListClips(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := strings.TrimSpace(gcpVideoLivestreamString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	_, channelID, ok := gcpVideoLivestreamParseChannelName(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	channelParent := strings.TrimSuffix(parent, "/channels/"+channelID)
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(channelParent)
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamClipFixture(project, location, channelID, "clip-1"),
		gcpVideoLivestreamClipFixture(project, location, channelID, "clip-2"),
	}
	return respondGCPVideoLivestreamList(w, "clips", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetClip(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, clipID, ok := gcpVideoLivestreamParseClipName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(clipID) {
		respondGCPVideoLivestreamNotFound(w, path, "clip not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamClipFixture(project, location, channelID, clipID))
	return true
}

func handleGCPVideoLivestreamCreateClip(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := strings.TrimSpace(gcpVideoLivestreamString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	parentLocation, channelID, ok := gcpVideoLivestreamParseChannelName(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	channelName := parentLocation + "/channels/" + channelID
	clip := gcpVideoLivestreamBodyMap(body, "clip")
	if len(clip) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "clip is required")
		return true
	}
	clipID := strings.TrimSpace(gcpVideoLivestreamString(body, "clipId", "clip_id"))
	if clipID == "" && ctx.Query != nil {
		clipID = strings.TrimSpace(ctx.Query.Get("clipId"))
		if clipID == "" {
			clipID = strings.TrimSpace(ctx.Query.Get("clip_id"))
		}
	}
	if clipID == "" {
		if name := strings.TrimSpace(gcpVideoLivestreamString(clip, "name")); name != "" {
			_, _, parsedID, ok := gcpVideoLivestreamParseClipName(name)
			if !ok {
				respondGCPVideoLivestreamInvalidArgument(w, path, "clip.name is invalid")
				return true
			}
			clipID = parsedID
		}
	}
	if clipID == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "clip_id is required")
		return true
	}
	if !gcpVideoLivestreamIDPattern.MatchString(clipID) {
		respondGCPVideoLivestreamInvalidArgument(w, path, "clip_id is invalid")
		return true
	}
	expectedName := channelName + "/clips/" + clipID
	if name := strings.TrimSpace(gcpVideoLivestreamString(clip, "name")); name != "" && name != expectedName {
		respondGCPVideoLivestreamInvalidArgument(w, path, "clip.name must match parent and clip_id")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parentLocation, "createClip."+clipID, expectedName, "create", true))
	return true
}

func handleGCPVideoLivestreamDeleteClip(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, _, clipID, ok := gcpVideoLivestreamParseClipName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(clipID) {
		respondGCPVideoLivestreamNotFound(w, path, "clip not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "deleteClip."+clipID, name, "delete", true))
	return true
}

func handleGCPVideoLivestreamCreateDvrSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := strings.TrimSpace(gcpVideoLivestreamString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	parentLocation, channelID, ok := gcpVideoLivestreamParseChannelName(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	channelName := parentLocation + "/channels/" + channelID
	dvrSession := gcpVideoLivestreamBodyMap(body, "dvrSession", "dvr_session")
	if len(dvrSession) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session is required")
		return true
	}
	dvrSessionID := strings.TrimSpace(gcpVideoLivestreamString(body, "dvrSessionId", "dvr_session_id"))
	if dvrSessionID == "" && ctx.Query != nil {
		dvrSessionID = strings.TrimSpace(ctx.Query.Get("dvrSessionId"))
		if dvrSessionID == "" {
			dvrSessionID = strings.TrimSpace(ctx.Query.Get("dvr_session_id"))
		}
	}
	if dvrSessionID == "" {
		if name := strings.TrimSpace(gcpVideoLivestreamString(dvrSession, "name")); name != "" {
			_, _, parsedID, ok := gcpVideoLivestreamParseDvrSessionName(name)
			if !ok {
				respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session.name is invalid")
				return true
			}
			dvrSessionID = parsedID
		}
	}
	if dvrSessionID == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session_id is required")
		return true
	}
	if !gcpVideoLivestreamIDPattern.MatchString(dvrSessionID) {
		respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session_id is invalid")
		return true
	}
	expectedName := channelName + "/dvrSessions/" + dvrSessionID
	if name := strings.TrimSpace(gcpVideoLivestreamString(dvrSession, "name")); name != "" && name != expectedName {
		respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session.name must match parent and dvr_session_id")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parentLocation, "createDvrSession."+dvrSessionID, expectedName, "create", true))
	return true
}

func handleGCPVideoLivestreamListDvrSessions(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := strings.TrimSpace(gcpVideoLivestreamString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	_, channelID, ok := gcpVideoLivestreamParseChannelName(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	channelParent := strings.TrimSuffix(parent, "/channels/"+channelID)
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(channelParent)
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamDvrSessionFixture(project, location, channelID, "dvr-session-1"),
		gcpVideoLivestreamDvrSessionFixture(project, location, channelID, "dvr-session-2"),
	}
	return respondGCPVideoLivestreamList(w, "dvrSessions", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetDvrSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, channelID, dvrSessionID, ok := gcpVideoLivestreamParseDvrSessionName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(dvrSessionID) {
		respondGCPVideoLivestreamNotFound(w, path, "dvr session not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamDvrSessionFixture(project, location, channelID, dvrSessionID))
	return true
}

func handleGCPVideoLivestreamDeleteDvrSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, _, dvrSessionID, ok := gcpVideoLivestreamParseDvrSessionName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(dvrSessionID) {
		respondGCPVideoLivestreamNotFound(w, path, "dvr session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "deleteDvrSession."+dvrSessionID, name, "delete", true))
	return true
}

func handleGCPVideoLivestreamUpdateDvrSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	dvrSession := gcpVideoLivestreamBodyMap(body, "dvrSession", "dvr_session")
	if len(dvrSession) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session is required")
		return true
	}
	if strings.TrimSpace(gcpVideoLivestreamString(body, "updateMask", "update_mask")) == "" && gcpVideoLivestreamUpdateMaskFromQuery(ctx.Query) == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "update_mask is required")
		return true
	}
	name := strings.TrimSpace(gcpVideoLivestreamString(dvrSession, "name"))
	if name == "" {
		name = gcpVideoLivestreamResolveName(body, ctx)
	}
	parent, _, dvrSessionID, ok := gcpVideoLivestreamParseDvrSessionName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session.name is required")
		return true
	}
	if ctx.Name != "" && name != ctx.Name {
		respondGCPVideoLivestreamInvalidArgument(w, path, "dvr_session.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "updateDvrSession."+dvrSessionID, name, "update", true))
	return true
}

func handleGCPVideoLivestreamCreateAsset(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := gcpVideoLivestreamResolveParent(body, ctx)
	if _, _, ok := gcpVideoLivestreamProjectLocationFromParent(parent); !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	asset := gcpVideoLivestreamBodyMap(body, "asset")
	if len(asset) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "asset is required")
		return true
	}
	assetID := strings.TrimSpace(gcpVideoLivestreamString(body, "assetId", "asset_id"))
	if assetID == "" && ctx.Query != nil {
		assetID = strings.TrimSpace(ctx.Query.Get("assetId"))
		if assetID == "" {
			assetID = strings.TrimSpace(ctx.Query.Get("asset_id"))
		}
	}
	if assetID == "" {
		if name := strings.TrimSpace(gcpVideoLivestreamString(asset, "name")); name != "" {
			_, parsedID, ok := gcpVideoLivestreamParseAssetName(name)
			if !ok {
				respondGCPVideoLivestreamInvalidArgument(w, path, "asset.name is invalid")
				return true
			}
			assetID = parsedID
		}
	}
	if assetID == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "asset_id is required")
		return true
	}
	if !gcpVideoLivestreamIDPattern.MatchString(assetID) {
		respondGCPVideoLivestreamInvalidArgument(w, path, "asset_id is invalid")
		return true
	}
	expectedName := parent + "/assets/" + assetID
	if name := strings.TrimSpace(gcpVideoLivestreamString(asset, "name")); name != "" && name != expectedName {
		respondGCPVideoLivestreamInvalidArgument(w, path, "asset.name must match parent and asset_id")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "createAsset."+assetID, expectedName, "create", true))
	return true
}

func handleGCPVideoLivestreamDeleteAsset(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, assetID, ok := gcpVideoLivestreamParseAssetName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(assetID) {
		respondGCPVideoLivestreamNotFound(w, path, "asset not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "deleteAsset."+assetID, name, "delete", true))
	return true
}

func handleGCPVideoLivestreamGetAsset(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, assetID, ok := gcpVideoLivestreamParseAssetName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(assetID) {
		respondGCPVideoLivestreamNotFound(w, path, "asset not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamAssetFixture(project, location, assetID))
	return true
}

func handleGCPVideoLivestreamListAssets(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	parent := gcpVideoLivestreamResolveParent(body, ctx)
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoLivestreamAssetFixture(project, location, "asset-1"),
		gcpVideoLivestreamAssetFixture(project, location, "asset-2"),
	}
	return respondGCPVideoLivestreamList(w, "assets", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamGetPool(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, poolID, ok := gcpVideoLivestreamParsePoolName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(poolID) {
		respondGCPVideoLivestreamNotFound(w, path, "pool not found")
		return true
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoLivestreamPoolFixture(project, location, poolID))
	return true
}

func handleGCPVideoLivestreamUpdatePool(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	pool := gcpVideoLivestreamBodyMap(body, "pool")
	if len(pool) == 0 {
		respondGCPVideoLivestreamInvalidArgument(w, path, "pool is required")
		return true
	}
	if strings.TrimSpace(gcpVideoLivestreamString(body, "updateMask", "update_mask")) == "" && gcpVideoLivestreamUpdateMaskFromQuery(ctx.Query) == "" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "update_mask is required")
		return true
	}
	name := strings.TrimSpace(gcpVideoLivestreamString(pool, "name"))
	if name == "" {
		name = gcpVideoLivestreamResolveName(body, ctx)
	}
	parent, poolID, ok := gcpVideoLivestreamParsePoolName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "pool.name is required")
		return true
	}
	if ctx.Name != "" && name != ctx.Name {
		respondGCPVideoLivestreamInvalidArgument(w, path, "pool.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, "updatePool."+poolID, name, "update", true))
	return true
}

func handleGCPVideoLivestreamGetOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	parent, opID, ok := gcpVideoLivestreamParseOperationName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(opID) {
		respondGCPVideoLivestreamNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoLivestreamOperationFixture(parent, opID, parent+"/operations/"+opID, "operate", true))
	return true
}

func handleGCPVideoLivestreamListOperations(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := strings.TrimSpace(gcpVideoLivestreamString(body, "name", "parent"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	if strings.HasSuffix(name, "/operations") {
		name = strings.TrimSuffix(name, "/operations")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoLivestreamPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	items := []map[string]any{
		gcpVideoLivestreamOperationFixture(parent, "createChannel.channel-1", parent+"/channels/channel-1", "create", true),
		gcpVideoLivestreamOperationFixture(parent, "createInput.input-1", parent+"/inputs/input-1", "create", true),
	}
	return respondGCPVideoLivestreamList(w, "operations", items, pageSize, offset, path)
}

func handleGCPVideoLivestreamCancelOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	_, opID, ok := gcpVideoLivestreamParseOperationName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(opID) {
		respondGCPVideoLivestreamNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVideoLivestreamDeleteOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoLivestreamRouteContext) bool {
	name := gcpVideoLivestreamResolveName(body, ctx)
	_, opID, ok := gcpVideoLivestreamParseOperationName(name)
	if !ok {
		respondGCPVideoLivestreamInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoLivestreamMissingID(opID) {
		respondGCPVideoLivestreamNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPVideoLivestreamLocationTail(path string) (project, location, tail string, ok bool) {
	const prefix = "/gcp/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}

	rest := strings.TrimPrefix(path, prefix)
	projectPart, afterProject, found := strings.Cut(rest, "/locations/")
	if !found {
		return "", "", "", false
	}
	project = strings.TrimSpace(projectPart)
	if !gcpVideoLivestreamProjectPattern.MatchString(project) {
		return "", "", "", false
	}
	afterProject = strings.TrimSpace(afterProject)
	if afterProject == "" {
		return "", "", "", false
	}
	split := strings.IndexAny(afterProject, "/:")
	if split < 0 {
		location = strings.TrimSpace(afterProject)
		tail = ""
	} else {
		location = strings.TrimSpace(afterProject[:split])
		tail = strings.TrimSpace(afterProject[split:])
	}
	if !gcpVideoLivestreamLocationPattern.MatchString(location) {
		return "", "", "", false
	}
	if tail == "/" || tail == ":" {
		return "", "", "", false
	}
	if tail != "" && !strings.HasPrefix(tail, "/") && !strings.HasPrefix(tail, ":") {
		return "", "", "", false
	}
	if strings.HasPrefix(tail, ":") && strings.Contains(tail, "/") {
		return "", "", "", false
	}
	return project, location, tail, true
}

func gcpVideoLivestreamTailParts(tail string) []string {
	value := strings.TrimSpace(strings.TrimPrefix(tail, "/"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil
		}
		out = append(out, trimmed)
	}
	return out
}

func gcpVideoLivestreamSplitIDAction(raw string) (id, action string, hasAction bool) {
	id = strings.TrimSpace(raw)
	if id == "" {
		return "", "", false
	}
	lhs, rhs, found := strings.Cut(id, ":")
	if !found {
		return id, "", false
	}
	lhs = strings.TrimSpace(lhs)
	rhs = strings.TrimSpace(rhs)
	if lhs == "" || rhs == "" {
		return "", "", false
	}
	return lhs, rhs, true
}

func decodeGCPVideoLivestreamJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPVideoLivestreamInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPVideoLivestreamPagination(w http.ResponseWriter, path string, body map[string]any, query url.Values) (int, int, bool) {
	pageSize := 50
	offset := 0

	rawPageSize := strings.TrimSpace(gcpVideoLivestreamString(body, "pageSize", "page_size"))
	if rawPageSize == "" && query != nil {
		rawPageSize = strings.TrimSpace(query.Get("pageSize"))
		if rawPageSize == "" {
			rawPageSize = strings.TrimSpace(query.Get("page_size"))
		}
	}
	if rawPageSize != "" {
		value, err := strconv.Atoi(rawPageSize)
		if err != nil || value < 0 || value > 1000 {
			respondGCPVideoLivestreamInvalidArgument(w, path, "pageSize must be between 0 and 1000")
			return 0, 0, false
		}
		pageSize = value
	}

	rawPageToken := strings.TrimSpace(gcpVideoLivestreamString(body, "pageToken", "page_token"))
	if rawPageToken == "" && query != nil {
		rawPageToken = strings.TrimSpace(query.Get("pageToken"))
		if rawPageToken == "" {
			rawPageToken = strings.TrimSpace(query.Get("page_token"))
		}
	}
	if rawPageToken != "" {
		value, err := strconv.Atoi(rawPageToken)
		if err != nil || value < 0 {
			respondGCPVideoLivestreamInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		offset = value
	}
	return pageSize, offset, true
}

func respondGCPVideoLivestreamList(w http.ResponseWriter, key string, items []map[string]any, pageSize, offset int, path string) bool {
	if offset > len(items) {
		respondGCPVideoLivestreamInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && offset+pageSize < end {
		end = offset + pageSize
	}
	response := map[string]any{
		key: items[offset:end],
	}
	if end < len(items) {
		response["nextPageToken"] = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func gcpVideoLivestreamResolveParent(body map[string]any, ctx gcpVideoLivestreamRouteContext) string {
	parent := strings.TrimSpace(gcpVideoLivestreamString(body, "parent"))
	if parent != "" {
		return parent
	}
	return strings.TrimSpace(ctx.Parent)
}

func gcpVideoLivestreamResolveName(body map[string]any, ctx gcpVideoLivestreamRouteContext) string {
	name := strings.TrimSpace(gcpVideoLivestreamString(body, "name"))
	if name != "" {
		return name
	}
	return strings.TrimSpace(ctx.Name)
}

func gcpVideoLivestreamUpdateMaskFromQuery(query url.Values) string {
	if query == nil {
		return ""
	}
	updateMask := strings.TrimSpace(query.Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(query.Get("update_mask"))
	}
	return updateMask
}

func gcpVideoLivestreamProjectFromName(name string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 2 || parts[0] != "projects" || !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) {
		return "", false
	}
	return parts[1], true
}

func gcpVideoLivestreamProjectLocationFromName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) {
		return "", "", false
	}
	return parts[1], parts[3], true
}

func gcpVideoLivestreamProjectLocationFromParent(parent string) (string, string, bool) {
	return gcpVideoLivestreamProjectLocationFromName(parent)
}

func gcpVideoLivestreamParseChannelName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "channels" {
		return "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpVideoLivestreamParseDistributionName(name string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "channels" || parts[6] != "distributions" {
		return "", "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) || !gcpVideoLivestreamIDPattern.MatchString(parts[7]) {
		return "", "", "", false
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3])
	return parent, parts[5], parts[7], true
}

func gcpVideoLivestreamParseInputName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "inputs" {
		return "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpVideoLivestreamParseEventName(name string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "channels" || parts[6] != "events" {
		return "", "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) || !gcpVideoLivestreamIDPattern.MatchString(parts[7]) {
		return "", "", "", false
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3])
	return parent, parts[5], parts[7], true
}

func gcpVideoLivestreamParseClipName(name string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "channels" || parts[6] != "clips" {
		return "", "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) || !gcpVideoLivestreamIDPattern.MatchString(parts[7]) {
		return "", "", "", false
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3])
	return parent, parts[5], parts[7], true
}

func gcpVideoLivestreamParseDvrSessionName(name string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "channels" || parts[6] != "dvrSessions" {
		return "", "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) || !gcpVideoLivestreamIDPattern.MatchString(parts[7]) {
		return "", "", "", false
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3])
	return parent, parts[5], parts[7], true
}

func gcpVideoLivestreamParseAssetName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "assets" {
		return "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpVideoLivestreamParsePoolName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "pools" {
		return "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamIDPattern.MatchString(parts[5]) {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func gcpVideoLivestreamParseOperationName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", false
	}
	if !gcpVideoLivestreamProjectPattern.MatchString(parts[1]) || !gcpVideoLivestreamLocationPattern.MatchString(parts[3]) || !gcpVideoLivestreamOperationIDPattern.MatchString(parts[5]) {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3]), parts[5], true
}

func isGCPVideoLivestreamMissingID(id string) bool {
	value := strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(value, "missing") || strings.Contains(value, "not-found") || strings.Contains(value, "does-not-exist")
}

func gcpVideoLivestreamLocationFixture(project, location string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s", project, location)
	return map[string]any{
		"name":        name,
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"labels": map[string]any{
			"cloud.googleapis.com/region": location,
		},
		"metadata": map[string]any{
			"supportsHls":  true,
			"supportsDash": true,
		},
	}
}

func gcpVideoLivestreamChannelFixture(project, location, channelID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/channels/%s", project, location, channelID)
	return map[string]any{
		"name":       name,
		"uid":        "uid-" + channelID,
		"createTime": gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
		"updateTime": gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		"streamingState": func() string {
			if strings.Contains(strings.ToLower(channelID), "stopped") {
				return "STOPPED"
			}
			return "AWAITING_INPUT"
		}(),
		"activeInput": "input-1",
		"output": map[string]any{
			"uri": fmt.Sprintf("https://example.com/%s/index.m3u8", channelID),
		},
	}
}

func gcpVideoLivestreamInputFixture(project, location, inputID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/inputs/%s", project, location, inputID)
	return map[string]any{
		"name":       name,
		"uid":        "uid-" + inputID,
		"type":       "RTMP_PUSH",
		"tier":       "SD",
		"uri":        fmt.Sprintf("rtmp://ingest.example.com/live/%s", inputID),
		"createTime": gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
		"updateTime": gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
	}
}

func gcpVideoLivestreamPreviewInputFixture(project, location, inputID string) map[string]any {
	return map[string]any{
		"uri": fmt.Sprintf("https://preview.example.com/projects/%s/locations/%s/inputs/%s/manifest.m3u8", project, location, inputID),
	}
}

func gcpVideoLivestreamEventFixture(project, location, channelID, eventID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/channels/%s/events/%s", project, location, channelID, eventID)
	return map[string]any{
		"name":       name,
		"state":      "SCHEDULED",
		"createTime": gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
		"updateTime": gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
	}
}

func gcpVideoLivestreamClipFixture(project, location, channelID, clipID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/channels/%s/clips/%s", project, location, channelID, clipID)
	return map[string]any{
		"name":       name,
		"state":      "SUCCEEDED",
		"outputUri":  fmt.Sprintf("gs://stackyard-live-stream/clips/%s.mp4", clipID),
		"createTime": gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
		"updateTime": gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
	}
}

func gcpVideoLivestreamDvrSessionFixture(project, location, channelID, dvrSessionID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/channels/%s/dvrSessions/%s", project, location, channelID, dvrSessionID)
	return map[string]any{
		"name":       name,
		"state":      "ACTIVE",
		"createTime": gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
		"updateTime": gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
	}
}

func gcpVideoLivestreamAssetFixture(project, location, assetID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/assets/%s", project, location, assetID)
	return map[string]any{
		"name":       name,
		"state":      "ACTIVE",
		"createTime": gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
		"updateTime": gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
	}
}

func gcpVideoLivestreamPoolFixture(project, location, poolID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/pools/%s", project, location, poolID)
	return map[string]any{
		"name":       name,
		"createTime": gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
		"updateTime": gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		"networkConfig": map[string]any{
			"peeredNetwork": fmt.Sprintf("projects/%s/global/networks/default", project),
		},
	}
}

func gcpVideoLivestreamOperationFixture(parent, operationID, target, verb string, done bool) map[string]any {
	out := map[string]any{
		"name": fmt.Sprintf("%s/operations/%s", strings.TrimSpace(parent), strings.TrimSpace(operationID)),
		"done": done,
		"metadata": map[string]any{
			"@type":                 "type.googleapis.com/google.cloud.video.livestream.v1.OperationMetadata",
			"createTime":            gcpVideoLivestreamReferenceTime.Format(time.RFC3339Nano),
			"endTime":               gcpVideoLivestreamReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
			"target":                strings.TrimSpace(target),
			"verb":                  strings.TrimSpace(verb),
			"requestedCancellation": false,
		},
	}
	if done {
		out["response"] = map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		}
	}
	return out
}

func gcpVideoLivestreamString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if body == nil {
			continue
		}
		if value, ok := body[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case json.Number:
				return typed.String()
			}
		}
	}
	return ""
}

func gcpVideoLivestreamBodyMap(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if body == nil {
			continue
		}
		if value, ok := body[key]; ok {
			if typed, ok := value.(map[string]any); ok {
				return typed
			}
		}
	}
	return map[string]any{}
}

func respondGCPVideoLivestreamInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVideoLivestreamNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVideoLivestreamFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_video_livestream(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "video-livestream") && !isGCPContractProbeRequestForService(r, path, "livestream") && !isGCPContractProbeRequestForService(r, path, "video_livestream") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPVideoLivestreamInvalidArgument(w, path, "pageSize must be between 0 and 1000")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/video-livestream",
			"service":  "video_livestream",
			"provider": providerGCP,
			"path":     path,
			"methods": []string{
				"CreateChannel",
				"CreateInput",
				"CreateEvent",
				"CreateClip",
				"CreateDvrSession",
				"CreateAsset",
				"GetPool",
				"ListOperations",
			},
		})
		return true
	}
	return false
}
