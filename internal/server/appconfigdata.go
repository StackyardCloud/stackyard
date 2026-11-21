package server

import (
	"bytes"
	"cmp"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	appConfigDataTokenTTL          = 24 * time.Hour
	appConfigDataDefaultPollSecond = 60
	appConfigDataMinPollSeconds    = 15
	appConfigDataMaxPollSeconds    = 86400
	appConfigDataContentKey        = "__content" // raw hosted version body, set by the AppConfig router
)

// appConfigDataSession is the state a configuration token stands for. Each
// token is single-use: GetLatestConfiguration consumes it and issues the next.
type appConfigDataSession struct {
	AppID, EnvID, ProfileID string
	MinPollSeconds          int
	Delivered               string // deployment/version last returned, "" before the first poll
	IssuedAt                time.Time
	NotBefore               time.Time
}

type appConfigDataError struct {
	Message      string            `json:"Message"`
	Reason       string            `json:"Reason,omitempty"`
	Details      any               `json:"Details,omitempty"`
	ResourceType string            `json:"ResourceType,omitempty"`
	ReferencedBy map[string]string `json:"ReferencedBy,omitempty"`
}

func (s *Server) handleAppConfigDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	action := appConfigDataAction(r)
	if action == "" {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "appconfig"); !ok {
		respondAppConfigDataError(w, status, code, appConfigDataError{Message: msg})
		return true
	}
	if action == "StartConfigurationSession" {
		s.appConfigDataStartSession(w, r)
		return true
	}
	s.appConfigDataGetLatest(w, r)
	return true
}

// appConfigDataAction claims only the two AppConfig Data routes. It must run
// before the AppConfig router, which claims every request signed "appconfig".
func appConfigDataAction(r *http.Request) string {
	hint := strings.ToLower(sigV4ServiceHint(r))
	if hint != "appconfig" && !strings.Contains(strings.ToLower(r.Host), "appconfigdata") {
		return ""
	}
	path := rawRequestPath(r)
	for _, op := range appConfigDataOperations {
		if op.Method == r.Method && op.URI == path {
			return op.Name
		}
	}
	return ""
}

func (s *Server) appConfigDataStartSession(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondAppConfigDataBadRequest(w, "unable to read request body")
		return
	}
	var req struct {
		ApplicationIdentifier                string
		EnvironmentIdentifier                string
		ConfigurationProfileIdentifier       string
		RequiredMinimumPollIntervalInSeconds *json.Number
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if len(bytes.TrimSpace(body)) > 0 && dec.Decode(&req) != nil {
		respondAppConfigDataBadRequest(w, "invalid JSON body")
		return
	}
	for _, field := range [][2]string{
		{"ApplicationIdentifier", req.ApplicationIdentifier},
		{"EnvironmentIdentifier", req.EnvironmentIdentifier},
		{"ConfigurationProfileIdentifier", req.ConfigurationProfileIdentifier},
	} {
		if field[1] == "" || len(field[1]) > 128 {
			respondAppConfigDataBadRequest(w, field[0]+" must be between 1 and 128 characters")
			return
		}
	}
	poll := 0
	if req.RequiredMinimumPollIntervalInSeconds != nil {
		n, err := strconv.Atoi(req.RequiredMinimumPollIntervalInSeconds.String())
		if err != nil || n < appConfigDataMinPollSeconds || n > appConfigDataMaxPollSeconds {
			respondAppConfigDataBadRequest(w, "RequiredMinimumPollIntervalInSeconds must be an integer between 15 and 86400")
			return
		}
		poll = n
	}

	st := s.appconfig
	st.mu.Lock()
	defer st.mu.Unlock()
	appID, ok := appConfigDataResolve(st.apps, req.ApplicationIdentifier)
	if !ok {
		respondAppConfigDataNotFound(w, "Application", "ApplicationIdentifier", req.ApplicationIdentifier)
		return
	}
	envID, ok := appConfigDataResolve(st.envs[appID], req.EnvironmentIdentifier)
	if !ok {
		respondAppConfigDataNotFound(w, "Environment", "EnvironmentIdentifier", req.EnvironmentIdentifier)
		return
	}
	profileID, ok := appConfigDataResolve(st.profiles[appID], req.ConfigurationProfileIdentifier)
	if !ok {
		respondAppConfigDataNotFound(w, "ConfigurationProfile", "ConfigurationProfileIdentifier", req.ConfigurationProfileIdentifier)
		return
	}
	now := time.Now().UTC()
	// ponytail: expired tokens are pruned here, O(sessions) per start; fine for an emulator.
	for token, session := range st.sessions {
		if now.Sub(session.IssuedAt) > appConfigDataTokenTTL {
			delete(st.sessions, token)
		}
	}
	token := appConfigDataNewToken()
	st.sessions[token] = appConfigDataSession{AppID: appID, EnvID: envID, ProfileID: profileID, MinPollSeconds: poll, IssuedAt: now}
	respondAppConfigJSON(w, http.StatusCreated, map[string]string{"InitialConfigurationToken": token})
}

func (s *Server) appConfigDataGetLatest(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("configuration_token")
	if token == "" {
		respondAppConfigDataBadRequest(w, "ConfigurationToken is required")
		return
	}
	st := s.appconfig
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	session, ok := st.sessions[token]
	switch {
	case !ok:
		respondAppConfigDataTokenProblem(w, "Corrupted", "Token is invalid or has already been used")
		return
	case now.Sub(session.IssuedAt) > appConfigDataTokenTTL:
		delete(st.sessions, token)
		respondAppConfigDataTokenProblem(w, "Expired", "Token has expired")
		return
	case now.Before(session.NotBefore):
		respondAppConfigDataTokenProblem(w, "PollIntervalNotSatisfied", "Request too early; wait for the poll interval to pass")
		return
	}

	number, deployment, ok := st.latestDeploymentLocked(session)
	if !ok {
		respondAppConfigDataNotFound(w, "Deployment", "ConfigurationToken", token)
		return
	}
	versions := st.versions[session.AppID+"/"+session.ProfileID]
	index, err := strconv.Atoi(deployment.Version)
	if err != nil || index < 1 || index > len(versions) {
		respondAppConfigDataNotFound(w, "Configuration", "ConfigurationToken", token)
		return
	}
	version := versions[index-1]

	interval := cmp.Or(session.MinPollSeconds, appConfigDataDefaultPollSecond)
	next := session
	next.Delivered = strconv.Itoa(number) + "/" + deployment.Version
	next.IssuedAt = now
	next.NotBefore = now.Add(time.Duration(session.MinPollSeconds) * time.Second)
	nextToken := appConfigDataNewToken()
	delete(st.sessions, token)
	st.sessions[nextToken] = next

	w.Header().Set("Next-Poll-Configuration-Token", nextToken)
	w.Header().Set("Next-Poll-Interval-In-Seconds", strconv.Itoa(interval))
	if next.Delivered == session.Delivered {
		// The client already has this version: empty body, no Version-Label.
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", cmp.Or(version.ContentType, "application/octet-stream"))
	if version.Label != "" {
		w.Header().Set("Version-Label", version.Label)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(version.Content)
}

// latestDeploymentLocked returns the newest deployment of the session's
// profile to its environment that was not stopped (rolled back).
func (s *appConfigStore) latestDeploymentLocked(session appConfigDataSession) (int, appConfigDeployment, bool) {
	deployments := s.deployments[session.AppID+"/"+session.EnvID]
	for i := len(deployments) - 1; i >= 0; i-- {
		if d := deployments[i]; d.ProfileID == session.ProfileID && !d.RolledBack {
			return i + 1, d, true
		}
	}
	return 0, appConfigDeployment{}, false
}

// recordDataPlaneLocked keeps the registry AppConfig Data reads in step with
// the control plane, whose own responses stay fabricated.
func (s *appConfigStore) recordDataPlaneLocked(action string, payload map[string]any, path map[string]string, result map[string]any) {
	str := func(key string) string { v, _ := result[key].(string); return v }
	switch action {
	case "CreateApplication":
		s.apps[str("Id")] = str("Name")
	case "UpdateApplication":
		if _, ok := s.apps[str("Id")]; ok {
			s.apps[str("Id")] = str("Name")
		}
	case "DeleteApplication":
		delete(s.apps, path["ApplicationId"])
	case "CreateEnvironment", "UpdateEnvironment":
		appConfigDataRecordName(s.envs, str("ApplicationId"), str("Id"), str("Name"), action == "CreateEnvironment")
	case "DeleteEnvironment":
		delete(s.envs[path["ApplicationId"]], path["EnvironmentId"])
	case "CreateConfigurationProfile", "UpdateConfigurationProfile":
		appConfigDataRecordName(s.profiles, str("ApplicationId"), str("Id"), str("Name"), action == "CreateConfigurationProfile")
	case "DeleteConfigurationProfile":
		delete(s.profiles[path["ApplicationId"]], path["ConfigurationProfileId"])
	case "CreateHostedConfigurationVersion":
		key := str("ApplicationId") + "/" + str("ConfigurationProfileId")
		content, _ := payload[appConfigDataContentKey].([]byte)
		label := appConfigDefaultString(payload, "VersionLabel", "")
		s.versions[key] = append(s.versions[key], appConfigHostedVersion{Content: bytes.Clone(content), ContentType: str("ContentType"), Label: label})
		result["VersionNumber"] = int64(len(s.versions[key]))
		if label != "" {
			result["VersionLabel"] = label
		}
	case "StartDeployment":
		key := str("ApplicationId") + "/" + str("EnvironmentId")
		version := appConfigDefaultString(payload, "ConfigurationVersion", "1")
		s.deployments[key] = append(s.deployments[key], appConfigDeployment{ProfileID: str("ConfigurationProfileId"), Version: version})
		result["DeploymentNumber"] = int64(len(s.deployments[key]))
		result["ConfigurationVersion"] = version
	case "StopDeployment":
		deployments := s.deployments[str("ApplicationId")+"/"+str("EnvironmentId")]
		if n, _ := result["DeploymentNumber"].(int64); n >= 1 && int(n) <= len(deployments) {
			deployments[n-1].RolledBack = true
		}
	}
}

func appConfigDataRecordName(byApp map[string]map[string]string, appID, id, name string, create bool) {
	if byApp[appID] == nil {
		if !create {
			return
		}
		byApp[appID] = map[string]string{}
	}
	if _, ok := byApp[appID][id]; ok || create {
		byApp[appID][id] = name
	}
}

// appConfigDataResolve matches an identifier against IDs first, then names.
func appConfigDataResolve(names map[string]string, identifier string) (string, bool) {
	if _, ok := names[identifier]; ok {
		return identifier, true
	}
	ids := make([]string, 0, len(names))
	for id, name := range names {
		if name == identifier {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "", false
	}
	slices.Sort(ids)
	return ids[0], true
}

func appConfigDataNewToken() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func respondAppConfigDataError(w http.ResponseWriter, status int, code string, body appConfigDataError) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAppConfigJSON(w, status, body)
}

func respondAppConfigDataBadRequest(w http.ResponseWriter, msg string) {
	respondAppConfigDataError(w, http.StatusBadRequest, "BadRequestException", appConfigDataError{Message: msg})
}

func respondAppConfigDataTokenProblem(w http.ResponseWriter, problem, msg string) {
	respondAppConfigDataError(w, http.StatusBadRequest, "BadRequestException", appConfigDataError{
		Message: msg,
		Reason:  "InvalidParameters",
		Details: map[string]any{"InvalidParameters": map[string]any{"ConfigurationToken": map[string]string{"Problem": problem}}},
	})
}

func respondAppConfigDataNotFound(w http.ResponseWriter, resourceType, param, value string) {
	respondAppConfigDataError(w, http.StatusNotFound, "ResourceNotFoundException", appConfigDataError{
		Message:      resourceType + " not found",
		ResourceType: resourceType,
		ReferencedBy: map[string]string{param: value},
	})
}
