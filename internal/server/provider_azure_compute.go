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

const azureComputeAPIVersion = "2025-04-01"

type azureComputeResource struct {
	SubscriptionID string
	ResourceGroup  string
	Segments       []string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	Identity       map[string]any
	SKU            map[string]any
	Plan           map[string]any
	Zones          []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

var azureComputeTopLevelCollections = map[string]string{
	"availabilitysets":          "availabilitySets",
	"proximityplacementgroups":  "proximityPlacementGroups",
	"sshpublickeys":             "sshPublicKeys",
	"capacityreservationgroups": "capacityReservationGroups",
	"hostgroups":                "hostGroups",
	"disks":                     "disks",
	"snapshots":                 "snapshots",
	"images":                    "images",
	"galleries":                 "galleries",
	"cloudservices":             "cloudServices",
	"virtualmachines":           "virtualMachines",
	"virtualmachinescalesets":   "virtualMachineScaleSets",
	"restorepointcollections":   "restorePointCollections",
}

var azureComputeNestedCollections = map[string]string{
	"capacityreservations": "capacityReservations",
	"hosts":                "hosts",
	"images":               "images",
	"applications":         "applications",
	"versions":             "versions",
	"extensions":           "extensions",
	"runcommands":          "runCommands",
	"virtualmachines":      "virtualMachines",
	"restorepoints":        "restorePoints",
}

func (s *Server) handleAzureComputeRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureComputeIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureComputeAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Compute") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeOperationsFixture()})
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureComputeRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "skus":
		if resourceGroup != "" || r.Method != http.MethodGet || len(tail) != 1 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeSKUsFixture()})
		return true
	case "locations":
		if resourceGroup != "" {
			respondAzureImplemented(w, path)
			return true
		}
		s.handleAzureComputeLocationRoute(w, r, path, subscriptionID, tail[1:])
		return true
	case "sharedgalleries":
		if resourceGroup != "" {
			respondAzureImplemented(w, path)
			return true
		}
		s.handleAzureComputeSharedGalleriesRoute(w, r, path, subscriptionID, tail[1:])
		return true
	}

	if canonical, ok := azureComputeCanonicalTopLevelCollection(tail[0]); ok {
		s.handleAzureComputeTopLevelCollectionRoute(w, r, path, subscriptionID, resourceGroup, canonical, tail)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func azureComputeIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Compute") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureComputeRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "skus", "locations", "sharedgalleries":
		return true
	default:
		_, found := azureComputeCanonicalTopLevelCollection(tail[0])
		return found
	}
}

func azureComputeAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureComputeRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
		for idx := 4; idx+1 < len(segments); idx++ {
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Compute") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Compute") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureComputeCanonicalTopLevelCollection(raw string) (string, bool) {
	canonical, ok := azureComputeTopLevelCollections[strings.ToLower(strings.TrimSpace(raw))]
	return canonical, ok
}

func azureComputeCanonicalNestedCollection(raw string) (string, bool) {
	canonical, ok := azureComputeNestedCollections[strings.ToLower(strings.TrimSpace(raw))]
	return canonical, ok
}

func (s *Server) handleAzureComputeLocationRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}
	location := strings.TrimSpace(tail[0])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "usages") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeUsagesFixture(location)})
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "vmSizes") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeVMSizesFixture(location)})
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "publishers") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputePublishersFixture(location)})
		return
	}
	if len(tail) >= 5 &&
		strings.EqualFold(tail[1], "publishers") &&
		strings.EqualFold(tail[3], "artifactTypes") &&
		strings.EqualFold(tail[4], "vmImage") {
		s.handleAzureComputeVMImageCatalogRoute(w, r, path, location, tail[2], tail[5:])
		return
	}
	if len(tail) >= 2 && strings.EqualFold(tail[1], "communityGalleries") {
		s.handleAzureComputeCommunityGalleriesRoute(w, r, path, subscriptionID, location, tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeVMImageCatalogRoute(w http.ResponseWriter, r *http.Request, path, location, publisher string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "offers") {
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeOffersFixture(location, publisher)})
		return
	}
	if len(tail) == 3 && strings.EqualFold(tail[0], "offers") && strings.EqualFold(tail[2], "skus") {
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeImageSKUsFixture(location, publisher, tail[1])})
		return
	}
	if len(tail) == 5 &&
		strings.EqualFold(tail[0], "offers") &&
		strings.EqualFold(tail[2], "skus") &&
		strings.EqualFold(tail[4], "versions") {
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeImageVersionsFixture(location, publisher, tail[1], tail[3])})
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeCommunityGalleriesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeCommunityGalleriesFixture(subscriptionID, location)})
		return
	}
	galleryName := strings.TrimSpace(tail[0])
	if galleryName == "" {
		respondAzureInvalidRequest(w, path, "community gallery name is required")
		return
	}
	if len(tail) == 1 {
		respondJSON(w, http.StatusOK, azureComputeCommunityGalleryFixture(subscriptionID, location, galleryName))
		return
	}
	if len(tail) == 3 && strings.EqualFold(tail[1], "images") {
		respondJSON(w, http.StatusOK, azureComputeCommunityGalleryImageFixture(subscriptionID, location, galleryName, tail[2]))
		return
	}
	if len(tail) == 5 && strings.EqualFold(tail[1], "images") && strings.EqualFold(tail[3], "versions") {
		respondJSON(w, http.StatusOK, azureComputeCommunityGalleryImageVersionFixture(subscriptionID, location, galleryName, tail[2], tail[4]))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeSharedGalleriesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeSharedGalleriesFixture(subscriptionID)})
		return
	}
	galleryName := strings.TrimSpace(tail[0])
	if galleryName == "" {
		respondAzureInvalidRequest(w, path, "shared gallery name is required")
		return
	}
	if len(tail) == 1 {
		respondJSON(w, http.StatusOK, azureComputeSharedGalleryFixture(subscriptionID, galleryName))
		return
	}
	if len(tail) == 3 && strings.EqualFold(tail[1], "images") {
		respondJSON(w, http.StatusOK, azureComputeSharedGalleryImageFixture(subscriptionID, galleryName, tail[2]))
		return
	}
	if len(tail) == 5 && strings.EqualFold(tail[1], "images") && strings.EqualFold(tail[3], "versions") {
		respondJSON(w, http.StatusOK, azureComputeSharedGalleryImageVersionFixture(subscriptionID, galleryName, tail[2], tail[4]))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeTopLevelCollectionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureComputeListTopLevel(w, subscriptionID, resourceGroup, collection)
		return
	}
	if resourceGroup == "" {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureComputeValidResourceName(name) {
		respondAzureInvalidRequest(w, path, collection+" name is invalid")
		return
	}
	resourceSegments := []string{collection, name}
	resourceKey := azureComputeResourceKey(subscriptionID, resourceGroup, resourceSegments)

	switch collection {
	case "capacityReservationGroups":
		if len(tail) >= 3 {
			if strings.EqualFold(tail[2], "capacityReservations") {
				s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, "capacityReservations", tail[3:])
				return
			}
			respondAzureImplemented(w, path)
			return
		}
	case "hostGroups":
		if len(tail) >= 3 {
			if strings.EqualFold(tail[2], "hosts") {
				s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, "hosts", tail[3:])
				return
			}
			respondAzureImplemented(w, path)
			return
		}
	case "galleries":
		if len(tail) >= 3 {
			switch strings.ToLower(strings.TrimSpace(tail[2])) {
			case "images":
				s.handleAzureComputeGalleryImagesRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, tail[3:])
				return
			case "applications":
				s.handleAzureComputeGalleryApplicationsRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, tail[3:])
				return
			default:
				respondAzureImplemented(w, path)
				return
			}
		}
	case "virtualMachines":
		if len(tail) >= 3 {
			s.handleAzureComputeVirtualMachineRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, tail[2:])
			return
		}
	case "virtualMachineScaleSets":
		if len(tail) >= 3 {
			s.handleAzureComputeScaleSetRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, tail[2:])
			return
		}
	case "restorePointCollections":
		if len(tail) >= 3 {
			if strings.EqualFold(tail[2], "restorePoints") {
				s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, "restorePoints", tail[3:])
				return
			}
			respondAzureImplemented(w, path)
			return
		}
	}

	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureComputeGetResource(w, path, subscriptionID, resourceGroup, resourceSegments, false)
	case http.MethodHead:
		s.handleAzureComputeGetResource(w, path, subscriptionID, resourceGroup, resourceSegments, true)
	case http.MethodPut:
		s.handleAzureComputeCreateOrUpdateResource(w, r, path, subscriptionID, resourceGroup, resourceSegments, resourceKey, "")
	case http.MethodPatch:
		s.handleAzureComputePatchResource(w, r, path, subscriptionID, resourceGroup, resourceSegments, resourceKey)
	case http.MethodDelete:
		s.handleAzureComputeDeleteResource(w, subscriptionID, resourceGroup, resourceSegments)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureComputeGalleryImagesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, gallerySegments []string, tail []string) {
	s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, gallerySegments, "images", tail)
}

func (s *Server) handleAzureComputeGalleryApplicationsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, gallerySegments []string, tail []string) {
	s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, gallerySegments, "applications", tail)
}

func (s *Server) handleAzureComputeVirtualMachineRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, vmSegments []string, tail []string) {
	if len(tail) == 1 {
		switch strings.ToLower(strings.TrimSpace(tail[0])) {
		case "start", "restart", "poweroff", "deallocate", "redeploy":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureComputeAction(w, path, subscriptionID, resourceGroup, vmSegments, tail[0], http.StatusAccepted)
			return
		case "instanceview":
			if r.Method != http.MethodGet && r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureComputeInstanceView(w, path, subscriptionID, resourceGroup, vmSegments)
			return
		}
	}
	if len(tail) >= 1 {
		switch strings.ToLower(strings.TrimSpace(tail[0])) {
		case "extensions":
			s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, vmSegments, "extensions", tail[1:])
			return
		case "runcommands":
			s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, vmSegments, "runCommands", tail[1:])
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeScaleSetRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, scaleSetSegments []string, tail []string) {
	if len(tail) == 1 {
		switch strings.ToLower(strings.TrimSpace(tail[0])) {
		case "start", "restart", "deallocate":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureComputeAction(w, path, subscriptionID, resourceGroup, scaleSetSegments, tail[0], http.StatusAccepted)
			return
		case "instanceview":
			if r.Method != http.MethodGet && r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureComputeInstanceView(w, path, subscriptionID, resourceGroup, scaleSetSegments)
			return
		}
	}
	if len(tail) >= 1 && strings.EqualFold(tail[0], "virtualMachines") {
		s.handleAzureComputeScaleSetVMsRoute(w, r, path, subscriptionID, resourceGroup, scaleSetSegments, tail[1:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeScaleSetVMsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, scaleSetSegments []string, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureComputeListChildren(w, subscriptionID, resourceGroup, scaleSetSegments, "virtualMachines")
		return
	}
	name := strings.TrimSpace(tail[0])
	if !azureComputeValidResourceName(name) {
		respondAzureInvalidRequest(w, path, "virtual machine scale set VM name is invalid")
		return
	}
	vmSegments := append(append([]string{}, scaleSetSegments...), "virtualMachines", name)
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureComputeGetResource(w, path, subscriptionID, resourceGroup, vmSegments, false)
		case http.MethodHead:
			s.handleAzureComputeGetResource(w, path, subscriptionID, resourceGroup, vmSegments, true)
		case http.MethodPut:
			s.handleAzureComputeCreateOrUpdateResource(w, r, path, subscriptionID, resourceGroup, vmSegments, azureComputeResourceKey(subscriptionID, resourceGroup, vmSegments), azureComputeResourceKey(subscriptionID, resourceGroup, scaleSetSegments))
		case http.MethodPatch:
			s.handleAzureComputePatchResource(w, r, path, subscriptionID, resourceGroup, vmSegments, azureComputeResourceKey(subscriptionID, resourceGroup, vmSegments))
		case http.MethodDelete:
			s.handleAzureComputeDeleteResource(w, subscriptionID, resourceGroup, vmSegments)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 2 {
		switch strings.ToLower(strings.TrimSpace(tail[1])) {
		case "instanceview":
			if r.Method != http.MethodGet && r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureComputeInstanceView(w, path, subscriptionID, resourceGroup, vmSegments)
			return
		case "restart", "deallocate":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			s.handleAzureComputeAction(w, path, subscriptionID, resourceGroup, vmSegments, tail[1], http.StatusAccepted)
			return
		}
	}
	if len(tail) >= 2 && strings.EqualFold(tail[1], "extensions") {
		s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, vmSegments, "extensions", tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeChildCollectionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, parentSegments []string, collection string, tail []string) {
	parentKey := azureComputeResourceKey(subscriptionID, resourceGroup, parentSegments)
	if !s.azureComputeResourceExists(parentKey) {
		respondJSON(w, http.StatusNotFound, azureComputeNotFound(path, parentSegments[len(parentSegments)-1]))
		return
	}

	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureComputeListChildren(w, subscriptionID, resourceGroup, parentSegments, collection)
		return
	}

	name := strings.TrimSpace(tail[0])
	if !azureComputeValidResourceName(name) {
		respondAzureInvalidRequest(w, path, collection+" name is invalid")
		return
	}
	resourceSegments := append(append([]string{}, parentSegments...), collection, name)
	resourceKey := azureComputeResourceKey(subscriptionID, resourceGroup, resourceSegments)

	if collection == "images" || collection == "applications" {
		if len(tail) >= 2 {
			if strings.EqualFold(tail[1], "versions") {
				s.handleAzureComputeChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, resourceSegments, "versions", tail[2:])
				return
			}
		}
	}

	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureComputeGetResource(w, path, subscriptionID, resourceGroup, resourceSegments, false)
	case http.MethodHead:
		s.handleAzureComputeGetResource(w, path, subscriptionID, resourceGroup, resourceSegments, true)
	case http.MethodPut:
		s.handleAzureComputeCreateOrUpdateResource(w, r, path, subscriptionID, resourceGroup, resourceSegments, resourceKey, parentKey)
	case http.MethodPatch:
		s.handleAzureComputePatchResource(w, r, path, subscriptionID, resourceGroup, resourceSegments, resourceKey)
	case http.MethodDelete:
		s.handleAzureComputeDeleteResource(w, subscriptionID, resourceGroup, resourceSegments)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) azureComputeResourceExists(key string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	_, ok := s.azureComputeResources[key]
	return ok
}

func (s *Server) handleAzureComputeListTopLevel(w http.ResponseWriter, subscriptionID, resourceGroup, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, resource := range s.azureComputeResources {
		if len(resource.Segments) != 2 || !strings.EqualFold(resource.Segments[0], collection) {
			continue
		}
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) {
			continue
		}
		if resourceGroup != "" && !strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			continue
		}
		items = append(items, azureComputeResourceFixture(resource))
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureComputeString(items[i]["id"])), strings.ToLower(azureComputeString(items[j]["id"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureComputeListChildren(w http.ResponseWriter, subscriptionID, resourceGroup string, parentSegments []string, collection string) {
	parentKey := azureComputeResourceKey(subscriptionID, resourceGroup, parentSegments)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, resource := range s.azureComputeResources {
		if len(resource.Segments) != len(parentSegments)+2 {
			continue
		}
		if !strings.EqualFold(resource.Segments[len(resource.Segments)-2], collection) {
			continue
		}
		if !strings.EqualFold(azureComputeResourceKey(resource.SubscriptionID, resource.ResourceGroup, resource.Segments[:len(resource.Segments)-2]), parentKey) {
			continue
		}
		items = append(items, azureComputeResourceFixture(resource))
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureComputeString(items[i]["id"])), strings.ToLower(azureComputeString(items[j]["id"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureComputeGetResource(w http.ResponseWriter, path, subscriptionID, resourceGroup string, segments []string, headOnly bool) {
	key := azureComputeResourceKey(subscriptionID, resourceGroup, segments)
	s.providerStorageMu.Lock()
	resource := s.azureComputeResources[key]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureComputeNotFound(path, segments[len(segments)-1]))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureComputeResourceFixture(resource))
}

func (s *Server) handleAzureComputeCreateOrUpdateResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, segments []string, key, parentKey string) {
	body, payload, ok := azureComputeReadRequestBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()

	location := strings.TrimSpace(azureComputeString(payload["location"]))
	properties := azureComputeObject(payload["properties"])
	if azureComputeCollectionRequiresLocation(segments) && location == "" {
		if parentKey == "" {
			respondAzureInvalidRequest(w, path, "location is required")
			return
		}
		s.providerStorageMu.Lock()
		parent := s.azureComputeResources[parentKey]
		s.providerStorageMu.Unlock()
		if parent == nil {
			respondJSON(w, http.StatusNotFound, azureComputeNotFound(path, segments[len(segments)-3]))
			return
		}
		location = parent.Location
	}
	if azureComputeCollectionRequiresProperties(segments) && len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}

	if parentKey != "" {
		s.providerStorageMu.Lock()
		parent := s.azureComputeResources[parentKey]
		s.providerStorageMu.Unlock()
		if parent == nil {
			respondJSON(w, http.StatusNotFound, azureComputeNotFound(path, segments[len(segments)-3]))
			return
		}
		if location == "" {
			location = parent.Location
		}
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureComputeResources[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		resource = &azureComputeResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Segments:       append([]string{}, segments...),
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
	}

	resource.Location = location
	resource.Tags = azureComputeStringMap(payload["tags"])
	resource.Properties = azureComputeNormalizeProperties(segments, properties, now)
	resource.Identity = azureComputeObject(payload["identity"])
	resource.SKU = azureComputeObject(payload["sku"])
	resource.Plan = azureComputeObject(payload["plan"])
	resource.Zones = azureComputeStringSlice(payload["zones"])
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureComputeETag(key, body, now)
	s.azureComputeResources[key] = resource

	respondJSON(w, status, azureComputeResourceFixture(resource))
}

func (s *Server) handleAzureComputePatchResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, segments []string, key string) {
	body, payload, ok := azureComputeReadRequestBody(w, r, path)
	if !ok {
		return
	}
	_ = body
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureComputeResources[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureComputeNotFound(path, segments[len(segments)-1]))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	if tags, ok := payload["tags"]; ok {
		resource.Tags = azureComputeStringMap(tags)
	}
	if identity, ok := payload["identity"]; ok {
		resource.Identity = azureComputeObject(identity)
	}
	if sku, ok := payload["sku"]; ok {
		resource.SKU = azureComputeObject(sku)
	}
	if plan, ok := payload["plan"]; ok {
		resource.Plan = azureComputeObject(plan)
	}
	if zones, ok := payload["zones"]; ok {
		resource.Zones = azureComputeStringSlice(zones)
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureComputeMergeObjects(resource.Properties, azureComputeObject(properties))
		resource.Properties = azureComputeNormalizeProperties(segments, resource.Properties, now)
	}
	resource.UpdatedAt = now
	resource.ETag = azureComputeETag(key, []byte(now.Format(time.RFC3339Nano)), now)

	respondJSON(w, http.StatusOK, azureComputeResourceFixture(resource))
}

func (s *Server) handleAzureComputeDeleteResource(w http.ResponseWriter, subscriptionID, resourceGroup string, segments []string) {
	key := azureComputeResourceKey(subscriptionID, resourceGroup, segments)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	delete(s.azureComputeResources, key)
	for childKey, resource := range s.azureComputeResources {
		if len(resource.Segments) <= len(segments) {
			continue
		}
		if strings.EqualFold(azureComputeResourceKey(resource.SubscriptionID, resource.ResourceGroup, resource.Segments[:len(segments)]), key) {
			delete(s.azureComputeResources, childKey)
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": "Deleted"})
}

func (s *Server) handleAzureComputeAction(w http.ResponseWriter, path, subscriptionID, resourceGroup string, segments []string, action string, status int) {
	key := azureComputeResourceKey(subscriptionID, resourceGroup, segments)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	resource := s.azureComputeResources[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureComputeNotFound(path, segments[len(segments)-1]))
		return
	}
	properties := azureComputeCloneObject(resource.Properties)
	powerState := "running"
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "deallocate":
		powerState = "deallocated"
	case "poweroff":
		powerState = "stopped"
	case "redeploy", "restart", "start":
		powerState = "running"
	}
	properties["powerState"] = powerState
	properties["provisioningState"] = "Succeeded"
	properties["lastAction"] = action
	resource.Properties = azureComputeNormalizeProperties(segments, properties, time.Now().UTC())
	resource.UpdatedAt = time.Now().UTC()
	respondJSON(w, status, map[string]any{
		"status":   "Accepted",
		"action":   action,
		"resource": azureComputeResourceFixture(resource),
	})
}

func (s *Server) handleAzureComputeInstanceView(w http.ResponseWriter, path, subscriptionID, resourceGroup string, segments []string) {
	key := azureComputeResourceKey(subscriptionID, resourceGroup, segments)
	s.providerStorageMu.Lock()
	resource := s.azureComputeResources[key]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureComputeNotFound(path, segments[len(segments)-1]))
		return
	}
	powerState := azureComputeString(resource.Properties["powerState"])
	if powerState == "" {
		powerState = "running"
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"statuses": []map[string]any{
			{
				"code":          "ProvisioningState/succeeded",
				"displayStatus": "Provisioning succeeded",
				"level":         "Info",
			},
			{
				"code":          "PowerState/" + powerState,
				"displayStatus": strings.Title(powerState),
				"level":         "Info",
			},
		},
	})
}

func azureComputeCollectionRequiresLocation(segments []string) bool {
	switch azureComputeLastCollection(segments) {
	case "extensions", "runCommands", "restorePoints", "versions":
		return false
	case "virtualMachines":
		return len(segments) == 2
	default:
		return true
	}
}

func azureComputeCollectionRequiresProperties(segments []string) bool {
	switch azureComputeLastCollection(segments) {
	case "virtualMachines", "virtualMachineScaleSets", "disks", "snapshots", "images", "cloudServices", "restorePoints", "extensions", "runCommands", "capacityReservations", "hosts", "versions":
		return true
	default:
		return false
	}
}

func azureComputeLastCollection(segments []string) string {
	if len(segments) == 0 {
		return ""
	}
	if len(segments)%2 == 0 {
		return segments[len(segments)-2]
	}
	return segments[len(segments)-1]
}

func azureComputeResourceKey(subscriptionID, resourceGroup string, segments []string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID) + "|" + strings.TrimSpace(resourceGroup) + "|" + strings.Join(segments, "/"))
}

func azureComputeResourceID(subscriptionID, resourceGroup string, segments []string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/" + strings.Join(segments, "/")
}

func azureComputeResourceType(segments []string) string {
	parts := make([]string, 0, len(segments)/2)
	for idx := 0; idx < len(segments); idx += 2 {
		parts = append(parts, azureComputeCanonicalCollectionSegment(segments[idx]))
	}
	return "Microsoft.Compute/" + strings.Join(parts, "/")
}

func azureComputeCanonicalCollectionSegment(raw string) string {
	if value, ok := azureComputeCanonicalTopLevelCollection(raw); ok {
		return value
	}
	if value, ok := azureComputeCanonicalNestedCollection(raw); ok {
		return value
	}
	return raw
}

func azureComputeResourceFixture(resource *azureComputeResource) map[string]any {
	out := map[string]any{
		"id":         azureComputeResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Segments),
		"name":       resource.Segments[len(resource.Segments)-1],
		"type":       azureComputeResourceType(resource.Segments),
		"location":   resource.Location,
		"tags":       resource.Tags,
		"properties": azureComputeCloneObject(resource.Properties),
		"etag":       resource.ETag,
		"systemData": map[string]any{
			"createdAt":          resource.CreatedAt.Format(time.RFC3339),
			"lastModifiedAt":     resource.UpdatedAt.Format(time.RFC3339),
			"createdByType":      "Application",
			"lastModifiedByType": "Application",
		},
	}
	if len(resource.Identity) > 0 {
		out["identity"] = azureComputeCloneObject(resource.Identity)
	}
	if len(resource.SKU) > 0 {
		out["sku"] = azureComputeCloneObject(resource.SKU)
	}
	if len(resource.Plan) > 0 {
		out["plan"] = azureComputeCloneObject(resource.Plan)
	}
	if len(resource.Zones) > 0 {
		out["zones"] = append([]string{}, resource.Zones...)
	}
	return out
}

func azureComputeNormalizeProperties(segments []string, properties map[string]any, now time.Time) map[string]any {
	out := azureComputeCloneObject(properties)
	if out == nil {
		out = map[string]any{}
	}
	if strings.TrimSpace(azureComputeString(out["provisioningState"])) == "" {
		out["provisioningState"] = "Succeeded"
	}
	switch azureComputeLastCollection(segments) {
	case "virtualMachines", "virtualMachineScaleSets":
		if strings.TrimSpace(azureComputeString(out["powerState"])) == "" {
			out["powerState"] = "running"
		}
		if strings.TrimSpace(azureComputeString(out["vmId"])) == "" {
			out["vmId"] = fmt.Sprintf("%x", sha1.Sum([]byte(strings.Join(segments, "/"))))
		}
	case "disks":
		if strings.TrimSpace(azureComputeString(out["diskState"])) == "" {
			out["diskState"] = "Unattached"
		}
	case "snapshots":
		if strings.TrimSpace(azureComputeString(out["completionPercent"])) == "" {
			out["completionPercent"] = 100
		}
	case "extensions":
		out["instanceView"] = map[string]any{
			"statuses": []map[string]any{{
				"code":          "ProvisioningState/succeeded",
				"displayStatus": "Provisioning succeeded",
				"level":         "Info",
			}},
		}
	case "runCommands":
		if _, ok := out["asyncExecution"]; !ok {
			out["asyncExecution"] = false
		}
	case "versions":
		if _, ok := out["publishingProfile"]; !ok {
			out["publishingProfile"] = map[string]any{
				"publishedDate": now.Format(time.RFC3339),
				"endOfLifeDate": now.Add(365 * 24 * time.Hour).Format(time.RFC3339),
			}
		}
	case "restorePoints":
		if _, ok := out["sourceMetadata"]; !ok {
			out["sourceMetadata"] = map[string]any{"createdAt": now.Format(time.RFC3339)}
		}
	}
	return out
}

func azureComputeNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("Compute resource %q was not found for path %s", name, path),
		},
	}
}

func azureComputeReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	return body, payload, true
}

func azureComputeString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func azureComputeObject(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return azureComputeCloneObject(m)
	}
	return map[string]any{}
}

func azureComputeCloneObject(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureComputeCloneObject(typed)
		case []any:
			out[key] = azureComputeCloneList(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func azureComputeCloneList(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		switch typed := item.(type) {
		case map[string]any:
			out = append(out, azureComputeCloneObject(typed))
		case []any:
			out = append(out, azureComputeCloneList(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}

func azureComputeMergeObjects(base, patch map[string]any) map[string]any {
	if len(base) == 0 {
		return azureComputeCloneObject(patch)
	}
	out := azureComputeCloneObject(base)
	for key, value := range patch {
		if nestedBase, ok := out[key].(map[string]any); ok {
			if nestedPatch, ok := value.(map[string]any); ok {
				out[key] = azureComputeMergeObjects(nestedBase, nestedPatch)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func azureComputeStringMap(v any) map[string]string {
	out := map[string]string{}
	if raw, ok := v.(map[string]any); ok {
		for key, value := range raw {
			if s, ok := value.(string); ok {
				out[key] = s
			}
		}
	}
	return out
}

func azureComputeStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func azureComputeETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+now.Format(time.RFC3339Nano)+"|"), body...))
	return `W/"` + hex.EncodeToString(sum[:]) + `"`
}

func azureComputeValidResourceName(name string) bool {
	name = strings.TrimSpace(name)
	return len(name) >= 1 && len(name) <= 80 && !strings.Contains(name, "/")
}

func azureComputeOperationsFixture() []map[string]any {
	return []map[string]any{
		{
			"name": "Microsoft.Compute/operations/read",
			"display": map[string]any{
				"provider":  "Microsoft Compute",
				"resource":  "Operations",
				"operation": "List compute operations",
			},
		},
		{
			"name": "Microsoft.Compute/virtualMachines/read",
			"display": map[string]any{
				"provider":  "Microsoft Compute",
				"resource":  "Virtual Machines",
				"operation": "Get virtual machine",
			},
		},
		{
			"name": "Microsoft.Compute/virtualMachineScaleSets/write",
			"display": map[string]any{
				"provider":  "Microsoft Compute",
				"resource":  "Virtual Machine Scale Sets",
				"operation": "Create or update virtual machine scale set",
			},
		},
		{
			"name": "Microsoft.Compute/disks/write",
			"display": map[string]any{
				"provider":  "Microsoft Compute",
				"resource":  "Disks",
				"operation": "Create or update disk",
			},
		},
	}
}

func azureComputeUsagesFixture(location string) []map[string]any {
	return []map[string]any{
		{
			"name": map[string]any{
				"value":          "cores",
				"localizedValue": "Total Regional vCPUs",
			},
			"unit":         "Count",
			"currentValue": 8,
			"limit":        100,
			"location":     location,
		},
		{
			"name": map[string]any{
				"value":          "virtualMachines",
				"localizedValue": "Virtual Machines",
			},
			"unit":         "Count",
			"currentValue": 2,
			"limit":        25000,
			"location":     location,
		},
	}
}

func azureComputeSKUsFixture() []map[string]any {
	return []map[string]any{
		{
			"resourceType": "virtualMachines",
			"name":         "Standard_DS1_v2",
			"tier":         "Standard",
			"size":         "DS1_v2",
			"family":       "standardDSv2Family",
			"locations":    []string{"eastus", "westus2"},
			"capabilities": []map[string]any{
				{"name": "vCPUs", "value": "1"},
				{"name": "MemoryGB", "value": "3.5"},
			},
		},
		{
			"resourceType": "disks",
			"name":         "Premium_LRS",
			"tier":         "Premium",
			"size":         "P10",
			"family":       "diskFamily",
			"locations":    []string{"eastus", "westus2"},
		},
	}
}

func azureComputeVMSizesFixture(location string) []map[string]any {
	return []map[string]any{
		{
			"name":                 "Standard_DS1_v2",
			"numberOfCores":        1,
			"memoryInMB":           3584,
			"maxDataDiskCount":     4,
			"osDiskSizeInMB":       1047552,
			"resourceDiskSizeInMB": 7168,
			"location":             location,
		},
		{
			"name":                 "Standard_D2s_v3",
			"numberOfCores":        2,
			"memoryInMB":           8192,
			"maxDataDiskCount":     8,
			"osDiskSizeInMB":       1047552,
			"resourceDiskSizeInMB": 16384,
			"location":             location,
		},
	}
}

func azureComputePublishersFixture(location string) []map[string]any {
	return []map[string]any{
		{"name": "Canonical", "location": location},
		{"name": "MicrosoftWindowsServer", "location": location},
	}
}

func azureComputeOffersFixture(location, publisher string) []map[string]any {
	return []map[string]any{
		{"name": "0001-com-ubuntu-server-jammy", "location": location, "publisher": publisher},
		{"name": "WindowsServer", "location": location, "publisher": publisher},
	}
}

func azureComputeImageSKUsFixture(location, publisher, offer string) []map[string]any {
	return []map[string]any{
		{"name": "22_04-lts-gen2", "location": location, "publisher": publisher, "offer": offer},
		{"name": "2022-datacenter-azure-edition", "location": location, "publisher": publisher, "offer": offer},
	}
}

func azureComputeImageVersionsFixture(location, publisher, offer, sku string) []map[string]any {
	return []map[string]any{
		{"name": "latest", "location": location, "publisher": publisher, "offer": offer, "sku": sku},
		{"name": "1.0.0", "location": location, "publisher": publisher, "offer": offer, "sku": sku},
	}
}

func azureComputeCommunityGalleriesFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		azureComputeCommunityGalleryFixture(subscriptionID, location, "community-gallery-a"),
	}
}

func azureComputeCommunityGalleryFixture(subscriptionID, location, galleryName string) map[string]any {
	return map[string]any{
		"id":       "/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/locations/" + location + "/communityGalleries/" + galleryName,
		"name":     galleryName,
		"type":     "Microsoft.Compute/communityGalleries",
		"location": location,
		"properties": map[string]any{
			"identifier": map[string]any{"uniqueId": "community-" + galleryName},
			"sharingProfile": map[string]any{
				"permissions": "Groups",
			},
		},
	}
}

func azureComputeCommunityGalleryImageFixture(subscriptionID, location, galleryName, imageName string) map[string]any {
	return map[string]any{
		"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/locations/" + location + "/communityGalleries/" + galleryName + "/images/" + imageName,
		"name": imageName,
		"type": "Microsoft.Compute/communityGalleries/images",
		"properties": map[string]any{
			"identifier": map[string]any{
				"publisher": "stackyard",
				"offer":     galleryName,
				"sku":       imageName,
			},
		},
	}
}

func azureComputeCommunityGalleryImageVersionFixture(subscriptionID, location, galleryName, imageName, version string) map[string]any {
	return map[string]any{
		"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/locations/" + location + "/communityGalleries/" + galleryName + "/images/" + imageName + "/versions/" + version,
		"name": version,
		"type": "Microsoft.Compute/communityGalleries/images/versions",
		"properties": map[string]any{
			"publishingProfile": map[string]any{
				"publishedDate": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
}

func azureComputeSharedGalleriesFixture(subscriptionID string) []map[string]any {
	return []map[string]any{
		azureComputeSharedGalleryFixture(subscriptionID, "shared-gallery-a"),
	}
}

func azureComputeSharedGalleryFixture(subscriptionID, galleryName string) map[string]any {
	return map[string]any{
		"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/sharedGalleries/" + galleryName,
		"name": galleryName,
		"type": "Microsoft.Compute/sharedGalleries",
		"properties": map[string]any{
			"uniqueId": "shared-" + galleryName,
		},
	}
}

func azureComputeSharedGalleryImageFixture(subscriptionID, galleryName, imageName string) map[string]any {
	return map[string]any{
		"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/sharedGalleries/" + galleryName + "/images/" + imageName,
		"name": imageName,
		"type": "Microsoft.Compute/sharedGalleries/images",
		"properties": map[string]any{
			"identifier": map[string]any{
				"publisher": "stackyard",
				"offer":     galleryName,
				"sku":       imageName,
			},
		},
	}
}

func azureComputeSharedGalleryImageVersionFixture(subscriptionID, galleryName, imageName, version string) map[string]any {
	return map[string]any{
		"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Compute/sharedGalleries/" + galleryName + "/images/" + imageName + "/versions/" + version,
		"name": version,
		"type": "Microsoft.Compute/sharedGalleries/images/versions",
		"properties": map[string]any{
			"publishingProfile": map[string]any{
				"publishedDate": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
}
