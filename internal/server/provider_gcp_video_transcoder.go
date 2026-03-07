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

const gcpVideoTranscoderGRPCPathPrefix = "/gcp/google.cloud.video.transcoder.v1.TranscoderService/"

var (
	gcpVideoTranscoderReferenceTime      = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpVideoTranscoderProjectPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)
	gcpVideoTranscoderLocationPattern    = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)
	gcpVideoTranscoderJobIDPattern       = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	gcpVideoTranscoderJobTemplatePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{3,62}$`)
)

type gcpVideoTranscoderRouteContext struct {
	Parent string
	Name   string
	Query  url.Values
}

func (s *Server) handleGCPVideoTranscoderRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_video_transcoder(w, r) {
		return true
	}

	path := normalizeGCPVideoTranscoderPath(rawRequestPath(r))
	if isGCPVideoTranscoderLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVideoTranscoderListLocations(w, r, path) {
			return true
		}
		if handleGCPVideoTranscoderGetLocation(w, path) {
			return true
		}
		return false
	}

	if strings.HasPrefix(path, gcpVideoTranscoderGRPCPathPrefix) {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPVideoTranscoderJSONBody(w, r, path)
		if !ok {
			return true
		}
		method := strings.TrimSpace(strings.TrimPrefix(path, gcpVideoTranscoderGRPCPathPrefix))
		if method == "" {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		if handleGCPVideoTranscoderRPCMethod(w, path, method, body, gcpVideoTranscoderRouteContext{Query: r.URL.Query()}) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if !isGCPVideoTranscoderPath(path, hasGCPVideoTranscoderHint(r)) {
		return false
	}

	method, ctx, needsBody, ok := mapGCPVideoTranscoderRESTToMethod(r, path)
	if !ok {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	body := map[string]any{}
	if needsBody {
		var valid bool
		body, valid = decodeGCPVideoTranscoderJSONBody(w, r, path)
		if !valid {
			return true
		}
	}

	if handleGCPVideoTranscoderRPCMethod(w, path, method, body, ctx) {
		return true
	}
	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func normalizeGCPVideoTranscoderPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVideoTranscoderHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "video_transcoder",
		"video-transcoder",
		"video-transcoder-apiv1",
		"video_transcoder_apiv1",
		"transcoder",
		"transcoder-apiv1",
		"gcp-video-transcoder",
		"gcp-transcoder":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-video-transcoder-apiv1") || strings.Contains(ua, "cloud.google.com/go/video/transcoder")
}

func isGCPVideoTranscoderLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVideoTranscoderHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPVideoTranscoderPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpVideoTranscoderGRPCPathPrefix) {
		return true
	}
	if _, _, _, ok := parseGCPVideoTranscoderLocationTail(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPProjectLocationPath(path); ok {
		return includeHint
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/projects/")
}

func mapGCPVideoTranscoderRESTToMethod(r *http.Request, path string) (string, gcpVideoTranscoderRouteContext, bool, bool) {
	ctx := gcpVideoTranscoderRouteContext{Query: r.URL.Query()}

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
		return "", gcpVideoTranscoderRouteContext{}, false, false
	}

	project, location, tail, ok := parseGCPVideoTranscoderLocationTail(path)
	if !ok {
		return "", gcpVideoTranscoderRouteContext{}, false, false
	}
	ctx.Parent = fmt.Sprintf("projects/%s/locations/%s", project, location)

	parts := gcpVideoTranscoderTailParts(tail)
	if len(parts) == 0 {
		return "", gcpVideoTranscoderRouteContext{}, false, false
	}

	switch parts[0] {
	case "jobs":
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				return "ListJobs", ctx, false, true
			case http.MethodPost:
				return "CreateJob", ctx, true, true
			default:
				return "", gcpVideoTranscoderRouteContext{}, false, false
			}
		}
		if len(parts) != 2 {
			return "", gcpVideoTranscoderRouteContext{}, false, false
		}
		jobID := strings.TrimSpace(parts[1])
		if jobID == "" {
			return "", gcpVideoTranscoderRouteContext{}, false, false
		}
		ctx.Name = ctx.Parent + "/jobs/" + jobID
		switch r.Method {
		case http.MethodGet:
			return "GetJob", ctx, false, true
		case http.MethodDelete:
			return "DeleteJob", ctx, false, true
		default:
			return "", gcpVideoTranscoderRouteContext{}, false, false
		}
	case "jobTemplates":
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				return "ListJobTemplates", ctx, false, true
			case http.MethodPost:
				return "CreateJobTemplate", ctx, true, true
			default:
				return "", gcpVideoTranscoderRouteContext{}, false, false
			}
		}
		if len(parts) != 2 {
			return "", gcpVideoTranscoderRouteContext{}, false, false
		}
		templateID := strings.TrimSpace(parts[1])
		if templateID == "" {
			return "", gcpVideoTranscoderRouteContext{}, false, false
		}
		ctx.Name = ctx.Parent + "/jobTemplates/" + templateID
		switch r.Method {
		case http.MethodGet:
			return "GetJobTemplate", ctx, false, true
		case http.MethodDelete:
			return "DeleteJobTemplate", ctx, false, true
		default:
			return "", gcpVideoTranscoderRouteContext{}, false, false
		}
	default:
		return "", gcpVideoTranscoderRouteContext{}, false, false
	}
}

func handleGCPVideoTranscoderRPCMethod(w http.ResponseWriter, path, method string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	switch method {
	case "ListLocations":
		return handleGCPVideoTranscoderListLocationsByMethod(w, path, body, ctx)
	case "GetLocation":
		return handleGCPVideoTranscoderGetLocationByMethod(w, path, body, ctx)
	case "CreateJob":
		return handleGCPVideoTranscoderCreateJob(w, path, body, ctx)
	case "ListJobs":
		return handleGCPVideoTranscoderListJobs(w, path, body, ctx)
	case "GetJob":
		return handleGCPVideoTranscoderGetJob(w, path, body, ctx)
	case "DeleteJob":
		return handleGCPVideoTranscoderDeleteJob(w, path, body, ctx)
	case "CreateJobTemplate":
		return handleGCPVideoTranscoderCreateJobTemplate(w, path, body, ctx)
	case "ListJobTemplates":
		return handleGCPVideoTranscoderListJobTemplates(w, path, body, ctx)
	case "GetJobTemplate":
		return handleGCPVideoTranscoderGetJobTemplate(w, path, body, ctx)
	case "DeleteJobTemplate":
		return handleGCPVideoTranscoderDeleteJobTemplate(w, path, body, ctx)
	default:
		return false
	}
}

func handleGCPVideoTranscoderListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, offset, valid := parseGCPVideoTranscoderPagination(w, path, nil, r.URL.Query())
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoTranscoderLocationFixture(project, "us-central1"),
		gcpVideoTranscoderLocationFixture(project, "global"),
	}
	return respondGCPVideoTranscoderList(w, "locations", items, pageSize, offset, path)
}

func handleGCPVideoTranscoderGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVideoTranscoderLocationFixture(project, location))
	return true
}

func handleGCPVideoTranscoderListLocationsByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	name := strings.TrimSpace(gcpVideoTranscoderString(body, "name", "parent"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project := strings.TrimPrefix(name, "projects/")
	if strings.Contains(project, "/") || !gcpVideoTranscoderProjectPattern.MatchString(project) {
		respondGCPVideoTranscoderInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoTranscoderPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoTranscoderLocationFixture(project, "us-central1"),
		gcpVideoTranscoderLocationFixture(project, "global"),
	}
	return respondGCPVideoTranscoderList(w, "locations", items, pageSize, offset, path)
}

func handleGCPVideoTranscoderGetLocationByMethod(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	name := strings.TrimSpace(gcpVideoTranscoderString(body, "name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(name)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoTranscoderLocationFixture(project, location))
	return true
}

func handleGCPVideoTranscoderCreateJob(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	parent := gcpVideoTranscoderResolveParent(body, ctx)
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "parent is required")
		return true
	}
	job := gcpVideoTranscoderBodyMap(body, "job")
	if len(job) == 0 {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job is required")
		return true
	}
	inputURI := strings.TrimSpace(gcpVideoTranscoderString(job, "inputUri", "input_uri"))
	if inputURI == "" {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job.input_uri is required")
		return true
	}
	outputURI := strings.TrimSpace(gcpVideoTranscoderString(job, "outputUri", "output_uri"))
	if outputURI == "" {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job.output_uri is required")
		return true
	}
	templateID := strings.TrimSpace(gcpVideoTranscoderString(job, "templateId", "template_id"))
	config := gcpVideoTranscoderBodyMap(job, "config")
	if templateID == "" && len(config) == 0 {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job.template_id or job.config is required")
		return true
	}

	jobID := "job-1"
	if name := strings.TrimSpace(gcpVideoTranscoderString(job, "name")); name != "" {
		parsedParent, parsedID, ok := gcpVideoTranscoderParseJobName(name)
		if !ok {
			respondGCPVideoTranscoderInvalidArgument(w, path, "job.name is invalid")
			return true
		}
		if parsedParent != parent {
			respondGCPVideoTranscoderInvalidArgument(w, path, "job.name must match parent")
			return true
		}
		jobID = parsedID
	}

	resp := gcpVideoTranscoderJobFixture(project, location, jobID)
	resp["inputUri"] = inputURI
	resp["outputUri"] = outputURI
	if templateID != "" {
		resp["templateId"] = templateID
	}
	if len(config) != 0 {
		resp["config"] = config
		delete(resp, "templateId")
	}
	if labels := gcpVideoTranscoderBodyMap(job, "labels"); len(labels) != 0 {
		resp["labels"] = labels
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPVideoTranscoderListJobs(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	parent := gcpVideoTranscoderResolveParent(body, ctx)
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoTranscoderPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoTranscoderJobFixture(project, location, "job-1"),
		gcpVideoTranscoderJobFixture(project, location, "job-2"),
	}
	return respondGCPVideoTranscoderList(w, "jobs", items, pageSize, offset, path)
}

func handleGCPVideoTranscoderGetJob(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	name := gcpVideoTranscoderResolveName(body, ctx)
	parent, jobID, ok := gcpVideoTranscoderParseJobName(name)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoTranscoderMissingID(jobID) {
		respondGCPVideoTranscoderNotFound(w, path, "job not found")
		return true
	}
	project, location, _ := gcpVideoTranscoderProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoTranscoderJobFixture(project, location, jobID))
	return true
}

func handleGCPVideoTranscoderDeleteJob(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	name := gcpVideoTranscoderResolveName(body, ctx)
	_, jobID, ok := gcpVideoTranscoderParseJobName(name)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "name is required")
		return true
	}
	allowMissing := gcpVideoTranscoderBool(body, "allowMissing", "allow_missing")
	if isGCPVideoTranscoderMissingID(jobID) && !allowMissing {
		respondGCPVideoTranscoderNotFound(w, path, "job not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVideoTranscoderCreateJobTemplate(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	parent := gcpVideoTranscoderResolveParent(body, ctx)
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "parent is required")
		return true
	}
	jobTemplate := gcpVideoTranscoderBodyMap(body, "jobTemplate", "job_template")
	if len(jobTemplate) == 0 {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job_template is required")
		return true
	}

	jobTemplateID := strings.TrimSpace(gcpVideoTranscoderString(body, "jobTemplateId", "job_template_id"))
	if jobTemplateID == "" && ctx.Query != nil {
		jobTemplateID = strings.TrimSpace(ctx.Query.Get("jobTemplateId"))
	}
	if jobTemplateID == "" && ctx.Query != nil {
		jobTemplateID = strings.TrimSpace(ctx.Query.Get("job_template_id"))
	}
	if jobTemplateID == "" {
		if name := strings.TrimSpace(gcpVideoTranscoderString(jobTemplate, "name")); name != "" {
			_, parsedID, ok := gcpVideoTranscoderParseJobTemplateName(name)
			if !ok {
				respondGCPVideoTranscoderInvalidArgument(w, path, "job_template.name is invalid")
				return true
			}
			jobTemplateID = parsedID
		}
	}
	if jobTemplateID == "" {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job_template_id is required")
		return true
	}
	if !gcpVideoTranscoderJobTemplatePattern.MatchString(jobTemplateID) {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job_template_id is invalid")
		return true
	}

	expectedName := fmt.Sprintf("projects/%s/locations/%s/jobTemplates/%s", project, location, jobTemplateID)
	if name := strings.TrimSpace(gcpVideoTranscoderString(jobTemplate, "name")); name != "" && name != expectedName {
		respondGCPVideoTranscoderInvalidArgument(w, path, "job_template.name must match parent and job_template_id")
		return true
	}

	resp := gcpVideoTranscoderJobTemplateFixture(project, location, jobTemplateID)
	if config := gcpVideoTranscoderBodyMap(jobTemplate, "config"); len(config) != 0 {
		resp["config"] = config
	}
	if labels := gcpVideoTranscoderBodyMap(jobTemplate, "labels"); len(labels) != 0 {
		resp["labels"] = labels
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPVideoTranscoderListJobTemplates(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	parent := gcpVideoTranscoderResolveParent(body, ctx)
	project, location, ok := gcpVideoTranscoderProjectLocationFromParent(parent)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, offset, valid := parseGCPVideoTranscoderPagination(w, path, body, ctx.Query)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoTranscoderJobTemplateFixture(project, location, "template-1"),
		gcpVideoTranscoderJobTemplateFixture(project, location, "template-2"),
	}
	return respondGCPVideoTranscoderList(w, "jobTemplates", items, pageSize, offset, path)
}

func handleGCPVideoTranscoderGetJobTemplate(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	name := gcpVideoTranscoderResolveName(body, ctx)
	parent, jobTemplateID, ok := gcpVideoTranscoderParseJobTemplateName(name)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVideoTranscoderMissingID(jobTemplateID) {
		respondGCPVideoTranscoderNotFound(w, path, "job_template not found")
		return true
	}
	project, location, _ := gcpVideoTranscoderProjectLocationFromParent(parent)
	respondJSON(w, http.StatusOK, gcpVideoTranscoderJobTemplateFixture(project, location, jobTemplateID))
	return true
}

func handleGCPVideoTranscoderDeleteJobTemplate(w http.ResponseWriter, path string, body map[string]any, ctx gcpVideoTranscoderRouteContext) bool {
	name := gcpVideoTranscoderResolveName(body, ctx)
	_, jobTemplateID, ok := gcpVideoTranscoderParseJobTemplateName(name)
	if !ok {
		respondGCPVideoTranscoderInvalidArgument(w, path, "name is required")
		return true
	}
	allowMissing := gcpVideoTranscoderBool(body, "allowMissing", "allow_missing")
	if isGCPVideoTranscoderMissingID(jobTemplateID) && !allowMissing {
		respondGCPVideoTranscoderNotFound(w, path, "job_template not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func decodeGCPVideoTranscoderJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
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
		respondGCPVideoTranscoderInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPVideoTranscoderPagination(w http.ResponseWriter, path string, body map[string]any, query url.Values) (int, int, bool) {
	pageSize := 50
	offset := 0

	if query != nil {
		if raw := strings.TrimSpace(query.Get("pageSize")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				respondGCPVideoTranscoderInvalidArgument(w, path, "pageSize must be between 0 and 1000")
				return 0, 0, false
			}
			pageSize = parsed
		}
		if raw := strings.TrimSpace(query.Get("pageToken")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 {
				respondGCPVideoTranscoderInvalidArgument(w, path, "pageToken must be a non-negative integer")
				return 0, 0, false
			}
			offset = parsed
		}
	}

	if body != nil {
		if raw := strings.TrimSpace(gcpVideoTranscoderString(body, "pageSize", "page_size")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				respondGCPVideoTranscoderInvalidArgument(w, path, "pageSize must be between 0 and 1000")
				return 0, 0, false
			}
			pageSize = parsed
		}
		if raw := strings.TrimSpace(gcpVideoTranscoderString(body, "pageToken", "page_token")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 {
				respondGCPVideoTranscoderInvalidArgument(w, path, "pageToken must be a non-negative integer")
				return 0, 0, false
			}
			offset = parsed
		}
	}

	if pageSize < 0 || pageSize > 1000 {
		respondGCPVideoTranscoderInvalidArgument(w, path, "pageSize must be between 0 and 1000")
		return 0, 0, false
	}
	return pageSize, offset, true
}

func respondGCPVideoTranscoderList(w http.ResponseWriter, field string, items []map[string]any, pageSize, offset int, path string) bool {
	if offset < 0 || offset > len(items) {
		respondGCPVideoTranscoderInvalidArgument(w, path, "pageToken out of range")
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

	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		field:           window,
		"nextPageToken": nextPageToken,
	})
	return true
}

func parseGCPVideoTranscoderLocationTail(path string) (project, location, tail string, ok bool) {
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
	if !gcpVideoTranscoderProjectPattern.MatchString(project) || !gcpVideoTranscoderLocationPattern.MatchString(location) {
		return "", "", "", false
	}
	if len(parts) > 6 {
		tail = "/" + strings.Join(parts[6:], "/")
	}
	return project, location, tail, true
}

func gcpVideoTranscoderTailParts(tail string) []string {
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

func gcpVideoTranscoderResolveParent(body map[string]any, ctx gcpVideoTranscoderRouteContext) string {
	parent := strings.TrimSpace(gcpVideoTranscoderString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(ctx.Parent)
	}
	return parent
}

func gcpVideoTranscoderResolveName(body map[string]any, ctx gcpVideoTranscoderRouteContext) string {
	name := strings.TrimSpace(gcpVideoTranscoderString(body, "name"))
	if name == "" {
		name = strings.TrimSpace(ctx.Name)
	}
	return name
}

func gcpVideoTranscoderProjectLocationFromParent(parent string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(parent), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project := strings.TrimSpace(parts[1])
	location := strings.TrimSpace(parts[3])
	if !gcpVideoTranscoderProjectPattern.MatchString(project) || !gcpVideoTranscoderLocationPattern.MatchString(location) {
		return "", "", false
	}
	return project, location, true
}

func gcpVideoTranscoderParseJobName(name string) (string, string, bool) {
	return gcpVideoTranscoderParseNamedResource(name, "jobs", gcpVideoTranscoderJobIDPattern)
}

func gcpVideoTranscoderParseJobTemplateName(name string) (string, string, bool) {
	return gcpVideoTranscoderParseNamedResource(name, "jobTemplates", gcpVideoTranscoderJobTemplatePattern)
}

func gcpVideoTranscoderParseNamedResource(name, collection string, idPattern *regexp.Regexp) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != collection {
		return "", "", false
	}
	project := strings.TrimSpace(parts[1])
	location := strings.TrimSpace(parts[3])
	resourceID := strings.TrimSpace(parts[5])
	if !gcpVideoTranscoderProjectPattern.MatchString(project) || !gcpVideoTranscoderLocationPattern.MatchString(location) || !idPattern.MatchString(resourceID) {
		return "", "", false
	}
	return fmt.Sprintf("projects/%s/locations/%s", project, location), resourceID, true
}

func isGCPVideoTranscoderMissingID(id string) bool {
	value := strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(value, "missing") || strings.Contains(value, "not-found") || strings.Contains(value, "does-not-exist")
}

func gcpVideoTranscoderLocationFixture(project, location string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s", project, location)
	return map[string]any{
		"name":        name,
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"metadata": map[string]any{
			"service": "video-transcoder",
		},
	}
}

func gcpVideoTranscoderJobFixture(project, location, jobID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID)
	return map[string]any{
		"name":      name,
		"inputUri":  fmt.Sprintf("gs://stackyard-inputs/%s.mp4", jobID),
		"outputUri": fmt.Sprintf("gs://stackyard-outputs/%s/", jobID),
		"templateId": func() string {
			if strings.Contains(strings.ToLower(jobID), "config") {
				return ""
			}
			return "preset/web-hd"
		}(),
		"state":      "SUCCEEDED",
		"createTime": gcpVideoTranscoderReferenceTime.Format(time.RFC3339Nano),
		"startTime":  gcpVideoTranscoderReferenceTime.Add(5 * time.Second).Format(time.RFC3339Nano),
		"endTime":    gcpVideoTranscoderReferenceTime.Add(15 * time.Second).Format(time.RFC3339Nano),
		"labels": map[string]any{
			"env": "staged",
			"id":  jobID,
		},
	}
}

func gcpVideoTranscoderJobTemplateFixture(project, location, jobTemplateID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/jobTemplates/%s", project, location, jobTemplateID)
	return map[string]any{
		"name": name,
		"config": map[string]any{
			"inputs": []map[string]any{
				{
					"key": "input0",
					"uri": "gs://stackyard-inputs/template-input.mp4",
				},
			},
			"output": map[string]any{
				"uri": fmt.Sprintf("gs://stackyard-outputs/templates/%s/", jobTemplateID),
			},
		},
		"labels": map[string]any{
			"env": "staged",
			"id":  jobTemplateID,
		},
	}
}

func gcpVideoTranscoderString(body map[string]any, keys ...string) string {
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

func gcpVideoTranscoderBool(body map[string]any, keys ...string) bool {
	for _, key := range keys {
		if body == nil {
			continue
		}
		value, ok := body[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			return strings.EqualFold(strings.TrimSpace(typed), "true")
		}
	}
	return false
}

func gcpVideoTranscoderBodyMap(body map[string]any, keys ...string) map[string]any {
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

func respondGCPVideoTranscoderInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVideoTranscoderNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_video_transcoder(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "video_transcoder") &&
		!isGCPContractProbeRequestForService(r, path, "video-transcoder") &&
		!isGCPContractProbeRequestForService(r, path, "transcoder") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPVideoTranscoderInvalidArgument(w, path, "pageSize must be between 0 and 1000")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/video-transcoder",
			"service":  "video_transcoder",
			"provider": providerGCP,
			"path":     path,
			"methods": []string{
				"CreateJob",
				"ListJobs",
				"GetJob",
				"DeleteJob",
				"CreateJobTemplate",
				"ListJobTemplates",
				"GetJobTemplate",
				"DeleteJobTemplate",
			},
		})
		return true
	}
	return false
}
