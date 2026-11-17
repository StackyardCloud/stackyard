package server

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"
)

var (
	azureDataBoxSKUs      = []string{"DataBox", "DataBoxDisk", "DataBoxHeavy", "DataBoxCustomerDisk"}
	azureDataBoxNewModels = []string{"AzureDataBox120", "AzureDataBox525"}
	azureDataBoxTransfers = []string{"ImportToAzure", "ExportFromAzure"}
	azureDataBoxShipments = []string{"MicrosoftManaged", "CustomerManaged"}
	// Characters allowed in device and share passwords besides letters and
	// digits; IilLoO0 are never allowed.
	azureDataBoxPasswordSpecial = "@#-$%^!+=;:_()]"
)

// azureDataBoxJobValidate checks a new job: its device, transfer, contact,
// shipping address, accounts and preferences.
func azureDataBoxJobValidate(payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	name := azureARMString(sku["name"])
	if !slices.Contains(azureDataBoxSKUs, name) {
		return "sku.name must be DataBox, DataBoxDisk, DataBoxHeavy or DataBoxCustomerDisk."
	}
	if model, present := sku["model"]; present && model != name && !(name == "DataBox" && slices.Contains(azureDataBoxNewModels, azureARMString(model))) {
		return "sku.model must be the sku name, or AzureDataBox120 or AzureDataBox525 for DataBox."
	}
	props := azureARMObject(payload["properties"])
	transfer := azureARMString(props["transferType"])
	if !slices.Contains(azureDataBoxTransfers, transfer) {
		return "properties.transferType must be ImportToAzure or ExportFromAzure."
	}
	switch props["deliveryType"] {
	case nil, "NonScheduled":
	case "Scheduled":
		if _, err := time.Parse(time.RFC3339, azureARMString(azureARMObject(props["deliveryInfo"])["scheduledDateTime"])); err != nil {
			return "A Scheduled job needs properties.deliveryInfo.scheduledDateTime."
		}
	default:
		return "properties.deliveryType must be NonScheduled or Scheduled."
	}
	details := azureARMObject(props["details"])
	if azureARMString(details["jobDetailsType"]) != name {
		return "properties.details.jobDetailsType must match sku.name."
	}
	if msg := azureDataBoxContactValidate(azureARMObject(details["contactDetails"])); msg != "" {
		return msg
	}
	if azureARMObject(details["contactDetails"])["contactName"] == nil || azureARMObject(details["contactDetails"])["phone"] == nil ||
		len(azureARMSlice(azureARMObject(details["contactDetails"])["emailList"])) == 0 {
		return "properties.details.contactDetails needs contactName, phone and emailList."
	}
	address := azureARMObject(details["shippingAddress"])
	if address["streetAddress1"] == nil || address["country"] == nil {
		return "properties.details.shippingAddress needs streetAddress1 and country."
	}
	if msg := azureDataBoxAddressValidate("properties.details.shippingAddress", address); msg != "" {
		return msg
	}
	if msg := azureDataBoxTransferValidate(transfer, details); msg != "" {
		return msg
	}
	if msg := azureDataBoxPreferencesValidate(azureARMObject(details["preferences"])); msg != "" {
		return msg
	}
	if raw := details["passkey"]; raw != nil && (name != "DataBoxDisk" || strings.TrimSpace(azureARMString(raw)) == "") {
		return "properties.details.passkey is a non-empty string, for DataBoxDisk jobs only."
	}
	return azureDataBoxKeyValidate(details["keyEncryptionKey"])
}

// azureDataBoxContactValidate checks the contact fields that are present,
// for both create and update.
func azureDataBoxContactValidate(contact map[string]any) string {
	for _, key := range []string{"contactName", "phone"} {
		if raw, present := contact[key]; present && strings.TrimSpace(azureARMString(raw)) == "" {
			return "properties.details.contactDetails." + key + " can't be empty."
		}
	}
	if raw, present := contact["emailList"]; present {
		emails := azureARMSlice(raw)
		if len(emails) == 0 {
			return "properties.details.contactDetails.emailList needs at least one email address."
		}
		for _, email := range emails {
			if !strings.Contains(azureARMString(email), "@") {
				return "properties.details.contactDetails.emailList must hold email addresses."
			}
		}
	}
	return ""
}

// azureDataBoxAddressValidate checks the address fields that are present.
func azureDataBoxAddressValidate(path string, address map[string]any) string {
	for _, key := range []string{"streetAddress1", "country"} {
		if raw, present := address[key]; present && strings.TrimSpace(azureARMString(raw)) == "" {
			return path + "." + key + " can't be empty."
		}
	}
	if raw, present := address["addressType"]; present && !slices.Contains([]string{"None", "Residential", "Commercial"}, azureARMString(raw)) {
		return path + ".addressType must be None, Residential or Commercial."
	}
	return ""
}

// azureDataBoxTransferValidate checks the accounts to import into or
// export from, and the passwords set for them.
func azureDataBoxTransferValidate(transfer string, details map[string]any) string {
	imports, exports := azureARMSlice(details["dataImportDetails"]), azureARMSlice(details["dataExportDetails"])
	items := imports
	if transfer == "ExportFromAzure" {
		if len(exports) == 0 || len(imports) != 0 {
			return "An ExportFromAzure job takes properties.details.dataExportDetails and no dataImportDetails."
		}
		if details["devicePassword"] != nil {
			return "properties.details.devicePassword can't be set for ExportFromAzure jobs."
		}
		items = exports
	} else if len(imports) == 0 || len(exports) != 0 {
		return "An ImportToAzure job takes properties.details.dataImportDetails and no dataExportDetails."
	}
	for _, item := range items {
		account := azureARMObject(azureARMObject(item)["accountDetails"])
		if msg := azureDataBoxAccountValidate(account, transfer); msg != "" {
			return msg
		}
		if transfer == "ExportFromAzure" {
			if msg := azureDataBoxExportConfigValidate(azureARMObject(azureARMObject(item)["transferConfiguration"])); msg != "" {
				return msg
			}
		}
	}
	if raw := details["devicePassword"]; raw != nil && !azureDataBoxPasswordValid(azureARMString(raw)) {
		return "properties.details.devicePassword must be 12 to 64 characters with an uppercase letter, a digit and one of " +
			azureDataBoxPasswordSpecial + ", and can't contain IilLoO0."
	}
	return ""
}

func azureDataBoxAccountValidate(account map[string]any, transfer string) string {
	kind := azureARMString(account["dataAccountType"])
	switch {
	case kind == "" || kind == "StorageAccount":
		if !azureDataBoxStorageAccountID(azureARMString(account["storageAccountId"])) {
			return "accountDetails.storageAccountId must be a storage account ID."
		}
	case kind == "ManagedDisk" && transfer == "ImportToAzure":
		group := strings.Split(strings.Trim(azureARMString(account["resourceGroupId"]), "/"), "/")
		if len(group) != 4 || !strings.EqualFold(group[0], "subscriptions") || !strings.EqualFold(group[2], "resourceGroups") ||
			!azureDataBoxStorageAccountID(azureARMString(account["stagingStorageAccountId"])) {
			return "ManagedDisk accountDetails need a resourceGroupId and a stagingStorageAccountId."
		}
	default:
		return "accountDetails.dataAccountType must be StorageAccount, or ManagedDisk for imports."
	}
	if raw := account["sharePassword"]; raw != nil {
		if transfer == "ExportFromAzure" {
			return "accountDetails.sharePassword can't be set for ExportFromAzure jobs."
		}
		if !azureDataBoxPasswordValid(azureARMString(raw)) {
			return "accountDetails.sharePassword must be 12 to 64 characters with an uppercase letter, a digit and one of " +
				azureDataBoxPasswordSpecial + ", and can't contain IilLoO0."
		}
	}
	return ""
}

func azureDataBoxStorageAccountID(id string) bool {
	parts := strings.Split(strings.Trim(id, "/"), "/")
	return len(parts) == 8 && strings.EqualFold(parts[0], "subscriptions") && strings.EqualFold(parts[2], "resourceGroups") &&
		strings.EqualFold(parts[5], "Microsoft.Storage") && strings.EqualFold(parts[6], "storageAccounts") && parts[7] != ""
}

func azureDataBoxExportConfigValidate(config map[string]any) string {
	switch config["transferConfigurationType"] {
	case "TransferAll":
		if len(azureARMObject(config["transferAllDetails"])) == 0 {
			return "A TransferAll transferConfiguration needs transferAllDetails."
		}
	case "TransferUsingFilter":
		if len(azureARMObject(config["transferFilterDetails"])) == 0 {
			return "A TransferUsingFilter transferConfiguration needs transferFilterDetails."
		}
	default:
		return "transferConfiguration.transferConfigurationType must be TransferAll or TransferUsingFilter."
	}
	return ""
}

// azureDataBoxPasswordValid applies the device and share password rules.
func azureDataBoxPasswordValid(password string) bool {
	if len(password) < 12 || len(password) > 64 || strings.ContainsAny(password, "IilLoO0") {
		return false
	}
	var upper, digit, special bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			upper = true
		case c >= '0' && c <= '9':
			digit = true
		case strings.ContainsRune(azureDataBoxPasswordSpecial, c):
			special = true
		case c < 'a' || c > 'z':
			return false
		}
	}
	return upper && digit && special
}

func azureDataBoxPreferencesValidate(preferences map[string]any) string {
	transport := azureARMObject(preferences["transportPreferences"])
	if raw, present := transport["preferredShipmentType"]; present && !slices.Contains(azureDataBoxShipments, azureARMString(raw)) {
		return "preferences.transportPreferences.preferredShipmentType must be MicrosoftManaged or CustomerManaged."
	}
	encryption := azureARMObject(preferences["encryptionPreferences"])
	if raw, present := encryption["doubleEncryption"]; present && raw != "Enabled" && raw != "Disabled" {
		return "preferences.encryptionPreferences.doubleEncryption must be Enabled or Disabled."
	}
	return ""
}

func azureDataBoxKeyValidate(raw any) string {
	key := azureARMObject(raw)
	switch key["kekType"] {
	case nil, "MicrosoftManaged":
	case "CustomerManaged":
		if strings.TrimSpace(azureARMString(key["kekUrl"])) == "" || strings.TrimSpace(azureARMString(key["kekVaultResourceID"])) == "" {
			return "A CustomerManaged keyEncryptionKey needs kekUrl and kekVaultResourceID."
		}
	default:
		return "keyEncryptionKey.kekType must be MicrosoftManaged or CustomerManaged."
	}
	return ""
}

// azureDataBoxRegisterServiceActions registers the pre-order checks at
// subscription and resource group scope.
func azureDataBoxRegisterServiceActions() {
	registerAzureARMGroupAction(azureDataBoxNamespace, "availableSkus", func(_ *Server, _, _ string, payload map[string]any) (int, any) {
		return azureDataBoxAvailableSkus(payload)
	})
	for _, action := range []struct {
		name string
		fn   func(map[string]any) (int, any)
	}{{"validateInputs", azureDataBoxValidateInputs}, {"regionConfiguration", azureDataBoxRegionConfiguration}} {
		fn := action.fn
		registerAzureARMProviderAction(azureDataBoxNamespace, "locations/"+action.name, func(_ *Server, payload map[string]any) (int, any) { return fn(payload) })
		registerAzureARMGroupAction(azureDataBoxNamespace, action.name, func(_ *Server, _, _ string, payload map[string]any) (int, any) { return fn(payload) })
	}
}

// ponytail: every SKU is available everywhere with sample capacities; per
// region availability is the upgrade if tests need disabled SKUs.
var azureDataBoxCapacity = map[string]map[string]any{
	"DataBox":             {"maximum": "100", "usable": "80", "individualSkuUsable": "80"},
	"DataBoxDisk":         {"maximum": "40", "usable": "35", "individualSkuUsable": "7"},
	"DataBoxHeavy":        {"maximum": "1000", "usable": "800", "individualSkuUsable": "400"},
	"DataBoxCustomerDisk": {"maximum": "40", "usable": "35", "individualSkuUsable": "7"},
}

func azureDataBoxAvailableSkus(payload map[string]any) (int, any) {
	country, location := azureARMString(payload["country"]), strings.TrimSpace(azureARMString(payload["location"]))
	var msg string
	switch {
	case !slices.Contains(azureDataBoxTransfers, azureARMString(payload["transferType"])):
		msg = "transferType must be ImportToAzure or ExportFromAzure."
	case len(country) != 2 || strings.Trim(strings.ToUpper(country), "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "":
		msg = "country must be a two-letter ISO country code."
	case location == "":
		msg = "location is required."
	}
	names := azureDataBoxSKUs
	if raw, present := payload["skuNames"]; present {
		items, isList := raw.([]any)
		if !isList {
			msg = "skuNames must be a list."
		}
		names = nil
		for _, item := range items {
			if !slices.Contains(azureDataBoxSKUs, azureARMString(item)) || slices.Contains(names, azureARMString(item)) {
				msg = "skuNames must hold distinct names among DataBox, DataBoxDisk, DataBoxHeavy and DataBoxCustomerDisk."
			}
			names = append(names, azureARMString(item))
		}
	}
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	items := []any{}
	for _, name := range names {
		items = append(items, map[string]any{
			"enabled": true, "sku": map[string]any{"name": name, "model": name},
			"properties": map[string]any{
				"apiVersions": []any{"2025-07-01"}, "capacity": maps.Clone(azureDataBoxCapacity[name]), "disabledReason": "None",
				"countriesWithinCommerceBoundary":  []any{strings.ToUpper(country)},
				"dataLocationToServiceLocationMap": []any{map[string]any{"dataLocation": location, "serviceLocation": location}},
			},
		})
	}
	return http.StatusOK, map[string]any{"value": items}
}

// azureDataBoxValidateInputs runs the pre-order checks the emulator can:
// address, transfer details and preferences; the subscription, order limit
// and SKU checks always pass.
func azureDataBoxValidateInputs(payload map[string]any) (int, any) {
	requests := azureARMSlice(payload["individualRequestDetails"])
	if payload["validationCategory"] != "JobCreationValidation" || len(requests) == 0 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "validationCategory must be JobCreationValidation, with individualRequestDetails.")
	}
	overall := "AllValidToProceed"
	responses := []any{}
	for _, raw := range requests {
		request := azureARMObject(raw)
		kind := azureARMString(request["validationType"])
		if raw, present := request["deviceType"]; present && !slices.Contains(azureDataBoxSKUs, azureARMString(raw)) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "deviceType must be DataBox, DataBoxDisk, DataBoxHeavy or DataBoxCustomerDisk.")
		}
		var msg string
		switch kind {
		case "ValidateAddress":
			address := azureARMObject(request["shippingAddress"])
			if address["streetAddress1"] == nil || address["country"] == nil {
				msg = "shippingAddress needs streetAddress1 and country."
			} else {
				msg = azureDataBoxAddressValidate("shippingAddress", address)
			}
		case "ValidateDataTransferDetails":
			msg = azureDataBoxTransferValidate(azureARMString(request["transferType"]), request)
			if !slices.Contains(azureDataBoxTransfers, azureARMString(request["transferType"])) {
				msg = "transferType must be ImportToAzure or ExportFromAzure."
			}
		case "ValidatePreferences":
			msg = azureDataBoxPreferencesValidate(azureARMObject(request["preference"]))
		case "ValidateSubscriptionIsAllowedToCreateJob", "ValidateCreateOrderLimit", "ValidateSkuAvailability":
		default:
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "validationType "+kind+" isn't supported.")
		}
		status, errBody := "Valid", map[string]any{"code": "Success", "details": []any{}, "additionalInfo": []any{}}
		if msg != "" {
			status, overall = "Invalid", "InputsRevisitRequired"
			errBody = map[string]any{"code": "InvalidRequestContent", "message": msg, "details": []any{}, "additionalInfo": []any{}}
		}
		response := map[string]any{"validationType": kind, "error": errBody, "status": status}
		if kind == "ValidateAddress" {
			delete(response, "status")
			response["validationStatus"] = status
			if msg == "" {
				response["alternateAddresses"] = []any{request["shippingAddress"]}
			}
		}
		responses = append(responses, response)
	}
	return http.StatusOK, map[string]any{"properties": map[string]any{"status": overall, "individualResponseDetails": responses}}
}

// azureDataBoxRegionConfiguration answers the transport, schedule and
// device capability requests; datacenter addresses aren't emulated.
func azureDataBoxRegionConfiguration(payload map[string]any) (int, any) {
	out := map[string]any{}
	for key, request := range payload {
		if request == nil {
			continue
		}
		if !slices.Contains(azureDataBoxSKUs, azureARMString(azureARMObject(request)["skuName"])) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", key+".skuName must be DataBox, DataBoxDisk, DataBoxHeavy or DataBoxCustomerDisk.")
		}
	}
	if payload["transportAvailabilityRequest"] != nil {
		out["transportAvailabilityResponse"] = map[string]any{"transportAvailabilityDetails": []any{
			map[string]any{"shipmentType": "MicrosoftManaged"}, map[string]any{"shipmentType": "CustomerManaged"}}}
	}
	if payload["scheduleAvailabilityRequest"] != nil {
		start := time.Now().UTC().Truncate(24 * time.Hour)
		dates := []any{}
		for day := 1; day <= 21; day++ {
			dates = append(dates, start.AddDate(0, 0, day).Format(time.RFC3339))
		}
		out["scheduleAvailabilityResponse"] = map[string]any{"availableDates": dates}
	}
	if request := azureARMObject(payload["deviceCapabilityRequest"]); len(request) != 0 {
		encryption := "Disabled"
		if request["skuName"] == "DataBoxDisk" {
			encryption = "Enabled"
		}
		out["deviceCapabilityResponse"] = map[string]any{"deviceCapabilityDetails": []any{map[string]any{"hardwareEncryption": encryption}}}
	}
	return http.StatusOK, out
}
