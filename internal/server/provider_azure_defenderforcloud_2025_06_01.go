package server

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const azureDefenderForCloud20250601APIVersion = "2025-06-01"

// Defender for Storage settings (Microsoft.Security 2025-06-01): the
// "current" setting on a storage account, served by the shared ARM handler
// as an extension resource, and on-demand malware scans, served below. No
// blobs are scanned: a scan stays Running until it is canceled.
const (
	azureDFCSettingType  = "defenderForStorageSettings"
	azureDFCMaxScans     = 20 // kept per setting, newest first
	azureDFCScanResource = "Microsoft.Security/defenderForStorageSettings/malwareScans"
)

var (
	azureDFCStorageScopeRE = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.Storage/storageAccounts/[a-z0-9]{3,24}$`)
	azureDFCScanIDRE       = regexp.MustCompile(`^(latest|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`)
	// azureDFCScanPathRE matches the scan routes after the setting ID.
	azureDFCScanPathRE = regexp.MustCompile(`(?i)^(.*/providers/Microsoft\.Security/defenderForStorageSettings/current)/malwareScans/([^/]+)(/cancelMalwareScan)?$`)
)

type azureDFCScans struct {
	mu        sync.Mutex
	bySetting map[string][]map[string]any
}

func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: "Microsoft.Security", Type: azureDFCSettingType, Extension: true,
		NoPatch: true, NoDelete: true,
		ValidateName: func(name string) string {
			if name != "current" {
				return "The setting name must be 'current'."
			}
			return ""
		},
		ValidateScope: func(scope string) string {
			if !azureDFCStorageScopeRE.MatchString(scope) {
				return "Defender for Storage settings attach to a storage account."
			}
			return ""
		},
		Validate: azureDFCValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			scanning := azureARMObject(rec.Properties["malwareScanning"])
			return map[string]any{
				"malwareScanning":        azureARMMerge(scanning, map[string]any{"operationStatus": map[string]any{"code": "Succeeded", "message": ""}}),
				"sensitiveDataDiscovery": azureARMMerge(azureARMObject(rec.Properties["sensitiveDataDiscovery"]), map[string]any{"operationStatus": map[string]any{"code": "Succeeded", "message": ""}}),
			}
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"startmalwarescan": azureDFCStartScan},
	})
}

func (s *Server) azureDFCScanState() *azureDFCScans {
	return azureDMSStateFor(s, "defender-storage-scans", func() *azureDFCScans {
		return &azureDFCScans{bySetting: map[string][]map[string]any{}}
	})
}

func azureDFCValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"isEnabled", "overrideSubscriptionLevelSettings"} {
		if raw, present := props[key]; present {
			if _, ok := raw.(bool); !ok {
				return "properties." + key + " must be a boolean."
			}
		}
	}
	upload := azureARMObject(azureARMObject(props["malwareScanning"])["onUpload"])
	if raw, present := upload["capGBPerMonth"]; present {
		if limit, ok := raw.(float64); !ok || (limit != -1 && limit < 0) || limit != float64(int64(limit)) {
			return "properties.malwareScanning.onUpload.capGBPerMonth must be -1 (unlimited) or a non-negative integer."
		}
	}
	if topic := azureARMString(azureARMObject(props["malwareScanning"])["scanResultsEventGridTopicResourceId"]); topic != "" && !strings.Contains(strings.ToLower(topic), "/providers/microsoft.eventgrid/topics/") {
		return "properties.malwareScanning.scanResultsEventGridTopicResourceId must be an Event Grid topic ID."
	}
	return ""
}

// azureDFCStartScan starts an on-demand scan; malware scanning must be on.
func azureDFCStartScan(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	if rec.Properties["isEnabled"] != true || azureARMObject(azureARMObject(rec.Properties["malwareScanning"])["onUpload"])["isEnabled"] != true {
		return http.StatusConflict, azureARMErrorBody("MalwareScanningDisabled", "Malware scanning must be enabled on the setting to start a scan.")
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	scan := map[string]any{
		"id": rec.ID + "/malwareScans/" + id, "name": id, "type": azureDFCScanResource,
		"properties": map[string]any{
			"scanId": id, "scanStatus": "Running", "scanStatusMessage": "The scan is running.", "scanStartTime": now,
			"scanSummary": map[string]any{
				"blobs":                map[string]any{"totalBlobsScanned": 0, "maliciousBlobsCount": 0, "skippedBlobsCount": 0, "failedBlobsCount": 0, "scannedBlobsInGB": 0},
				"estimatedScanCostUSD": 0,
			},
		},
	}
	st := s.azureDFCScanState()
	st.mu.Lock()
	defer st.mu.Unlock()
	key := azureARMKey(rec.ID)
	scans := append([]map[string]any{scan}, st.bySetting[key]...)
	st.bySetting[key] = scans[:min(len(scans), azureDFCMaxScans)]
	return http.StatusOK, azureARMMerge(scan, nil)
}

// handleAzureDefenderForCloud20250601Router serves malware scans under a
// setting; the setting itself is served by the shared ARM handler.
func (s *Server) handleAzureDefenderForCloud20250601Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimPrefix(rawRequestPath(r), "/azure")
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.security/defenderforstoragesettings/") {
		return false
	}
	m := azureDFCScanPathRE.FindStringSubmatch(path)
	switch {
	case m == nil:
		azureARMError(w, http.StatusNotFound, "InvalidResourceType", "The resource type could not be found in the namespace 'Microsoft.Security'.")
		return true
	case hasAzureInvalidAPIVersion(r):
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	case (m[3] == "" && r.Method != http.MethodGet) || (m[3] != "" && r.Method != http.MethodPost):
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	case !azureDFCScanIDRE.MatchString(m[2]):
		azureARMError(w, http.StatusBadRequest, "InvalidScanId", "scanId must be 'latest' or a GUID.")
		return true
	}
	settingID := m[1]
	s.providerStorageMu.Lock()
	setting := s.azureARMResources[azureARMKey(settingID)]
	s.providerStorageMu.Unlock()
	if setting == nil {
		azureARMNotFound(w, "ResourceNotFound", settingID, "")
		return true
	}
	st := s.azureDFCScanState()
	st.mu.Lock()
	defer st.mu.Unlock()
	var scan map[string]any
	for _, candidate := range st.bySetting[azureARMKey(setting.ID)] {
		if strings.EqualFold(m[2], "latest") || strings.EqualFold(azureARMString(candidate["name"]), m[2]) {
			scan = candidate
			break
		}
	}
	if scan == nil {
		azureARMNotFound(w, "ResourceNotFound", setting.ID+"/malwareScans/"+m[2], "")
		return true
	}
	if m[3] != "" {
		props := azureARMObject(scan["properties"])
		if props["scanStatus"] != "Running" {
			azureARMError(w, http.StatusConflict, "ScanNotRunning", "Only a running scan can be canceled.")
			return true
		}
		props["scanStatus"], props["scanStatusMessage"], props["scanEndTime"] = "Canceled", "The scan was canceled.", time.Now().UTC().Format(time.RFC3339)
		scan["properties"] = props
	}
	respondJSON(w, http.StatusOK, azureARMMerge(scan, nil))
	return true
}
