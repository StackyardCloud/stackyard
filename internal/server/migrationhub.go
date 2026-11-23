package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Migration Hub: awsJson1_1, X-Amz-Target AWSMigrationHub.*, signed mgh.
// The other Migration Hub services (Orchestrator, Strategy, Refactor Spaces)
// use their own targets and signing names.

const (
	mghTargetPrefix = "AWSMigrationHub."
	mghMaxBody      = 1 << 20
)

type mghHandler func(s *Server, in map[string]any) (map[string]any, *mghErr)

var mghHandlers map[string]mghHandler

func init() {
	mghHandlers = map[string]mghHandler{
		"CreateProgressUpdateStream": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return nil, s.migrationHub.createStream(mghStr(in, "ProgressUpdateStreamName"), mghDry(in))
		},
		"DeleteProgressUpdateStream": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return nil, s.migrationHub.deleteStream(mghStr(in, "ProgressUpdateStreamName"), mghDry(in))
		},
		"ListProgressUpdateStreams": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return s.migrationHub.listStreams(mghStr(in, "NextToken"), mghMax(in, mghDefaultPageSize))
		},
		"ImportMigrationTask": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return nil, s.migrationHub.importTask(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghDry(in), time.Now().UTC())
		},
		"NotifyMigrationTaskState": (*Server).mghNotifyMigrationTaskState,
		"PutResourceAttributes":    (*Server).mghPutResourceAttributes,
		"DescribeMigrationTask": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return s.migrationHub.describeTask(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"))
		},
		"ListMigrationTasks": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return s.migrationHub.listTasks(mghStr(in, "ResourceName"), mghStr(in, "NextToken"), mghMax(in, mghDefaultPageSize))
		},
		"ListMigrationTaskUpdates": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return s.migrationHub.listUpdates(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghStr(in, "NextToken"), mghMax(in, mghDefaultPageSize))
		},
		"AssociateCreatedArtifact": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			a := mghMap(in, "CreatedArtifact")
			return nil, s.migrationHub.associate(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghArtifacts,
				mghNamed{name: mghStr(a, "Name"), description: mghStr(a, "Description")}, mghDry(in))
		},
		"DisassociateCreatedArtifact": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return nil, s.migrationHub.disassociate(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghArtifacts,
				mghStr(in, "CreatedArtifactName"), "Created artifact", mghDry(in))
		},
		"ListCreatedArtifacts": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return s.migrationHub.listAssociations(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghArtifacts,
				func(n mghNamed) map[string]any { return mghNameDesc("Name", n) }, "CreatedArtifactList", mghStr(in, "NextToken"), mghMax(in, 10))
		},
		"AssociateDiscoveredResource": (*Server).mghAssociateDiscoveredResource,
		"DisassociateDiscoveredResource": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return nil, s.migrationHub.disassociate(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghDiscovered,
				mghStr(in, "ConfigurationId"), "Discovered resource", mghDry(in))
		},
		"ListDiscoveredResources": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return s.migrationHub.listAssociations(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghDiscovered,
				func(n mghNamed) map[string]any { return mghNameDesc("ConfigurationId", n) }, "DiscoveredResourceList", mghStr(in, "NextToken"), mghMax(in, 10))
		},
		"AssociateSourceResource": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			r := mghMap(in, "SourceResource")
			return nil, s.migrationHub.associate(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghSources,
				mghNamed{name: mghStr(r, "Name"), description: mghStr(r, "Description"), statusDetail: mghStr(r, "StatusDetail")}, mghDry(in))
		},
		"DisassociateSourceResource": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return nil, s.migrationHub.disassociate(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghSources,
				mghStr(in, "SourceResourceName"), "Source resource", mghDry(in))
		},
		"ListSourceResources": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			return s.migrationHub.listAssociations(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghSources,
				func(n mghNamed) map[string]any {
					out := mghNameDesc("Name", n)
					if n.statusDetail != "" {
						out["StatusDetail"] = n.statusDetail
					}
					return out
				}, "SourceResourceList", mghStr(in, "NextToken"), mghMax(in, 10))
		},
		"NotifyApplicationState": (*Server).mghNotifyApplicationState,
		"DescribeApplicationState": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			id := mghStr(in, "ApplicationId")
			if !s.discovery.hasApplication(id) {
				return nil, mghError("ResourceNotFoundException", "Application %s does not exist in Application Discovery Service.", id)
			}
			return s.migrationHub.describeApplication(id)
		},
		"ListApplicationStates": func(s *Server, in map[string]any) (map[string]any, *mghErr) {
			var ids []string
			if list, ok := in["ApplicationIds"].([]any); ok {
				for _, v := range list {
					if id, ok := v.(string); ok {
						ids = append(ids, id)
					}
				}
			}
			return s.migrationHub.listApplications(ids, mghStr(in, "NextToken"), mghMax(in, mghDefaultPageSize))
		},
	}
}

func (s *Server) handleMigrationHubJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), mghTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > mghMaxBody {
		respondMigrationHubError(w, http.StatusBadRequest, "InvalidInputException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "mgh"); !ok {
		respondMigrationHubError(w, status, code, msg)
		return true
	}
	h := mghHandlers[op]
	if _, known := migrationHubOperationByName[op]; !known || h == nil {
		respondMigrationHubError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return true
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondMigrationHubError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
		return true
	}
	model := awsmodels.MigrationHubModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondMigrationHubError(w, http.StatusBadRequest, "InvalidInputException", strings.Join(problems, "; "))
		return true
	}
	// Conform drops members the input shape doesn't define.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	out, merr := h(s, conformed)
	if merr != nil {
		respondMigrationHubError(w, http.StatusBadRequest, merr.code, merr.msg)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	// out is built fresh under the store lock; nothing in it is shared.
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondMigrationHubError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "Message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (s *Server) mghNotifyMigrationTaskState(in map[string]any) (map[string]any, *mghErr) {
	at, err := mghTime(in["UpdateDateTime"])
	if err != nil {
		return nil, err
	}
	task := mghMap(in, "Task")
	u := mghUpdate{at: at, status: mghStr(task, "Status"), statusDetail: mghStr(task, "StatusDetail")}
	if p, ok := task["ProgressPercent"].(float64); ok {
		n := int(p) // the model bounds it to 0-100
		u.percent = &n
	}
	return nil, s.migrationHub.notifyTask(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), u, mghDry(in))
}

func (s *Server) mghPutResourceAttributes(in map[string]any) (map[string]any, *mghErr) {
	list, _ := in["ResourceAttributeList"].([]any)
	attrs := make([]map[string]any, 0, len(list))
	for _, item := range list {
		a, _ := item.(map[string]any)
		typ, value := mghStr(a, "Type"), mghStr(a, "Value")
		if err := mghCheckAttribute(typ, value); err != nil {
			return nil, err
		}
		attrs = append(attrs, map[string]any{"Type": typ, "Value": value})
	}
	return nil, s.migrationHub.putAttributes(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), attrs, mghDry(in))
}

func (s *Server) mghAssociateDiscoveredResource(in map[string]any) (map[string]any, *mghErr) {
	res := mghMap(in, "DiscoveredResource")
	id := mghStr(res, "ConfigurationId")
	// Checked before the Migration Hub lock: the two stores never nest locks.
	if !s.discovery.hasConfiguration(id) {
		return nil, mghError("ResourceNotFoundException", "Configuration %s does not exist in Application Discovery Service.", id)
	}
	return nil, s.migrationHub.associate(mghStr(in, "ProgressUpdateStream"), mghStr(in, "MigrationTaskName"), mghDiscovered,
		mghNamed{name: id, description: mghStr(res, "Description")}, mghDry(in))
}

func (s *Server) mghNotifyApplicationState(in map[string]any) (map[string]any, *mghErr) {
	id := mghStr(in, "ApplicationId")
	if !s.discovery.hasApplication(id) {
		return nil, mghError("ResourceNotFoundException", "Application %s does not exist in Application Discovery Service.", id)
	}
	at := time.Now().UTC()
	if v, ok := in["UpdateDateTime"]; ok {
		t, err := mghTime(v)
		if err != nil {
			return nil, err
		}
		at = t
	}
	return nil, s.migrationHub.notifyApplication(id, mghStr(in, "Status"), at, mghDry(in))
}

// hasConfiguration reports whether Application Discovery Service knows id.
func (s *discoveryStore) hasConfiguration(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.configurations[id]
	return ok
}

// hasApplication reports whether id is an Application Discovery Service application.
func (s *discoveryStore) hasApplication(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.applications[id]
	return ok
}

// ---- request helpers ----

func mghStr(m map[string]any, k string) string { v, _ := m[k].(string); return v }

func mghMap(m map[string]any, k string) map[string]any { v, _ := m[k].(map[string]any); return v }

func mghDry(m map[string]any) bool { v, _ := m["DryRun"].(bool); return v }

// mghMax returns MaxResults (already range-checked by the model) or def.
func mghMax(m map[string]any, def int) int {
	if v, ok := m["MaxResults"].(float64); ok && v >= 1 {
		return min(int(v), def)
	}
	return def
}

func mghNameDesc(key string, n mghNamed) map[string]any {
	out := map[string]any{key: n.name}
	if n.description != "" {
		out["Description"] = n.description
	}
	return out
}

// mghTime parses a JSON 1.1 timestamp: epoch seconds, or an RFC 3339 string.
func mghTime(v any) (time.Time, *mghErr) {
	switch t := v.(type) {
	case float64:
		if t < 0 || t > 1e11 { // beyond year ~5000: not a real timestamp
			break
		}
		sec := int64(t)
		return time.Unix(sec, int64((t-float64(sec))*1e9)).UTC(), nil
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, mghError("InvalidInputException", "UpdateDateTime must be a timestamp.")
}
