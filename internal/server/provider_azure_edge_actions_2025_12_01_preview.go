package server

import (
	"cmp"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"
)

const azureEdgeActions20251201PreviewAPIVersion = "2025-12-01-preview"

// Azure Front Door edge actions (Microsoft.Cdn/edgeActions
// 2025-12-01-preview) on the shared ARM handler: edge actions, their
// versions with deployed code and default-version swaps, and execution
// filters that route requests to a version. No code runs.

const (
	edgeaNamespace     = "Microsoft.Cdn"
	edgeaVersionType   = edgeaNamespace + "/edgeActions/versions"
	edgeaFilterType    = edgeaNamespace + "/edgeActions/executionFilters"
	edgeaMaxCodeBytes  = 4 << 20  // decoded bytes of one version's code
	edgeaMaxTotalCode  = 64 << 20 // encoded bytes of all deployed code per server
	edgeaMaxVersions   = 100      // versions per edge action
	edgeaMaxFilters    = 100      // execution filters per edge action
	edgeaMaxHeaderSize = 256
)

var (
	edgeaDeployTypes = []string{"zip", "file", "others"}
	edgeaTrueFalse   = []string{"True", "False"}
)

// edgeaCode holds each server's deployed version code, keyed by the
// lower-cased version ID.
// ponytail: package-level, keyed by server; entries go with their version.
var edgeaCode = struct {
	sync.Mutex
	m map[*Server]map[string][2]string // {name, base64 content}
}{m: map[*Server]map[string][2]string{}}

func init() {
	ns := edgeaNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "edgeActions",
		Validate: func(payload map[string]any) string {
			sku, ok := payload["sku"].(map[string]any)
			if !ok {
				return "sku is required."
			}
			return azureWorkloadsRequired(sku, "sku.", "name", "tier")
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"attachments": []any{}}
		},
		Deleted: func(s *Server, removed *azureARMResource) { edgeaForgetCode(s, removed.ID) },
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "edgeActions/versions",
		Validate: func(payload map[string]any) string {
			return azureWorkloadsEnums(azureARMObject(payload["properties"]), "properties.", []string{"deploymentType", "isDefaultVersion"},
				map[string][]string{"deploymentType": edgeaDeployTypes, "isDefaultVersion": edgeaTrueFalse})
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return payload, nil, cmp.Or(
				azureLabEnum(azureARMObject(payload["properties"]), "properties.", "deploymentType", edgeaDeployTypes),
				azureLabEnum(azureARMObject(payload["properties"]), "properties.", "isDefaultVersion", edgeaTrueFalse),
			)
		},
		StateProperties: map[string]any{"validationStatus": "Pending", "lastPackageUpdateTime": nil},
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			return esanSiblingLimit(s, existing, rec, "versions", edgeaMaxVersions)
		},
		Reconcile: func(s *Server, _, rec *azureARMResource) (map[string]any, string) {
			if azureARMString(rec.Properties["isDefaultVersion"]) == "True" {
				s.edgeaClearDefaultsLocked(rec.ID)
			}
			return nil, ""
		},
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			for _, f := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool { return strings.EqualFold(r.Type, edgeaFilterType) }) {
				if strings.EqualFold(azureARMString(f.Properties["versionId"]), existing.ID) {
					return "Version " + existing.Name + " is used by execution filter " + f.Name + "; delete or repoint it first."
				}
			}
			return ""
		},
		Deleted: func(s *Server, removed *azureARMResource) { edgeaForgetCode(s, removed.ID) },
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"deployversioncode": edgeaDeployCode,
			"getversioncode":    edgeaGetCode,
			"swapdefault":       edgeaSwapDefault,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "edgeActions/executionFilters",
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			return cmp.Or(
				azureWorkloadsRequired(props, "properties.", "versionId", "executionFilterIdentifierHeaderName", "executionFilterIdentifierHeaderValue"),
				azureLabLength(props, "properties.", "executionFilterIdentifierHeaderName", 1, edgeaMaxHeaderSize),
				azureLabLength(props, "properties.", "executionFilterIdentifierHeaderValue", 1, edgeaMaxHeaderSize),
			)
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"lastUpdateTime": rec.UpdatedAt.Format(time.RFC3339)}
		},
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			return esanSiblingLimit(s, existing, rec, "execution filters", edgeaMaxFilters)
		},
		Reconcile: func(s *Server, _, rec *azureARMResource) (map[string]any, string) {
			versionID := azureARMString(rec.Properties["versionId"])
			version := s.azureARMResources[azureARMKey(versionID)]
			if version == nil || !strings.EqualFold(version.Type, edgeaVersionType) || !strings.EqualFold(esanParentID(version.ID), esanParentID(rec.ID)) {
				return nil, "properties.versionId is invalid: it must name a version of this edge action."
			}
			return map[string]any{"versionId": version.ID}, ""
		},
	})
}

// handleAzureEdgeActions20251201PreviewRouter stays as the stage-0 dispatch
// hook; the shared ARM handler, which runs first, serves every route.
func (s *Server) handleAzureEdgeActions20251201PreviewRouter(http.ResponseWriter, *http.Request) bool {
	return false
}

// edgeaClearDefaultsLocked marks every other version of the edge action as
// not the default. Caller holds providerStorageMu.
func (s *Server) edgeaClearDefaultsLocked(versionID string) {
	prefix := azureARMKey(esanParentID(versionID) + "/versions/")
	for _, v := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		return strings.EqualFold(r.Type, edgeaVersionType) && strings.HasPrefix(azureARMKey(r.ID), prefix) && !strings.EqualFold(r.ID, versionID)
	}) {
		if azureARMString(v.Properties["isDefaultVersion"]) == "True" {
			updated := *v
			updated.Properties = azureARMMerge(v.Properties, map[string]any{"isDefaultVersion": "False"})
			updated.UpdatedAt = time.Now().UTC()
			s.azureARMResources[azureARMKey(v.ID)] = &updated
		}
	}
}

// edgeaUpdateVersion stores a copy of the version with props merged in.
func (s *Server) edgeaUpdateVersion(id string, props map[string]any, clearOthers bool) (*azureARMResource, bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(id)]
	if current == nil {
		return nil, false
	}
	updated := *current
	updated.Properties = azureARMMerge(current.Properties, props)
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(id)] = &updated
	if clearOthers {
		s.edgeaClearDefaultsLocked(id)
	}
	return &updated, true
}

func edgeaDeployCode(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureWorkloadsRequired(payload, "", "name", "content"); msg != "" {
		return azureOracleBad(msg)
	}
	content := azureARMString(payload["content"])
	if base64.StdEncoding.DecodedLen(len(content)) > edgeaMaxCodeBytes {
		return azureOracleBad("content is invalid: version code is limited to 4 MiB.")
	}
	if _, err := base64.StdEncoding.DecodeString(content); err != nil {
		return azureOracleBad("content is invalid: it must be base64.")
	}
	edgeaCode.Lock()
	total := len(content)
	for k, code := range edgeaCode.m[s] {
		if k != azureARMKey(rec.ID) {
			total += len(code[1])
		}
	}
	edgeaCode.Unlock()
	if total > edgeaMaxTotalCode {
		return http.StatusConflict, azureARMErrorBody("QuotaExceeded", "Deployed edge action code is limited to 64 MiB in total.")
	}
	updated, ok := s.edgeaUpdateVersion(rec.ID, map[string]any{"validationStatus": "Succeeded", "lastPackageUpdateTime": time.Now().UTC().Format(time.RFC3339)}, false)
	if !ok {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	edgeaCode.Lock()
	if edgeaCode.m[s] == nil {
		edgeaCode.m[s] = map[string][2]string{}
	}
	edgeaCode.m[s][azureARMKey(rec.ID)] = [2]string{azureARMString(payload["name"]), content}
	edgeaCode.Unlock()
	return http.StatusOK, updated.fixture()
}

func edgeaGetCode(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	edgeaCode.Lock()
	code, ok := edgeaCode.m[s][azureARMKey(rec.ID)]
	edgeaCode.Unlock()
	if !ok {
		return http.StatusNotFound, azureARMErrorBody("NotFound", "No code has been deployed to version "+rec.Name+".")
	}
	return http.StatusOK, map[string]any{"name": code[0], "content": code[1]}
}

func edgeaSwapDefault(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	if _, ok := s.edgeaUpdateVersion(rec.ID, map[string]any{"isDefaultVersion": "True"}, true); !ok {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	return http.StatusAccepted, nil
}

// edgeaForgetCode drops the deployed code of a version, or of every
// version of an edge action.
func edgeaForgetCode(s *Server, id string) {
	key := azureARMKey(id)
	edgeaCode.Lock()
	defer edgeaCode.Unlock()
	for k := range edgeaCode.m[s] {
		if k == key || strings.HasPrefix(k, key+"/") {
			delete(edgeaCode.m[s], k)
		}
	}
}
