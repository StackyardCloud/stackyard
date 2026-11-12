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

const azureBillingAPIVersion = "2024-04-01"

type azureBillingAccount struct {
	Name          string
	DisplayName   string
	AgreementType string
	Tags          map[string]string
	Properties    map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ETag          string
}

type azureBillingResource struct {
	ScopeID    string
	Account    string
	Collection string
	Name       string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureBillingProperty struct {
	SubscriptionID string
	Properties     map[string]any
	UpdatedAt      time.Time
	ETag           string
}

var azureBillingNestedCollections = map[string]struct{}{
	"billingprofiles":    {},
	"invoicesections":    {},
	"customers":          {},
	"departments":        {},
	"enrollmentaccounts": {},
}

var azureBillingReadOnlyCollections = map[string]struct{}{
	"agreements":             {},
	"availablebalances":      {},
	"billingpermissions":     {},
	"billingroledefinitions": {},
	"invoices":               {},
	"paymentmethods":         {},
	"transactions":           {},
	"reservationorders":      {},
	"savingsplanorders":      {},
}

func (s *Server) handleAzureBillingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsBillingPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureBillingAllowedMethod(r.Method) {
		return false
	}

	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Billing") {
		s.handleAzureBillingProviderRoute(w, r, path, segments[2:])
		return true
	}

	subscriptionID, tail, ok := azureBillingSubscriptionContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	s.handleAzureBillingSubscriptionRoute(w, r, path, subscriptionID, tail)
	return true
}

func azureIsBillingPath(segments []string) bool {
	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Billing") {
		return true
	}
	_, tail, ok := azureBillingSubscriptionContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "billingProperty")
}

func azureBillingAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureBillingSubscriptionContext(segments []string) (subscriptionID string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Billing") {
			return subscriptionID, segments[idx+2:], true
		}
	}
	return "", nil, false
}

func (s *Server) handleAzureBillingProviderRoute(w http.ResponseWriter, r *http.Request, path string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "validateaddress":
		if r.Method != http.MethodPost || len(tail) != 1 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBillingValidateAddress(w, r, path)
	case "billingaccounts":
		s.handleAzureBillingAccountsRoute(w, r, path, tail[1:])
	case "billingrequests", "transfers", "recipienttransfers":
		s.handleAzureBillingRootCollectionRoute(w, r, path, tail[0], tail[1:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureBillingSubscriptionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) == 2 &&
		strings.EqualFold(tail[0], "billingProperty") &&
		strings.EqualFold(tail[1], "default") {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBillingPropertyGet(w, subscriptionID)
		case http.MethodPatch:
			s.handleAzureBillingPropertyPatch(w, r, path, subscriptionID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureBillingValidateAddress(w http.ResponseWriter, r *http.Request, path string) {
	_, payload, ok := azureBillingReadRequestBody(w, r, path)
	if !ok {
		return
	}
	address := azureBillingMap(payload["address"])
	if len(address) == 0 {
		address = payload
	}
	if len(address) == 0 {
		respondAzureBillingError(w, http.StatusBadRequest, "InvalidAddress", "address payload is required")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"status":            "Valid",
		"validationMessage": "Address is valid",
		"address":           address,
	})
}

func (s *Server) handleAzureBillingAccountsRoute(w http.ResponseWriter, r *http.Request, path string, tail []string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	s.azureBillingEnsureSeededLocked()

	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		accounts := make([]map[string]any, 0, len(s.azureBillingAccounts))
		for _, account := range s.azureBillingAccounts {
			accounts = append(accounts, azureBillingAccountFixture(account))
		}
		sort.Slice(accounts, func(i, j int) bool {
			return azureBillingString(accounts[i]["name"]) < azureBillingString(accounts[j]["name"])
		})
		respondJSON(w, http.StatusOK, map[string]any{"value": accounts})
		return
	}

	accountName := strings.TrimSpace(tail[0])
	account := s.azureBillingAccounts[strings.ToLower(accountName)]
	if account == nil {
		respondAzureBillingError(w, http.StatusNotFound, "ResourceNotFound", "billing account not found")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, azureBillingAccountFixture(account))
		case http.MethodHead:
			w.WriteHeader(http.StatusOK)
		case http.MethodPatch:
			s.handleAzureBillingAccountPatch(w, r, path, account)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "validatepaymentterms":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"isValid": true,
			"message": "Payment terms are valid for the billing account",
		})
		return
	case "listinvoicesectionsbycreatesubscriptionpermission":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		scopeID := azureBillingAccountID(account.Name)
		s.azureBillingEnsureDefaultsForCollectionLocked(scopeID, account.Name, "billingprofiles")
		values := s.azureBillingInvoiceSectionsByPermissionLocked(account.Name)
		respondJSON(w, http.StatusOK, map[string]any{"value": values})
		return
	default:
		s.handleAzureBillingScopeRouteUnlocked(w, r, path, account.Name, nil, tail[1:])
	}
}

func (s *Server) handleAzureBillingAccountPatch(w http.ResponseWriter, r *http.Request, path string, account *azureBillingAccount) {
	_, payload, ok := azureBillingReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		if account.Tags == nil {
			account.Tags = map[string]string{}
		}
		for k, v := range tags {
			account.Tags[k] = azureBillingString(v)
		}
	}
	if props, ok := payload["properties"].(map[string]any); ok {
		account.Properties = azureBillingMergeMaps(account.Properties, props)
	}
	account.Properties = azureBillingNormalizeAccountProperties(account.AgreementType, account.Name, account.Properties)
	account.UpdatedAt = time.Now().UTC()
	account.ETag = azureBillingETag(account.Name, account.UpdatedAt)
	respondJSON(w, http.StatusOK, azureBillingAccountFixture(account))
}

func (s *Server) handleAzureBillingRootCollectionRoute(w http.ResponseWriter, r *http.Request, path, collection string, tail []string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	s.azureBillingEnsureSeededLocked()

	scopeID := "/providers/Microsoft.Billing"
	normalized := strings.ToLower(strings.TrimSpace(collection))
	s.azureBillingEnsureDefaultsForCollectionLocked(scopeID, "", normalized)

	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": s.azureBillingListResourcesLocked(scopeID, normalized)})
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "resource name is required")
		return
	}
	if len(tail) == 1 {
		s.handleAzureBillingCollectionItemLocked(w, r, path, scopeID, "", normalized, name)
		return
	}
	s.handleAzureBillingCollectionActionLocked(w, r, path, scopeID, "", normalized, name, tail[1:])
}

func (s *Server) handleAzureBillingScopeRoute(w http.ResponseWriter, r *http.Request, path, accountName string, scopePairs []string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}
	scopeID := azureBillingScopeID(accountName, scopePairs)
	collection := strings.ToLower(strings.TrimSpace(tail[0]))
	if collection == "" {
		respondAzureInvalidRequest(w, path, "collection is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	s.azureBillingEnsureSeededLocked()

	if collection == "availablebalances" {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, azureBillingAvailableBalanceFixture(scopeID))
		return
	}

	s.azureBillingEnsureDefaultsForCollectionLocked(scopeID, accountName, collection)

	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": s.azureBillingListResourcesLocked(scopeID, collection)})
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "resource name is required")
		return
	}

	if _, ok := azureBillingNestedCollections[collection]; ok && len(tail) > 2 {
		nextScope := append(append([]string{}, scopePairs...), collection, name)
		s.handleAzureBillingScopeRouteUnlocked(w, r, path, accountName, nextScope, tail[2:])
		return
	}

	if len(tail) == 2 {
		s.handleAzureBillingCollectionItemLocked(w, r, path, scopeID, accountName, collection, name)
		return
	}
	s.handleAzureBillingCollectionActionLocked(w, r, path, scopeID, accountName, collection, name, tail[2:])
}

func (s *Server) handleAzureBillingScopeRouteUnlocked(w http.ResponseWriter, r *http.Request, path, accountName string, scopePairs []string, tail []string) {
	scopeID := azureBillingScopeID(accountName, scopePairs)
	collection := strings.ToLower(strings.TrimSpace(tail[0]))
	if collection == "availablebalances" {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, azureBillingAvailableBalanceFixture(scopeID))
		return
	}

	s.azureBillingEnsureDefaultsForCollectionLocked(scopeID, accountName, collection)

	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": s.azureBillingListResourcesLocked(scopeID, collection)})
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "resource name is required")
		return
	}
	if _, ok := azureBillingNestedCollections[collection]; ok && len(tail) > 2 {
		nextScope := append(append([]string{}, scopePairs...), collection, name)
		s.handleAzureBillingScopeRouteUnlocked(w, r, path, accountName, nextScope, tail[2:])
		return
	}
	if len(tail) == 2 {
		s.handleAzureBillingCollectionItemLocked(w, r, path, scopeID, accountName, collection, name)
		return
	}
	s.handleAzureBillingCollectionActionLocked(w, r, path, scopeID, accountName, collection, name, tail[2:])
}

func (s *Server) handleAzureBillingCollectionItemLocked(w http.ResponseWriter, r *http.Request, path, scopeID, accountName, collection, name string) {
	key := azureBillingResourceStoreKey(scopeID, collection, name)
	resource := s.azureBillingResources[key]

	switch r.Method {
	case http.MethodGet:
		if resource == nil {
			if _, ok := azureBillingReadOnlyCollections[collection]; ok {
				s.azureBillingEnsureDefaultsForCollectionLocked(scopeID, accountName, collection)
				resource = s.azureBillingResources[key]
			}
		}
		if resource == nil {
			respondAzureBillingError(w, http.StatusNotFound, "ResourceNotFound", "resource not found")
			return
		}
		respondJSON(w, http.StatusOK, azureBillingResourceFixture(resource))
	case http.MethodHead:
		if resource == nil {
			respondAzureBillingError(w, http.StatusNotFound, "ResourceNotFound", "resource not found")
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodPut, http.MethodPatch:
		if _, readOnly := azureBillingReadOnlyCollections[collection]; readOnly {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, payload, ok := azureBillingReadRequestBody(w, r, path)
		if !ok {
			return
		}
		created := false
		if resource == nil {
			resource = &azureBillingResource{
				ScopeID:    scopeID,
				Account:    accountName,
				Collection: collection,
				Name:       name,
				Tags:       map[string]string{},
				Properties: map[string]any{},
				CreatedAt:  time.Now().UTC(),
			}
			created = true
		}
		if tags, ok := payload["tags"].(map[string]any); ok {
			if resource.Tags == nil {
				resource.Tags = map[string]string{}
			}
			for k, v := range tags {
				resource.Tags[k] = azureBillingString(v)
			}
		}
		if props, ok := payload["properties"].(map[string]any); ok {
			resource.Properties = azureBillingMergeMaps(resource.Properties, props)
		} else {
			resource.Properties = azureBillingMergeMaps(resource.Properties, payload)
		}
		resource.Properties = azureBillingNormalizeResourceProperties(collection, name, resource.Properties)
		resource.UpdatedAt = time.Now().UTC()
		resource.ETag = azureBillingETag(resource.ScopeID+"|"+collection+"|"+name, resource.UpdatedAt)
		s.azureBillingResources[key] = resource
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		respondJSON(w, status, azureBillingResourceFixture(resource))
	case http.MethodDelete:
		if _, readOnly := azureBillingReadOnlyCollections[collection]; readOnly {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if resource == nil {
			respondAzureBillingError(w, http.StatusNotFound, "ResourceNotFound", "resource not found")
			return
		}
		delete(s.azureBillingResources, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureBillingCollectionActionLocked(w http.ResponseWriter, r *http.Request, path, scopeID, accountName, collection, name string, tail []string) {
	action := strings.ToLower(strings.TrimSpace(tail[0]))
	resource := s.azureBillingResources[azureBillingResourceStoreKey(scopeID, collection, name)]
	if resource == nil {
		s.azureBillingEnsureDefaultsForCollectionLocked(scopeID, accountName, collection)
		resource = s.azureBillingResources[azureBillingResourceStoreKey(scopeID, collection, name)]
	}
	if resource == nil {
		resource = &azureBillingResource{
			ScopeID:    scopeID,
			Account:    accountName,
			Collection: collection,
			Name:       name,
			Tags:       map[string]string{},
			Properties: azureBillingNormalizeResourceProperties(collection, name, map[string]any{}),
			CreatedAt:  time.Now().UTC(),
		}
	}

	switch collection {
	case "billingsubscriptions":
		switch action {
		case "validatemoveeligibility":
			respondJSON(w, http.StatusOK, map[string]any{"isMoveEligible": true, "message": "subscription can be moved"})
			return
		case "cancel":
			resource.Properties["status"] = "Canceled"
		case "move", "merge", "split":
			resource.Properties["lastAction"] = action
		}
	case "products":
		switch action {
		case "validatemoveeligibility":
			respondJSON(w, http.StatusOK, map[string]any{"isMoveEligible": true, "message": "product can be moved"})
			return
		case "move":
			resource.Properties["lastAction"] = action
		}
	case "recipienttransfers":
		switch action {
		case "accept":
			resource.Properties["status"] = "Accepted"
		case "decline":
			resource.Properties["status"] = "Declined"
		}
	case "transfers":
		if action == "initiate" {
			resource.Properties["status"] = "PendingAcceptance"
		}
	case "invoices":
		if strings.HasPrefix(action, "download") {
			respondJSON(w, http.StatusOK, map[string]any{
				"url":    "https://example.invalid/billing/" + name + "/download",
				"status": "Ready",
			})
			return
		}
		if action == "amend" {
			resource.Properties["status"] = "Amended"
		}
	case "reservations", "savingsplans":
		resource.Properties["lastAction"] = action
	default:
		resource.Properties["lastAction"] = action
	}

	resource.Properties = azureBillingNormalizeResourceProperties(collection, name, resource.Properties)
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureBillingETag(resource.ScopeID+"|"+collection+"|"+name, resource.UpdatedAt)
	s.azureBillingResources[azureBillingResourceStoreKey(scopeID, collection, name)] = resource
	respondJSON(w, http.StatusOK, azureBillingResourceFixture(resource))
}

func (s *Server) handleAzureBillingPropertyGet(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	s.azureBillingEnsureSeededLocked()
	property := s.azureBillingEnsurePropertyLocked(subscriptionID)
	respondJSON(w, http.StatusOK, azureBillingPropertyFixture(property))
}

func (s *Server) handleAzureBillingPropertyPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	s.azureBillingEnsureSeededLocked()

	_, payload, ok := azureBillingReadRequestBody(w, r, path)
	if !ok {
		return
	}
	property := s.azureBillingEnsurePropertyLocked(subscriptionID)
	if props, ok := payload["properties"].(map[string]any); ok {
		property.Properties = azureBillingMergeMaps(property.Properties, props)
	} else {
		property.Properties = azureBillingMergeMaps(property.Properties, payload)
	}
	property.Properties = azureBillingNormalizePropertyProperties(subscriptionID, property.Properties)
	property.UpdatedAt = time.Now().UTC()
	property.ETag = azureBillingETag(subscriptionID, property.UpdatedAt)
	respondJSON(w, http.StatusOK, azureBillingPropertyFixture(property))
}

func (s *Server) azureBillingEnsureSeededLocked() {
	if len(s.azureBillingAccounts) > 0 {
		return
	}
	now := time.Now().UTC()
	for _, seeded := range []struct {
		Name          string
		DisplayName   string
		AgreementType string
	}{
		{Name: "mca-account", DisplayName: "Contoso MCA Billing", AgreementType: "MicrosoftCustomerAgreement"},
		{Name: "mpa-account", DisplayName: "Fabrikam MPA Billing", AgreementType: "MicrosoftPartnerAgreement"},
		{Name: "ea-account", DisplayName: "AdventureWorks EA Billing", AgreementType: "EnterpriseAgreement"},
	} {
		account := &azureBillingAccount{
			Name:          seeded.Name,
			DisplayName:   seeded.DisplayName,
			AgreementType: seeded.AgreementType,
			Tags:          map[string]string{"seed": "true"},
			Properties:    azureBillingNormalizeAccountProperties(seeded.AgreementType, seeded.Name, map[string]any{"displayName": seeded.DisplayName}),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		account.ETag = azureBillingETag(account.Name, now)
		s.azureBillingAccounts[strings.ToLower(account.Name)] = account
	}
	s.azureBillingEnsurePropertyLocked("00000000-0000-0000-0000-000000000000")
}

func (s *Server) azureBillingEnsurePropertyLocked(subscriptionID string) *azureBillingProperty {
	key := strings.ToLower(strings.TrimSpace(subscriptionID))
	property := s.azureBillingProperties[key]
	if property != nil {
		return property
	}
	now := time.Now().UTC()
	property = &azureBillingProperty{
		SubscriptionID: subscriptionID,
		Properties: azureBillingNormalizePropertyProperties(subscriptionID, map[string]any{
			"billingTenantId":  "11111111-1111-1111-1111-111111111111",
			"billingAccountId": azureBillingAccountID("mca-account"),
		}),
		UpdatedAt: now,
		ETag:      azureBillingETag(subscriptionID, now),
	}
	s.azureBillingProperties[key] = property
	return property
}

func (s *Server) azureBillingEnsureDefaultsForCollectionLocked(scopeID, accountName, collection string) {
	if collection == "" {
		return
	}
	if collection == "availablebalances" {
		return
	}
	if len(s.azureBillingListResourcesLocked(scopeID, collection)) > 0 {
		return
	}
	switch collection {
	case "agreements":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "default", map[string]any{"agreementType": azureBillingAgreementType(accountName)})
	case "billingprofiles":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "default", map[string]any{"currency": "USD"})
	case "invoicesections":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "default", map[string]any{"state": "Active"})
	case "customers":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "default", map[string]any{"enabledAzurePlans": []any{"AzurePlan"}})
	case "billingroledefinitions":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "owner", map[string]any{"permissions": []any{map[string]any{"actions": []any{"*"}, "notActions": []any{}}}})
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "reader", map[string]any{"permissions": []any{map[string]any{"actions": []any{"Microsoft.Billing/*/read"}, "notActions": []any{}}}})
	case "billingpermissions":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "default", map[string]any{"actions": []any{"Microsoft.Billing/*/read", "Microsoft.Billing/billingSubscriptions/write"}})
	case "paymentmethods":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "pm-default", map[string]any{"paymentMethodType": "CheckWire", "status": "Active"})
	case "policies":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "default", map[string]any{"marketplacePurchases": "Enabled", "reservationPurchases": "Enabled"})
	case "billingsubscriptions":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "sub-default", map[string]any{"status": "Active", "skuId": "AzurePlan"})
	case "products":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "product-default", map[string]any{"status": "Active", "productTypeId": "AzureSubscription"})
	case "invoices":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "invoice-202404", map[string]any{"amount": map[string]any{"currency": "USD", "value": 128.44}, "status": "Due"})
	case "transactions":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "txn-202404", map[string]any{"transactionType": "Purchase", "amount": map[string]any{"currency": "USD", "value": 42.50}})
	case "billingrequests":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "request-default", map[string]any{"status": "Completed"})
	case "transfers":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "transfer-default", map[string]any{"status": "Pending"})
	case "recipienttransfers":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "recipient-transfer-default", map[string]any{"status": "Pending"})
	case "associatedtenants":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "tenant-default", map[string]any{"tenantId": "22222222-2222-2222-2222-222222222222", "provisioningState": "Succeeded"})
	case "departments":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "department-default", map[string]any{"status": "Active"})
	case "enrollmentaccounts":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "enrollment-default", map[string]any{"status": "Active", "principalName": "owner@example.com"})
	case "reservationorders":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "reservation-order-default", map[string]any{"status": "Succeeded"})
	case "reservations":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "reservation-default", map[string]any{"status": "Active"})
	case "savingsplanorders":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "savings-plan-order-default", map[string]any{"status": "Succeeded"})
	case "savingsplans":
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "savings-plan-default", map[string]any{"status": "Active"})
	case "billingroleassignments":
		// no implicit role assignments
	default:
		s.azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, "default", map[string]any{})
	}
}

func (s *Server) azureBillingPutDefaultResourceLocked(scopeID, accountName, collection, name string, properties map[string]any) {
	now := time.Now().UTC()
	resource := &azureBillingResource{
		ScopeID:    scopeID,
		Account:    accountName,
		Collection: collection,
		Name:       name,
		Tags:       map[string]string{"seed": "true"},
		Properties: azureBillingNormalizeResourceProperties(collection, name, properties),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	resource.ETag = azureBillingETag(resource.ScopeID+"|"+collection+"|"+name, now)
	s.azureBillingResources[azureBillingResourceStoreKey(scopeID, collection, name)] = resource
}

func (s *Server) azureBillingListResourcesLocked(scopeID, collection string) []map[string]any {
	values := make([]map[string]any, 0)
	for _, resource := range s.azureBillingResources {
		if strings.EqualFold(resource.ScopeID, scopeID) && strings.EqualFold(resource.Collection, collection) {
			values = append(values, azureBillingResourceFixture(resource))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureBillingString(values[i]["name"]) < azureBillingString(values[j]["name"])
	})
	return values
}

func (s *Server) azureBillingInvoiceSectionsByPermissionLocked(accountName string) []map[string]any {
	values := make([]map[string]any, 0)
	prefix := strings.ToLower(azureBillingAccountID(accountName) + "/billingprofiles/")
	for _, resource := range s.azureBillingResources {
		if !strings.EqualFold(resource.Collection, "invoicesections") {
			continue
		}
		scope := strings.ToLower(resource.ScopeID)
		if strings.HasPrefix(scope, prefix) {
			values = append(values, azureBillingResourceFixture(resource))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureBillingString(values[i]["name"]) < azureBillingString(values[j]["name"])
	})
	return values
}

func azureBillingAccountFixture(account *azureBillingAccount) map[string]any {
	return map[string]any{
		"id":         azureBillingAccountID(account.Name),
		"name":       account.Name,
		"type":       "Microsoft.Billing/billingAccounts",
		"etag":       account.ETag,
		"tags":       azureBillingCopyStringMap(account.Tags),
		"properties": azureBillingCloneMap(account.Properties),
	}
}

func azureBillingPropertyFixture(property *azureBillingProperty) map[string]any {
	return map[string]any{
		"id":         "/subscriptions/" + property.SubscriptionID + "/providers/Microsoft.Billing/billingProperty/default",
		"name":       "default",
		"type":       "Microsoft.Billing/billingProperty",
		"etag":       property.ETag,
		"properties": azureBillingCloneMap(property.Properties),
	}
}

func azureBillingResourceFixture(resource *azureBillingResource) map[string]any {
	return map[string]any{
		"id":         azureBillingResourceID(resource.ScopeID, resource.Collection, resource.Name),
		"name":       resource.Name,
		"type":       azureBillingResourceType(resource.ScopeID, resource.Collection),
		"etag":       resource.ETag,
		"tags":       azureBillingCopyStringMap(resource.Tags),
		"properties": azureBillingCloneMap(resource.Properties),
	}
}

func azureBillingAvailableBalanceFixture(scopeID string) map[string]any {
	name := "default"
	return map[string]any{
		"id":   azureBillingResourceID(scopeID, "availableBalances", name),
		"name": name,
		"type": azureBillingResourceType(scopeID, "availableBalances"),
		"properties": map[string]any{
			"amount": map[string]any{
				"currency": "USD",
				"value":    5000.00,
			},
			"creditType": "General",
		},
	}
}

func azureBillingNormalizeAccountProperties(agreementType, name string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	if strings.TrimSpace(azureBillingString(properties["displayName"])) == "" {
		properties["displayName"] = strings.ReplaceAll(strings.Title(strings.ReplaceAll(name, "-", " ")), " ", "")
	}
	if strings.TrimSpace(azureBillingString(properties["accountStatus"])) == "" {
		properties["accountStatus"] = "Active"
	}
	if strings.TrimSpace(azureBillingString(properties["agreementType"])) == "" {
		properties["agreementType"] = agreementType
	}
	if properties["soldTo"] == nil {
		properties["soldTo"] = map[string]any{"companyName": "Contoso"}
	}
	return properties
}

func azureBillingNormalizePropertyProperties(subscriptionID string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	if strings.TrimSpace(azureBillingString(properties["billingTenantId"])) == "" {
		properties["billingTenantId"] = "11111111-1111-1111-1111-111111111111"
	}
	if strings.TrimSpace(azureBillingString(properties["billingAccountId"])) == "" {
		properties["billingAccountId"] = azureBillingAccountID("mca-account")
	}
	if strings.TrimSpace(azureBillingString(properties["subscriptionId"])) == "" {
		properties["subscriptionId"] = subscriptionID
	}
	if strings.TrimSpace(azureBillingString(properties["invoiceEmailOptIn"])) == "" {
		properties["invoiceEmailOptIn"] = "Enabled"
	}
	return properties
}

func azureBillingNormalizeResourceProperties(collection, name string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	if strings.TrimSpace(azureBillingString(properties["displayName"])) == "" {
		properties["displayName"] = name
	}
	switch collection {
	case "billingprofiles":
		if strings.TrimSpace(azureBillingString(properties["status"])) == "" {
			properties["status"] = "Active"
		}
		if strings.TrimSpace(azureBillingString(properties["currency"])) == "" {
			properties["currency"] = "USD"
		}
	case "invoicesections":
		if strings.TrimSpace(azureBillingString(properties["state"])) == "" {
			properties["state"] = "Active"
		}
	case "customers":
		if properties["enabledAzurePlans"] == nil {
			properties["enabledAzurePlans"] = []any{"AzurePlan"}
		}
	case "billingroleassignments":
		if strings.TrimSpace(azureBillingString(properties["principalId"])) == "" {
			properties["principalId"] = "33333333-3333-3333-3333-333333333333"
		}
		if strings.TrimSpace(azureBillingString(properties["roleDefinitionId"])) == "" {
			properties["roleDefinitionId"] = "owner"
		}
	case "billingroledefinitions":
		if properties["permissions"] == nil {
			properties["permissions"] = []any{map[string]any{"actions": []any{"Microsoft.Billing/*/read"}, "notActions": []any{}}}
		}
	case "billingpermissions":
		if properties["actions"] == nil {
			properties["actions"] = []any{"Microsoft.Billing/*/read"}
		}
	case "paymentmethods":
		if strings.TrimSpace(azureBillingString(properties["paymentMethodType"])) == "" {
			properties["paymentMethodType"] = "CheckWire"
		}
		if strings.TrimSpace(azureBillingString(properties["status"])) == "" {
			properties["status"] = "Active"
		}
	case "policies":
		if strings.TrimSpace(azureBillingString(properties["marketplacePurchases"])) == "" {
			properties["marketplacePurchases"] = "Enabled"
		}
	case "billingsubscriptions":
		if strings.TrimSpace(azureBillingString(properties["status"])) == "" {
			properties["status"] = "Active"
		}
		if strings.TrimSpace(azureBillingString(properties["skuId"])) == "" {
			properties["skuId"] = "AzurePlan"
		}
	case "products":
		if strings.TrimSpace(azureBillingString(properties["status"])) == "" {
			properties["status"] = "Active"
		}
		if strings.TrimSpace(azureBillingString(properties["productTypeId"])) == "" {
			properties["productTypeId"] = "AzureSubscription"
		}
	case "invoices":
		if properties["amount"] == nil {
			properties["amount"] = map[string]any{"currency": "USD", "value": 128.44}
		}
		if strings.TrimSpace(azureBillingString(properties["status"])) == "" {
			properties["status"] = "Due"
		}
	case "transactions":
		if properties["amount"] == nil {
			properties["amount"] = map[string]any{"currency": "USD", "value": 42.50}
		}
		if strings.TrimSpace(azureBillingString(properties["transactionType"])) == "" {
			properties["transactionType"] = "Purchase"
		}
	case "billingrequests", "transfers", "recipienttransfers":
		if strings.TrimSpace(azureBillingString(properties["status"])) == "" {
			properties["status"] = "Pending"
		}
	case "associatedtenants":
		if strings.TrimSpace(azureBillingString(properties["tenantId"])) == "" {
			properties["tenantId"] = "22222222-2222-2222-2222-222222222222"
		}
		if strings.TrimSpace(azureBillingString(properties["provisioningState"])) == "" {
			properties["provisioningState"] = "Succeeded"
		}
	case "departments", "enrollmentaccounts", "reservationorders", "reservations", "savingsplanorders", "savingsplans":
		if strings.TrimSpace(azureBillingString(properties["status"])) == "" {
			properties["status"] = "Active"
		}
	}
	return properties
}

func azureBillingAgreementType(accountName string) string {
	switch strings.ToLower(strings.TrimSpace(accountName)) {
	case "mpa-account":
		return "MicrosoftPartnerAgreement"
	case "ea-account":
		return "EnterpriseAgreement"
	default:
		return "MicrosoftCustomerAgreement"
	}
}

func azureBillingAccountID(name string) string {
	return "/providers/Microsoft.Billing/billingAccounts/" + name
}

func azureBillingScopeID(accountName string, scopePairs []string) string {
	id := azureBillingAccountID(accountName)
	for i := 0; i+1 < len(scopePairs); i += 2 {
		id += "/" + scopePairs[i] + "/" + scopePairs[i+1]
	}
	return id
}

func azureBillingResourceID(scopeID, collection, name string) string {
	return strings.TrimRight(scopeID, "/") + "/" + collection + "/" + name
}

func azureBillingResourceType(scopeID, collection string) string {
	segments := splitPathSegments(strings.TrimPrefix(scopeID, "/providers/Microsoft.Billing"))
	parts := []string{"Microsoft.Billing"}
	for i := 0; i < len(segments); i += 2 {
		parts = append(parts, segments[i])
	}
	parts = append(parts, collection)
	return strings.Join(parts, "/")
}

func azureBillingResourceStoreKey(scopeID, collection, name string) string {
	return strings.ToLower(scopeID + "|" + collection + "|" + name)
}

func azureBillingETag(seed string, at time.Time) string {
	sum := sha1.Sum([]byte(seed + "|" + at.UTC().Format(time.RFC3339Nano)))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureBillingReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
		return nil, map[string]any{}, true
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

func respondAzureBillingError(w http.ResponseWriter, status int, code, message string) {
	if strings.TrimSpace(code) == "" {
		code = "Error"
	}
	if strings.TrimSpace(message) == "" {
		message = "billing request failed"
	}
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azureBillingString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func azureBillingMap(v any) map[string]any {
	if out, ok := v.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func azureBillingMergeMaps(base map[string]any, updates map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range updates {
		if nested, ok := v.(map[string]any); ok {
			if existing, ok := base[k].(map[string]any); ok {
				base[k] = azureBillingMergeMaps(existing, nested)
				continue
			}
		}
		base[k] = v
	}
	return base
}

func azureBillingCloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		if nested, ok := v.(map[string]any); ok {
			dst[k] = azureBillingCloneMap(nested)
			continue
		}
		if list, ok := v.([]any); ok {
			cloned := make([]any, len(list))
			copy(cloned, list)
			dst[k] = cloned
			continue
		}
		dst[k] = v
	}
	return dst
}

func azureBillingCopyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
