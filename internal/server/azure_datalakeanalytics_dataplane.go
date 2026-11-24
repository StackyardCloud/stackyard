package server

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// The Data Lake Analytics data plane: jobs, pipelines, recurrences and the
// U-SQL catalog, served at /azure/{Jobs,BuildJob,pipelines,recurrences,
// catalog}. Real clients reach {account}.azuredatalakeanalytics.net, so the
// account is the Host's first label there and "default" otherwise. Jobs
// finish at once with result Succeeded; no U-SQL runs, so the catalog holds
// only the master database, its dbo schema, and the credentials and
// secrets clients create.

const (
	azureDLAJobType     = azureDLANamespace + "/jobs"
	azureDLASecretType  = azureDLANamespace + "/catalog/secrets"
	azureDLACredType    = azureDLANamespace + "/catalog/credentials"
	azureDLAMaxJobs     = 1000 // stored jobs across every account
	azureDLAMaxSecrets  = 500  // stored catalog credentials and secrets across every account
	azureDLAScriptLimit = 1 << 20
)

var azureDLAJobTypes = []string{"USql", "Hive", "Scope"}

// azureDLAAccount names the data plane account a request is for.
func azureDLAAccount(r *http.Request) string {
	host := strings.ToLower(r.Host)
	if label, rest, ok := strings.Cut(host, "."); ok && strings.HasPrefix(rest, "azuredatalakeanalytics.") && azureDataPlatAccountNameError(label) == "" {
		return label
	}
	return "default"
}

func (s *Server) serveAzureDLADataPlane(w http.ResponseWriter, r *http.Request, path string) bool {
	rest := strings.TrimPrefix(path, "/azure/")
	seg := splitPathSegments(rest)
	if len(seg) == 0 {
		return false
	}
	switch seg[0] {
	case "Jobs", "BuildJob", "pipelines", "recurrences", "catalog":
	default:
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	account := azureDLAAccount(r)
	switch seg[0] {
	case "Jobs":
		s.serveAzureDLAJobs(w, r, account, seg[1:])
	case "BuildJob":
		if r.Method != http.MethodPost || len(seg) != 1 {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return true
		}
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return true
		}
		if msg := azureDLAJobValidate(payload); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return true
		}
		job := azureDLAJobInfo(payload, "", account, time.Now().UTC())
		job["state"], job["result"] = "Ended", "Succeeded"
		respondJSON(w, http.StatusOK, job)
	case "pipelines", "recurrences":
		// No job declares pipeline or recurrence metadata, so none exist.
		switch {
		case r.Method != http.MethodGet:
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		case len(seg) == 1:
			respondJSON(w, http.StatusOK, map[string]any{"value": []any{}})
		default:
			azureARMError(w, http.StatusNotFound, "ResourceNotFound", "The "+strings.TrimSuffix(seg[0], "s")+" '"+seg[1]+"' was not found.")
		}
	case "catalog":
		s.serveAzureDLACatalog(w, r, account, seg[1:])
	}
	return true
}

func azureDLAJobValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	script := azureARMString(props["script"])
	switch {
	case strings.TrimSpace(azureARMString(payload["name"])) == "":
		return "name is required."
	case !slices.Contains(azureDLAJobTypes, azureARMString(payload["type"])):
		return "type must be one of " + strings.Join(azureDLAJobTypes, ", ") + "."
	case strings.TrimSpace(script) == "":
		return "properties.script is required."
	case len(script) > azureDLAScriptLimit:
		return "properties.script must be at most 1 MiB."
	}
	for _, key := range []string{"degreeOfParallelism", "priority"} {
		if raw, present := payload[key]; present {
			if n, ok := raw.(float64); !ok || n < 1 || n != float64(int32(n)) {
				return key + " must be a positive integer."
			}
		}
	}
	return ""
}

// azureDLAJobInfo builds the JobInformation for a submitted job.
func azureDLAJobInfo(payload map[string]any, jobID, account string, now time.Time) map[string]any {
	props := azureARMObject(payload["properties"])
	jobType := azureARMString(payload["type"])
	runtime := azureARMString(props["runtimeVersion"])
	if runtime == "" {
		runtime = "default"
	}
	parallelism, priority := payload["degreeOfParallelism"], payload["priority"]
	if parallelism == nil {
		parallelism = 1
	}
	if priority == nil {
		priority = 1000
	}
	at := now.Format(time.RFC3339)
	job := map[string]any{
		"name": azureARMString(payload["name"]), "type": jobType, "submitter": "stackyard@" + account,
		"degreeOfParallelism": parallelism, "priority": priority,
		"submitTime": at, "startTime": at, "endTime": at,
		"state": "Ended", "result": "Succeeded", "errorMessage": []any{},
		"stateAuditRecords": []any{
			map[string]any{"newState": "New", "timeStamp": at, "requestedByUser": "stackyard"},
			map[string]any{"newState": "Ended", "timeStamp": at, "requestedByUser": "stackyard", "details": "result:Succeeded"},
		},
		"properties": map[string]any{"type": jobType, "script": azureARMString(props["script"]), "runtimeVersion": runtime},
	}
	if jobID != "" {
		job["jobId"] = jobID
	}
	return job
}

func (s *Server) serveAzureDLAJobs(w http.ResponseWriter, r *http.Request, account string, seg []string) {
	switch {
	case len(seg) == 0 && r.Method == http.MethodGet:
		s.providerStorageMu.Lock()
		var jobs []any
		for _, rec := range s.azureDLAJobsLocked(account) {
			jobs = append(jobs, rec.Properties)
		}
		s.providerStorageMu.Unlock()
		if jobs == nil {
			jobs = []any{}
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": jobs})
		return
	case len(seg) == 0:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	jobID := strings.ToLower(seg[0])
	if _, err := uuid.Parse(jobID); err != nil {
		azureARMError(w, http.StatusBadRequest, "InvalidJobId", "The job identity must be a GUID.")
		return
	}
	id := azureDLAJobID(account, jobID)
	switch {
	case len(seg) == 1 && r.Method == http.MethodPut:
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		if msg := azureDLAJobValidate(payload); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return
		}
		now := time.Now().UTC()
		job := azureDLAJobInfo(payload, jobID, account, now)
		s.providerStorageMu.Lock()
		if s.azureARMResources[azureARMKey(id)] != nil {
			s.providerStorageMu.Unlock()
			azureARMError(w, http.StatusConflict, "JobAlreadyExists", "A job with identity '"+jobID+"' already exists.")
			return
		}
		s.azureDLAEvictLocked(azureDLAJobType, azureDLAMaxJobs)
		s.azureARMResources[azureARMKey(id)] = &azureARMResource{ID: id, Name: jobID, Type: azureDLAJobType, Properties: job, CreatedAt: now, UpdatedAt: now}
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, job)
		return
	}
	s.providerStorageMu.Lock()
	rec := s.azureDataPlatStoredLocked(id, azureDLAJobType)
	s.providerStorageMu.Unlock()
	if rec == nil {
		azureARMError(w, http.StatusNotFound, "JobNotFound", "The job '"+jobID+"' was not found.")
		return
	}
	verb := ""
	if len(seg) == 2 {
		verb = seg[1]
	}
	switch {
	case len(seg) == 1 && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, rec.Properties)
	case verb == "CancelJob" && r.Method == http.MethodPost:
		// ponytail: jobs end at submission, so there is nothing to cancel.
		respondJSON(w, http.StatusOK, map[string]any{})
	case verb == "GetStatistics" && r.Method == http.MethodGet:
		end := azureARMString(rec.Properties["endTime"])
		respondJSON(w, http.StatusOK, map[string]any{"lastUpdateTimeUtc": end, "finalizingTimeUtc": end, "stages": []any{}})
	case verb == "GetDebugDataPath" && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"jobId": jobID, "command": "", "paths": []any{}})
	case len(seg) > 2 || verb != "" && verb != "CancelJob" && verb != "GetStatistics" && verb != "GetDebugDataPath":
		azureARMError(w, http.StatusNotFound, "ResourceNotFound", "The route was not found.")
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func azureDLAJobID(account, jobID string) string {
	return "/dataplane/" + azureDLANamespace + "/" + account + "/jobs/" + jobID
}

// azureDLAJobsLocked returns the account's jobs, oldest first. Caller holds
// providerStorageMu.
func (s *Server) azureDLAJobsLocked(account string) []*azureARMResource {
	prefix := azureARMKey(azureDLAJobID(account, ""))
	var out []*azureARMResource
	for key, rec := range s.azureARMResources {
		if strings.HasPrefix(key, prefix) && rec.Type == azureDLAJobType {
			out = append(out, rec)
		}
	}
	slices.SortFunc(out, func(a, b *azureARMResource) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out
}

// azureDLAEvictLocked drops the oldest records of fullType so one more fits
// under limit. Data plane accounts come from the Host header, so the limit
// spans all of them. Caller holds providerStorageMu.
func (s *Server) azureDLAEvictLocked(fullType string, limit int) {
	var all []*azureARMResource
	for _, rec := range s.azureARMResources {
		if rec.Type == fullType {
			all = append(all, rec)
		}
	}
	if len(all) < limit {
		return
	}
	slices.SortFunc(all, func(a, b *azureARMResource) int { return a.CreatedAt.Compare(b.CreatedAt) })
	for _, old := range all[:len(all)-limit+1] {
		delete(s.azureARMResources, azureARMKey(old.ID))
	}
}

// serveAzureDLACatalog serves /catalog/usql/...: the master database and
// its dbo schema, credentials and secrets, and ACLs. Every other catalog
// item type lists empty and is never found, since no U-SQL runs.
func (s *Server) serveAzureDLACatalog(w http.ResponseWriter, r *http.Request, account string, seg []string) {
	if len(seg) == 0 || seg[0] != "usql" {
		azureARMError(w, http.StatusNotFound, "ResourceNotFound", "The route was not found.")
		return
	}
	seg = seg[1:]
	switch {
	case len(seg) == 1 && seg[0] == "acl":
		s.serveAzureDLAACL(w, r)
		return
	case len(seg) == 0 || seg[0] != "databases":
		s.serveAzureDLAEmptyCatalog(w, r, seg)
		return
	case len(seg) == 1:
		s.serveAzureDLAEmptyCatalog(w, r, nil)
		return
	}
	db := seg[1]
	if !strings.EqualFold(db, "master") {
		azureARMError(w, http.StatusNotFound, "DatabaseNotFound", "The database '"+db+"' was not found.")
		return
	}
	switch {
	case len(seg) == 2:
		if r.Method != http.MethodGet {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return
		}
		respondJSON(w, http.StatusOK, azureDLAMasterDatabase())
	case seg[2] == "acl" && len(seg) == 3:
		s.serveAzureDLAACL(w, r)
	case seg[2] == "credentials" || seg[2] == "secrets":
		s.serveAzureDLACatalogSecret(w, r, account, seg[2], seg[3:])
	case seg[2] == "schemas" && len(seg) == 3 && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"value": []any{azureDLADboSchema()}})
	case seg[2] == "schemas" && len(seg) == 4 && r.Method == http.MethodGet && strings.EqualFold(seg[3], "dbo"):
		respondJSON(w, http.StatusOK, azureDLADboSchema())
	default:
		s.serveAzureDLAEmptyCatalog(w, r, seg[2:])
	}
}

// serveAzureDLAEmptyCatalog lists an empty collection (an odd number of
// segments after the database, or the database list) or reports an item
// missing. The database list holds master.
func (s *Server) serveAzureDLAEmptyCatalog(w http.ResponseWriter, r *http.Request, seg []string) {
	switch {
	case r.Method != http.MethodGet:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	case seg == nil:
		respondJSON(w, http.StatusOK, map[string]any{"value": []any{azureDLAMasterDatabase()}})
	case len(seg)%2 == 1:
		respondJSON(w, http.StatusOK, map[string]any{"value": []any{}})
	default:
		azureARMError(w, http.StatusNotFound, "CatalogItemNotFound", "The catalog item '"+strings.Join(seg, "/")+"' was not found.")
	}
}

func azureDLAMasterDatabase() map[string]any {
	return map[string]any{"computeAccountName": "", "databaseName": "master", "version": azureDataPlatGUID("master")}
}

func azureDLADboSchema() map[string]any {
	return map[string]any{"databaseName": "master", "schemaName": "dbo", "version": azureDataPlatGUID("master/dbo")}
}

// serveAzureDLAACL lists no ACL entries and accepts grant and revoke
// requests: GET, or POST with op GrantAce or RevokeAce.
func (s *Server) serveAzureDLAACL(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"value": []any{}})
	case http.MethodPost:
		op := r.URL.Query().Get("op")
		if op != "GrantAce" && op != "RevokeAce" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "op must be GrantAce or RevokeAce.")
			return
		}
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		if !slices.Contains([]string{"User", "Group", "Other", "UserObj", "GroupObj"}, azureARMString(payload["aceType"])) {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "aceType must be User, Group, Other, UserObj or GroupObj.")
			return
		}
		if _, err := uuid.Parse(azureARMString(payload["principalId"])); err != nil {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "principalId must be a GUID.")
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

// serveAzureDLACatalogSecret serves master's credentials and the
// deprecated secrets, which share a shape: PUT creates, PATCH updates,
// GET reads or lists, DELETE removes (on the collection, every secret).
// Passwords are accepted but never returned.
func (s *Server) serveAzureDLACatalogSecret(w http.ResponseWriter, r *http.Request, account, kind string, seg []string) {
	fullType := azureDLACredType
	if kind == "secrets" {
		fullType = azureDLASecretType
	}
	base := "/dataplane/" + azureDLANamespace + "/" + account + "/catalog/master/" + kind
	if len(seg) == 0 {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		var items []*azureARMResource
		for key, rec := range s.azureARMResources {
			if strings.HasPrefix(key, azureARMKey(base+"/")) && rec.Type == fullType {
				items = append(items, rec)
			}
		}
		slices.SortFunc(items, func(a, b *azureARMResource) int { return strings.Compare(a.Name, b.Name) })
		switch {
		case r.Method == http.MethodGet:
			value := make([]any, 0, len(items))
			for _, rec := range items {
				value = append(value, rec.Properties)
			}
			respondJSON(w, http.StatusOK, map[string]any{"value": value})
		case r.Method == http.MethodDelete && kind == "secrets":
			for _, rec := range items {
				delete(s.azureARMResources, azureARMKey(rec.ID))
			}
			w.WriteHeader(http.StatusOK)
		default:
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		}
		return
	}
	if len(seg) != 1 {
		azureARMError(w, http.StatusNotFound, "ResourceNotFound", "The route was not found.")
		return
	}
	name := seg[0]
	id := base + "/" + name
	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		password := azureARMString(payload["password"])
		if kind == "credentials" && r.Method == http.MethodPatch {
			password = azureARMString(payload["newPassword"])
		}
		uri := azureARMString(payload["uri"])
		if password == "" || uri == "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "password and uri are required.")
			return
		}
		item := map[string]any{"databaseName": "master", "name": name, "uri": uri, "creationTime": time.Now().UTC().Format(time.RFC3339)}
		if kind == "credentials" {
			item = map[string]any{"databaseName": "master", "credentialName": name, "userName": azureARMString(payload["userId"]), "version": azureDataPlatGUID(id)}
			if azureARMString(payload["userId"]) == "" {
				azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "userId is required.")
				return
			}
		}
		s.providerStorageMu.Lock()
		existing := s.azureDataPlatStoredLocked(id, fullType)
		switch {
		case r.Method == http.MethodPut && existing != nil:
			s.providerStorageMu.Unlock()
			azureARMError(w, http.StatusConflict, "Conflict", "The "+strings.TrimSuffix(kind, "s")+" '"+name+"' already exists.")
			return
		case r.Method == http.MethodPatch && existing == nil:
			s.providerStorageMu.Unlock()
			azureARMError(w, http.StatusNotFound, "ResourceNotFound", "The "+strings.TrimSuffix(kind, "s")+" '"+name+"' was not found.")
			return
		}
		now := time.Now().UTC()
		if existing == nil {
			s.azureDLAEvictLocked(fullType, azureDLAMaxSecrets)
		}
		s.azureARMResources[azureARMKey(id)] = &azureARMResource{ID: id, Name: name, Type: fullType, Properties: item, CreatedAt: now, UpdatedAt: now}
		s.providerStorageMu.Unlock()
		w.WriteHeader(http.StatusOK)
	case http.MethodGet, http.MethodDelete:
		s.providerStorageMu.Lock()
		rec := s.azureDataPlatStoredLocked(id, fullType)
		if rec != nil && r.Method == http.MethodDelete {
			delete(s.azureARMResources, azureARMKey(id))
		}
		s.providerStorageMu.Unlock()
		switch {
		case rec == nil:
			azureARMError(w, http.StatusNotFound, "ResourceNotFound", "The "+strings.TrimSuffix(kind, "s")+" '"+name+"' was not found.")
		case r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, rec.Properties)
		default:
			w.WriteHeader(http.StatusOK)
		}
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}
