package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type azureAVSPrivateCloud struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureAVSAuthorization struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSGlobalReachConnection struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSLicense struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSAddon struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSMaintenance struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Properties       map[string]any
}

type azureAVSScriptExecution struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSIscsiPath struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Properties       map[string]any
	UpdatedAt        time.Time
}

type azureAVSCluster struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSDatastore struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	ClusterName      string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSPlacementPolicy struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	ClusterName      string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSWorkloadNetworkDNSZone struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureAVSWorkloadNetworkVMGroup struct {
	SubscriptionID   string
	ResourceGroup    string
	PrivateCloudName string
	Name             string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s *Server) handleAzureAVSRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsAVSPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureAVSAllowedMethod(r.Method) {
		return false
	}

	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AVS") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureAVSOperationsFixture()})
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureAVSRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		switch {
		case len(tail) == 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "checkQuotaAvailability") && r.Method == http.MethodPost:
			respondJSON(w, http.StatusOK, azureAVSQuotaFixture(subscriptionID, tail[1]))
			return true
		case len(tail) == 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "checkTrialAvailability") && r.Method == http.MethodPost:
			respondJSON(w, http.StatusOK, azureAVSTrialFixture(subscriptionID, tail[1]))
			return true
		case len(tail) == 1 && strings.EqualFold(tail[0], "privateClouds") && r.Method == http.MethodGet:
			s.handleAzureAVSPrivateCloudListInSubscription(w, subscriptionID)
			return true
		default:
			respondAzureImplemented(w, path)
			return true
		}
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "privateClouds") && r.Method == http.MethodGet:
		s.handleAzureAVSPrivateCloudListByResourceGroup(w, subscriptionID, resourceGroup)
		return true
	case len(tail) >= 2 && strings.EqualFold(tail[0], "privateClouds"):
		privateCloudName := strings.TrimSpace(tail[1])
		if privateCloudName == "" {
			respondAzureInvalidRequest(w, path, "private cloud name is required")
			return true
		}
		if len(tail) == 2 {
			switch r.Method {
			case http.MethodGet:
				s.handleAzureAVSPrivateCloudGet(w, path, subscriptionID, resourceGroup, privateCloudName, false)
			case http.MethodHead:
				s.handleAzureAVSPrivateCloudGet(w, path, subscriptionID, resourceGroup, privateCloudName, true)
			case http.MethodPut:
				s.handleAzureAVSPrivateCloudCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName)
			case http.MethodPatch:
				s.handleAzureAVSPrivateCloudPatch(w, r, path, subscriptionID, resourceGroup, privateCloudName)
			case http.MethodDelete:
				s.handleAzureAVSPrivateCloudDelete(w, path, subscriptionID, resourceGroup, privateCloudName)
			default:
				respondAzureImplemented(w, path)
			}
			return true
		}
		switch strings.ToLower(strings.TrimSpace(tail[2])) {
		case "listadmincredentials":
			if r.Method == http.MethodPost {
				s.handleAzureAVSPrivateCloudListAdminCredentials(w, path, subscriptionID, resourceGroup, privateCloudName)
				return true
			}
		case "getvcflicense":
			if r.Method == http.MethodPost {
				s.handleAzureAVSPrivateCloudGetVCFLicense(w, path, subscriptionID, resourceGroup, privateCloudName)
				return true
			}
		case "rotatensxtpassword":
			if r.Method == http.MethodPost {
				s.handleAzureAVSPrivateCloudPasswordRotation(w, path, subscriptionID, resourceGroup, privateCloudName, "rotateNsxtPassword")
				return true
			}
		case "rotatevcenterpassword":
			if r.Method == http.MethodPost {
				s.handleAzureAVSPrivateCloudPasswordRotation(w, path, subscriptionID, resourceGroup, privateCloudName, "rotateVcenterPassword")
				return true
			}
		case "authorizations":
			s.handleAzureAVSAuthorizationRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "globalreachconnections":
			s.handleAzureAVSGlobalReachRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "licenses":
			s.handleAzureAVSLicenseRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "addons":
			s.handleAzureAVSAddonRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "maintenances":
			s.handleAzureAVSMaintenanceRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "scriptexecutions":
			s.handleAzureAVSScriptExecutionRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "iscsipaths":
			s.handleAzureAVSIscsiPathRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "clusters":
			s.handleAzureAVSClusterRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		case "workloadnetworks":
			s.handleAzureAVSWorkloadNetworkRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[3:])
			return true
		}
		respondAzureImplemented(w, path)
		return true
	default:
		respondAzureImplemented(w, path)
		return true
	}
}

func (s *Server) handleAzureAVSAuthorizationRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSAuthorizationList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "authorization name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSAuthorizationGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodPut:
		s.handleAzureAVSAuthorizationCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodDelete:
		s.handleAzureAVSAuthorizationDelete(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSGlobalReachRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSGlobalReachList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "global reach connection name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSGlobalReachGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodPut:
		s.handleAzureAVSGlobalReachCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodDelete:
		s.handleAzureAVSGlobalReachDelete(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSLicenseRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSLicenseList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "license name is required")
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "getProperties") && r.Method == http.MethodPost {
		s.handleAzureAVSLicenseGetProperties(w, path, subscriptionID, resourceGroup, privateCloudName, name)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSLicenseGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodPut:
		s.handleAzureAVSLicenseCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodDelete:
		s.handleAzureAVSLicenseDelete(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSAddonRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSAddonList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "addon name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSAddonGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodPut:
		s.handleAzureAVSAddonCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodDelete:
		s.handleAzureAVSAddonDelete(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSMaintenanceRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSMaintenanceList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "maintenance name is required")
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureAVSMaintenanceGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "initiateChecks") && r.Method == http.MethodPost {
		s.handleAzureAVSMaintenanceInitiateChecks(w, path, subscriptionID, resourceGroup, privateCloudName, name)
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "reschedule") && r.Method == http.MethodPost {
		s.handleAzureAVSMaintenanceReschedule(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "schedule") && r.Method == http.MethodGet {
		s.handleAzureAVSMaintenanceSchedule(w, path, subscriptionID, resourceGroup, privateCloudName, name)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureAVSScriptExecutionRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSScriptExecutionList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "script execution name is required")
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "getExecutionLogs") && r.Method == http.MethodPost {
		s.handleAzureAVSScriptExecutionLogs(w, path, subscriptionID, resourceGroup, privateCloudName, name)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSScriptExecutionGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodPut:
		s.handleAzureAVSScriptExecutionCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodDelete:
		s.handleAzureAVSScriptExecutionDelete(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSIscsiPathRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSIscsiPathList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) != 1 || !strings.EqualFold(tail[0], "default") {
		respondAzureImplemented(w, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSIscsiPathGet(w, path, subscriptionID, resourceGroup, privateCloudName)
	case http.MethodPut:
		s.handleAzureAVSIscsiPathCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName)
	case http.MethodDelete:
		s.handleAzureAVSIscsiPathDelete(w, path, subscriptionID, resourceGroup, privateCloudName)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSClusterRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSClusterList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	clusterName := strings.TrimSpace(tail[0])
	if clusterName == "" {
		respondAzureInvalidRequest(w, path, "cluster name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureAVSClusterGet(w, path, subscriptionID, resourceGroup, privateCloudName, clusterName)
		case http.MethodPut:
			s.handleAzureAVSClusterCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName)
		case http.MethodPatch:
			s.handleAzureAVSClusterPatch(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName)
		case http.MethodDelete:
			s.handleAzureAVSClusterDelete(w, path, subscriptionID, resourceGroup, privateCloudName, clusterName)
		default:
			respondAzureImplemented(w, path)
		}
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "listzones":
		if r.Method == http.MethodPost {
			s.handleAzureAVSClusterListZones(w, path, subscriptionID, resourceGroup, privateCloudName, clusterName)
			return
		}
	case "hosts":
		s.handleAzureAVSHostRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName, tail[2:])
		return
	case "datastores":
		s.handleAzureAVSDatastoreRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName, tail[2:])
		return
	case "placementpolicies":
		s.handleAzureAVSPlacementPolicyRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName, tail[2:])
		return
	case "virtualmachines":
		s.handleAzureAVSClusterVirtualMachineRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName, tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureAVSHostRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, clusterName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureAVSHostListFixture(subscriptionID, resourceGroup, privateCloudName, clusterName)})
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, azureAVSHostFixture(subscriptionID, resourceGroup, privateCloudName, clusterName, tail[0]))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureAVSDatastoreRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, clusterName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSDatastoreList(w, subscriptionID, resourceGroup, privateCloudName, clusterName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "datastore name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSDatastoreGet(w, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	case http.MethodPut:
		s.handleAzureAVSDatastoreCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	case http.MethodDelete:
		s.handleAzureAVSDatastoreDelete(w, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSPlacementPolicyRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, clusterName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSPlacementPolicyList(w, subscriptionID, resourceGroup, privateCloudName, clusterName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "placement policy name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSPlacementPolicyGet(w, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	case http.MethodPut:
		s.handleAzureAVSPlacementPolicyCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	case http.MethodPatch:
		s.handleAzureAVSPlacementPolicyPatch(w, r, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	case http.MethodDelete:
		s.handleAzureAVSPlacementPolicyDelete(w, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSClusterVirtualMachineRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, clusterName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureAVSClusterVirtualMachineListFixture(subscriptionID, resourceGroup, privateCloudName, clusterName)})
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	vmName := strings.TrimSpace(tail[0])
	if vmName == "" {
		respondAzureInvalidRequest(w, path, "virtual machine id is required")
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, azureAVSClusterVirtualMachineFixture(subscriptionID, resourceGroup, privateCloudName, clusterName, vmName))
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "restrictMovement") && r.Method == http.MethodPost {
		respondJSON(w, http.StatusOK, map[string]any{
			"id":   azureAVSClusterVirtualMachineID(subscriptionID, resourceGroup, privateCloudName, clusterName, vmName),
			"name": vmName,
			"type": "Microsoft.AVS/privateClouds/clusters/virtualMachines",
			"properties": map[string]any{
				"restrictMovement":  true,
				"provisioningState": "Succeeded",
			},
		})
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureAVSWorkloadNetworkRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 || !strings.EqualFold(tail[0], "default") {
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, azureAVSWorkloadNetworkRootFixture(subscriptionID, resourceGroup, privateCloudName))
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "dnszones":
		s.handleAzureAVSDNSZoneRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[2:])
	case "vmgroups":
		s.handleAzureAVSVMGroupRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[2:])
	case "virtualmachines":
		s.handleAzureAVSWorkloadVirtualMachineRouter(w, r, path, subscriptionID, resourceGroup, privateCloudName, tail[2:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSDNSZoneRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSDNSZoneList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "dns zone id is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSDNSZoneGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodPut:
		s.handleAzureAVSDNSZoneCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodDelete:
		s.handleAzureAVSDNSZoneDelete(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSVMGroupRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAVSVMGroupList(w, subscriptionID, resourceGroup, privateCloudName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "vm group id is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureAVSVMGroupGet(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodPut:
		s.handleAzureAVSVMGroupCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, privateCloudName, name)
	case http.MethodDelete:
		s.handleAzureAVSVMGroupDelete(w, path, subscriptionID, resourceGroup, privateCloudName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAVSWorkloadVirtualMachineRouter(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureAVSWorkloadVirtualMachineListFixture(subscriptionID, resourceGroup, privateCloudName)})
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, azureAVSWorkloadVirtualMachineFixture(subscriptionID, resourceGroup, privateCloudName, tail[0]))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureAVSPrivateCloudCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureAVSString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()
	key := azureAVSPrivateCloudKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	resource := s.azureAVSPrivateClouds[key]
	if resource == nil {
		resource = &azureAVSPrivateCloud{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			CreatedAt:      now,
		}
	}
	resource.Location = location
	resource.Tags = azureAVSStringMap(payload["tags"])
	resource.Properties = azureAVSObject(payload["properties"])
	resource.UpdatedAt = now
	s.azureAVSPrivateClouds[key] = resource
	s.providerStorageMu.Unlock()

	_ = body
	respondJSON(w, http.StatusOK, azureAVSPrivateCloudFixture(resource))
}

func (s *Server) handleAzureAVSPrivateCloudPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureAVSPrivateCloudKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	resource := s.azureAVSPrivateClouds[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	if location := azureAVSString(payload["location"]); location != "" {
		resource.Location = location
	}
	if tags, ok := payload["tags"]; ok {
		resource.Tags = azureAVSStringMap(tags)
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureAVSMergeObject(resource.Properties, azureAVSObject(properties))
	}
	resource.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureAVSPrivateCloudFixture(resource))
}

func (s *Server) handleAzureAVSPrivateCloudGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, head bool) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSPrivateClouds[azureAVSPrivateCloudKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		if head {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureAVSPrivateCloudFixture(resource))
}

func (s *Server) handleAzureAVSPrivateCloudDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureAVSPrivateCloudKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSPrivateClouds[key]
	delete(s.azureAVSPrivateClouds, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSPrivateCloud{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			Location:       "eastus",
			Tags:           map[string]string{},
			Properties:     map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSPrivateCloudFixture(resource))
}

func (s *Server) handleAzureAVSPrivateCloudListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSPrivateClouds {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			continue
		}
		items = append(items, azureAVSPrivateCloudFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSPrivateCloudListInSubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSPrivateClouds {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) {
			continue
		}
		items = append(items, azureAVSPrivateCloudFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSPrivateCloudListAdminCredentials(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	if !s.azureAVSPrivateCloudExists(subscriptionID, resourceGroup, name) {
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"nsxtPassword":    "Stackyard-Nsxt-Password-123!",
		"vcenterPassword": "Stackyard-Vcenter-Password-123!",
	})
}

func (s *Server) handleAzureAVSPrivateCloudGetVCFLicense(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	if !s.azureAVSPrivateCloudExists(subscriptionID, resourceGroup, name) {
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"licenseKey": "STACKYARD-VCF-LICENSE-KEY",
	})
}

func (s *Server) handleAzureAVSPrivateCloudPasswordRotation(w http.ResponseWriter, path, subscriptionID, resourceGroup, name, operation string) {
	if !s.azureAVSPrivateCloudExists(subscriptionID, resourceGroup, name) {
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"operation": operation,
		"status":    "Succeeded",
		"target":    name,
	})
}

func (s *Server) handleAzureAVSAuthorizationCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	body, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureAVSPrivateCloudExists(subscriptionID, resourceGroup, privateCloudName) {
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	now := time.Now().UTC()
	key := azureAVSAuthorizationKey(subscriptionID, resourceGroup, privateCloudName, name)

	s.providerStorageMu.Lock()
	resource := &azureAVSAuthorization{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"authorizationKey": azureAVSHash("auth", key)}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSAuthorizations[key] = resource
	s.providerStorageMu.Unlock()

	_ = body
	respondJSON(w, http.StatusOK, azureAVSAuthorizationFixture(resource))
}

func (s *Server) handleAzureAVSAuthorizationGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSAuthorizations[azureAVSAuthorizationKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "authorization not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSAuthorizationFixture(resource))
}

func (s *Server) handleAzureAVSAuthorizationDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSAuthorizationKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSAuthorizations[key]
	delete(s.azureAVSAuthorizations, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSAuthorization{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             name,
			Location:         "eastus",
			Tags:             map[string]string{},
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSAuthorizationFixture(resource))
}

func (s *Server) handleAzureAVSAuthorizationList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSAuthorizations {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSAuthorizationFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSGlobalReachCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureAVSPrivateCloudExists(subscriptionID, resourceGroup, privateCloudName) {
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	now := time.Now().UTC()
	key := azureAVSGlobalReachKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSGlobalReachConnection{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded"}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSGlobalReachConnections[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSGlobalReachFixture(resource))
}

func (s *Server) handleAzureAVSGlobalReachGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSGlobalReachConnections[azureAVSGlobalReachKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "global reach connection not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSGlobalReachFixture(resource))
}

func (s *Server) handleAzureAVSGlobalReachDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSGlobalReachKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSGlobalReachConnections[key]
	delete(s.azureAVSGlobalReachConnections, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSGlobalReachConnection{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             name,
			Location:         "eastus",
			Tags:             map[string]string{},
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSGlobalReachFixture(resource))
}

func (s *Server) handleAzureAVSGlobalReachList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSGlobalReachConnections {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSGlobalReachFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSLicenseCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureAVSPrivateCloudExists(subscriptionID, resourceGroup, privateCloudName) {
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	now := time.Now().UTC()
	key := azureAVSLicenseKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSLicense{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded", "licenseType": "WindowsServer"}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSLicenses[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSLicenseFixture(resource))
}

func (s *Server) handleAzureAVSLicenseGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSLicenses[azureAVSLicenseKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "license not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSLicenseFixture(resource))
}

func (s *Server) handleAzureAVSLicenseGetProperties(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSLicenses[azureAVSLicenseKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "license not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"licenseType": azureAVSCoalesceString(azureAVSString(resource.Properties["licenseType"]), "WindowsServer"),
		"edition":     azureAVSCoalesceString(azureAVSString(resource.Properties["edition"]), "Datacenter"),
		"status":      "Active",
	})
}

func (s *Server) handleAzureAVSLicenseDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSLicenseKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSLicenses[key]
	delete(s.azureAVSLicenses, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSLicense{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             name,
			Location:         "eastus",
			Tags:             map[string]string{},
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSLicenseFixture(resource))
}

func (s *Server) handleAzureAVSLicenseList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSLicenses {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSLicenseFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSAddonCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureAVSAddonKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSAddon{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded"}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSAddons[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSAddonFixture(resource))
}

func (s *Server) handleAzureAVSAddonGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSAddons[azureAVSAddonKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = azureAVSDefaultAddon(subscriptionID, resourceGroup, privateCloudName, name)
	}
	respondJSON(w, http.StatusOK, azureAVSAddonFixture(resource))
}

func (s *Server) handleAzureAVSAddonDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSAddonKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSAddons[key]
	delete(s.azureAVSAddons, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = azureAVSDefaultAddon(subscriptionID, resourceGroup, privateCloudName, name)
	}
	respondJSON(w, http.StatusOK, azureAVSAddonFixture(resource))
}

func (s *Server) handleAzureAVSAddonList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSAddons {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSAddonFixture(resource))
	}
	s.providerStorageMu.Unlock()
	if len(items) == 0 {
		items = append(items, azureAVSAddonFixture(azureAVSDefaultAddon(subscriptionID, resourceGroup, privateCloudName, "srm")))
	}
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSMaintenanceList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	items := []map[string]any{
		azureAVSMaintenanceFixture(azureAVSDefaultMaintenance(subscriptionID, resourceGroup, privateCloudName, "default")),
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSMaintenanceGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	respondJSON(w, http.StatusOK, azureAVSMaintenanceFixture(azureAVSDefaultMaintenance(subscriptionID, resourceGroup, privateCloudName, name)))
}

func (s *Server) handleAzureAVSMaintenanceInitiateChecks(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":      "Succeeded",
		"maintenance": name,
	})
}

func (s *Server) handleAzureAVSMaintenanceReschedule(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":   azureAVSMaintenanceID(subscriptionID, resourceGroup, privateCloudName, name) + "/reschedule",
		"name": name,
		"type": "Microsoft.AVS/privateClouds/maintenances/reschedule",
		"properties": map[string]any{
			"kind":           "Reschedule",
			"status":         "Succeeded",
			"rescheduleTime": azureAVSCoalesceString(azureAVSString(payload["rescheduleTime"]), time.Now().UTC().Add(24*time.Hour).Format(time.RFC3339)),
		},
	})
}

func (s *Server) handleAzureAVSMaintenanceSchedule(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	respondJSON(w, http.StatusOK, map[string]any{
		"scheduledByMicrosoft": true,
		"scheduledStartTime":   time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
	})
}

func (s *Server) handleAzureAVSScriptExecutionCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureAVSScriptExecutionKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSScriptExecution{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded", "scriptCmdletId": "Get-Cluster"}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSScriptExecutions[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSScriptExecutionFixture(resource))
}

func (s *Server) handleAzureAVSScriptExecutionGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSScriptExecutions[azureAVSScriptExecutionKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "script execution not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSScriptExecutionFixture(resource))
}

func (s *Server) handleAzureAVSScriptExecutionDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSScriptExecutionKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSScriptExecutions[key]
	delete(s.azureAVSScriptExecutions, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSScriptExecution{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             name,
			Location:         "eastus",
			Tags:             map[string]string{},
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSScriptExecutionFixture(resource))
}

func (s *Server) handleAzureAVSScriptExecutionList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSScriptExecutions {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSScriptExecutionFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSScriptExecutionLogs(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	if !s.azureAVSScriptExecutionExists(subscriptionID, resourceGroup, privateCloudName, name) {
		azureAVSNotFound(w, path, "script execution not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"scriptOutputStream":  "stdout",
		"scriptExecutionLogs": []string{"Stackyard execution log line"},
	})
}

func (s *Server) handleAzureAVSIscsiPathCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureAVSIscsiPathKey(subscriptionID, resourceGroup, privateCloudName)
	s.providerStorageMu.Lock()
	resource := &azureAVSIscsiPath{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded"}, azureAVSObject(payload["properties"])),
		UpdatedAt:        time.Now().UTC(),
	}
	s.azureAVSIscsiPaths[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSIscsiPathFixture(resource))
}

func (s *Server) handleAzureAVSIscsiPathGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSIscsiPaths[azureAVSIscsiPathKey(subscriptionID, resourceGroup, privateCloudName)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSIscsiPath{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Properties: map[string]any{
				"provisioningState": "Succeeded",
				"networkType":       "Dynamic",
				"target":            "10.0.0.10",
			},
			UpdatedAt: time.Now().UTC(),
		}
	}
	respondJSON(w, http.StatusOK, azureAVSIscsiPathFixture(resource))
}

func (s *Server) handleAzureAVSIscsiPathDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName string) {
	key := azureAVSIscsiPathKey(subscriptionID, resourceGroup, privateCloudName)
	s.providerStorageMu.Lock()
	resource := s.azureAVSIscsiPaths[key]
	delete(s.azureAVSIscsiPaths, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSIscsiPath{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Properties:       map[string]any{},
			UpdatedAt:        time.Now().UTC(),
		}
	}
	respondJSON(w, http.StatusOK, azureAVSIscsiPathFixture(resource))
}

func (s *Server) handleAzureAVSIscsiPathList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSIscsiPaths[azureAVSIscsiPathKey(subscriptionID, resourceGroup, privateCloudName)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSIscsiPath{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Properties: map[string]any{
				"provisioningState": "Succeeded",
				"networkType":       "Dynamic",
				"target":            "10.0.0.10",
			},
			UpdatedAt: time.Now().UTC(),
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": []map[string]any{azureAVSIscsiPathFixture(resource)}})
}

func (s *Server) handleAzureAVSClusterCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureAVSPrivateCloudExists(subscriptionID, resourceGroup, privateCloudName) {
		azureAVSNotFound(w, path, "private cloud not found")
		return
	}
	now := time.Now().UTC()
	key := azureAVSClusterKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSCluster{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded", "clusterSize": 3}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSClusters[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSClusterFixture(resource))
}

func (s *Server) handleAzureAVSClusterPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureAVSClusterKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSClusters[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		azureAVSNotFound(w, path, "cluster not found")
		return
	}
	if tags, ok := payload["tags"]; ok {
		resource.Tags = azureAVSStringMap(tags)
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureAVSMergeObject(resource.Properties, azureAVSObject(properties))
	}
	resource.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSClusterFixture(resource))
}

func (s *Server) handleAzureAVSClusterGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSClusters[azureAVSClusterKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "cluster not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSClusterFixture(resource))
}

func (s *Server) handleAzureAVSClusterDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSClusterKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSClusters[key]
	delete(s.azureAVSClusters, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSCluster{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             name,
			Location:         "eastus",
			Tags:             map[string]string{},
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSClusterFixture(resource))
}

func (s *Server) handleAzureAVSClusterList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSClusters {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSClusterFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSClusterListZones(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	if !s.azureAVSClusterExists(subscriptionID, resourceGroup, privateCloudName, name) {
		azureAVSNotFound(w, path, "cluster not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"zones": []map[string]any{
			{"zone": "1", "status": "Available"},
			{"zone": "2", "status": "Available"},
		},
	})
}

func (s *Server) handleAzureAVSDatastoreCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureAVSClusterExists(subscriptionID, resourceGroup, privateCloudName, clusterName) {
		azureAVSNotFound(w, path, "cluster not found")
		return
	}
	now := time.Now().UTC()
	key := azureAVSDatastoreKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSDatastore{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		ClusterName:      clusterName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded", "netAppVolumeId": "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.NetApp/netAppAccounts/na-a/capacityPools/pool-a/volumes/vol-a"}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSDatastores[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSDatastoreFixture(resource))
}

func (s *Server) handleAzureAVSDatastoreGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSDatastores[azureAVSDatastoreKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "datastore not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSDatastoreFixture(resource))
}

func (s *Server) handleAzureAVSDatastoreDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name string) {
	key := azureAVSDatastoreKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSDatastores[key]
	delete(s.azureAVSDatastores, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSDatastore{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			ClusterName:      clusterName,
			Name:             name,
			Location:         "eastus",
			Tags:             map[string]string{},
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSDatastoreFixture(resource))
}

func (s *Server) handleAzureAVSDatastoreList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName, clusterName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSDatastores {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) || !strings.EqualFold(resource.ClusterName, clusterName) {
			continue
		}
		items = append(items, azureAVSDatastoreFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSPlacementPolicyCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureAVSPlacementPolicyKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSPlacementPolicy{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		ClusterName:      clusterName,
		Name:             name,
		Location:         azureAVSCoalesceString(azureAVSString(payload["location"]), "eastus"),
		Tags:             azureAVSStringMap(payload["tags"]),
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded", "affinityType": "VmVm"}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSPlacementPolicies[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSPlacementPolicyFixture(resource))
}

func (s *Server) handleAzureAVSPlacementPolicyPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureAVSPlacementPolicyKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSPlacementPolicies[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		azureAVSNotFound(w, path, "placement policy not found")
		return
	}
	if tags, ok := payload["tags"]; ok {
		resource.Tags = azureAVSStringMap(tags)
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureAVSMergeObject(resource.Properties, azureAVSObject(properties))
	}
	resource.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSPlacementPolicyFixture(resource))
}

func (s *Server) handleAzureAVSPlacementPolicyGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSPlacementPolicies[azureAVSPlacementPolicyKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "placement policy not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSPlacementPolicyFixture(resource))
}

func (s *Server) handleAzureAVSPlacementPolicyDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, clusterName, name string) {
	key := azureAVSPlacementPolicyKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSPlacementPolicies[key]
	delete(s.azureAVSPlacementPolicies, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSPlacementPolicy{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			ClusterName:      clusterName,
			Name:             name,
			Location:         "eastus",
			Tags:             map[string]string{},
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSPlacementPolicyFixture(resource))
}

func (s *Server) handleAzureAVSPlacementPolicyList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName, clusterName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSPlacementPolicies {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) || !strings.EqualFold(resource.ClusterName, clusterName) {
			continue
		}
		items = append(items, azureAVSPlacementPolicyFixture(resource))
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSDNSZoneCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureAVSDNSZoneKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSWorkloadNetworkDNSZone{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded", "domain": name + ".local"}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSDNSZones[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSDNSZoneFixture(resource))
}

func (s *Server) handleAzureAVSDNSZoneGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSDNSZones[azureAVSDNSZoneKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "dns zone not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSDNSZoneFixture(resource))
}

func (s *Server) handleAzureAVSDNSZoneDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSDNSZoneKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSDNSZones[key]
	delete(s.azureAVSDNSZones, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSWorkloadNetworkDNSZone{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             name,
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSDNSZoneFixture(resource))
}

func (s *Server) handleAzureAVSDNSZoneList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSDNSZones {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSDNSZoneFixture(resource))
	}
	s.providerStorageMu.Unlock()
	if len(items) == 0 {
		items = append(items, azureAVSDNSZoneFixture(&azureAVSWorkloadNetworkDNSZone{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             "corp.local",
			Properties: map[string]any{
				"domain":            "corp.local",
				"dnsServerIP":       "10.0.0.20",
				"provisioningState": "Succeeded",
			},
		}))
	}
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAVSVMGroupCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	_, payload, ok := azureAVSReadRequestBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureAVSVMGroupKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := &azureAVSWorkloadNetworkVMGroup{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Properties:       azureAVSMergeObject(map[string]any{"provisioningState": "Succeeded", "displayName": name}, azureAVSObject(payload["properties"])),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.azureAVSVMGroups[key] = resource
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureAVSVMGroupFixture(resource))
}

func (s *Server) handleAzureAVSVMGroupGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureAVSVMGroups[azureAVSVMGroupKey(subscriptionID, resourceGroup, privateCloudName, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureAVSNotFound(w, path, "vm group not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAVSVMGroupFixture(resource))
}

func (s *Server) handleAzureAVSVMGroupDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, privateCloudName, name string) {
	key := azureAVSVMGroupKey(subscriptionID, resourceGroup, privateCloudName, name)
	s.providerStorageMu.Lock()
	resource := s.azureAVSVMGroups[key]
	delete(s.azureAVSVMGroups, key)
	s.providerStorageMu.Unlock()
	if resource == nil {
		resource = &azureAVSWorkloadNetworkVMGroup{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             name,
			Properties:       map[string]any{},
		}
	}
	respondJSON(w, http.StatusOK, azureAVSVMGroupFixture(resource))
}

func (s *Server) handleAzureAVSVMGroupList(w http.ResponseWriter, subscriptionID, resourceGroup, privateCloudName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureAVSVMGroups {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.ResourceGroup, resourceGroup) || !strings.EqualFold(resource.PrivateCloudName, privateCloudName) {
			continue
		}
		items = append(items, azureAVSVMGroupFixture(resource))
	}
	s.providerStorageMu.Unlock()
	if len(items) == 0 {
		items = append(items, azureAVSVMGroupFixture(&azureAVSWorkloadNetworkVMGroup{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			PrivateCloudName: privateCloudName,
			Name:             "app-group",
			Properties: map[string]any{
				"displayName":       "app-group",
				"members":           []string{"vm-1"},
				"provisioningState": "Succeeded",
			},
		}))
	}
	sort.Slice(items, func(i, j int) bool { return azureAVSString(items[i]["id"]) < azureAVSString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func azureIsAVSPath(segments []string) bool {
	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AVS") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureAVSRouteContext(segments)
	return ok && len(tail) > 0
}

func azureAVSRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.AVS") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureAVSAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func azureAVSReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "invalid JSON request body")
		return nil, nil, false
	}
	return body, payload, true
}

func (s *Server) azureAVSPrivateCloudExists(subscriptionID, resourceGroup, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureAVSPrivateClouds[azureAVSPrivateCloudKey(subscriptionID, resourceGroup, name)] != nil
}

func (s *Server) azureAVSClusterExists(subscriptionID, resourceGroup, privateCloudName, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureAVSClusters[azureAVSClusterKey(subscriptionID, resourceGroup, privateCloudName, name)] != nil
}

func (s *Server) azureAVSScriptExecutionExists(subscriptionID, resourceGroup, privateCloudName, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureAVSScriptExecutions[azureAVSScriptExecutionKey(subscriptionID, resourceGroup, privateCloudName, name)] != nil
}

func azureAVSNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerAzure,
		"path":     path,
	})
}

func azureAVSOperationsFixture() []map[string]any {
	return []map[string]any{
		{"name": "Microsoft.AVS/privateClouds/read", "display": map[string]any{"provider": "Microsoft.AVS", "resource": "privateClouds", "operation": "Get Private Cloud"}},
		{"name": "Microsoft.AVS/privateClouds/write", "display": map[string]any{"provider": "Microsoft.AVS", "resource": "privateClouds", "operation": "Create or Update Private Cloud"}},
		{"name": "Microsoft.AVS/privateClouds/delete", "display": map[string]any{"provider": "Microsoft.AVS", "resource": "privateClouds", "operation": "Delete Private Cloud"}},
		{"name": "Microsoft.AVS/privateClouds/clusters/read", "display": map[string]any{"provider": "Microsoft.AVS", "resource": "clusters", "operation": "Get Cluster"}},
		{"name": "Microsoft.AVS/privateClouds/workloadNetworks/vmGroups/write", "display": map[string]any{"provider": "Microsoft.AVS", "resource": "vmGroups", "operation": "Update VM Group"}},
	}
}

func azureAVSQuotaFixture(subscriptionID, location string) map[string]any {
	return map[string]any{
		"hostsRemaining":          96,
		"quotaEnabled":            true,
		"subscriptionId":          subscriptionID,
		"availabilityZoneType":    "SingleZone",
		"location":                location,
		"clusterMaximumNodeCount": 16,
	}
}

func azureAVSTrialFixture(subscriptionID, location string) map[string]any {
	return map[string]any{
		"status":         "Available",
		"subscriptionId": subscriptionID,
		"location":       location,
	}
}

func azureAVSPrivateCloudFixture(resource *azureAVSPrivateCloud) map[string]any {
	properties := azureAVSMergeObject(map[string]any{
		"provisioningState": "Succeeded",
		"circuit": map[string]any{
			"expressRouteID": "/subscriptions/" + resource.SubscriptionID + "/resourceGroups/" + resource.ResourceGroup + "/providers/Microsoft.Network/expressRouteCircuits/" + resource.Name + "-er",
		},
		"endpoints": map[string]any{
			"nsxtManager": "https://" + resource.Name + ".nsxt.local",
			"vcsa":        "https://" + resource.Name + ".vcsa.local",
		},
	}, resource.Properties)
	return map[string]any{
		"id":         azureAVSPrivateCloudID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.AVS/privateClouds",
		"location":   resource.Location,
		"tags":       azureAVSAnyStringMap(resource.Tags),
		"properties": properties,
	}
}

func azureAVSAuthorizationFixture(resource *azureAVSAuthorization) map[string]any {
	return map[string]any{
		"id":       azureAVSAuthorizationID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/authorizations",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"authorizationKey":  azureAVSCoalesceString(azureAVSString(resource.Properties["authorizationKey"]), azureAVSHash("authorization", resource.Name)),
			"expressRouteId":    "/subscriptions/" + resource.SubscriptionID + "/resourceGroups/" + resource.ResourceGroup + "/providers/Microsoft.Network/expressRouteCircuits/" + resource.Name,
		}, resource.Properties),
	}
}

func azureAVSGlobalReachFixture(resource *azureAVSGlobalReachConnection) map[string]any {
	return map[string]any{
		"id":       azureAVSGlobalReachID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/globalReachConnections",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState":         "Succeeded",
			"authorizationKey":          azureAVSHash("globalreach", resource.Name),
			"peerExpressRouteCircuit":   "/subscriptions/" + resource.SubscriptionID + "/resourceGroups/" + resource.ResourceGroup + "/providers/Microsoft.Network/expressRouteCircuits/peer-a",
			"expressRouteID":            "/subscriptions/" + resource.SubscriptionID + "/resourceGroups/" + resource.ResourceGroup + "/providers/Microsoft.Network/expressRouteCircuits/local-a",
			"circuitConnectionStatus":   "Connected",
			"remoteExpressRouteCircuit": "remote-a",
		}, resource.Properties),
	}
}

func azureAVSLicenseFixture(resource *azureAVSLicense) map[string]any {
	return map[string]any{
		"id":       azureAVSLicenseID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/licenses",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"licenseType":       "WindowsServer",
			"edition":           "Datacenter",
		}, resource.Properties),
	}
}

func azureAVSAddonFixture(resource *azureAVSAddon) map[string]any {
	return map[string]any{
		"id":       azureAVSAddonID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/addons",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"addonType":         "SRM",
			"status":            "Enabled",
		}, resource.Properties),
	}
}

func azureAVSMaintenanceFixture(resource *azureAVSMaintenance) map[string]any {
	return map[string]any{
		"id":   azureAVSMaintenanceID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.Name),
		"name": resource.Name,
		"type": "Microsoft.AVS/privateClouds/maintenances",
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState":  "Succeeded",
			"maintenanceState":   "Scheduled",
			"scheduledStartTime": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		}, resource.Properties),
	}
}

func azureAVSScriptExecutionFixture(resource *azureAVSScriptExecution) map[string]any {
	return map[string]any{
		"id":       azureAVSScriptExecutionID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/scriptExecutions",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"executionStatus":   "Succeeded",
			"scriptCmdletId":    "Get-Cluster",
			"timeout":           "PT30M",
		}, resource.Properties),
	}
}

func azureAVSIscsiPathFixture(resource *azureAVSIscsiPath) map[string]any {
	return map[string]any{
		"id":   azureAVSIscsiPathID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName),
		"name": "default",
		"type": "Microsoft.AVS/privateClouds/iscsiPaths",
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"networkType":       "Dynamic",
			"target":            "10.0.0.10",
		}, resource.Properties),
	}
}

func azureAVSClusterFixture(resource *azureAVSCluster) map[string]any {
	return map[string]any{
		"id":       azureAVSClusterID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/clusters",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"clusterSize":       3,
			"hosts":             3,
		}, resource.Properties),
	}
}

func azureAVSHostListFixture(subscriptionID, resourceGroup, privateCloudName, clusterName string) []map[string]any {
	return []map[string]any{
		azureAVSHostFixture(subscriptionID, resourceGroup, privateCloudName, clusterName, "host-1"),
	}
}

func azureAVSHostFixture(subscriptionID, resourceGroup, privateCloudName, clusterName, hostID string) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/hosts/%s", azureAVSClusterID(subscriptionID, resourceGroup, privateCloudName, clusterName), hostID),
		"name": hostID,
		"type": "Microsoft.AVS/privateClouds/clusters/hosts",
		"properties": map[string]any{
			"status":       "Connected",
			"serialNumber": "SN-" + hostID,
		},
	}
}

func azureAVSDatastoreFixture(resource *azureAVSDatastore) map[string]any {
	return map[string]any{
		"id":       azureAVSDatastoreID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.ClusterName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/clusters/datastores",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"netAppVolumeId":    "/subscriptions/" + resource.SubscriptionID + "/resourceGroups/" + resource.ResourceGroup + "/providers/Microsoft.NetApp/netAppAccounts/na-a/capacityPools/pool-a/volumes/vol-a",
		}, resource.Properties),
	}
}

func azureAVSPlacementPolicyFixture(resource *azureAVSPlacementPolicy) map[string]any {
	return map[string]any{
		"id":       azureAVSPlacementPolicyID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName, resource.ClusterName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.AVS/privateClouds/clusters/placementPolicies",
		"location": resource.Location,
		"tags":     azureAVSAnyStringMap(resource.Tags),
		"properties": azureAVSMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"affinityType":      "VmVm",
			"state":             "Enabled",
		}, resource.Properties),
	}
}

func azureAVSClusterVirtualMachineListFixture(subscriptionID, resourceGroup, privateCloudName, clusterName string) []map[string]any {
	return []map[string]any{
		azureAVSClusterVirtualMachineFixture(subscriptionID, resourceGroup, privateCloudName, clusterName, "vm-1"),
	}
}

func azureAVSClusterVirtualMachineFixture(subscriptionID, resourceGroup, privateCloudName, clusterName, vmID string) map[string]any {
	return map[string]any{
		"id":   azureAVSClusterVirtualMachineID(subscriptionID, resourceGroup, privateCloudName, clusterName, vmID),
		"name": vmID,
		"type": "Microsoft.AVS/privateClouds/clusters/virtualMachines",
		"properties": map[string]any{
			"powerState": "PoweredOn",
		},
	}
}

func azureAVSWorkloadNetworkRootFixture(subscriptionID, resourceGroup, privateCloudName string) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/workloadNetworks/default", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName)),
		"name": "default",
		"type": "Microsoft.AVS/privateClouds/workloadNetworks",
		"properties": map[string]any{
			"provisioningState": "Succeeded",
			"segmentCount":      2,
		},
	}
}

func azureAVSDNSZoneFixture(resource *azureAVSWorkloadNetworkDNSZone) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/workloadNetworks/default/dnsZones/%s", azureAVSPrivateCloudID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName), resource.Name),
		"name": resource.Name,
		"type": "Microsoft.AVS/privateClouds/workloadNetworks/dnsZones",
		"properties": azureAVSMergeObject(map[string]any{
			"domain":            resource.Name,
			"dnsServerIP":       "10.0.0.20",
			"provisioningState": "Succeeded",
		}, resource.Properties),
	}
}

func azureAVSVMGroupFixture(resource *azureAVSWorkloadNetworkVMGroup) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/workloadNetworks/default/vmGroups/%s", azureAVSPrivateCloudID(resource.SubscriptionID, resource.ResourceGroup, resource.PrivateCloudName), resource.Name),
		"name": resource.Name,
		"type": "Microsoft.AVS/privateClouds/workloadNetworks/vmGroups",
		"properties": azureAVSMergeObject(map[string]any{
			"displayName":       resource.Name,
			"members":           []string{"vm-1"},
			"provisioningState": "Succeeded",
		}, resource.Properties),
	}
}

func azureAVSWorkloadVirtualMachineListFixture(subscriptionID, resourceGroup, privateCloudName string) []map[string]any {
	return []map[string]any{
		azureAVSWorkloadVirtualMachineFixture(subscriptionID, resourceGroup, privateCloudName, "vm-1"),
	}
}

func azureAVSWorkloadVirtualMachineFixture(subscriptionID, resourceGroup, privateCloudName, vmID string) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/workloadNetworks/default/virtualMachines/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), vmID),
		"name": vmID,
		"type": "Microsoft.AVS/privateClouds/workloadNetworks/virtualMachines",
		"properties": map[string]any{
			"displayName": "app-" + vmID,
			"powerState":  "PoweredOn",
		},
	}
}

func azureAVSDefaultAddon(subscriptionID, resourceGroup, privateCloudName, name string) *azureAVSAddon {
	return &azureAVSAddon{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Location:         "eastus",
		Tags:             map[string]string{},
		Properties: map[string]any{
			"addonType":         "SRM",
			"status":            "Enabled",
			"provisioningState": "Succeeded",
		},
	}
}

func azureAVSDefaultMaintenance(subscriptionID, resourceGroup, privateCloudName, name string) *azureAVSMaintenance {
	return &azureAVSMaintenance{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		PrivateCloudName: privateCloudName,
		Name:             name,
		Properties: map[string]any{
			"provisioningState":  "Succeeded",
			"maintenanceState":   "Scheduled",
			"scheduledStartTime": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		},
	}
}

func azureAVSPrivateCloudKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + name)
}

func azureAVSAuthorizationKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSGlobalReachKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSLicenseKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSAddonKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSScriptExecutionKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSIscsiPathKey(subscriptionID, resourceGroup, privateCloudName string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/default")
}

func azureAVSClusterKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSDatastoreKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + clusterName + "/" + name)
}

func azureAVSPlacementPolicyKey(subscriptionID, resourceGroup, privateCloudName, clusterName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + clusterName + "/" + name)
}

func azureAVSDNSZoneKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSVMGroupKey(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + privateCloudName + "/" + name)
}

func azureAVSPrivateCloudID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.AVS/privateClouds/%s", subscriptionID, resourceGroup, name)
}

func azureAVSAuthorizationID(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return fmt.Sprintf("%s/authorizations/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), name)
}

func azureAVSGlobalReachID(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return fmt.Sprintf("%s/globalReachConnections/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), name)
}

func azureAVSLicenseID(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return fmt.Sprintf("%s/licenses/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), name)
}

func azureAVSAddonID(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return fmt.Sprintf("%s/addons/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), name)
}

func azureAVSMaintenanceID(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return fmt.Sprintf("%s/maintenances/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), name)
}

func azureAVSScriptExecutionID(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return fmt.Sprintf("%s/scriptExecutions/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), name)
}

func azureAVSIscsiPathID(subscriptionID, resourceGroup, privateCloudName string) string {
	return fmt.Sprintf("%s/iscsiPaths/default", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName))
}

func azureAVSClusterID(subscriptionID, resourceGroup, privateCloudName, name string) string {
	return fmt.Sprintf("%s/clusters/%s", azureAVSPrivateCloudID(subscriptionID, resourceGroup, privateCloudName), name)
}

func azureAVSDatastoreID(subscriptionID, resourceGroup, privateCloudName, clusterName, name string) string {
	return fmt.Sprintf("%s/datastores/%s", azureAVSClusterID(subscriptionID, resourceGroup, privateCloudName, clusterName), name)
}

func azureAVSPlacementPolicyID(subscriptionID, resourceGroup, privateCloudName, clusterName, name string) string {
	return fmt.Sprintf("%s/placementPolicies/%s", azureAVSClusterID(subscriptionID, resourceGroup, privateCloudName, clusterName), name)
}

func azureAVSClusterVirtualMachineID(subscriptionID, resourceGroup, privateCloudName, clusterName, vmID string) string {
	return fmt.Sprintf("%s/virtualMachines/%s", azureAVSClusterID(subscriptionID, resourceGroup, privateCloudName, clusterName), vmID)
}

func azureAVSHash(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "/")))
	return hex.EncodeToString(sum[:16])
}

func azureAVSString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func azureAVSObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if m, ok := value.(map[string]any); ok {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return map[string]any{}
}

func azureAVSStringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = azureAVSString(v)
	}
	return out
}

func azureAVSAnyStringMap(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for k, v := range value {
		out[k] = v
	}
	return out
}

func azureAVSMergeObject(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		if existing, ok := out[k].(map[string]any); ok {
			if incoming, ok := v.(map[string]any); ok {
				out[k] = azureAVSMergeObject(existing, incoming)
				continue
			}
		}
		out[k] = v
	}
	return out
}

func azureAVSCoalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
