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

const gcpVideoStitcherGRPCPathPrefix = "/gcp/google.cloud.video.stitcher.v1.VideoStitcherService/"

var (
	gcpVideoStitcherReferenceTime      = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpVideoStitcherProjectPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)
	gcpVideoStitcherLocationPattern    = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)
	gcpVideoStitcherIDPattern          = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	gcpVideoStitcherOperationIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
)

type gcpVideoStitcherRouteContext struct {
	Parent string
	Name   string
	Query  url.Values
}

type gcpVideoStitcherManagedResourceSpec struct {
	collection     string
	singular       string
	idKeys         []string
	bodyKeys       []string
	idLabel        string
	listField      string
	listFixtureIDs []string
	fixture        func(project, location, id string) map[string]any
	createOpPrefix string
	updateOpPrefix string
	deleteOpPrefix string
	validateCreate func(resource map[string]any) string
}

func (s *Server) handleGCPVideoStitcherRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_video_stitcher(w, r) {
		return true
	}

	path := normalizeGCPVideoStitcherPath(rawRequestPath(r))
	if isGCPVideoStitcherLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVideoStitcherListLocations(w, r, path) {
			return true
		}
		if handleGCPVideoStitcherGetLocation(w, path) {
			return true
		}
		return false
	}

	if strings.HasPrefix(path, gcpVideoStitcherGRPCPathPrefix) {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPVideoStitcherJSONBody(w, r, path)
		if !ok {
			return true
		}
		method := strings.TrimSpace(strings.TrimPrefix(path, gcpVideoStitcherGRPCPathPrefix))
		if method == "" {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		if handleGCPVideoStitcherRPCMethod(w, path, method, body, gcpVideoStitcherRouteContext{Query: r.URL.Query()}) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if !isGCPVideoStitcherPath(path, hasGCPVideoStitcherHint(r)) {
		return false
	}

	rpcMethod, ctx, needsBody, ok := mapGCPVideoStitcherRESTToMethod(r, path)
	if !ok {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	body := map[string]any{}
	if needsBody {
		var valid bool
		body, valid = decodeGCPVideoStitcherJSONBody(w, r, path)
		if !valid {
			return true
		}
	}

	if handleGCPVideoStitcherRPCMethod(w, path, rpcMethod, body, ctx) {
		return true
	}
	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func normalizeGCPVideoStitcherPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVideoStitcherHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "video_stitcher",
		"video-stitcher",
		"video-stitcher-apiv1",
		"video_stitcher_apiv1",
		"stitcher",
		"stitcher-apiv1",
		"gcp-video-stitcher",
		"gcp-stitcher":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-video-stitcher-apiv1") || strings.Contains(ua, "cloud.google.com/go/video/stitcher")
}

func isGCPVideoStitcherLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVideoStitcherHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPVideoStitcherPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpVideoStitcherGRPCPathPrefix) {
		return true
	}
	if !includeHint {
		return false
	}
	if _, _, _, ok := parseGCPProjectLocationPath(path); ok {
		return true
	}
	_, _, tail, ok := parseGCPVideoStitcherLocationTail(path)
	if !ok {
		return strings.HasPrefix(path, "/gcp/v1/projects/")
	}
	parts := gcpVideoStitcherTailParts(tail)
	if len(parts) == 0 {
		return true
	}
	switch parts[0] {
	case "cdnKeys", "slates", "liveConfigs", "vodConfigs", "vodSessions", "liveSessions":
		return true
	case "operations":
		return true
	default:
		return true
	}
}

func mapGCPVideoStitcherRESTToMethod(r *http.Request, path string) (string, gcpVideoStitcherRouteContext, bool, bool) {
	ctx := gcpVideoStitcherRouteContext{Query: r.URL.Query()}

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
		return "", gcpVideoStitcherRouteContext{}, false, false
	}

	project, location, tail, ok := parseGCPVideoStitcherLocationTail(path)
	if !ok {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	ctx.Parent = fmt.Sprintf("projects/%s/locations/%s", project, location)

	parts := gcpVideoStitcherTailParts(tail)
	if len(parts) == 0 {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}

	switch parts[0] {
	case "cdnKeys":
		return mapGCPVideoStitcherRESTManagedResourceMethod(r, ctx, parts, "cdnKeys", "ListCdnKeys", "CreateCdnKey", "GetCdnKey", "UpdateCdnKey", "DeleteCdnKey")
	case "slates":
		return mapGCPVideoStitcherRESTManagedResourceMethod(r, ctx, parts, "slates", "ListSlates", "CreateSlate", "GetSlate", "UpdateSlate", "DeleteSlate")
	case "liveConfigs":
		return mapGCPVideoStitcherRESTManagedResourceMethod(r, ctx, parts, "liveConfigs", "ListLiveConfigs", "CreateLiveConfig", "GetLiveConfig", "UpdateLiveConfig", "DeleteLiveConfig")
	case "vodConfigs":
		return mapGCPVideoStitcherRESTManagedResourceMethod(r, ctx, parts, "vodConfigs", "ListVodConfigs", "CreateVodConfig", "GetVodConfig", "UpdateVodConfig", "DeleteVodConfig")
	case "vodSessions":
		return mapGCPVideoStitcherRESTVodSessionMethod(r, ctx, parts)
	case "liveSessions":
		return mapGCPVideoStitcherRESTLiveSessionMethod(r, ctx, parts)
	case "operations":
		return mapGCPVideoStitcherRESTOperationMethod(r, ctx, parts)
	default:
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
}

func mapGCPVideoStitcherRESTManagedResourceMethod(
	r *http.Request,
	ctx gcpVideoStitcherRouteContext,
	parts []string,
	collection string,
	listMethod string,
	createMethod string,
	getMethod string,
	updateMethod string,
	deleteMethod string,
) (string, gcpVideoStitcherRouteContext, bool, bool) {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return listMethod, ctx, false, true
		case http.MethodPost:
			return createMethod, ctx, true, true
		default:
			return "", gcpVideoStitcherRouteContext{}, false, false
		}
	}
	if len(parts) != 2 {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	resourceID := strings.TrimSpace(parts[1])
	if resourceID == "" {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	ctx.Name = ctx.Parent + "/" + collection + "/" + resourceID
	switch r.Method {
	case http.MethodGet:
		return getMethod, ctx, false, true
	case http.MethodPatch:
		return updateMethod, ctx, true, true
	case http.MethodDelete:
		return deleteMethod, ctx, false, true
	default:
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
}

func mapGCPVideoStitcherRESTVodSessionMethod(r *http.Request, ctx gcpVideoStitcherRouteContext, parts []string) (string, gcpVideoStitcherRouteContext, bool, bool) {
	if len(parts) == 1 {
		if r.Method == http.MethodPost {
			return "CreateVodSession", ctx, true, true
		}
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	sessionID := strings.TrimSpace(parts[1])
	if sessionID == "" {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	sessionName := ctx.Parent + "/vodSessions/" + sessionID
	if len(parts) == 2 {
		if r.Method == http.MethodGet {
			ctx.Name = sessionName
			return "GetVodSession", ctx, false, true
		}
		return "", gcpVideoStitcherRouteContext{}, false, false
	}

	switch parts[2] {
	case "vodStitchDetails":
		if len(parts) == 3 && r.Method == http.MethodGet {
			ctx.Parent = sessionName
			return "ListVodStitchDetails", ctx, false, true
		}
		if len(parts) == 4 && r.Method == http.MethodGet {
			detailID := strings.TrimSpace(parts[3])
			if detailID == "" {
				return "", gcpVideoStitcherRouteContext{}, false, false
			}
			ctx.Name = sessionName + "/vodStitchDetails/" + detailID
			return "GetVodStitchDetail", ctx, false, true
		}
	case "vodAdTagDetails":
		if len(parts) == 3 && r.Method == http.MethodGet {
			ctx.Parent = sessionName
			return "ListVodAdTagDetails", ctx, false, true
		}
		if len(parts) == 4 && r.Method == http.MethodGet {
			detailID := strings.TrimSpace(parts[3])
			if detailID == "" {
				return "", gcpVideoStitcherRouteContext{}, false, false
			}
			ctx.Name = sessionName + "/vodAdTagDetails/" + detailID
			return "GetVodAdTagDetail", ctx, false, true
		}
	}
	return "", gcpVideoStitcherRouteContext{}, false, false
}

func mapGCPVideoStitcherRESTLiveSessionMethod(r *http.Request, ctx gcpVideoStitcherRouteContext, parts []string) (string, gcpVideoStitcherRouteContext, bool, bool) {
	if len(parts) == 1 {
		if r.Method == http.MethodPost {
			return "CreateLiveSession", ctx, true, true
		}
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	sessionID := strings.TrimSpace(parts[1])
	if sessionID == "" {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	sessionName := ctx.Parent + "/liveSessions/" + sessionID
	if len(parts) == 2 {
		if r.Method == http.MethodGet {
			ctx.Name = sessionName
			return "GetLiveSession", ctx, false, true
		}
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	if parts[2] != "liveAdTagDetails" {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		ctx.Parent = sessionName
		return "ListLiveAdTagDetails", ctx, false, true
	}
	if len(parts) == 4 && r.Method == http.MethodGet {
		detailID := strings.TrimSpace(parts[3])
		if detailID == "" {
			return "", gcpVideoStitcherRouteContext{}, false, false
		}
		ctx.Name = sessionName + "/liveAdTagDetails/" + detailID
		return "GetLiveAdTagDetail", ctx, false, true
	}
	return "", gcpVideoStitcherRouteContext{}, false, false
}

func mapGCPVideoStitcherRESTOperationMethod(r *http.Request, ctx gcpVideoStitcherRouteContext, parts []string) (string, gcpVideoStitcherRouteContext, bool, bool) {
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			ctx.Name = ctx.Parent
			return "ListOperations", ctx, false, true
		}
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	if len(parts) != 2 {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	opID, action, hasAction := gcpVideoStitcherSplitIDAction(parts[1])
	if strings.TrimSpace(opID) == "" {
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	ctx.Name = ctx.Parent + "/operations/" + opID
	if hasAction {
		if r.Method == http.MethodPost && action == "cancel" {
			return "CancelOperation", ctx, true, true
		}
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
	switch r.Method {
	case http.MethodGet:
		return "GetOperation", ctx, false, true
	case http.MethodDelete:
		return "DeleteOperation", ctx, false, true
	default:
		return "", gcpVideoStitcherRouteContext{}, false, false
	}
}

func handleGCPVideoStitcherRPCMethod(w http.ResponseWriter, path, method string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	switch method {
	case "ListLocations":
		return handleGCPVideoStitcherListLocationsByMethod(w, path, body, ctx)
	case "GetLocation":
		return handleGCPVideoStitcherGetLocationByMethod(w, path, body, ctx)
	case "CreateCdnKey":
		return handleGCPVideoStitcherCreateCdnKey(w, path, body, ctx)
	case "ListCdnKeys":
		return handleGCPVideoStitcherListCdnKeys(w, path, body, ctx)
	case "GetCdnKey":
		return handleGCPVideoStitcherGetCdnKey(w, path, body, ctx)
	case "DeleteCdnKey":
		return handleGCPVideoStitcherDeleteCdnKey(w, path, body, ctx)
	case "UpdateCdnKey":
		return handleGCPVideoStitcherUpdateCdnKey(w, path, body, ctx)
	case "CreateVodSession":
		return handleGCPVideoStitcherCreateVodSession(w, path, body, ctx)
	case "GetVodSession":
		return handleGCPVideoStitcherGetVodSession(w, path, body, ctx)
	case "ListVodStitchDetails":
		return handleGCPVideoStitcherListVodStitchDetails(w, path, body, ctx)
	case "GetVodStitchDetail":
		return handleGCPVideoStitcherGetVodStitchDetail(w, path, body, ctx)
	case "ListVodAdTagDetails":
		return handleGCPVideoStitcherListVodAdTagDetails(w, path, body, ctx)
	case "GetVodAdTagDetail":
		return handleGCPVideoStitcherGetVodAdTagDetail(w, path, body, ctx)
	case "ListLiveAdTagDetails":
		return handleGCPVideoStitcherListLiveAdTagDetails(w, path, body, ctx)
	case "GetLiveAdTagDetail":
		return handleGCPVideoStitcherGetLiveAdTagDetail(w, path, body, ctx)
	case "CreateSlate":
		return handleGCPVideoStitcherCreateSlate(w, path, body, ctx)
	case "ListSlates":
		return handleGCPVideoStitcherListSlates(w, path, body, ctx)
	case "GetSlate":
		return handleGCPVideoStitcherGetSlate(w, path, body, ctx)
	case "UpdateSlate":
		return handleGCPVideoStitcherUpdateSlate(w, path, body, ctx)
	case "DeleteSlate":
		return handleGCPVideoStitcherDeleteSlate(w, path, body, ctx)
	case "CreateLiveSession":
		return handleGCPVideoStitcherCreateLiveSession(w, path, body, ctx)
	case "GetLiveSession":
		return handleGCPVideoStitcherGetLiveSession(w, path, body, ctx)
	case "CreateLiveConfig":
		return handleGCPVideoStitcherCreateLiveConfig(w, path, body, ctx)
	case "ListLiveConfigs":
		return handleGCPVideoStitcherListLiveConfigs(w, path, body, ctx)
	case "GetLiveConfig":
		return handleGCPVideoStitcherGetLiveConfig(w, path, body, ctx)
	case "DeleteLiveConfig":
		return handleGCPVideoStitcherDeleteLiveConfig(w, path, body, ctx)
	case "UpdateLiveConfig":
		return handleGCPVideoStitcherUpdateLiveConfig(w, path, body, ctx)
	case "CreateVodConfig":
		return handleGCPVideoStitcherCreateVodConfig(w, path, body, ctx)
	case "ListVodConfigs":
		return handleGCPVideoStitcherListVodConfigs(w, path, body, ctx)
	case "GetVodConfig":
		return handleGCPVideoStitcherGetVodConfig(w, path, body, ctx)
	case "DeleteVodConfig":
		return handleGCPVideoStitcherDeleteVodConfig(w, path, body, ctx)
	case "UpdateVodConfig":
		return handleGCPVideoStitcherUpdateVodConfig(w, path, body, ctx)
	case "GetOperation":
		return handleGCPVideoStitcherGetOperation(w, path, body, ctx)
	case "ListOperations":
		return handleGCPVideoStitcherListOperations(w, path, body, ctx)
	case "CancelOperation":
		return handleGCPVideoStitcherCancelOperation(w, path, body, ctx)
	case "DeleteOperation":
		return handleGCPVideoStitcherDeleteOperation(w, path, body, ctx)
	default:
		return false
	}
}

func handleGCPVideoStitcherListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, offset, valid := parseGCPVideoStitcherPagination(w, path, nil, r.URL.Query())
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoStitcherLocationFixture(project, "us-central1"),
		gcpVideoStitcherLocationFixture(project, "global"),
	}
	return respondGCPVideoStitcherList(w, "locations", items, pageSize, offset, path)
}

func handleGCPVideoStitcherGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVideoStitcherLocationFixture(project, location))
	return true
}

func handleGCPVideoStitcherListLocationsByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := strings.TrimSpace(gcpVideoStitcherString(body, "name", "parent"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project := strings.TrimPrefix(name, "projects/")
	if strings.Contains(project, "/") || !gcpVideoStitcherProjectPattern.MatchString(project) {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoStitcherPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoStitcherLocationFixture(project, "us-central1"),
		gcpVideoStitcherLocationFixture(project, "global"),
	}
	return respondGCPVideoStitcherList(w, "locations", items, pageSize, offset, path)
}

func handleGCPVideoStitcherGetLocationByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := strings.TrimSpace(gcpVideoStitcherString(body, "name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoStitcherLocationFixture(project, location))
	return true
}

func handleGCPVideoStitcherCreateCdnKey(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherCreateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedCdnKeySpec())
}

func handleGCPVideoStitcherListCdnKeys(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherListManagedResource(w, path, body, ctx, gcpVideoStitcherManagedCdnKeySpec())
}

func handleGCPVideoStitcherGetCdnKey(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherGetManagedResource(w, path, body, ctx, gcpVideoStitcherManagedCdnKeySpec())
}

func handleGCPVideoStitcherDeleteCdnKey(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherDeleteManagedResource(w, path, body, ctx, gcpVideoStitcherManagedCdnKeySpec())
}

func handleGCPVideoStitcherUpdateCdnKey(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherUpdateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedCdnKeySpec())
}

func handleGCPVideoStitcherCreateSlate(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherCreateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedSlateSpec())
}

func handleGCPVideoStitcherListSlates(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherListManagedResource(w, path, body, ctx, gcpVideoStitcherManagedSlateSpec())
}

func handleGCPVideoStitcherGetSlate(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherGetManagedResource(w, path, body, ctx, gcpVideoStitcherManagedSlateSpec())
}

func handleGCPVideoStitcherDeleteSlate(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherDeleteManagedResource(w, path, body, ctx, gcpVideoStitcherManagedSlateSpec())
}

func handleGCPVideoStitcherUpdateSlate(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherUpdateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedSlateSpec())
}

func handleGCPVideoStitcherCreateLiveConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherCreateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedLiveConfigSpec())
}

func handleGCPVideoStitcherListLiveConfigs(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherListManagedResource(w, path, body, ctx, gcpVideoStitcherManagedLiveConfigSpec())
}

func handleGCPVideoStitcherGetLiveConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherGetManagedResource(w, path, body, ctx, gcpVideoStitcherManagedLiveConfigSpec())
}

func handleGCPVideoStitcherDeleteLiveConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherDeleteManagedResource(w, path, body, ctx, gcpVideoStitcherManagedLiveConfigSpec())
}

func handleGCPVideoStitcherUpdateLiveConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherUpdateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedLiveConfigSpec())
}

func handleGCPVideoStitcherCreateVodConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherCreateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedVodConfigSpec())
}

func handleGCPVideoStitcherListVodConfigs(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherListManagedResource(w, path, body, ctx, gcpVideoStitcherManagedVodConfigSpec())
}

func handleGCPVideoStitcherGetVodConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherGetManagedResource(w, path, body, ctx, gcpVideoStitcherManagedVodConfigSpec())
}

func handleGCPVideoStitcherDeleteVodConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherDeleteManagedResource(w, path, body, ctx, gcpVideoStitcherManagedVodConfigSpec())
}

func handleGCPVideoStitcherUpdateVodConfig(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	return handleGCPVideoStitcherUpdateManagedResource(w, path, body, ctx, gcpVideoStitcherManagedVodConfigSpec())
}

func handleGCPVideoStitcherCreateManagedResource(
	w http.ResponseWriter,
	path string,
	body map[string]any,
	ctx gcpVideoStitcherRouteContext,
	spec gcpVideoStitcherManagedResourceSpec,
) bool {
	parent := gcpVideoStitcherResolveParent(body, ctx)
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "parent is required")
		return true
	}
	resource := gcpVideoStitcherBodyMap(body, spec.bodyKeys...)
	if len(resource) == 0 {
		respondGCPVideoStitcherInvalidArgument(w, path, spec.singular+" is required")
		return true
	}
	if spec.validateCreate != nil {
		if msg := strings.TrimSpace(spec.validateCreate(resource)); msg != "" {
			respondGCPVideoStitcherInvalidArgument(w, path, msg)
			return true
		}
	}
	resourceID := gcpVideoStitcherResolveCreateID(body, resource, ctx.Query, spec)
	if resourceID == "" {
		respondGCPVideoStitcherInvalidArgument(w, path, spec.idLabel+" is required")
		return true
	}
	if !gcpVideoStitcherIDPattern.MatchString(resourceID) {
		respondGCPVideoStitcherInvalidArgument(w, path, spec.idLabel+" is invalid")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/%s/%s", project, location, spec.collection, resourceID)
	if name := strings.TrimSpace(gcpVideoStitcherString(resource, "name")); name != "" && name != expectedName {
		respondGCPVideoStitcherInvalidArgument(w, path, spec.singular+".name must match parent and "+spec.idLabel)
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoStitcherOperationFixture(parent, spec.createOpPrefix+"."+resourceID, expectedName, "create", true))
	return true
}

func handleGCPVideoStitcherListManagedResource(
	w http.ResponseWriter,
	path string,
	body map[string]any,
	ctx gcpVideoStitcherRouteContext,
	spec gcpVideoStitcherManagedResourceSpec,
) bool {
	parent := gcpVideoStitcherResolveParent(body, ctx)
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoStitcherPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := make([]map[string]any, 0, len(spec.listFixtureIDs))
	for _, id := range spec.listFixtureIDs {
		items = append(items, spec.fixture(project, location, id))
	}
	return respondGCPVideoStitcherList(w, spec.listField, items, pageSize, offset, path)
}

func handleGCPVideoStitcherGetManagedResource(
	w http.ResponseWriter,
	path string,
	body map[string]any,
	ctx gcpVideoStitcherRouteContext,
	spec gcpVideoStitcherManagedResourceSpec,
) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, resourceID, ok := gcpVideoStitcherParseManagedResourceName(name, spec.collection)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(resourceID) {
		respondGCPVideoStitcherNotFound(w, path, spec.singular+" not found")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, spec.fixture(project, location, resourceID))
	return true
}

func handleGCPVideoStitcherDeleteManagedResource(
	w http.ResponseWriter,
	path string,
	body map[string]any,
	ctx gcpVideoStitcherRouteContext,
	spec gcpVideoStitcherManagedResourceSpec,
) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, resourceID, ok := gcpVideoStitcherParseManagedResourceName(name, spec.collection)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(resourceID) {
		respondGCPVideoStitcherNotFound(w, path, spec.singular+" not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoStitcherOperationFixture(parent, spec.deleteOpPrefix+"."+resourceID, name, "delete", true))
	return true
}

func handleGCPVideoStitcherUpdateManagedResource(
	w http.ResponseWriter,
	path string,
	body map[string]any,
	ctx gcpVideoStitcherRouteContext,
	spec gcpVideoStitcherManagedResourceSpec,
) bool {
	resource := gcpVideoStitcherBodyMap(body, spec.bodyKeys...)
	if len(resource) == 0 {
		respondGCPVideoStitcherInvalidArgument(w, path, spec.singular+" is required")
		return true
	}
	if strings.TrimSpace(gcpVideoStitcherString(body, "updateMask", "update_mask")) == "" && gcpVideoStitcherUpdateMaskFromQuery(ctx.Query) == "" {
		respondGCPVideoStitcherInvalidArgument(w, path, "update_mask is required")
		return true
	}
	name := strings.TrimSpace(gcpVideoStitcherString(resource, "name"))
	if name == "" {
		name = gcpVideoStitcherResolveName(body, ctx)
	}
	parent, resourceID, ok := gcpVideoStitcherParseManagedResourceName(name, spec.collection)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, spec.singular+".name is required")
		return true
	}
	if ctx.Name != "" && name != ctx.Name {
		respondGCPVideoStitcherInvalidArgument(w, path, spec.singular+".name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoStitcherOperationFixture(parent, spec.updateOpPrefix+"."+resourceID, name, "update", true))
	return true
}

func handleGCPVideoStitcherCreateVodSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	parent := gcpVideoStitcherResolveParent(body, ctx)
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "parent is required")
		return true
	}
	vodSession := gcpVideoStitcherBodyMap(body, "vodSession", "vod_session")
	if len(vodSession) == 0 {
		respondGCPVideoStitcherInvalidArgument(w, path, "vod_session is required")
		return true
	}
	vodConfig := strings.TrimSpace(gcpVideoStitcherString(vodSession, "vodConfig", "vod_config"))
	sourceURI := strings.TrimSpace(gcpVideoStitcherString(vodSession, "sourceUri", "source_uri"))
	if vodConfig == "" && sourceURI == "" {
		respondGCPVideoStitcherInvalidArgument(w, path, "vod_session.vod_config or vod_session.source_uri is required")
		return true
	}
	sessionID := "vod-session-1"
	if name := strings.TrimSpace(gcpVideoStitcherString(vodSession, "name")); name != "" {
		parsedParent, parsedID, ok := gcpVideoStitcherParseVodSessionName(name)
		if !ok {
			respondGCPVideoStitcherInvalidArgument(w, path, "vod_session.name is invalid")
			return true
		}
		if parsedParent != parent {
			respondGCPVideoStitcherInvalidArgument(w, path, "vod_session.name must match parent")
			return true
		}
		sessionID = parsedID
	}
	resp := gcpVideoStitcherVodSessionFixture(project, location, sessionID)
	if vodConfig != "" {
		resp["vodConfig"] = vodConfig
	}
	if sourceURI != "" {
		resp["sourceUri"] = sourceURI
	}
	if adTagURI := strings.TrimSpace(gcpVideoStitcherString(vodSession, "adTagUri", "ad_tag_uri")); adTagURI != "" {
		resp["adTagUri"] = adTagURI
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPVideoStitcherGetVodSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, sessionID, ok := gcpVideoStitcherParseVodSessionName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(sessionID) {
		respondGCPVideoStitcherNotFound(w, path, "vod_session not found")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoStitcherVodSessionFixture(project, location, sessionID))
	return true
}

func handleGCPVideoStitcherListVodStitchDetails(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	parent := gcpVideoStitcherResolveParent(body, ctx)
	_, sessionID, ok := gcpVideoStitcherParseVodSessionName(parent)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "parent is required")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(strings.TrimSuffix(parent, "/vodSessions/"+sessionID))
	pageSize, offset, valid := parseGCPVideoStitcherPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoStitcherVodStitchDetailFixture(project, location, sessionID, "stitch-1"),
		gcpVideoStitcherVodStitchDetailFixture(project, location, sessionID, "stitch-2"),
	}
	return respondGCPVideoStitcherList(w, "vodStitchDetails", items, pageSize, offset, path)
}

func handleGCPVideoStitcherGetVodStitchDetail(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, sessionID, detailID, ok := gcpVideoStitcherParseVodStitchDetailName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(detailID) {
		respondGCPVideoStitcherNotFound(w, path, "vod_stitch_detail not found")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoStitcherVodStitchDetailFixture(project, location, sessionID, detailID))
	return true
}

func handleGCPVideoStitcherListVodAdTagDetails(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	parent := gcpVideoStitcherResolveParent(body, ctx)
	_, sessionID, ok := gcpVideoStitcherParseVodSessionName(parent)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "parent is required")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(strings.TrimSuffix(parent, "/vodSessions/"+sessionID))
	pageSize, offset, valid := parseGCPVideoStitcherPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoStitcherVodAdTagDetailFixture(project, location, sessionID, "adtag-1"),
		gcpVideoStitcherVodAdTagDetailFixture(project, location, sessionID, "adtag-2"),
	}
	return respondGCPVideoStitcherList(w, "vodAdTagDetails", items, pageSize, offset, path)
}

func handleGCPVideoStitcherGetVodAdTagDetail(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, sessionID, detailID, ok := gcpVideoStitcherParseVodAdTagDetailName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(detailID) {
		respondGCPVideoStitcherNotFound(w, path, "vod_ad_tag_detail not found")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoStitcherVodAdTagDetailFixture(project, location, sessionID, detailID))
	return true
}

func handleGCPVideoStitcherCreateLiveSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	parent := gcpVideoStitcherResolveParent(body, ctx)
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "parent is required")
		return true
	}
	liveSession := gcpVideoStitcherBodyMap(body, "liveSession", "live_session")
	if len(liveSession) == 0 {
		respondGCPVideoStitcherInvalidArgument(w, path, "live_session is required")
		return true
	}
	liveConfig := strings.TrimSpace(gcpVideoStitcherString(liveSession, "liveConfig", "live_config"))
	if liveConfig == "" {
		respondGCPVideoStitcherInvalidArgument(w, path, "live_session.live_config is required")
		return true
	}
	sessionID := "live-session-1"
	if name := strings.TrimSpace(gcpVideoStitcherString(liveSession, "name")); name != "" {
		parsedParent, parsedID, ok := gcpVideoStitcherParseLiveSessionName(name)
		if !ok {
			respondGCPVideoStitcherInvalidArgument(w, path, "live_session.name is invalid")
			return true
		}
		if parsedParent != parent {
			respondGCPVideoStitcherInvalidArgument(w, path, "live_session.name must match parent")
			return true
		}
		sessionID = parsedID
	}
	resp := gcpVideoStitcherLiveSessionFixture(project, location, sessionID)
	resp["liveConfig"] = liveConfig
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPVideoStitcherGetLiveSession(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, sessionID, ok := gcpVideoStitcherParseLiveSessionName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(sessionID) {
		respondGCPVideoStitcherNotFound(w, path, "live_session not found")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoStitcherLiveSessionFixture(project, location, sessionID))
	return true
}

func handleGCPVideoStitcherListLiveAdTagDetails(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	parent := gcpVideoStitcherResolveParent(body, ctx)
	_, sessionID, ok := gcpVideoStitcherParseLiveSessionName(parent)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "parent is required")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(strings.TrimSuffix(parent, "/liveSessions/"+sessionID))
	pageSize, offset, valid := parseGCPVideoStitcherPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoStitcherLiveAdTagDetailFixture(project, location, sessionID, "adtag-1"),
		gcpVideoStitcherLiveAdTagDetailFixture(project, location, sessionID, "adtag-2"),
	}
	return respondGCPVideoStitcherList(w, "liveAdTagDetails", items, pageSize, offset, path)
}

func handleGCPVideoStitcherGetLiveAdTagDetail(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, sessionID, detailID, ok := gcpVideoStitcherParseLiveAdTagDetailName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(detailID) {
		respondGCPVideoStitcherNotFound(w, path, "live_ad_tag_detail not found")
		return true
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoStitcherLiveAdTagDetailFixture(project, location, sessionID, detailID))
	return true
}

func handleGCPVideoStitcherGetOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	parent, opID, ok := gcpVideoStitcherParseOperationName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(opID) {
		respondGCPVideoStitcherNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoStitcherOperationFixture(parent, opID, parent+"/operations/"+opID, "operate", true))
	return true
}

func handleGCPVideoStitcherListOperations(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := strings.TrimSpace(gcpVideoStitcherString(body, "name", "parent"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	if name == "" {
		name = strings.TrimSpace(ctx.Parent)
	}
	if strings.HasSuffix(name, "/operations") {
		name = strings.TrimSuffix(name, "/operations")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoStitcherPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	items := []map[string]any{
		gcpVideoStitcherOperationFixture(parent, "createCdnKey.cdn-key-1", parent+"/cdnKeys/cdn-key-1", "create", true),
		gcpVideoStitcherOperationFixture(parent, "createVodConfig.vod-config-1", parent+"/vodConfigs/vod-config-1", "create", true),
	}
	return respondGCPVideoStitcherList(w, "operations", items, pageSize, offset, path)
}

func handleGCPVideoStitcherCancelOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	_, opID, ok := gcpVideoStitcherParseOperationName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(opID) {
		respondGCPVideoStitcherNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVideoStitcherDeleteOperation(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoStitcherRouteContext) bool {
	name := gcpVideoStitcherResolveName(body, ctx)
	_, opID, ok := gcpVideoStitcherParseOperationName(name)
	if !ok {
		respondGCPVideoStitcherInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoStitcherMissingID(opID) {
		respondGCPVideoStitcherNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func decodeGCPVideoStitcherJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
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
		respondGCPVideoStitcherInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPVideoStitcherPagination(w http.ResponseWriter, path string, body map[string]any, query url.Values) (int, int, bool) {
	pageSize := 50
	offset := 0

	if query != nil {
		if raw := strings.TrimSpace(query.Get("pageSize")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				respondGCPVideoStitcherInvalidArgument(w, path, "pageSize must be between 0 and 1000")
				return 0, 0, false
			}
			pageSize = parsed
		}
		if raw := strings.TrimSpace(query.Get("pageToken")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 {
				respondGCPVideoStitcherInvalidArgument(w, path, "pageToken must be a non-negative integer")
				return 0, 0, false
			}
			offset = parsed
		}
	}

	if body != nil {
		if raw := strings.TrimSpace(gcpVideoStitcherString(body, "pageSize", "page_size")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				respondGCPVideoStitcherInvalidArgument(w, path, "pageSize must be between 0 and 1000")
				return 0, 0, false
			}
			pageSize = parsed
		}
		if raw := strings.TrimSpace(gcpVideoStitcherString(body, "pageToken", "page_token")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 {
				respondGCPVideoStitcherInvalidArgument(w, path, "pageToken must be a non-negative integer")
				return 0, 0, false
			}
			offset = parsed
		}
	}

	if pageSize < 0 || pageSize > 1000 {
		respondGCPVideoStitcherInvalidArgument(w, path, "pageSize must be between 0 and 1000")
		return 0, 0, false
	}
	return pageSize, offset, true
}

func respondGCPVideoStitcherList(w http.ResponseWriter, field string, items []map[string]any, pageSize, offset int, path string) bool {
	if offset < 0 || offset > len(items) {
		respondGCPVideoStitcherInvalidArgument(w, path, "pageToken out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && offset+pageSize < end {
		end = offset + pageSize
	}
	if pageSize == 0 {
		end = offset
	}

	window := make([]map[string]any, 0, end-offset)
	if offset < end {
		window = append(window, items[offset:end]...)
	}

	nextToken := ""
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		field:           window,
		"nextPageToken": nextToken,
	})
	return true
}

func parseGCPVideoStitcherLocationTail(path string) (project, location, tail string, ok bool) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 6 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if !gcpVideoStitcherProjectPattern.MatchString(project) || !gcpVideoStitcherLocationPattern.MatchString(location) {
		return "", "", "", false
	}
	if len(parts) > 6 {
		tail = "/" + strings.Join(parts[6:], "/")
	}
	return project, location, tail, true
}

func gcpVideoStitcherTailParts(tail string) []string {
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

func gcpVideoStitcherSplitIDAction(raw string) (id, action string, hasAction bool) {
	id = strings.TrimSpace(raw)
	if id == "" {
		return "", "", false
	}
	lhs, rhs, ok := strings.Cut(id, ":")
	if !ok {
		return id, "", false
	}
	lhs = strings.TrimSpace(lhs)
	rhs = strings.TrimSpace(rhs)
	if lhs == "" || rhs == "" {
		return "", "", false
	}
	return lhs, rhs, true
}

func gcpVideoStitcherResolveParent(body map[string]any, ctx gcpVideoStitcherRouteContext) string {
	parent := strings.TrimSpace(gcpVideoStitcherString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	return parent
}

func gcpVideoStitcherResolveName(body map[string]any, ctx gcpVideoStitcherRouteContext) string {
	name := strings.TrimSpace(gcpVideoStitcherString(body, "name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	return name
}

func gcpVideoStitcherProjectLocationFromParent(parent string) (string, string, bool) {
	return gcpVideoStitcherProjectLocationFromName(parent)
}

func gcpVideoStitcherProjectLocationFromName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project := strings.TrimSpace(parts[1])
	location := strings.TrimSpace(parts[3])
	if !gcpVideoStitcherProjectPattern.MatchString(project) || !gcpVideoStitcherLocationPattern.MatchString(location) {
		return "", "", false
	}
	return project, location, true
}

func gcpVideoStitcherParseManagedResourceName(name, collection string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != collection {
		return "", "", false
	}
	if !gcpVideoStitcherProjectPattern.MatchString(parts[1]) || !gcpVideoStitcherLocationPattern.MatchString(parts[3]) || !gcpVideoStitcherIDPattern.MatchString(parts[5]) {
		return "", "", false
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3])
	return parent, parts[5], true
}

func gcpVideoStitcherParseVodSessionName(name string) (string, string, bool) {
	return gcpVideoStitcherParseManagedResourceName(name, "vodSessions")
}

func gcpVideoStitcherParseLiveSessionName(name string) (string, string, bool) {
	return gcpVideoStitcherParseManagedResourceName(name, "liveSessions")
}

func gcpVideoStitcherParseDetailName(name, sessionCollection, detailCollection string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != sessionCollection || parts[6] != detailCollection {
		return "", "", "", false
	}
	if !gcpVideoStitcherProjectPattern.MatchString(parts[1]) || !gcpVideoStitcherLocationPattern.MatchString(parts[3]) || !gcpVideoStitcherIDPattern.MatchString(parts[5]) || !gcpVideoStitcherIDPattern.MatchString(parts[7]) {
		return "", "", "", false
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3])
	return parent, parts[5], parts[7], true
}

func gcpVideoStitcherParseVodStitchDetailName(name string) (string, string, string, bool) {
	return gcpVideoStitcherParseDetailName(name, "vodSessions", "vodStitchDetails")
}

func gcpVideoStitcherParseVodAdTagDetailName(name string) (string, string, string, bool) {
	return gcpVideoStitcherParseDetailName(name, "vodSessions", "vodAdTagDetails")
}

func gcpVideoStitcherParseLiveAdTagDetailName(name string) (string, string, string, bool) {
	return gcpVideoStitcherParseDetailName(name, "liveSessions", "liveAdTagDetails")
}

func gcpVideoStitcherParseOperationName(name string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", false
	}
	if !gcpVideoStitcherProjectPattern.MatchString(parts[1]) || !gcpVideoStitcherLocationPattern.MatchString(parts[3]) || !gcpVideoStitcherOperationIDPattern.MatchString(parts[5]) {
		return "", "", false
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", parts[1], parts[3])
	return parent, parts[5], true
}

func isGCPVideoStitcherMissingID(id string) bool {
	value := strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(value, "missing") || strings.Contains(value, "not-found") || strings.Contains(value, "does-not-exist")
}

func gcpVideoStitcherManagedCdnKeySpec() gcpVideoStitcherManagedResourceSpec {
	return gcpVideoStitcherManagedResourceSpec{
		collection:     "cdnKeys",
		singular:       "cdn_key",
		idKeys:         []string{"cdnKeyId", "cdn_key_id"},
		bodyKeys:       []string{"cdnKey", "cdn_key"},
		idLabel:        "cdn_key_id",
		listField:      "cdnKeys",
		listFixtureIDs: []string{"cdn-key-1", "cdn-key-2"},
		fixture:        gcpVideoStitcherCdnKeyFixture,
		createOpPrefix: "createCdnKey",
		updateOpPrefix: "updateCdnKey",
		deleteOpPrefix: "deleteCdnKey",
	}
}

func gcpVideoStitcherManagedSlateSpec() gcpVideoStitcherManagedResourceSpec {
	return gcpVideoStitcherManagedResourceSpec{
		collection:     "slates",
		singular:       "slate",
		idKeys:         []string{"slateId", "slate_id"},
		bodyKeys:       []string{"slate"},
		idLabel:        "slate_id",
		listField:      "slates",
		listFixtureIDs: []string{"slate-1", "slate-2"},
		fixture:        gcpVideoStitcherSlateFixture,
		createOpPrefix: "createSlate",
		updateOpPrefix: "updateSlate",
		deleteOpPrefix: "deleteSlate",
		validateCreate: func(resource map[string]any) string {
			if strings.TrimSpace(gcpVideoStitcherString(resource, "uri")) == "" {
				return "slate.uri is required"
			}
			return ""
		},
	}
}

func gcpVideoStitcherManagedLiveConfigSpec() gcpVideoStitcherManagedResourceSpec {
	return gcpVideoStitcherManagedResourceSpec{
		collection:     "liveConfigs",
		singular:       "live_config",
		idKeys:         []string{"liveConfigId", "live_config_id"},
		bodyKeys:       []string{"liveConfig", "live_config"},
		idLabel:        "live_config_id",
		listField:      "liveConfigs",
		listFixtureIDs: []string{"live-config-1", "live-config-2"},
		fixture:        gcpVideoStitcherLiveConfigFixture,
		createOpPrefix: "createLiveConfig",
		updateOpPrefix: "updateLiveConfig",
		deleteOpPrefix: "deleteLiveConfig",
		validateCreate: func(resource map[string]any) string {
			if strings.TrimSpace(gcpVideoStitcherString(resource, "sourceUri", "source_uri")) == "" {
				return "live_config.source_uri is required"
			}
			return ""
		},
	}
}

func gcpVideoStitcherManagedVodConfigSpec() gcpVideoStitcherManagedResourceSpec {
	return gcpVideoStitcherManagedResourceSpec{
		collection:     "vodConfigs",
		singular:       "vod_config",
		idKeys:         []string{"vodConfigId", "vod_config_id"},
		bodyKeys:       []string{"vodConfig", "vod_config"},
		idLabel:        "vod_config_id",
		listField:      "vodConfigs",
		listFixtureIDs: []string{"vod-config-1", "vod-config-2"},
		fixture:        gcpVideoStitcherVodConfigFixture,
		createOpPrefix: "createVodConfig",
		updateOpPrefix: "updateVodConfig",
		deleteOpPrefix: "deleteVodConfig",
		validateCreate: func(resource map[string]any) string {
			if strings.TrimSpace(gcpVideoStitcherString(resource, "sourceUri", "source_uri")) == "" {
				return "vod_config.source_uri is required"
			}
			if strings.TrimSpace(gcpVideoStitcherString(resource, "adTagUri", "ad_tag_uri")) == "" {
				return "vod_config.ad_tag_uri is required"
			}
			return ""
		},
	}
}

func gcpVideoStitcherResolveCreateID(body, resource map[string]any, query url.Values, spec gcpVideoStitcherManagedResourceSpec) string {
	for _, key := range spec.idKeys {
		if value := strings.TrimSpace(gcpVideoStitcherString(body, key)); value != "" {
			return value
		}
	}
	if query != nil {
		for _, key := range spec.idKeys {
			if value := strings.TrimSpace(query.Get(key)); value != "" {
				return value
			}
		}
	}
	if name := strings.TrimSpace(gcpVideoStitcherString(resource, "name")); name != "" {
		_, id, ok := gcpVideoStitcherParseManagedResourceName(name, spec.collection)
		if ok {
			return id
		}
	}
	return ""
}

func gcpVideoStitcherLocationFixture(project, location string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s", project, location)
	return map[string]any{
		"name":        name,
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"metadata": map[string]any{
			"service": "video-stitcher",
		},
	}
}

func gcpVideoStitcherCdnKeyFixture(project, location, id string) map[string]any {
	return map[string]any{
		"name":     fmt.Sprintf("projects/%s/locations/%s/cdnKeys/%s", project, location, id),
		"hostname": fmt.Sprintf("%s.example.com", id),
		"googleCdnKey": map[string]any{
			"keyName": "key-" + id,
		},
	}
}

func gcpVideoStitcherSlateFixture(project, location, id string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/slates/%s", project, location, id),
		"uri":  fmt.Sprintf("https://cdn.example.com/slates/%s.mp4", id),
	}
}

func gcpVideoStitcherLiveConfigFixture(project, location, id string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/locations/%s/liveConfigs/%s", project, location, id),
		"sourceUri": fmt.Sprintf("https://origin.example.com/live/%s.m3u8", id),
		"adTagUri":  fmt.Sprintf("https://ads.example.com/live/%s", id),
		"state":     "READY",
	}
}

func gcpVideoStitcherVodConfigFixture(project, location, id string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/locations/%s/vodConfigs/%s", project, location, id),
		"sourceUri": fmt.Sprintf("https://origin.example.com/vod/%s.m3u8", id),
		"adTagUri":  fmt.Sprintf("https://ads.example.com/vod/%s", id),
		"state":     "READY",
	}
}

func gcpVideoStitcherVodSessionFixture(project, location, id string) map[string]any {
	vodConfig := fmt.Sprintf("projects/%s/locations/%s/vodConfigs/vod-config-1", project, location)
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/locations/%s/vodSessions/%s", project, location, id),
		"playUri":   fmt.Sprintf("https://play.example.com/vod/%s/master.m3u8", id),
		"vodConfig": vodConfig,
		"sourceUri": "https://origin.example.com/vod/source.m3u8",
		"adTagUri":  "https://ads.example.com/vod/default",
	}
}

func gcpVideoStitcherLiveSessionFixture(project, location, id string) map[string]any {
	liveConfig := fmt.Sprintf("projects/%s/locations/%s/liveConfigs/live-config-1", project, location)
	return map[string]any{
		"name":       fmt.Sprintf("projects/%s/locations/%s/liveSessions/%s", project, location, id),
		"playUri":    fmt.Sprintf("https://play.example.com/live/%s/master.m3u8", id),
		"liveConfig": liveConfig,
	}
}

func gcpVideoStitcherVodStitchDetailFixture(project, location, sessionID, detailID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/vodSessions/%s/vodStitchDetails/%s", project, location, sessionID, detailID),
		"adStitchDetails": []map[string]any{
			{"adBreakId": "break-1"},
		},
	}
}

func gcpVideoStitcherVodAdTagDetailFixture(project, location, sessionID, detailID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/vodSessions/%s/vodAdTagDetails/%s", project, location, sessionID, detailID),
		"adRequests": []map[string]any{
			{"uri": "https://ads.example.com/vod/request"},
		},
	}
}

func gcpVideoStitcherLiveAdTagDetailFixture(project, location, sessionID, detailID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/liveSessions/%s/liveAdTagDetails/%s", project, location, sessionID, detailID),
		"adRequests": []map[string]any{
			{"uri": "https://ads.example.com/live/request"},
		},
	}
}

func gcpVideoStitcherOperationFixture(parent, opID, target, verb string, done bool) map[string]any {
	name := parent + "/operations/" + opID
	return map[string]any{
		"name": name,
		"done": done,
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.video.stitcher.v1.OperationMetadata",
			"createTime": gcpVideoStitcherReferenceTime.Format(time.RFC3339Nano),
			"endTime":    gcpVideoStitcherReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
			"target":     target,
			"verb":       verb,
		},
		"response": map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	}
}

func gcpVideoStitcherString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if body == nil {
			continue
		}
		if value, ok := body[key]; ok {
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return strings.TrimSpace(typed)
				}
			case json.Number:
				return strings.TrimSpace(typed.String())
			}
		}
	}
	return ""
}

func gcpVideoStitcherBodyMap(body map[string]any, keys ...string) map[string]any {
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

func gcpVideoStitcherUpdateMaskFromQuery(query url.Values) string {
	if query == nil {
		return ""
	}
	if value := strings.TrimSpace(query.Get("updateMask")); value != "" {
		return value
	}
	if value := strings.TrimSpace(query.Get("update_mask")); value != "" {
		return value
	}
	return ""
}

func respondGCPVideoStitcherInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVideoStitcherNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_video_stitcher(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "video_stitcher") &&
		!isGCPContractProbeRequestForService(r, path, "video-stitcher") &&
		!isGCPContractProbeRequestForService(r, path, "stitcher") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPVideoStitcherInvalidArgument(w, path, "pageSize must be between 0 and 1000")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/video-stitcher",
			"service":  "video_stitcher",
			"provider": providerGCP,
			"path":     path,
			"methods": []string{
				"CreateCdnKey",
				"CreateVodSession",
				"CreateSlate",
				"CreateLiveConfig",
				"CreateVodConfig",
				"ListOperations",
			},
		})
		return true
	}
	return false
}
