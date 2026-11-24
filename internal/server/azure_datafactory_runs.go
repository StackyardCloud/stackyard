package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Data Factory pipeline runs, activity runs, run queries, trigger queries
// and integration runtime actions. Runs are stored as records under their
// factory (so deleting it removes them) but are not ARM resources: their
// bodies are flat PipelineRun objects, served by handleAzureDataFactoryV2Router.

const (
	azureADFRunType      = azureADFFactoryType + "/pipelineruns"
	azureADFMaxRuns      = 1000             // per factory; the oldest go first
	azureADFActivityKey  = "\x00activities" // run-only key; clients can't send a NUL key name
	azureADFMaxQueryPage = 100
)

// azureADFFactoryOf returns the factory ID a child resource ID is under.
func azureADFFactoryOf(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) < 9 {
		return id
	}
	return strings.Join(parts[:9], "/")
}

// azureADFCreateRun starts a run of the pipeline: the body holds its
// parameters. The run succeeds at once.
func azureADFCreateRun(s *Server, pipeline *azureARMResource, payload map[string]any) (int, any) {
	params := map[string]any{}
	for key, value := range payload {
		text, ok := value.(string)
		if !ok {
			b, _ := json.Marshal(value)
			text = string(b)
		}
		params[key] = text
	}
	var activities []any
	defined, _ := pipeline.Properties["activities"].([]any)
	for _, item := range defined {
		activity := azureARMObject(item)
		activities = append(activities, map[string]any{"name": azureARMString(activity["name"]), "type": azureARMString(activity["type"])})
	}
	now := time.Now().UTC()
	at := now.Format(time.RFC3339Nano)
	runID := uuid.NewString()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	factory := s.azureDataPlatStoredLocked(azureADFFactoryOf(pipeline.ID), azureADFFactoryType)
	if factory == nil {
		return http.StatusNotFound, azureARMErrorBody("ParentResourceNotFound", "The factory was not found.")
	}
	s.azureDataPlatPutChildLocked(factory, azureADFRunType, "pipelineruns", runID, map[string]any{
		"runId": runID, "runGroupId": runID, "isLatest": true, "pipelineName": pipeline.Name, "parameters": params,
		"invokedBy": map[string]any{"name": "Manual", "id": azureDataPlatGUID(runID + "/invoker"), "invokedByType": "Manual"},
		"runStart":  at, "runEnd": at, "lastUpdated": at, "durationInMs": 0,
		"status": "Succeeded", "message": "", "annotations": []any{}, "runDimensions": map[string]any{},
		azureADFActivityKey: activities,
	}, azureADFMaxRuns)
	return http.StatusOK, map[string]any{"runId": runID}
}

// azureADFRunBody returns a run without its server-only fields.
func azureADFRunBody(rec *azureARMResource) map[string]any {
	body := azureARMMerge(rec.Properties, nil)
	delete(body, azureADFActivityKey)
	return body
}

// azureADFWindow reads the required lastUpdatedAfter and lastUpdatedBefore
// of a run query.
func azureADFWindow(payload map[string]any) (after, before time.Time, msg string) {
	after, err1 := time.Parse(time.RFC3339Nano, azureARMString(payload["lastUpdatedAfter"]))
	before, err2 := time.Parse(time.RFC3339Nano, azureARMString(payload["lastUpdatedBefore"]))
	switch {
	case err1 != nil:
		return after, before, "lastUpdatedAfter must be an RFC 3339 time."
	case err2 != nil:
		return after, before, "lastUpdatedBefore must be an RFC 3339 time."
	case !after.Before(before):
		return after, before, "lastUpdatedAfter must be before lastUpdatedBefore."
	}
	return after, before, ""
}

// azureADFRunMatches applies a query's filters (operand, operator, values)
// to a run. Filters on operands a run lacks don't match.
func azureADFRunMatches(run map[string]any, filters []any) (bool, string) {
	fields := map[string]string{"PipelineName": "pipelineName", "Status": "status", "RunId": "runId", "RunGroupId": "runGroupId"}
	for _, raw := range filters {
		filter := azureARMObject(raw)
		field, known := fields[azureARMString(filter["operand"])]
		if !known {
			return false, "filters operand must be one of PipelineName, Status, RunId or RunGroupId."
		}
		values, _ := filter["values"].([]any)
		in := slices.ContainsFunc(values, func(v any) bool { return azureARMString(v) == azureARMString(run[field]) })
		switch azureARMString(filter["operator"]) {
		case "Equals", "In":
			if !in {
				return false, ""
			}
		case "NotEquals", "NotIn":
			if in {
				return false, ""
			}
		default:
			return false, "filters operator must be Equals, NotEquals, In or NotIn."
		}
	}
	return true, ""
}

func azureADFQueryPipelineRuns(s *Server, factory *azureARMResource, payload map[string]any) (int, any) {
	after, before, msg := azureADFWindow(payload)
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	filters, _ := payload["filters"].([]any)
	s.providerStorageMu.Lock()
	runs := s.azureDataPlatChildrenLocked(factory.ID, azureADFRunType)
	s.providerStorageMu.Unlock()
	value := []any{}
	for _, rec := range runs {
		if rec.UpdatedAt.Before(after) || !rec.UpdatedAt.Before(before) {
			continue
		}
		ok, msg := azureADFRunMatches(rec.Properties, filters)
		if msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
		}
		if ok && len(value) < azureADFMaxQueryPage {
			value = append(value, azureADFRunBody(rec))
		}
	}
	return http.StatusOK, map[string]any{"value": value}
}

// azureADFQueryTriggerRuns reports no trigger runs: triggers never fire.
func azureADFQueryTriggerRuns(_ *Server, _ *azureARMResource, payload map[string]any) (int, any) {
	if _, _, msg := azureADFWindow(payload); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	return http.StatusOK, map[string]any{"value": []any{}}
}

// azureADFQueryTriggers lists the factory's triggers, optionally those
// whose parentTrigger is parentTriggerName.
func azureADFQueryTriggers(s *Server, factory *azureARMResource, payload map[string]any) (int, any) {
	parent := azureARMString(payload["parentTriggerName"])
	s.providerStorageMu.Lock()
	triggers := s.azureDataPlatChildrenLocked(factory.ID, azureADFFactoryType+"/triggers")
	value := []any{}
	for _, rec := range triggers {
		if parent == "" || azureARMString(rec.Properties["parentTrigger"]) == parent {
			value = append(value, rec.fixture())
		}
	}
	s.providerStorageMu.Unlock()
	return http.StatusOK, map[string]any{"value": value}
}

// serveAzureADFRun serves one run: GET it, or POST cancel or
// queryActivityruns.
func (s *Server) serveAzureADFRun(w http.ResponseWriter, r *http.Request, factoryID, runID string, rest []string) {
	s.providerStorageMu.Lock()
	run := s.azureDataPlatStoredLocked(factoryID+"/pipelineruns/"+runID, azureADFRunType)
	s.providerStorageMu.Unlock()
	verb := ""
	if len(rest) == 1 {
		verb = strings.ToLower(rest[0])
	}
	switch {
	case len(rest) == 0 && r.Method != http.MethodGet, len(rest) == 1 && r.Method != http.MethodPost:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	case verb != "" && verb != "cancel" && verb != "queryactivityruns":
		azureARMError(w, http.StatusNotFound, "ResourceNotFound", "The route was not found.")
	case run == nil:
		azureARMNotFound(w, "PipelineRunNotFound", factoryID+"/pipelineruns/"+runID, "")
	case verb == "":
		respondJSON(w, http.StatusOK, azureADFRunBody(run))
	case verb == "cancel":
		// ponytail: runs succeed at creation, so cancelling changes nothing.
		w.WriteHeader(http.StatusOK)
	default:
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		if _, _, msg := azureADFWindow(payload); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return
		}
		at := azureARMString(run.Properties["runStart"])
		value := []any{}
		activities, _ := run.Properties[azureADFActivityKey].([]any)
		for _, raw := range activities {
			activity := azureARMObject(raw)
			value = append(value, map[string]any{
				"activityName": activity["name"], "activityType": activity["type"],
				"pipelineName": run.Properties["pipelineName"], "pipelineRunId": runID,
				"activityRunId": azureDataPlatGUID(runID + "/" + azureARMString(activity["name"])),
				"status":        "Succeeded", "activityRunStart": at, "activityRunEnd": at, "durationInMs": 0,
				"input": map[string]any{}, "output": map[string]any{}, "error": map[string]any{},
			})
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": value})
	}
}

// azureADFRuntimeActions are the integration runtime POST operations.
func azureADFRuntimeActions() map[string]azureARMAction {
	status := func(rec *azureARMResource) map[string]any {
		return map[string]any{"name": rec.Name, "properties": map[string]any{
			"type": rec.Properties["type"], "state": rec.Properties["state"], "dataFactoryName": azureDataPlatLastSegment(azureADFFactoryOf(rec.ID)),
		}}
	}
	ok := func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, nil }
	authKeys := func(rec *azureARMResource) map[string]any {
		return map[string]any{
			"authKey1": "IR@" + azureDataPlatGUID(rec.ID+"/authKey1"),
			"authKey2": "IR@" + azureDataPlatGUID(rec.ID+"/authKey2"),
		}
	}
	return map[string]azureARMAction{
		"getstatus": func(rec *azureARMResource, _ map[string]any) (int, any) { return http.StatusOK, status(rec) },
		"getconnectioninfo": func(rec *azureARMResource, _ map[string]any) (int, any) {
			return http.StatusOK, map[string]any{
				"serviceToken": azureDataPlatGUID(rec.ID + "/serviceToken"), "identityCertThumbprint": strings.ToUpper(strings.ReplaceAll(azureDataPlatGUID(rec.ID+"/cert"), "-", "")),
				"hostServiceUri": "https://" + strings.ToLower(rec.Name) + ".frontend.clouddatahub.net", "version": "5.0", "publicKey": "", "isIdentityCertExprired": false,
			}
		},
		"listauthkeys": func(rec *azureARMResource, _ map[string]any) (int, any) {
			if rec.Properties["type"] != "SelfHosted" {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Only self-hosted integration runtimes have authentication keys.")
			}
			return http.StatusOK, authKeys(rec)
		},
		"regenerateauthkey": func(rec *azureARMResource, payload map[string]any) (int, any) {
			name := azureARMString(payload["keyName"])
			switch {
			case rec.Properties["type"] != "SelfHosted":
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Only self-hosted integration runtimes have authentication keys.")
			case name != "authKey1" && name != "authKey2":
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "keyName must be authKey1 or authKey2.")
			}
			// ponytail: keys are derived from the ID, so a regenerated key
			// equals the old one; store a generation counter if clients
			// check that the key changed.
			return http.StatusOK, map[string]any{name: authKeys(rec)[name]}
		},
		"getmonitoringdata": func(rec *azureARMResource, _ map[string]any) (int, any) {
			return http.StatusOK, map[string]any{"name": rec.Name, "nodes": []any{}}
		},
		"synccredentials": ok,
		"upgrade":         ok,
		"removelinks": func(_ *azureARMResource, payload map[string]any) (int, any) {
			if azureARMString(payload["linkedFactoryName"]) == "" {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "linkedFactoryName is required.")
			}
			return http.StatusOK, nil
		},
		"linkedintegrationruntime": func(rec *azureARMResource, payload map[string]any) (int, any) {
			if azureARMString(payload["name"]) == "" {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required.")
			}
			return http.StatusOK, status(rec)
		},
	}
}

// azureDataPlatLastSegment returns the last segment of a resource ID.
func azureDataPlatLastSegment(id string) string { return id[strings.LastIndex(id, "/")+1:] }
