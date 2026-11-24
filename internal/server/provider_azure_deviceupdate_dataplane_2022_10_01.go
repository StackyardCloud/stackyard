package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const azureDeviceUpdateDataPlane20221001APIVersion = "2022-10-01"
const azureDeviceUpdateDataPlane20221001Prefix = "/azure/deviceupdate-dataplane/2022-10-01/"

// Device Update for IoT Hub data plane, per instance under
// {prefix}deviceUpdate/{instanceId}/. Nothing is downloaded or deployed:
// importing reads the update ID from the import manifest's file name
// ({provider}.{name}.{version}.importmanifest.json), importing devices adds
// two sample devices of one device class to the "default" group (no IoT Hub
// exists to read them from), and deployments complete at once. Device
// management code is in azure_deviceupdate_dataplane_2022_10_01_management.go.
func (s *Server) handleAzureDeviceUpdateDataPlane20221001Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(strings.ToLower(path), azureDeviceUpdateDataPlane20221001Prefix) {
		return false
	}
	if r.URL.Query().Get("api-version") != azureDeviceUpdateDataPlane20221001APIVersion {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "api-version must be 2022-10-01.")
		return true
	}
	seg := splitPathSegments(strings.Trim(path[len(azureDeviceUpdateDataPlane20221001Prefix):], "/"))
	if len(seg) < 3 || !strings.EqualFold(seg[0], "deviceUpdate") || !aduDPIDPattern.MatchString(seg[1]) {
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
		return true
	}
	base := "/dataplane/deviceupdate/" + strings.ToLower(seg[1])
	head, action, _ := strings.Cut(seg[2], ":")
	switch strings.ToLower(head) {
	case "updates":
		s.aduDPUpdates(w, r, base, seg[3:], action)
	case "management":
		s.aduDPManagement(w, r, base, seg[3:])
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
	return true
}

var aduDPIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

const (
	aduDPMaxImport = 10
	aduDPMaxFiles  = 10
)

func (s *Server) aduDPUpdates(w http.ResponseWriter, r *http.Request, base string, seg []string, action string) {
	if strings.EqualFold(action, "import") {
		s.aduDPImport(w, r, base)
		return
	}
	if len(seg) == 0 {
		if dcdevGetOnly(w, r) {
			dcdevRespondList(w, dcdevViews(s.azureDPList(base+"/updates", "adu/update"), dcdevRecordView))
		}
		return
	}
	switch strings.ToLower(seg[0]) {
	case "operations":
		s.dcdevChildren(w, r, base+"/updates/operations", "adu/operation", seg[1:])
	case "providers":
		s.aduDPProviders(w, r, base, seg[1:])
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

// aduDPProviders serves providers/{p}/names/{n}/versions/{v}[/files[/{id}]].
func (s *Server) aduDPProviders(w http.ResponseWriter, r *http.Request, base string, seg []string) {
	updates := s.azureDPList(base+"/updates", "adu/update")
	// Each level lists the distinct values of one update ID part among the
	// updates that match the levels above it.
	parts := []string{"provider", "name", "version"}
	match := func(u *azureARMResource, depth int) bool {
		id := azureARMObject(u.Properties["updateId"])
		for i := 0; i < depth; i++ {
			if !strings.EqualFold(azureARMString(id[parts[i]]), seg[i*2]) {
				return false
			}
		}
		return true
	}
	// seg: [p] [names] [n] [versions] [v] [files] [id]
	keywords := []string{"", "names", "", "versions", "", "files"}
	for i := 1; i < len(seg) && i < len(keywords); i += 2 {
		if !strings.EqualFold(seg[i], keywords[i]) {
			azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
			return
		}
	}
	if len(seg)%2 == 0 && len(seg) <= 4 {
		if !dcdevGetOnly(w, r) {
			return
		}
		depth := len(seg) / 2
		seen := map[string]bool{}
		var out []any
		for _, u := range updates {
			if match(u, depth) {
				v := azureARMString(azureARMObject(u.Properties["updateId"])[parts[depth]])
				if !seen[strings.ToLower(v)] {
					seen[strings.ToLower(v)] = true
					out = append(out, v)
				}
			}
		}
		if depth > 0 && len(out) == 0 {
			azureARMError(w, http.StatusNotFound, "NotFound", "update not found")
			return
		}
		dcdevRespondList(w, out)
		return
	}
	if len(seg) < 5 {
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
		return
	}
	var update *azureARMResource
	for _, u := range updates {
		if match(u, 3) {
			update = u
		}
	}
	if update == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "update not found")
		return
	}
	switch len(seg) {
	case 5:
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, dcdevRecordView(update))
		case http.MethodDelete:
			s.azureDPDelete(update.ID)
			if s.aduDPOperation(w, base+"/updates/operations", update.Properties["updateId"]) {
				w.WriteHeader(http.StatusAccepted)
			}
		default:
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		}
	case 6, 7:
		if !dcdevGetOnly(w, r) {
			return
		}
		files, _ := update.Properties["_files"].([]any)
		if len(seg) == 6 {
			var ids []any
			for _, f := range files {
				ids = append(ids, azureARMObject(f)["fileId"])
			}
			dcdevRespondList(w, ids)
			return
		}
		for _, f := range files {
			if strings.EqualFold(azureARMString(azureARMObject(f)["fileId"]), seg[6]) {
				respondJSON(w, http.StatusOK, f)
				return
			}
		}
		azureARMError(w, http.StatusNotFound, "NotFound", "file not found")
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

var aduDPManifestPattern = regexp.MustCompile(`(?i)([A-Za-z0-9-]+)\.([A-Za-z0-9-]+)\.([0-9][0-9.]*[0-9])\.importmanifest\.json$`)

func (s *Server) aduDPImport(w http.ResponseWriter, r *http.Request, base string) {
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	// The body is a JSON array of import items ({"value": [...]} works too).
	body, err := readBodyBytes(r)
	var items []any
	if err == nil && json.Unmarshal(body, &items) != nil {
		var wrapped struct {
			Value []any `json:"value"`
		}
		_ = json.Unmarshal(body, &wrapped)
		items = wrapped.Value
	}
	if err != nil || len(items) == 0 || len(items) > aduDPMaxImport {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "the import must list 1 to 10 updates.")
		return
	}
	var last any
	for _, item := range items {
		manifest := azureARMObject(azureARMObject(item)["importManifest"])
		url := azureARMString(manifest["url"])
		m := aduDPManifestPattern.FindStringSubmatch(url)
		if m == nil {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "importManifest.url must end in {provider}.{name}.{version}.importmanifest.json.")
			return
		}
		files, _ := azureARMObject(item)["files"].([]any)
		if len(files) > aduDPMaxFiles {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "an update may list at most 10 files.")
			return
		}
		var stored []any
		for i, f := range files {
			name := azureARMString(azureARMObject(f)["filename"])
			if name == "" || azureARMString(azureARMObject(f)["url"]) == "" {
				azureARMError(w, http.StatusBadRequest, "InvalidRequest", "files[].filename and files[].url are required.")
				return
			}
			stored = append(stored, map[string]any{"fileId": "f" + string(rune('0'+i)), "fileName": name, "sizeInBytes": 0, "hashes": map[string]any{}})
		}
		updateID := map[string]any{"provider": m[1], "name": m[2], "version": m[3]}
		id := base + "/updates/" + strings.ToLower(m[1]+"/"+m[2]+"/"+m[3])
		now := time.Now().UTC().Format(time.RFC3339)
		rec := azureDPRecord(id, "adu/update", m[3], map[string]any{
			"updateId": updateID, "description": azureARMObject(item)["friendlyName"], "isDeployable": true, "updateType": "microsoft/swupdate:2",
			"installedCriteria": m[3], "compatibility": []any{aduDPCompat}, "manifestVersion": "5.0", "importedDateTime": now, "createdDateTime": now,
			"etag": `"` + uuid.NewString() + `"`, "_files": stored,
		}, s.azureDPGet(id))
		if _, ok := s.azureDPPut(rec); !ok {
			azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
			return
		}
		last = updateID
	}
	if s.aduDPOperation(w, base+"/updates/operations", last) {
		w.WriteHeader(http.StatusAccepted)
	}
}

// aduDPOperation records a Succeeded operation under prefix and points
// Operation-Location at it.
func (s *Server) aduDPOperation(w http.ResponseWriter, prefix string, update any) bool {
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	op := azureDPRecord(prefix+"/"+id, "adu/operation", id, map[string]any{
		"operationId": id, "status": "Succeeded", "update": map[string]any{"updateId": update}, "createdDateTime": now, "lastActionDateTime": now,
		"etag": `"` + id + `"`,
	}, nil)
	if _, ok := s.azureDPPut(op); !ok {
		azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
		return false
	}
	rel := strings.TrimPrefix(prefix, "/dataplane/deviceupdate/")
	instance, rest, _ := strings.Cut(rel, "/")
	w.Header().Set("Operation-Location", azureDeviceUpdateDataPlane20221001Prefix+"deviceUpdate/"+instance+"/"+rest+"/"+id+"?api-version="+azureDeviceUpdateDataPlane20221001APIVersion)
	return true
}

var aduDPCompat = map[string]any{"manufacturer": "Contoso", "model": "Sensor"}

// aduDPSorted returns the records' views ordered by name.
func aduDPSorted(recs []*azureARMResource) []any {
	sort.Slice(recs, func(i, j int) bool { return recs[i].Name < recs[j].Name })
	return dcdevViews(recs, dcdevRecordView)
}
