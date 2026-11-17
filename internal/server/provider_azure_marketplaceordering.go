package server

import (
	"net/http"
	"strings"
	"time"
)

// Microsoft.MarketplaceOrdering: the terms of a marketplace virtual machine
// image and the subscription's agreement to them. Terms exist for every
// publisher, offer and plan, unaccepted until a PUT accepts them or the
// agreement is signed; accepting signs the agreement and declining cancels
// it. Agreements are stored as records of an unregistered type, so only
// this router serves them.
// ponytail: any publisher, offer and plan has terms; a catalog of real
// images is the upgrade if tests need unknown plans rejected.

const (
	azureMarketplaceNamespace     = "Microsoft.MarketplaceOrdering"
	azureMarketplaceAgreementType = azureMarketplaceNamespace + "/agreements"
)

func (s *Server) handleAzureMarketplaceOrderingRouter(w http.ResponseWriter, r *http.Request) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	if len(seg) < 5 || !strings.EqualFold(seg[0], "subscriptions") || !strings.EqualFold(seg[2], "providers") || !strings.EqualFold(seg[3], azureMarketplaceNamespace) {
		return false
	}
	sub, rest := seg[1], seg[4:]
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	is := func(i int, word string) bool { return strings.EqualFold(rest[i], word) }
	switch {
	case len(rest) == 10 && is(0, "offerTypes") && is(2, "publishers") && is(4, "offers") && is(6, "plans") && is(8, "agreements") && is(9, "current"):
		if !strings.EqualFold(rest[1], "virtualmachine") {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "offerType must be virtualmachine.")
			return true
		}
		s.serveAzureMarketplaceTerms(w, r, sub, rest[3], rest[5], rest[7])
	case len(rest) == 1 && is(0, "agreements"):
		if r.Method != http.MethodGet {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return true
		}
		prefix := azureARMKey("/subscriptions/" + sub + "/providers/" + azureMarketplaceAgreementType + "/")
		items := []any{}
		for _, rec := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			return rec.Type == azureMarketplaceAgreementType && strings.HasPrefix(azureARMKey(rec.ID), prefix)
		}) {
			items = append(items, azureMarketplaceAgreement(rec))
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
	case (len(rest) == 6 || len(rest) == 7) && is(0, "agreements") && is(2, "offers") && is(4, "plans"):
		verb := ""
		if len(rest) == 7 {
			verb = strings.ToLower(rest[6])
		}
		s.serveAzureMarketplaceAgreement(w, r, sub, rest[1], rest[3], rest[5], verb)
	default:
		return false
	}
	return true
}

func azureMarketplaceAgreementID(sub, publisher, offer, plan string) string {
	return "/subscriptions/" + sub + "/providers/" + azureMarketplaceAgreementType + "/" + publisher + "/offers/" + offer + "/plans/" + plan
}

func (s *Server) serveAzureMarketplaceTerms(w http.ResponseWriter, r *http.Request, sub, publisher, offer, plan string) {
	id := azureMarketplaceAgreementID(sub, publisher, offer, plan)
	switch r.Method {
	case http.MethodGet:
		s.providerStorageMu.Lock()
		rec := s.azureARMResources[azureARMKey(id)]
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, azureMarketplaceTerms(sub, publisher, offer, plan, rec))
	case http.MethodPut:
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		accepted, isBool := azureARMObject(payload["properties"])["accepted"].(bool)
		if !isBool {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "properties.accepted must be true or false.")
			return
		}
		state := "Canceled"
		if accepted {
			state = "Active"
		}
		s.providerStorageMu.Lock()
		rec := s.azureARMResources[azureARMKey(id)]
		// Declining terms never accepted leaves no agreement behind.
		if accepted || rec != nil {
			rec = s.azureMarketplaceSetStateLocked(id, sub, publisher, offer, plan, state)
		}
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, azureMarketplaceTerms(sub, publisher, offer, plan, rec))
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) serveAzureMarketplaceAgreement(w http.ResponseWriter, r *http.Request, sub, publisher, offer, plan, verb string) {
	id := azureMarketplaceAgreementID(sub, publisher, offer, plan)
	var state string
	switch {
	case verb == "" && r.Method == http.MethodGet:
	case verb == "sign" && r.Method == http.MethodPost:
		state = "Active"
	case verb == "cancel" && r.Method == http.MethodPost:
		state = "Canceled"
	case verb == "" || verb == "sign" || verb == "cancel":
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	default:
		azureARMNotFound(w, "ResourceNotFound", id+"/"+verb, "")
		return
	}
	s.providerStorageMu.Lock()
	rec := s.azureARMResources[azureARMKey(id)]
	if state == "Active" || (state == "Canceled" && rec != nil) {
		rec = s.azureMarketplaceSetStateLocked(id, sub, publisher, offer, plan, state)
	}
	s.providerStorageMu.Unlock()
	if rec == nil {
		// Never signed, so there is no agreement to read or cancel.
		azureARMNotFound(w, "ResourceNotFound", id, "")
		return
	}
	respondJSON(w, http.StatusOK, azureMarketplaceAgreement(rec))
}

// azureMarketplaceSetStateLocked stores the agreement in state and returns
// it, keeping its sign or cancel date when it is already in that state.
// Caller holds providerStorageMu.
func (s *Server) azureMarketplaceSetStateLocked(id, sub, publisher, offer, plan, state string) *azureARMResource {
	existing := s.azureARMResources[azureARMKey(id)]
	if existing != nil && existing.Properties["state"] == state {
		return existing
	}
	now := time.Now().UTC()
	rec := &azureARMResource{
		ID: id, SubscriptionID: sub, Name: plan, Type: azureMarketplaceAgreementType, CreatedAt: now, UpdatedAt: now,
		Properties: map[string]any{"publisher": publisher, "offer": offer, "plan": plan, "state": state},
	}
	if existing != nil {
		rec.ID, rec.Name, rec.CreatedAt = existing.ID, existing.Name, existing.CreatedAt
		rec.Properties = azureARMMerge(existing.Properties, map[string]any{"state": state})
	}
	if state == "Active" {
		rec.Properties["signDate"] = now.Format(time.RFC3339Nano)
	} else {
		rec.Properties["cancelDate"] = now.Format(time.RFC3339Nano)
	}
	s.azureARMResources[azureARMKey(id)] = rec
	return rec
}

func azureMarketplaceAgreement(rec *azureARMResource) map[string]any {
	props := map[string]any{"id": rec.ID, "publisher": rec.Properties["publisher"], "offer": rec.Properties["offer"], "state": rec.Properties["state"]}
	if rec.Properties["state"] == "Active" {
		props["signDate"] = rec.Properties["signDate"]
	} else {
		props["cancelDate"] = rec.Properties["cancelDate"]
	}
	return map[string]any{"id": rec.ID, "name": rec.Name, "type": azureMarketplaceAgreementType, "properties": props}
}

// azureMarketplaceTerms returns a plan's terms, accepted while its
// agreement is active. The signature is derived from the plan, and the
// links are placeholders, since the emulator has no terms documents.
func azureMarketplaceTerms(sub, publisher, offer, plan string, rec *azureARMResource) map[string]any {
	accepted := rec != nil && rec.Properties["state"] == "Active"
	retrieved := ""
	if accepted {
		retrieved = azureARMString(rec.Properties["signDate"])
	}
	key := strings.ToUpper(publisher + "%253a24" + offer + "%253a24" + plan)
	signature := strings.ToUpper(strings.NewReplacer("+", "", "/", "", "=", "").Replace(azureARMDerivedKey(key, "terms")))
	return map[string]any{
		"id": "/subscriptions/" + sub + "/providers/" + azureMarketplaceNamespace + "/offerTypes/VirtualMachine/publishers/" + publisher +
			"/offers/" + offer + "/plans/" + plan + "/agreements/current",
		"name": plan, "type": azureMarketplaceNamespace + "/offertypes",
		"properties": map[string]any{
			"publisher": publisher, "product": offer, "plan": plan, "accepted": accepted, "retrieveDatetime": retrieved, "signature": signature,
			"licenseTextLink":      "https://marketplace-terms.stackyard.invalid/legalterms/" + key + ".txt",
			"privacyPolicyLink":    "https://marketplace-terms.stackyard.invalid/privacy/" + key + ".txt",
			"marketplaceTermsLink": "https://marketplace-terms.stackyard.invalid/marketplaceterms/" + signature + ".txt",
		},
	}
}
