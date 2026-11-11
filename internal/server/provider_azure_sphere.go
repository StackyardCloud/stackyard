package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureSphereAPIVersion = "2024-04-01"

type azureSphereCatalog struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureSphereProduct struct {
	SubscriptionID string
	ResourceGroup  string
	CatalogName    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureSphereDeviceGroup struct {
	SubscriptionID string
	ResourceGroup  string
	CatalogName    string
	ProductName    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureSphereDevice struct {
	SubscriptionID string
	ResourceGroup  string
	CatalogName    string
	ProductName    string
	DeviceGroup    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureSphereDeployment struct {
	SubscriptionID string
	ResourceGroup  string
	CatalogName    string
	ProductName    string
	DeviceGroup    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureSphereImage struct {
	SubscriptionID string
	ResourceGroup  string
	CatalogName    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

func (s *Server) handleAzureSphereRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsSpherePath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureSphereAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureSphere") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureSphereOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureSphereRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	if !strings.EqualFold(tail[0], "catalogs") {
		return false
	}
	s.handleAzureSphereCatalogsRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsSpherePath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureSphere") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureSphereRouteContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "catalogs")
}

func azureSphereAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureSphereRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AzureSphere") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AzureSphere") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureSphereCatalogsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureSphereCatalogListBySubscription(w, r, subscriptionID)
			return
		}
		s.handleAzureSphereCatalogListByResourceGroup(w, r, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	catalogName := strings.TrimSpace(tail[1])
	if catalogName == "" {
		respondAzureInvalidRequest(w, path, "catalog name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSphereCatalogGet(w, path, subscriptionID, resourceGroup, catalogName, false)
		case http.MethodHead:
			s.handleAzureSphereCatalogGet(w, path, subscriptionID, resourceGroup, catalogName, true)
		case http.MethodPut:
			s.handleAzureSphereCatalogCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, catalogName)
		case http.MethodPatch:
			s.handleAzureSphereCatalogPatch(w, r, path, subscriptionID, resourceGroup, catalogName)
		case http.MethodDelete:
			s.handleAzureSphereCatalogDelete(w, subscriptionID, resourceGroup, catalogName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "products":
		s.handleAzureSphereProductsRoute(w, r, path, subscriptionID, resourceGroup, catalogName, tail[2:])
	case "images":
		s.handleAzureSphereImagesRoute(w, r, path, subscriptionID, resourceGroup, catalogName, tail[2:])
	case "listdevices":
		if r.Method == http.MethodPost {
			s.handleAzureSphereCatalogListDevices(w, r, subscriptionID, resourceGroup, catalogName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "listdevicegroups":
		if r.Method == http.MethodPost {
			s.handleAzureSphereCatalogListDeviceGroups(w, r, subscriptionID, resourceGroup, catalogName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "listdeployments":
		if r.Method == http.MethodPost {
			s.handleAzureSphereCatalogListDeployments(w, r, subscriptionID, resourceGroup, catalogName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "listdeviceinsights":
		if r.Method == http.MethodPost {
			s.handleAzureSphereCatalogListDeviceInsights(w, r, subscriptionID, resourceGroup, catalogName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "uploadimage":
		if r.Method == http.MethodPost {
			s.handleAzureSphereCatalogUploadImage(w, r, path, subscriptionID, resourceGroup, catalogName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureSphereProductsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureSphereProductListByCatalog(w, r, subscriptionID, resourceGroup, catalogName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "product name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSphereProductGet(w, path, subscriptionID, resourceGroup, catalogName, name, false)
		case http.MethodHead:
			s.handleAzureSphereProductGet(w, path, subscriptionID, resourceGroup, catalogName, name, true)
		case http.MethodPut:
			s.handleAzureSphereProductCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, catalogName, name)
		case http.MethodPatch:
			s.handleAzureSphereProductPatch(w, r, path, subscriptionID, resourceGroup, catalogName, name)
		case http.MethodDelete:
			s.handleAzureSphereProductDelete(w, subscriptionID, resourceGroup, catalogName, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "devicegroups":
		s.handleAzureSphereDeviceGroupsRoute(w, r, path, subscriptionID, resourceGroup, catalogName, name, tail[2:])
	case "countdevices":
		if r.Method == http.MethodPost {
			s.handleAzureSphereProductCountDevices(w, subscriptionID, resourceGroup, catalogName, name)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "generatedefaultdevicegroups":
		if r.Method == http.MethodPost {
			s.handleAzureSphereProductGenerateDefaultDeviceGroups(w, subscriptionID, resourceGroup, catalogName, name)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureSphereDeviceGroupsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureSphereDeviceGroupListByProduct(w, r, subscriptionID, resourceGroup, catalogName, productName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "device group name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSphereDeviceGroupGet(w, path, subscriptionID, resourceGroup, catalogName, productName, name, false)
		case http.MethodHead:
			s.handleAzureSphereDeviceGroupGet(w, path, subscriptionID, resourceGroup, catalogName, productName, name, true)
		case http.MethodPut:
			s.handleAzureSphereDeviceGroupCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, catalogName, productName, name)
		case http.MethodPatch:
			s.handleAzureSphereDeviceGroupPatch(w, r, path, subscriptionID, resourceGroup, catalogName, productName, name)
		case http.MethodDelete:
			s.handleAzureSphereDeviceGroupDelete(w, subscriptionID, resourceGroup, catalogName, productName, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "devices":
		s.handleAzureSphereDevicesRoute(w, r, path, subscriptionID, resourceGroup, catalogName, productName, name, tail[2:])
	case "deployments":
		s.handleAzureSphereDeploymentsRoute(w, r, path, subscriptionID, resourceGroup, catalogName, productName, name, tail[2:])
	case "countdevices":
		if r.Method == http.MethodPost {
			s.handleAzureSphereDeviceGroupCountDevices(w, subscriptionID, resourceGroup, catalogName, productName, name)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "claimdevices":
		if r.Method == http.MethodPost {
			s.handleAzureSphereDeviceGroupClaimDevices(w, r, path, subscriptionID, resourceGroup, catalogName, productName, name)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureSphereDevicesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureSphereDeviceListByDeviceGroup(w, r, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "device name is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureSphereDeviceGet(w, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name, false)
	case http.MethodHead:
		s.handleAzureSphereDeviceGet(w, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name, true)
	case http.MethodPut:
		s.handleAzureSphereDeviceCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	case http.MethodPatch:
		s.handleAzureSphereDevicePatch(w, r, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	case http.MethodDelete:
		s.handleAzureSphereDeviceDelete(w, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureSphereDeploymentsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureSphereDeploymentListByDeviceGroup(w, r, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "deployment name is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureSphereDeploymentGet(w, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name, false)
	case http.MethodHead:
		s.handleAzureSphereDeploymentGet(w, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name, true)
	case http.MethodPut:
		s.handleAzureSphereDeploymentCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	case http.MethodDelete:
		s.handleAzureSphereDeploymentDelete(w, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureSphereImagesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureSphereImageListByCatalog(w, r, subscriptionID, resourceGroup, catalogName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "image name is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureSphereImageGet(w, path, subscriptionID, resourceGroup, catalogName, name, false)
	case http.MethodHead:
		s.handleAzureSphereImageGet(w, path, subscriptionID, resourceGroup, catalogName, name, true)
	case http.MethodPut:
		s.handleAzureSphereImageCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, catalogName, name)
	case http.MethodDelete:
		s.handleAzureSphereImageDelete(w, subscriptionID, resourceGroup, catalogName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureSphereCatalogListBySubscription(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, catalog := range s.azureSphereCatalogs {
		if strings.EqualFold(catalog.SubscriptionID, subscriptionID) {
			items = append(items, azureSphereCatalogFixture(catalog))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereCatalogListByResourceGroup(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, catalog := range s.azureSphereCatalogs {
		if strings.EqualFold(catalog.SubscriptionID, subscriptionID) && strings.EqualFold(catalog.ResourceGroup, resourceGroup) {
			items = append(items, azureSphereCatalogFixture(catalog))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereCatalogGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	catalog, ok := s.azureSphereCatalogs[azureSphereCatalogKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSphereCatalogFixture(catalog))
}

func (s *Server) handleAzureSphereCatalogCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureSphereString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureSphereCatalogKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()
	status := http.StatusCreated
	catalog, exists := s.azureSphereCatalogs[key]
	if !exists {
		catalog = &azureSphereCatalog{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			CreatedAt:      now,
		}
		s.azureSphereCatalogs[key] = catalog
	} else {
		status = http.StatusOK
	}
	catalog.Location = location
	catalog.Tags = azureSphereStringMap(payload["tags"])
	catalog.Properties = azureSphereNormalizeCatalogProperties(name, subscriptionID, resourceGroup, azureSphereObject(payload["properties"]), catalog.CreatedAt, now)
	catalog.UpdatedAt = now
	catalog.ETag = azureSphereStableTag("catalog", subscriptionID, resourceGroup, name, now)
	respondJSON(w, status, azureSphereCatalogFixture(catalog))
}

func (s *Server) handleAzureSphereCatalogPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	catalog, exists := s.azureSphereCatalogs[azureSphereCatalogKey(subscriptionID, resourceGroup, name)]
	if !exists {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}

	if location := strings.TrimSpace(azureSphereString(payload["location"])); location != "" {
		catalog.Location = location
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		catalog.Tags = azureSphereStringMap(tags)
	}
	props := azureSphereObject(payload["properties"])
	catalog.Properties = azureSphereNormalizeCatalogProperties(name, subscriptionID, resourceGroup, azureSphereMergeObjects(catalog.Properties, props), catalog.CreatedAt, time.Now().UTC())
	catalog.UpdatedAt = time.Now().UTC()
	catalog.ETag = azureSphereStableTag("catalog", subscriptionID, resourceGroup, name, catalog.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSphereCatalogFixture(catalog))
}

func (s *Server) handleAzureSphereCatalogDelete(w http.ResponseWriter, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureSphereCatalogKey(subscriptionID, resourceGroup, name)
	if _, ok := s.azureSphereCatalogs[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSphereCatalogs, key)
	for id, product := range s.azureSphereProducts {
		if strings.EqualFold(product.SubscriptionID, subscriptionID) && strings.EqualFold(product.ResourceGroup, resourceGroup) && strings.EqualFold(product.CatalogName, name) {
			delete(s.azureSphereProducts, id)
		}
	}
	for id, group := range s.azureSphereDeviceGroups {
		if strings.EqualFold(group.SubscriptionID, subscriptionID) && strings.EqualFold(group.ResourceGroup, resourceGroup) && strings.EqualFold(group.CatalogName, name) {
			delete(s.azureSphereDeviceGroups, id)
		}
	}
	for id, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, name) {
			delete(s.azureSphereDevices, id)
		}
	}
	for id, deployment := range s.azureSphereDeployments {
		if strings.EqualFold(deployment.SubscriptionID, subscriptionID) && strings.EqualFold(deployment.ResourceGroup, resourceGroup) && strings.EqualFold(deployment.CatalogName, name) {
			delete(s.azureSphereDeployments, id)
		}
	}
	for id, image := range s.azureSphereImages {
		if strings.EqualFold(image.SubscriptionID, subscriptionID) && strings.EqualFold(image.ResourceGroup, resourceGroup) && strings.EqualFold(image.CatalogName, name) {
			delete(s.azureSphereImages, id)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSphereProductListByCatalog(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSphereCatalogs[azureSphereCatalogKey(subscriptionID, resourceGroup, catalogName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(rawRequestPath(r), catalogName))
		return
	}
	items := make([]map[string]any, 0)
	for _, product := range s.azureSphereProducts {
		if strings.EqualFold(product.SubscriptionID, subscriptionID) && strings.EqualFold(product.ResourceGroup, resourceGroup) && strings.EqualFold(product.CatalogName, catalogName) {
			items = append(items, azureSphereProductFixture(product))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereProductGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, catalogName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	product, ok := s.azureSphereProducts[azureSphereProductKey(subscriptionID, resourceGroup, catalogName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSphereProductFixture(product))
}

func (s *Server) handleAzureSphereProductCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	catalog, ok := s.azureSphereCatalogs[azureSphereCatalogKey(subscriptionID, resourceGroup, catalogName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, catalogName))
		return
	}

	key := azureSphereProductKey(subscriptionID, resourceGroup, catalogName, name)
	now := time.Now().UTC()
	status := http.StatusCreated
	product, exists := s.azureSphereProducts[key]
	if !exists {
		product = &azureSphereProduct{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			CatalogName:    catalogName,
			Name:           name,
			CreatedAt:      now,
		}
		s.azureSphereProducts[key] = product
	} else {
		status = http.StatusOK
	}
	product.Location = azureSphereCoalesceLocation(payload, catalog.Location)
	product.Tags = azureSphereStringMap(payload["tags"])
	product.Properties = azureSphereNormalizeProductProperties(name, azureSphereObject(payload["properties"]), product.CreatedAt, now)
	product.UpdatedAt = now
	product.ETag = azureSphereStableTag("product", subscriptionID, resourceGroup, name, now)
	respondJSON(w, status, azureSphereProductFixture(product))
}

func (s *Server) handleAzureSphereProductPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	product, exists := s.azureSphereProducts[azureSphereProductKey(subscriptionID, resourceGroup, catalogName, name)]
	if !exists {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if location := strings.TrimSpace(azureSphereString(payload["location"])); location != "" {
		product.Location = location
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		product.Tags = azureSphereStringMap(tags)
	}
	product.Properties = azureSphereNormalizeProductProperties(name, azureSphereMergeObjects(product.Properties, azureSphereObject(payload["properties"])), product.CreatedAt, time.Now().UTC())
	product.UpdatedAt = time.Now().UTC()
	product.ETag = azureSphereStableTag("product", subscriptionID, resourceGroup, name, product.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSphereProductFixture(product))
}

func (s *Server) handleAzureSphereProductDelete(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureSphereProductKey(subscriptionID, resourceGroup, catalogName, name)
	if _, ok := s.azureSphereProducts[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSphereProducts, key)
	for id, group := range s.azureSphereDeviceGroups {
		if strings.EqualFold(group.SubscriptionID, subscriptionID) && strings.EqualFold(group.ResourceGroup, resourceGroup) && strings.EqualFold(group.CatalogName, catalogName) && strings.EqualFold(group.ProductName, name) {
			delete(s.azureSphereDeviceGroups, id)
		}
	}
	for id, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, catalogName) && strings.EqualFold(device.ProductName, name) {
			delete(s.azureSphereDevices, id)
		}
	}
	for id, deployment := range s.azureSphereDeployments {
		if strings.EqualFold(deployment.SubscriptionID, subscriptionID) && strings.EqualFold(deployment.ResourceGroup, resourceGroup) && strings.EqualFold(deployment.CatalogName, catalogName) && strings.EqualFold(deployment.ProductName, name) {
			delete(s.azureSphereDeployments, id)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSphereDeviceGroupListByProduct(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName, productName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSphereProducts[azureSphereProductKey(subscriptionID, resourceGroup, catalogName, productName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(rawRequestPath(r), productName))
		return
	}
	items := make([]map[string]any, 0)
	for _, group := range s.azureSphereDeviceGroups {
		if strings.EqualFold(group.SubscriptionID, subscriptionID) && strings.EqualFold(group.ResourceGroup, resourceGroup) && strings.EqualFold(group.CatalogName, catalogName) && strings.EqualFold(group.ProductName, productName) {
			items = append(items, azureSphereDeviceGroupFixture(group, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, group.Name)))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereDeviceGroupGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, catalogName, productName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	group, ok := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSphereDeviceGroupFixture(group, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, name)))
}

func (s *Server) handleAzureSphereDeviceGroupCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	product, ok := s.azureSphereProducts[azureSphereProductKey(subscriptionID, resourceGroup, catalogName, productName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, productName))
		return
	}

	key := azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, name)
	now := time.Now().UTC()
	status := http.StatusCreated
	group, exists := s.azureSphereDeviceGroups[key]
	if !exists {
		group = &azureSphereDeviceGroup{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			CatalogName:    catalogName,
			ProductName:    productName,
			Name:           name,
			CreatedAt:      now,
		}
		s.azureSphereDeviceGroups[key] = group
	} else {
		status = http.StatusOK
	}
	group.Location = azureSphereCoalesceLocation(payload, product.Location)
	group.Tags = azureSphereStringMap(payload["tags"])
	group.Properties = azureSphereNormalizeDeviceGroupProperties(name, azureSphereObject(payload["properties"]), group.CreatedAt, now, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, name))
	group.UpdatedAt = now
	group.ETag = azureSphereStableTag("devicegroup", subscriptionID, resourceGroup, name, now)
	respondJSON(w, status, azureSphereDeviceGroupFixture(group, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, name)))
}

func (s *Server) handleAzureSphereDeviceGroupPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	group, exists := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, name)]
	if !exists {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if location := strings.TrimSpace(azureSphereString(payload["location"])); location != "" {
		group.Location = location
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		group.Tags = azureSphereStringMap(tags)
	}
	group.Properties = azureSphereNormalizeDeviceGroupProperties(name, azureSphereMergeObjects(group.Properties, azureSphereObject(payload["properties"])), group.CreatedAt, time.Now().UTC(), s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, name))
	group.UpdatedAt = time.Now().UTC()
	group.ETag = azureSphereStableTag("devicegroup", subscriptionID, resourceGroup, name, group.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSphereDeviceGroupFixture(group, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, name)))
}

func (s *Server) handleAzureSphereDeviceGroupDelete(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, productName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, name)
	if _, ok := s.azureSphereDeviceGroups[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSphereDeviceGroups, key)
	for id, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, catalogName) && strings.EqualFold(device.ProductName, productName) && strings.EqualFold(device.DeviceGroup, name) {
			delete(s.azureSphereDevices, id)
		}
	}
	for id, deployment := range s.azureSphereDeployments {
		if strings.EqualFold(deployment.SubscriptionID, subscriptionID) && strings.EqualFold(deployment.ResourceGroup, resourceGroup) && strings.EqualFold(deployment.CatalogName, catalogName) && strings.EqualFold(deployment.ProductName, productName) && strings.EqualFold(deployment.DeviceGroup, name) {
			delete(s.azureSphereDeployments, id)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSphereDeviceListByDeviceGroup(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(rawRequestPath(r), deviceGroupName))
		return
	}
	items := make([]map[string]any, 0)
	for _, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, catalogName) && strings.EqualFold(device.ProductName, productName) && strings.EqualFold(device.DeviceGroup, deviceGroupName) {
			items = append(items, azureSphereDeviceFixture(device))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereDeviceGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	device, ok := s.azureSphereDevices[azureSphereDeviceKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSphereDeviceFixture(device))
}

func (s *Server) handleAzureSphereDeviceCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	group, ok := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, deviceGroupName))
		return
	}

	key := azureSphereDeviceKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	now := time.Now().UTC()
	status := http.StatusCreated
	device, exists := s.azureSphereDevices[key]
	if !exists {
		device = &azureSphereDevice{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			CatalogName:    catalogName,
			ProductName:    productName,
			DeviceGroup:    deviceGroupName,
			Name:           name,
			CreatedAt:      now,
		}
		s.azureSphereDevices[key] = device
	} else {
		status = http.StatusOK
	}
	device.Location = azureSphereCoalesceLocation(payload, group.Location)
	device.Tags = azureSphereStringMap(payload["tags"])
	device.Properties = azureSphereNormalizeDeviceProperties(name, azureSphereObject(payload["properties"]), device.CreatedAt, now)
	device.UpdatedAt = now
	device.ETag = azureSphereStableTag("device", subscriptionID, resourceGroup, name, now)
	respondJSON(w, status, azureSphereDeviceFixture(device))
}

func (s *Server) handleAzureSphereDevicePatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	device, exists := s.azureSphereDevices[azureSphereDeviceKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)]
	if !exists {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if location := strings.TrimSpace(azureSphereString(payload["location"])); location != "" {
		device.Location = location
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		device.Tags = azureSphereStringMap(tags)
	}
	device.Properties = azureSphereNormalizeDeviceProperties(name, azureSphereMergeObjects(device.Properties, azureSphereObject(payload["properties"])), device.CreatedAt, time.Now().UTC())
	device.UpdatedAt = time.Now().UTC()
	device.ETag = azureSphereStableTag("device", subscriptionID, resourceGroup, name, device.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSphereDeviceFixture(device))
}

func (s *Server) handleAzureSphereDeviceDelete(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureSphereDeviceKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	if _, ok := s.azureSphereDevices[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSphereDevices, key)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSphereDeploymentListByDeviceGroup(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(rawRequestPath(r), deviceGroupName))
		return
	}
	items := make([]map[string]any, 0)
	for _, deployment := range s.azureSphereDeployments {
		if strings.EqualFold(deployment.SubscriptionID, subscriptionID) && strings.EqualFold(deployment.ResourceGroup, resourceGroup) && strings.EqualFold(deployment.CatalogName, catalogName) && strings.EqualFold(deployment.ProductName, productName) && strings.EqualFold(deployment.DeviceGroup, deviceGroupName) {
			items = append(items, azureSphereDeploymentFixture(deployment))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereDeploymentGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	deployment, ok := s.azureSphereDeployments[azureSphereDeploymentKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSphereDeploymentFixture(deployment))
}

func (s *Server) handleAzureSphereDeploymentCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	group, ok := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, deviceGroupName))
		return
	}

	key := azureSphereDeploymentKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	now := time.Now().UTC()
	status := http.StatusCreated
	deployment, exists := s.azureSphereDeployments[key]
	if !exists {
		deployment = &azureSphereDeployment{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			CatalogName:    catalogName,
			ProductName:    productName,
			DeviceGroup:    deviceGroupName,
			Name:           name,
			CreatedAt:      now,
		}
		s.azureSphereDeployments[key] = deployment
	} else {
		status = http.StatusOK
	}
	deployment.Location = azureSphereCoalesceLocation(payload, group.Location)
	deployment.Tags = azureSphereStringMap(payload["tags"])
	deployment.Properties = azureSphereNormalizeDeploymentProperties(name, azureSphereObject(payload["properties"]), deployment.CreatedAt, now)
	deployment.UpdatedAt = now
	deployment.ETag = azureSphereStableTag("deployment", subscriptionID, resourceGroup, name, now)
	if existing := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)]; existing != nil {
		existing.Properties = azureSphereNormalizeDeviceGroupProperties(existing.Name, existing.Properties, existing.CreatedAt, existing.UpdatedAt, true)
	}
	respondJSON(w, status, azureSphereDeploymentFixture(deployment))
}

func (s *Server) handleAzureSphereDeploymentDelete(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureSphereDeploymentKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name)
	if _, ok := s.azureSphereDeployments[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSphereDeployments, key)
	if existing := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)]; existing != nil {
		existing.Properties = azureSphereNormalizeDeviceGroupProperties(existing.Name, existing.Properties, existing.CreatedAt, time.Now().UTC(), s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName))
		existing.UpdatedAt = time.Now().UTC()
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSphereImageListByCatalog(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSphereCatalogs[azureSphereCatalogKey(subscriptionID, resourceGroup, catalogName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(rawRequestPath(r), catalogName))
		return
	}
	items := make([]map[string]any, 0)
	for _, image := range s.azureSphereImages {
		if strings.EqualFold(image.SubscriptionID, subscriptionID) && strings.EqualFold(image.ResourceGroup, resourceGroup) && strings.EqualFold(image.CatalogName, catalogName) {
			items = append(items, azureSphereImageFixture(image))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereImageGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, catalogName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	image, ok := s.azureSphereImages[azureSphereImageKey(subscriptionID, resourceGroup, catalogName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSphereImageFixture(image))
}

func (s *Server) handleAzureSphereImageCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, name string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	catalog, ok := s.azureSphereCatalogs[azureSphereCatalogKey(subscriptionID, resourceGroup, catalogName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, catalogName))
		return
	}

	key := azureSphereImageKey(subscriptionID, resourceGroup, catalogName, name)
	now := time.Now().UTC()
	status := http.StatusCreated
	image, exists := s.azureSphereImages[key]
	if !exists {
		image = &azureSphereImage{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			CatalogName:    catalogName,
			Name:           name,
			CreatedAt:      now,
		}
		s.azureSphereImages[key] = image
	} else {
		status = http.StatusOK
	}
	image.Location = azureSphereCoalesceLocation(payload, catalog.Location)
	image.Tags = azureSphereStringMap(payload["tags"])
	image.Properties = azureSphereNormalizeImageProperties(name, azureSphereObject(payload["properties"]), image.CreatedAt, now)
	image.UpdatedAt = now
	image.ETag = azureSphereStableTag("image", subscriptionID, resourceGroup, name, now)
	respondJSON(w, status, azureSphereImageFixture(image))
}

func (s *Server) handleAzureSphereImageDelete(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureSphereImageKey(subscriptionID, resourceGroup, catalogName, name)
	if _, ok := s.azureSphereImages[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSphereImages, key)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSphereCatalogListDevices(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, catalogName) {
			items = append(items, azureSphereDeviceFixture(device))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereCatalogListDeviceGroups(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, group := range s.azureSphereDeviceGroups {
		if strings.EqualFold(group.SubscriptionID, subscriptionID) && strings.EqualFold(group.ResourceGroup, resourceGroup) && strings.EqualFold(group.CatalogName, catalogName) {
			items = append(items, azureSphereDeviceGroupFixture(group, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, group.ProductName, group.Name)))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereCatalogListDeployments(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, deployment := range s.azureSphereDeployments {
		if strings.EqualFold(deployment.SubscriptionID, subscriptionID) && strings.EqualFold(deployment.ResourceGroup, resourceGroup) && strings.EqualFold(deployment.CatalogName, catalogName) {
			items = append(items, azureSphereDeploymentFixture(deployment))
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereCatalogListDeviceInsights(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, catalogName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, catalogName) {
			items = append(items, map[string]any{
				"deviceId":         device.Name,
				"deviceGroupName":  device.DeviceGroup,
				"severity":         "Low",
				"description":      "Device is reporting healthy telemetry in Stackyard.",
				"occurredTimeUtc":  device.UpdatedAt.Format(time.RFC3339),
				"lastModifiedUtc":  device.UpdatedAt.Format(time.RFC3339),
				"recommendation":   "No action required.",
				"catalogItemCount": 1,
			})
		}
	}
	respondJSON(w, http.StatusOK, azureSpherePaginatedResponse(r, items))
}

func (s *Server) handleAzureSphereCatalogUploadImage(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	catalog, ok := s.azureSphereCatalogs[azureSphereCatalogKey(subscriptionID, resourceGroup, catalogName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, catalogName))
		return
	}

	props := azureSphereObject(payload["properties"])
	name := strings.TrimSpace(azureSphereString(payload["name"]))
	if name == "" {
		name = strings.TrimSpace(azureSphereString(props["imageName"]))
	}
	if name == "" {
		sum := sha1.Sum([]byte(path + time.Now().UTC().Format(time.RFC3339Nano)))
		name = "uploaded-" + strings.ToLower(hex.EncodeToString(sum[:])[:8])
	}
	now := time.Now().UTC()
	image := &azureSphereImage{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		CatalogName:    catalogName,
		Name:           name,
		Location:       catalog.Location,
		Tags:           map[string]string{},
		Properties:     azureSphereNormalizeImageProperties(name, props, now, now),
		CreatedAt:      now,
		UpdatedAt:      now,
		ETag:           azureSphereStableTag("image", subscriptionID, resourceGroup, name, now),
	}
	s.azureSphereImages[azureSphereImageKey(subscriptionID, resourceGroup, catalogName, name)] = image
	w.Header().Set("Location", azureSphereImageID(subscriptionID, resourceGroup, catalogName, name)+"?api-version="+azureSphereAPIVersion)
	w.Header().Set("Retry-After", "0")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSphereProductCountDevices(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, productName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	count := 0
	for _, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, catalogName) && strings.EqualFold(device.ProductName, productName) {
			count++
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": count})
}

func (s *Server) handleAzureSphereProductGenerateDefaultDeviceGroups(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, productName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	product, ok := s.azureSphereProducts[azureSphereProductKey(subscriptionID, resourceGroup, catalogName, productName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound("", productName))
		return
	}

	now := time.Now().UTC()
	defaultNames := []string{".default", ".unassigned"}
	items := make([]map[string]any, 0, len(defaultNames))
	for _, name := range defaultNames {
		key := azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, name)
		group, exists := s.azureSphereDeviceGroups[key]
		if !exists {
			group = &azureSphereDeviceGroup{
				SubscriptionID: subscriptionID,
				ResourceGroup:  resourceGroup,
				CatalogName:    catalogName,
				ProductName:    productName,
				Name:           name,
				Location:       product.Location,
				Tags:           map[string]string{},
				CreatedAt:      now,
			}
			s.azureSphereDeviceGroups[key] = group
		}
		group.Properties = azureSphereNormalizeDeviceGroupProperties(name, group.Properties, group.CreatedAt, now, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, name))
		group.UpdatedAt = now
		group.ETag = azureSphereStableTag("devicegroup", subscriptionID, resourceGroup, name, now)
		items = append(items, azureSphereDeviceGroupFixture(group, s.azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, name)))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSphereDeviceGroupCountDevices(w http.ResponseWriter, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	count := 0
	for _, device := range s.azureSphereDevices {
		if strings.EqualFold(device.SubscriptionID, subscriptionID) && strings.EqualFold(device.ResourceGroup, resourceGroup) && strings.EqualFold(device.CatalogName, catalogName) && strings.EqualFold(device.ProductName, productName) && strings.EqualFold(device.DeviceGroup, deviceGroupName) {
			count++
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": count})
}

func (s *Server) handleAzureSphereDeviceGroupClaimDevices(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, catalogName, productName, deviceGroupName string) {
	_, payload, ok := azureSphereReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	group, ok := s.azureSphereDeviceGroups[azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureSphereNotFound(path, deviceGroupName))
		return
	}

	ids := azureSphereDeviceIdentifierList(payload)
	if len(ids) == 0 {
		ids = []string{strings.Repeat("0", 128)}
	}
	now := time.Now().UTC()
	for _, id := range ids {
		for existingKey, existingDevice := range s.azureSphereDevices {
			if strings.EqualFold(existingDevice.SubscriptionID, subscriptionID) &&
				strings.EqualFold(existingDevice.ResourceGroup, resourceGroup) &&
				strings.EqualFold(existingDevice.CatalogName, catalogName) &&
				strings.EqualFold(existingDevice.Name, id) {
				delete(s.azureSphereDevices, existingKey)
			}
		}
		deviceKey := azureSphereDeviceKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, id)
		s.azureSphereDevices[deviceKey] = &azureSphereDevice{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			CatalogName:    catalogName,
			ProductName:    productName,
			DeviceGroup:    deviceGroupName,
			Name:           id,
			Location:       group.Location,
			Tags:           map[string]string{},
			Properties:     azureSphereNormalizeDeviceProperties(id, map[string]any{"claimedDeviceId": id}, now, now),
			CreatedAt:      now,
			UpdatedAt:      now,
			ETag:           azureSphereStableTag("device", subscriptionID, resourceGroup, id, now),
		}
	}
	w.Header().Set("Location", azureSphereDeviceGroupID(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName)+"/claimDevices?api-version="+azureSphereAPIVersion)
	w.Header().Set("Retry-After", "0")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) azureSphereGroupHasDeployment(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName string) bool {
	for _, deployment := range s.azureSphereDeployments {
		if strings.EqualFold(deployment.SubscriptionID, subscriptionID) &&
			strings.EqualFold(deployment.ResourceGroup, resourceGroup) &&
			strings.EqualFold(deployment.CatalogName, catalogName) &&
			strings.EqualFold(deployment.ProductName, productName) &&
			strings.EqualFold(deployment.DeviceGroup, deviceGroupName) {
			return true
		}
	}
	return false
}

func azureSphereOperationsFixture() []map[string]any {
	paths := []string{
		"Microsoft.AzureSphere/catalogs/read",
		"Microsoft.AzureSphere/catalogs/write",
		"Microsoft.AzureSphere/catalogs/delete",
		"Microsoft.AzureSphere/catalogs/listDevices/action",
		"Microsoft.AzureSphere/catalogs/listDeviceGroups/action",
		"Microsoft.AzureSphere/catalogs/listDeployments/action",
		"Microsoft.AzureSphere/catalogs/listDeviceInsights/action",
		"Microsoft.AzureSphere/catalogs/uploadImage/action",
		"Microsoft.AzureSphere/catalogs/products/read",
		"Microsoft.AzureSphere/catalogs/products/write",
		"Microsoft.AzureSphere/catalogs/products/delete",
		"Microsoft.AzureSphere/catalogs/products/countDevices/action",
		"Microsoft.AzureSphere/catalogs/products/generateDefaultDeviceGroups/action",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/read",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/write",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/delete",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/countDevices/action",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/claimDevices/action",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/devices/read",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/devices/write",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/devices/delete",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/deployments/read",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/deployments/write",
		"Microsoft.AzureSphere/catalogs/products/deviceGroups/deployments/delete",
		"Microsoft.AzureSphere/catalogs/images/read",
		"Microsoft.AzureSphere/catalogs/images/write",
		"Microsoft.AzureSphere/catalogs/images/delete",
	}
	out := make([]map[string]any, 0, len(paths))
	for _, name := range paths {
		out = append(out, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":  "Microsoft Azure Sphere",
				"resource":  "Azure Sphere",
				"operation": name,
			},
		})
	}
	return out
}

func azureSphereCatalogFixture(catalog *azureSphereCatalog) map[string]any {
	return map[string]any{
		"id":         azureSphereCatalogID(catalog.SubscriptionID, catalog.ResourceGroup, catalog.Name),
		"name":       catalog.Name,
		"type":       "Microsoft.AzureSphere/catalogs",
		"location":   catalog.Location,
		"tags":       azureSphereAnyStringMap(catalog.Tags),
		"systemData": azureSphereSystemData(catalog.CreatedAt, catalog.UpdatedAt),
		"properties": azureSphereMergeObjects(map[string]any{}, catalog.Properties),
	}
}

func azureSphereProductFixture(product *azureSphereProduct) map[string]any {
	return map[string]any{
		"id":         azureSphereProductID(product.SubscriptionID, product.ResourceGroup, product.CatalogName, product.Name),
		"name":       product.Name,
		"type":       "Microsoft.AzureSphere/catalogs/products",
		"location":   product.Location,
		"tags":       azureSphereAnyStringMap(product.Tags),
		"systemData": azureSphereSystemData(product.CreatedAt, product.UpdatedAt),
		"properties": azureSphereMergeObjects(map[string]any{}, product.Properties),
	}
}

func azureSphereDeviceGroupFixture(group *azureSphereDeviceGroup, hasDeployment bool) map[string]any {
	properties := azureSphereMergeObjects(map[string]any{}, group.Properties)
	properties["hasDeployment"] = hasDeployment
	return map[string]any{
		"id":         azureSphereDeviceGroupID(group.SubscriptionID, group.ResourceGroup, group.CatalogName, group.ProductName, group.Name),
		"name":       group.Name,
		"type":       "Microsoft.AzureSphere/catalogs/products/deviceGroups",
		"location":   group.Location,
		"tags":       azureSphereAnyStringMap(group.Tags),
		"systemData": azureSphereSystemData(group.CreatedAt, group.UpdatedAt),
		"properties": properties,
	}
}

func azureSphereDeviceFixture(device *azureSphereDevice) map[string]any {
	return map[string]any{
		"id":         azureSphereDeviceID(device.SubscriptionID, device.ResourceGroup, device.CatalogName, device.ProductName, device.DeviceGroup, device.Name),
		"name":       device.Name,
		"type":       "Microsoft.AzureSphere/catalogs/products/deviceGroups/devices",
		"location":   device.Location,
		"tags":       azureSphereAnyStringMap(device.Tags),
		"systemData": azureSphereSystemData(device.CreatedAt, device.UpdatedAt),
		"properties": azureSphereMergeObjects(map[string]any{}, device.Properties),
	}
}

func azureSphereDeploymentFixture(deployment *azureSphereDeployment) map[string]any {
	return map[string]any{
		"id":         azureSphereDeploymentID(deployment.SubscriptionID, deployment.ResourceGroup, deployment.CatalogName, deployment.ProductName, deployment.DeviceGroup, deployment.Name),
		"name":       deployment.Name,
		"type":       "Microsoft.AzureSphere/catalogs/products/deviceGroups/deployments",
		"location":   deployment.Location,
		"tags":       azureSphereAnyStringMap(deployment.Tags),
		"systemData": azureSphereSystemData(deployment.CreatedAt, deployment.UpdatedAt),
		"properties": azureSphereMergeObjects(map[string]any{}, deployment.Properties),
	}
}

func azureSphereImageFixture(image *azureSphereImage) map[string]any {
	return map[string]any{
		"id":         azureSphereImageID(image.SubscriptionID, image.ResourceGroup, image.CatalogName, image.Name),
		"name":       image.Name,
		"type":       "Microsoft.AzureSphere/catalogs/images",
		"location":   image.Location,
		"tags":       azureSphereAnyStringMap(image.Tags),
		"systemData": azureSphereSystemData(image.CreatedAt, image.UpdatedAt),
		"properties": azureSphereMergeObjects(map[string]any{}, image.Properties),
	}
}

func azureSphereNormalizeCatalogProperties(name, subscriptionID, resourceGroup string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureSphereMergeObjects(map[string]any{}, payload)
	if azureSphereString(props["tenantId"]) == "" {
		props["tenantId"] = azureSphereStableGUID(azureSphereCatalogID(subscriptionID, resourceGroup, name))
	}
	props["provisioningState"] = "Succeeded"
	if azureSphereString(props["description"]) == "" {
		props["description"] = "Stackyard Azure Sphere catalog"
	}
	props["createdTimeUtc"] = createdAt.Format(time.RFC3339)
	props["lastModifiedTimeUtc"] = updatedAt.Format(time.RFC3339)
	return props
}

func azureSphereNormalizeProductProperties(name string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureSphereMergeObjects(map[string]any{}, payload)
	if azureSphereString(props["description"]) == "" {
		props["description"] = "Stackyard Azure Sphere product " + name
	}
	props["provisioningState"] = "Succeeded"
	props["createdTimeUtc"] = createdAt.Format(time.RFC3339)
	props["lastModifiedTimeUtc"] = updatedAt.Format(time.RFC3339)
	return props
}

func azureSphereNormalizeDeviceGroupProperties(name string, payload map[string]any, createdAt, updatedAt time.Time, hasDeployment bool) map[string]any {
	props := azureSphereMergeObjects(map[string]any{}, payload)
	if _, ok := props["allowCrashDumpsCollection"]; !ok {
		props["allowCrashDumpsCollection"] = false
	}
	if azureSphereString(props["description"]) == "" {
		props["description"] = "Stackyard Azure Sphere device group " + name
	}
	if azureSphereString(props["osFeedType"]) == "" {
		props["osFeedType"] = "Retail"
	}
	if azureSphereString(props["regionalDataBoundary"]) == "" {
		props["regionalDataBoundary"] = "None"
	}
	if _, ok := props["updatePolicy"]; !ok {
		props["updatePolicy"] = "UpdateAll"
	}
	props["hasDeployment"] = hasDeployment
	props["provisioningState"] = "Succeeded"
	props["createdTimeUtc"] = createdAt.Format(time.RFC3339)
	props["lastModifiedTimeUtc"] = updatedAt.Format(time.RFC3339)
	return props
}

func azureSphereNormalizeDeviceProperties(name string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureSphereMergeObjects(map[string]any{}, payload)
	if azureSphereString(props["deviceId"]) == "" {
		props["deviceId"] = name
	}
	if azureSphereString(props["claimedDeviceId"]) == "" {
		props["claimedDeviceId"] = name
	}
	if azureSphereString(props["chipModel"]) == "" {
		props["chipModel"] = "MT3620"
	}
	props["provisioningState"] = "Succeeded"
	props["lastAddressedUtc"] = updatedAt.Format(time.RFC3339)
	props["lastModifiedUtc"] = updatedAt.Format(time.RFC3339)
	props["createdTimeUtc"] = createdAt.Format(time.RFC3339)
	return props
}

func azureSphereNormalizeDeploymentProperties(name string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureSphereMergeObjects(map[string]any{}, payload)
	if azureSphereString(props["deploymentId"]) == "" {
		props["deploymentId"] = azureSphereStableGUID(name + "::deployment")
	}
	if _, ok := props["deployedImages"]; !ok {
		props["deployedImages"] = []any{
			map[string]any{
				"imageId":   azureSphereStableGUID(name + "::image"),
				"imageName": "stackyard-os",
			},
		}
	}
	props["provisioningState"] = "Succeeded"
	props["deploymentDateUtc"] = updatedAt.Format(time.RFC3339)
	props["createdTimeUtc"] = createdAt.Format(time.RFC3339)
	return props
}

func azureSphereNormalizeImageProperties(name string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureSphereMergeObjects(map[string]any{}, payload)
	if azureSphereString(props["imageId"]) == "" {
		props["imageId"] = azureSphereStableGUID(name + "::image")
	}
	if azureSphereString(props["imageName"]) == "" {
		props["imageName"] = name
	}
	if azureSphereString(props["description"]) == "" {
		props["description"] = "Stackyard Azure Sphere image " + name
	}
	if azureSphereString(props["imageType"]) == "" {
		props["imageType"] = "Application"
	}
	if azureSphereString(props["componentId"]) == "" {
		props["componentId"] = azureSphereStableGUID(name + "::component")
	}
	if azureSphereString(props["uri"]) == "" {
		props["uri"] = "https://stackyard.example/images/" + strings.ToLower(name) + ".bin"
	}
	if azureSphereString(props["regionalDataBoundary"]) == "" {
		props["regionalDataBoundary"] = "None"
	}
	props["provisioningState"] = "Succeeded"
	props["createdTimeUtc"] = createdAt.Format(time.RFC3339)
	props["lastModifiedTimeUtc"] = updatedAt.Format(time.RFC3339)
	return props
}

func azureSpherePaginatedResponse(r *http.Request, items []map[string]any) map[string]any {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureSphereString(items[i]["id"])) < strings.ToLower(azureSphereString(items[j]["id"]))
	})
	start := azureSphereQueryInt(r, "$skip")
	if start < 0 {
		start = 0
	}
	limit := azureSphereQueryInt(r, "$top")
	if maxPage := azureSphereQueryInt(r, "$maxpagesize"); maxPage > 0 && (limit <= 0 || maxPage < limit) {
		limit = maxPage
	}
	if start > len(items) {
		start = len(items)
	}
	end := len(items)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return map[string]any{"value": items[start:end]}
}

func azureSphereQueryInt(r *http.Request, key string) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func azureSphereDeviceIdentifierList(payload map[string]any) []string {
	candidates := []string{"deviceIdentifiers", "deviceIds", "devices"}
	for _, key := range candidates {
		if values, ok := payload[key].([]any); ok {
			out := make([]string, 0, len(values))
			for _, value := range values {
				if id := strings.TrimSpace(azureSphereString(value)); id != "" {
					out = append(out, id)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

func azureSphereCatalogKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(azureSphereCatalogID(subscriptionID, resourceGroup, name))
}

func azureSphereProductKey(subscriptionID, resourceGroup, catalogName, name string) string {
	return strings.ToLower(azureSphereProductID(subscriptionID, resourceGroup, catalogName, name))
}

func azureSphereDeviceGroupKey(subscriptionID, resourceGroup, catalogName, productName, name string) string {
	return strings.ToLower(azureSphereDeviceGroupID(subscriptionID, resourceGroup, catalogName, productName, name))
}

func azureSphereDeviceKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) string {
	return strings.ToLower(azureSphereDeviceID(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name))
}

func azureSphereDeploymentKey(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) string {
	return strings.ToLower(azureSphereDeploymentID(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name))
}

func azureSphereImageKey(subscriptionID, resourceGroup, catalogName, name string) string {
	return strings.ToLower(azureSphereImageID(subscriptionID, resourceGroup, catalogName, name))
}

func azureSphereCatalogID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.AzureSphere/catalogs/%s", subscriptionID, resourceGroup, name)
}

func azureSphereProductID(subscriptionID, resourceGroup, catalogName, name string) string {
	return azureSphereCatalogID(subscriptionID, resourceGroup, catalogName) + "/products/" + name
}

func azureSphereDeviceGroupID(subscriptionID, resourceGroup, catalogName, productName, name string) string {
	return azureSphereProductID(subscriptionID, resourceGroup, catalogName, productName) + "/deviceGroups/" + name
}

func azureSphereDeviceID(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) string {
	return azureSphereDeviceGroupID(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName) + "/devices/" + name
}

func azureSphereDeploymentID(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName, name string) string {
	return azureSphereDeviceGroupID(subscriptionID, resourceGroup, catalogName, productName, deviceGroupName) + "/deployments/" + name
}

func azureSphereImageID(subscriptionID, resourceGroup, catalogName, name string) string {
	return azureSphereCatalogID(subscriptionID, resourceGroup, catalogName) + "/images/" + name
}

func azureSphereStableGUID(seed string) string {
	sum := sha1.Sum([]byte(seed))
	hexValue := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}

func azureSphereStableTag(prefix, subscriptionID, resourceGroup, name string, now time.Time) string {
	sum := sha1.Sum([]byte(prefix + "::" + azureSphereCatalogID(subscriptionID, resourceGroup, name) + "::" + now.Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:8])
}

func azureSphereSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
	}
}

func azureSphereNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("resource %q was not found for path %s", name, path),
		},
	}
}

func azureSphereReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
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
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureSphereString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureSphereObject(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return azureSphereMergeObjects(map[string]any{}, typed)
}

func azureSphereStringMap(value any) map[string]string {
	raw, _ := value.(map[string]any)
	if raw == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if str := azureSphereString(val); str != "" {
			out[key] = str
		}
	}
	return out
}

func azureSphereAnyStringMap(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for key, val := range value {
		out[key] = val
	}
	return out
}

func azureSphereMergeObjects(base, overlay map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	out := make(map[string]any, len(base)+len(overlay))
	for key, val := range base {
		out[key] = val
	}
	for key, val := range overlay {
		if existing, ok := out[key].(map[string]any); ok {
			if incoming, ok := val.(map[string]any); ok {
				out[key] = azureSphereMergeObjects(existing, incoming)
				continue
			}
		}
		out[key] = val
	}
	return out
}

func azureSphereCoalesceLocation(payload map[string]any, fallback string) string {
	if location := strings.TrimSpace(azureSphereString(payload["location"])); location != "" {
		return location
	}
	return fallback
}
