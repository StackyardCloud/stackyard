package server

import (
	"cmp"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

const azureEdgeHardwareCenter20240201APIVersion = "2024-02-01"

// Azure Edge Hardware Center (Microsoft.EdgeOrder 2024-02-01) on the shared
// ARM handler: shipping addresses, order items with cancel and return,
// the orders their orderId names (a view over the items), the product
// catalog actions and the operations list. Nothing ships: an item stays
// Placed until it is cancelled or returned.

const (
	ehwcNamespace    = "Microsoft.EdgeOrder"
	ehwcItemType     = ehwcNamespace + "/orderItems"
	ehwcMaxEmails    = 20
	ehwcMaxListItems = 1000
)

var (
	ehwcOrderIDRE        = regexp.MustCompile(`(?i)^/subscriptions/([^/]+)/resourceGroups/([^/]+)/providers/Microsoft\.EdgeOrder/locations/([^/]+)/orders/([^/]+)$`)
	ehwcClassifications  = []string{"Shipping", "Site"}
	ehwcAddressTypes     = []string{"None", "Residential", "Commercial"}
	ehwcItemTypes        = []string{"Purchase", "Rental", "External"}
	ehwcShipmentTypes    = []string{"CustomerManaged", "MicrosoftManaged"}
	ehwcPatchProperties  = []string{"forwardAddress", "preferences", "notificationEmailList"}
	ehwcReadOnlyDetails  = []string{"currentStage", "cancellationReason", "returnReason", "cancellationStatus", "returnStatus", "deletionStatus"}
	ehwcCatalogProducts  = map[string][]string{"azurestackedgegpu": {"edgep_base", "edgep_high"}, "azurestackedgepro2": {"edgep2_base"}}
	ehwcCatalogFamily    = "azurestackedge"
	ehwcCatalogLine      = "azurestackedge"
	ehwcCatalogCountries = []any{"US", "CA", "GB", "DE"}
)

func init() {
	ns := ehwcNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "addresses",
		Validate: func(payload map[string]any) string {
			return ehwcValidateAddress(azureARMObject(payload["properties"]), "properties.")
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			msg := ""
			if contact, present := props["contactDetails"]; present {
				msg = ehwcValidateContact(azureARMObject(contact), "properties.contactDetails.")
			}
			if ship, present := props["shippingAddress"]; msg == "" && present {
				msg = ehwcValidateShipping(azureARMObject(ship), "properties.shippingAddress.")
			}
			return payload, nil, msg
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := map[string]any{"addressValidationStatus": "Valid"}
			if _, set := rec.Properties["addressClassification"]; !set {
				props["addressClassification"] = "Shipping"
			}
			return props
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "orderItems",
		Validate:           ehwcValidateItem,
		PatchBody:          ehwcItemPatch,
		ReadOnlyProperties: ehwcItemReadOnly,
		Reconcile:          ehwcItemReconcile,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"cancel": func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
				return s.ehwcItemAction(rec, payload, "reason", "cancellationReason", http.StatusNoContent)
			},
			"return": func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
				return s.ehwcItemAction(rec, payload, "returnReason", "returnReason", http.StatusOK)
			},
		},
		ValidateDelete: func(_ *Server, existing *azureARMResource) string {
			if ehwcStage(azureARMObject(existing.Properties["orderItemDetails"])) == "Placed" {
				return "Order item " + existing.Name + " must be cancelled or returned before it is deleted."
			}
			return ""
		},
	})
	orders := func(s *Server, scope string) []map[string]any { return s.ehwcOrders(scope) }
	registerAzureARMView(ns, "orders", orders)
	registerAzureARMView(ns, "locations/orders", orders)
	registerAzureARMProviderAction(ns, "listProductFamilies", func(_ *Server, payload map[string]any) (int, any) {
		if len(azureARMObject(payload["filterableProperties"])) == 0 {
			return azureOracleBad("filterableProperties is required.")
		}
		return http.StatusOK, map[string]any{"value": []any{ehwcFamily(true)}}
	})
	registerAzureARMProviderAction(ns, "productFamiliesMetadata", func(*Server, map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"value": []any{ehwcFamily(false)}}
	})
	registerAzureARMProviderAction(ns, "listConfigurations", ehwcListConfigurations)
	registerAzureARMView(ns, "operations", elasOperations("Microsoft Edge Order", ns, "addresses", "orderItems", "orders"))
}

func ehwcValidateContact(contact map[string]any, path string) string {
	emails := azureARMSlice(contact["emailList"])
	msg := azureWorkloadsRequired(contact, path, "contactName", "phone")
	switch {
	case msg != "":
	case len(emails) == 0:
		msg = path + "emailList is required."
	case len(emails) > ehwcMaxEmails:
		msg = path + "emailList is invalid: at most 20 addresses."
	}
	for _, raw := range emails {
		if email, _ := raw.(string); msg == "" && !strings.Contains(email, "@") {
			msg = path + "emailList is invalid: '" + azureARMString(raw) + "' isn't an email address."
		}
	}
	return msg
}

func ehwcValidateShipping(ship map[string]any, path string) string {
	return cmp.Or(
		azureWorkloadsRequired(ship, path, "streetAddress1", "country"),
		azureLabEnum(ship, path, "addressType", ehwcAddressTypes),
	)
}

// ehwcValidateAddress checks an address: contact details always, and a
// shipping address unless it is a site address.
func ehwcValidateAddress(props map[string]any, path string) string {
	contact, ok := props["contactDetails"].(map[string]any)
	if !ok {
		return path + "contactDetails is required."
	}
	msg := cmp.Or(azureLabEnum(props, path, "addressClassification", ehwcClassifications), ehwcValidateContact(contact, path+"contactDetails."))
	if ship, present := props["shippingAddress"]; msg == "" && present {
		msg = ehwcValidateShipping(azureARMObject(ship), path+"shippingAddress.")
	} else if msg == "" && azureARMString(props["addressClassification"]) != "Site" {
		msg = path + "shippingAddress is required."
	}
	return msg
}

func ehwcValidateItem(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	details, ok := props["orderItemDetails"].(map[string]any)
	if !ok {
		return "properties.orderItemDetails is required."
	}
	for _, key := range ehwcReadOnlyDetails {
		if _, present := details[key]; present {
			return "properties.orderItemDetails." + key + " is read-only."
		}
	}
	hier := azureARMObject(azureARMObject(details["productDetails"])["hierarchyInformation"])
	msg := cmp.Or(
		azureWorkloadsRequired(props, "properties.", "orderId"),
		azureWorkloadsEnums(details, "properties.orderItemDetails.", []string{"orderItemType"}, map[string][]string{"orderItemType": ehwcItemTypes}),
		azureWorkloadsRequired(hier, "properties.orderItemDetails.productDetails.hierarchyInformation.", "productFamilyName", "productLineName", "productName", "configurationName"),
		azureLabEnum(azureARMObject(azureARMObject(details["preferences"])["transportPreferences"]), "properties.orderItemDetails.preferences.transportPreferences.", "preferredShipmentType", ehwcShipmentTypes),
	)
	if msg == "" && !ehwcOrderIDRE.MatchString(azureARMString(props["orderId"])) {
		msg = "properties.orderId is invalid: it must be an order ID ending in /locations/{location}/orders/{name}."
	}
	if msg == "" && !ehwcInCatalog(hier) {
		msg = "properties.orderItemDetails.productDetails.hierarchyInformation is invalid: no such product configuration."
	}
	if msg == "" {
		forward, ok := azureARMObject(props["addressDetails"])["forwardAddress"].(map[string]any)
		if !ok {
			return "properties.addressDetails.forwardAddress is required."
		}
		msg = ehwcValidateAddress(forward, "properties.addressDetails.forwardAddress.")
	}
	return msg
}

// ehwcItemPatch maps the order item update model, whose preferences and
// notification emails sit at the top of properties, onto the resource.
func ehwcItemPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	props := azureARMObject(payload["properties"])
	for key := range props {
		if !slices.Contains(ehwcPatchProperties, key) {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	body := map[string]any{}
	if tags, present := payload["tags"]; present {
		body["tags"] = tags
	}
	update := map[string]any{}
	details := map[string]any{}
	if forward, present := props["forwardAddress"]; present {
		if msg := ehwcValidateAddress(azureARMObject(forward), "properties.forwardAddress."); msg != "" {
			return nil, nil, msg
		}
		update["addressDetails"] = map[string]any{"forwardAddress": forward}
	}
	if prefs, present := props["preferences"]; present {
		if msg := azureLabEnum(azureARMObject(azureARMObject(prefs)["transportPreferences"]), "properties.preferences.transportPreferences.", "preferredShipmentType", ehwcShipmentTypes); msg != "" {
			return nil, nil, msg
		}
		details["preferences"] = prefs
	}
	if emails, present := props["notificationEmailList"]; present {
		if len(azureARMSlice(emails)) > ehwcMaxEmails {
			return nil, nil, "properties.notificationEmailList is invalid: at most 20 addresses."
		}
		details["notificationEmailList"] = emails
	}
	if len(details) > 0 {
		update["orderItemDetails"] = details
	}
	body["properties"] = update
	return body, nil, ""
}

// ehwcStage returns an item's stage: Placed until cancelled or returned.
func ehwcStage(details map[string]any) string {
	switch {
	case azureARMString(details["returnReason"]) != "":
		return "ReturnInitiated"
	case azureARMString(details["cancellationReason"]) != "":
		return "Cancelled"
	}
	return "Placed"
}

func ehwcItemReadOnly(rec *azureARMResource) map[string]any {
	details := azureARMObject(rec.Properties["orderItemDetails"])
	stage := ehwcStage(details)
	open := map[bool]string{true: "Cancellable", false: "NotCancellable"}[stage == "Placed"]
	returnable := map[bool]string{true: "Returnable", false: "NotReturnable"}[stage == "Placed"]
	return map[string]any{
		"startTime": rec.CreatedAt.Format(time.RFC3339),
		"orderItemDetails": map[string]any{
			"currentStage":       map[string]any{"stageName": stage, "stageStatus": "Succeeded", "startTime": rec.UpdatedAt.Format(time.RFC3339)},
			"cancellationStatus": open,
			"returnStatus":       returnable,
			"deletionStatus":     map[bool]string{true: "NotAllowed", false: "Allowed"}[stage == "Placed"],
		},
	}
}

// ehwcItemReconcile keeps an item in its order's subscription, resource
// group and location, fixes its order, and keeps the stage a cancel or
// return set.
func ehwcItemReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	m := ehwcOrderIDRE.FindStringSubmatch(azureARMString(rec.Properties["orderId"]))
	if m == nil {
		return nil, "properties.orderId is invalid."
	}
	if !strings.EqualFold(m[1], rec.SubscriptionID) || !strings.EqualFold(m[2], rec.ResourceGroup) || !strings.EqualFold(strings.ReplaceAll(m[3], " ", ""), strings.ReplaceAll(rec.Location, " ", "")) {
		return nil, "properties.orderId is invalid: the order must be in the order item's subscription, resource group and location."
	}
	orderID := "/subscriptions/" + m[1] + "/resourceGroups/" + m[2] + "/providers/" + ehwcNamespace + "/locations/" + m[3] + "/orders/" + m[4]
	props := map[string]any{"orderId": orderID}
	if existing != nil {
		if !strings.EqualFold(azureARMString(existing.Properties["orderId"]), orderID) {
			return nil, "properties.orderId can't be changed."
		}
		old := azureARMObject(existing.Properties["orderItemDetails"])
		keep := map[string]any{}
		for _, key := range ehwcReadOnlyDetails {
			if v, present := old[key]; present {
				keep[key] = v
			}
		}
		props["orderItemDetails"] = keep
	}
	return props, ""
}

// ehwcItemAction cancels or returns a Placed order item, recording reason.
func (s *Server) ehwcItemAction(rec *azureARMResource, payload map[string]any, field, reasonKey string, status int) (int, any) {
	reason := strings.TrimSpace(azureARMString(payload[field]))
	if reason == "" {
		return azureOracleBad(field + " is required.")
	}
	typ := azureARMResourceTypes[strings.ToLower(ehwcItemType)]
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	if stage := ehwcStage(azureARMObject(current.Properties["orderItemDetails"])); stage != "Placed" {
		return http.StatusConflict, azureARMErrorBody("Conflict", "Order item "+current.Name+" is "+stage+".")
	}
	updated := *current
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(current.Properties, map[string]any{"orderItemDetails": map[string]any{reasonKey: reason}})
	updated.Properties = azureARMProperties(typ, &updated)
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	if status == http.StatusNoContent {
		return status, nil
	}
	return status, updated.fixture()
}

// ehwcOrders lists the orders under scope (a subscription, resource group
// or location) that order items name, with the items each holds.
func (s *Server) ehwcOrders(scope string) []map[string]any {
	prefix := azureARMKey(scope + "/")
	items := s.azureARMResourcesWhere(func(r *azureARMResource) bool {
		return strings.EqualFold(r.Type, ehwcItemType) && strings.HasPrefix(azureARMKey(azureARMString(r.Properties["orderId"])), prefix)
	})
	byOrder := map[string]map[string]any{}
	var out []map[string]any
	for _, item := range items {
		orderID := azureARMString(item.Properties["orderId"])
		order := byOrder[azureARMKey(orderID)]
		if order == nil {
			if len(out) >= ehwcMaxListItems {
				break
			}
			m := ehwcOrderIDRE.FindStringSubmatch(orderID)
			order = map[string]any{
				"id": orderID, "name": m[4], "type": ehwcNamespace + "/orders",
				"properties": map[string]any{"orderItemIds": []any{}, "orderMode": "Default", "location": m[3], "_cancelled": true},
				"systemData": azureARMSystemData(item.CreatedAt, item.UpdatedAt),
			}
			byOrder[azureARMKey(orderID)] = order
			out = append(out, order)
		}
		props := order["properties"].(map[string]any)
		props["orderItemIds"] = append(props["orderItemIds"].([]any), item.ID)
		if ehwcStage(azureARMObject(item.Properties["orderItemDetails"])) != "Cancelled" {
			props["_cancelled"] = false
		}
	}
	for _, order := range out {
		props := order["properties"].(map[string]any)
		stage := map[bool]string{true: "Cancelled", false: "Placed"}[props["_cancelled"] == true]
		delete(props, "_cancelled")
		props["currentStage"] = map[string]any{"stageName": stage, "stageStatus": "Succeeded"}
	}
	return out
}

func ehwcInCatalog(hier map[string]any) bool {
	configs, ok := ehwcCatalogProducts[azureARMString(hier["productName"])]
	return ok && azureARMString(hier["productFamilyName"]) == ehwcCatalogFamily && azureARMString(hier["productLineName"]) == ehwcCatalogLine &&
		slices.Contains(configs, azureARMString(hier["configurationName"]))
}

func ehwcHier(product, config string) map[string]any {
	h := map[string]any{"productFamilyName": ehwcCatalogFamily, "productLineName": ehwcCatalogLine}
	if product != "" {
		h["productName"] = product
	}
	if config != "" {
		h["configurationName"] = config
	}
	return h
}

func ehwcConfiguration(product, config string) map[string]any {
	return map[string]any{"properties": map[string]any{
		"displayName": config, "hierarchyInformation": ehwcHier(product, config),
		"availabilityInformation": map[string]any{"availabilityStage": "Available", "disabledReason": "None"},
		"filterableProperties":    []any{map[string]any{"type": "ShipToCountries", "supportedValues": ehwcCatalogCountries}},
		"dimensions":              map[string]any{"length": 15.5, "width": 17.3, "height": 3.5, "weight": 21, "lengthHeightUnit": "IN", "weightUnit": "LBS"},
	}}
}

// ehwcFamily returns the catalog's product family, with its product line,
// products and (when configs is set) their configurations.
// ponytail: listProductFamilies always expands configurations; the
// provider action hook doesn't see the $expand query.
func ehwcFamily(configs bool) map[string]any {
	var products []any
	for _, product := range slices.Sorted(maps.Keys(ehwcCatalogProducts)) {
		props := map[string]any{"displayName": product, "hierarchyInformation": ehwcHier(product, ""), "availabilityInformation": map[string]any{"availabilityStage": "Available", "disabledReason": "None"}}
		if configs {
			var list []any
			for _, config := range ehwcCatalogProducts[product] {
				list = append(list, ehwcConfiguration(product, config))
			}
			props["configurations"] = list
		}
		products = append(products, map[string]any{"properties": props})
	}
	return map[string]any{"properties": map[string]any{
		"displayName": "Azure Stack Edge", "hierarchyInformation": ehwcHier("", ""),
		"availabilityInformation": map[string]any{"availabilityStage": "Available", "disabledReason": "None"},
		"productLines": []any{map[string]any{"properties": map[string]any{
			"displayName": "Azure Stack Edge", "hierarchyInformation": ehwcHier("", ""), "products": products,
		}}},
	}}
}

func ehwcListConfigurations(_ *Server, payload map[string]any) (int, any) {
	filter, ok := payload["configurationFilter"].(map[string]any)
	if !ok {
		return azureOracleBad("configurationFilter is required.")
	}
	hier, ok := filter["hierarchyInformation"].(map[string]any)
	if !ok {
		return azureOracleBad("configurationFilter.hierarchyInformation is required.")
	}
	value := []any{}
	for _, product := range slices.Sorted(maps.Keys(ehwcCatalogProducts)) {
		configs := ehwcCatalogProducts[product]
		if name := azureARMString(hier["productName"]); name != "" && name != product {
			continue
		}
		if azureARMString(hier["productFamilyName"]) != ehwcCatalogFamily {
			continue
		}
		for _, config := range configs {
			value = append(value, ehwcConfiguration(product, config))
		}
	}
	return http.StatusOK, map[string]any{"value": value}
}
