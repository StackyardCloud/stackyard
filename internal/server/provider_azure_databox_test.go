package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

const azureDataBoxTestAPIVersion = "?api-version=2025-07-01"

const azureDataBoxTestAccount = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-data/providers/Microsoft.Storage/storageAccounts/ingest"

// azureDataBoxTestJob returns a DataBox import job body with changes
// applied to its details.
func azureDataBoxTestJob(t *testing.T, change func(job, details map[string]any)) string {
	t.Helper()
	details := map[string]any{
		"jobDetailsType": "DataBox",
		"contactDetails": map[string]any{"contactName": "Avery Chen", "phone": "5550100", "emailList": []any{"ops@contoso.com"}},
		"shippingAddress": map[string]any{"streetAddress1": "1 Main St", "city": "Redmond", "stateOrProvince": "WA", "country": "US",
			"postalCode": "98052", "addressType": "Commercial"},
		"dataImportDetails": []any{map[string]any{"accountDetails": map[string]any{"dataAccountType": "StorageAccount",
			"storageAccountId": azureDataBoxTestAccount, "sharePassword": "Share#Pa55wrdXY"}}},
		"devicePassword": "Secure#Pa55wrdX",
	}
	job := map[string]any{"location": "westus", "sku": map[string]any{"name": "DataBox", "model": "DataBox"},
		"properties": map[string]any{"transferType": "ImportToAzure", "details": details}}
	if change != nil {
		change(job, details)
	}
	body, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestAzureDataBoxJobLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-databox")
	job := sub + "/resourceGroups/rg-databox/providers/Microsoft.DataBox/jobs/import-1"

	created := do(http.MethodPut, job+azureDataBoxTestAPIVersion, azureDataBoxTestJob(t, nil))
	azureResourcesExpectStatus(t, created, http.StatusOK)
	body := providerContractJSONMap(t, created)
	props := body["properties"].(map[string]any)
	details := props["details"].(map[string]any)
	stages := details["jobStages"].([]any)
	if body["type"] != "Microsoft.DataBox/jobs" || props["status"] != "DeviceOrdered" || props["isCancellable"] != true || props["isDeletable"] != false ||
		props["isShippingAddressEditable"] != true || props["deliveryType"] != "NonScheduled" || props["startTime"] == nil || len(stages) != 8 ||
		stages[0].(map[string]any)["stageStatus"] != "Succeeded" || stages[6].(map[string]any)["stageName"] != "DataCopy" {
		t.Fatalf("unexpected job: %#v", body)
	}
	if details["devicePassword"] != nil || details["dataImportDetails"].([]any)[0].(map[string]any)["accountDetails"].(map[string]any)["sharePassword"] != nil {
		t.Fatalf("passwords must not be returned: %#v", details)
	}
	// PUT only creates; PATCH updates.
	azureSentinelExpectRejected(t, do(http.MethodPut, job+azureDataBoxTestAPIVersion, azureDataBoxTestJob(t, nil)), "update it with PATCH")

	patched := providerContractJSONMap(t, do(http.MethodPatch, job+azureDataBoxTestAPIVersion,
		`{"tags":{"team":"data"},"properties":{"details":{"contactDetails":{"phone":"5550199"},"shippingAddress":{"streetAddress1":"2 Main St"}}}}`))
	patchedDetails := patched["properties"].(map[string]any)["details"].(map[string]any)
	if patched["tags"].(map[string]any)["team"] != "data" || patchedDetails["contactDetails"].(map[string]any)["phone"] != "5550199" ||
		patchedDetails["contactDetails"].(map[string]any)["contactName"] != "Avery Chen" || patchedDetails["shippingAddress"].(map[string]any)["streetAddress1"] != "2 Main St" {
		t.Fatalf("unexpected patched job: %#v", patched)
	}
	// A PATCH without tags keeps them.
	kept := providerContractJSONMap(t, do(http.MethodPatch, job+azureDataBoxTestAPIVersion, `{"properties":{"details":{"contactDetails":{"phone":"5550123"}}}}`))
	if kept["tags"].(map[string]any)["team"] != "data" {
		t.Fatalf("PATCH without tags should keep them: %#v", kept)
	}
	for payload, want := range map[string]string{
		`{"properties":{"transferType":"ExportFromAzure"}}`:                             "properties.transferType can't be updated",
		`{"properties":{"details":{"devicePassword":"x"}}}`:                             "properties.details.devicePassword can't be updated",
		`{"properties":{"details":{"contactDetails":{"emailList":[]}}}}`:                "at least one email",
		`{"properties":{"details":{"shippingAddress":{"addressType":"Office"}}}}`:       "addressType",
		`{"properties":{"details":{"keyEncryptionKey":{"kekType":"CustomerManaged"}}}}`: "kekUrl",
		`{"properties":{"details":"x"}}`:                                                "properties.details must be an object",
		`{"properties":{"details":{"contactDetails":null}}}`:                            "contactDetails must be an object",
		`{"properties":{"details":{"preferences":"x"}}}`:                                "preferences must be an object",
		`{"properties":"x"}`: "properties must be an object",
		`{"tags":["x"]}`:     "tags must be an object",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPatch, job+azureDataBoxTestAPIVersion, payload), want)
	}

	azureResourcesExpectErrorCode(t, do(http.MethodDelete, job+azureDataBoxTestAPIVersion, ""), http.StatusConflict, "Conflict")
	azureSentinelExpectRejected(t, do(http.MethodPost, job+"/cancel"+azureDataBoxTestAPIVersion, `{"reason":" "}`), "reason is required")
	azureResourcesExpectStatus(t, do(http.MethodPost, job+"/cancel"+azureDataBoxTestAPIVersion, `{"reason":"No longer needed"}`), http.StatusNoContent)
	cancelled := providerContractJSONMap(t, do(http.MethodGet, job+azureDataBoxTestAPIVersion, ""))
	cp := cancelled["properties"].(map[string]any)
	cancelledStages := cp["details"].(map[string]any)["jobStages"].([]any)
	if cp["status"] != "Cancelled" || cp["cancellationReason"] != "No longer needed" || cp["isCancellable"] != false || cp["isDeletable"] != true ||
		cp["isShippingAddressEditable"] != false || len(cancelledStages) != 2 || cancelledStages[1].(map[string]any)["stageName"] != "Cancelled" {
		t.Fatalf("unexpected cancelled job: %#v", cp)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, job+"/cancel"+azureDataBoxTestAPIVersion, `{"reason":"again"}`), http.StatusConflict, "Conflict")
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, job+azureDataBoxTestAPIVersion, `{"properties":{"details":{"contactDetails":{"phone":"1"}}}}`), http.StatusConflict, "Conflict")
	cancelTime := cancelledStages[1].(map[string]any)["stageTime"]
	time.Sleep(1100 * time.Millisecond)
	retagged := providerContractJSONMap(t, do(http.MethodPatch, job+azureDataBoxTestAPIVersion, `{"tags":{"state":"done"}}`))
	if got := retagged["properties"].(map[string]any)["details"].(map[string]any)["jobStages"].([]any)[1].(map[string]any)["stageTime"]; got != cancelTime {
		t.Fatalf("a tag update moved the cancellation time from %v to %v", cancelTime, got)
	}

	list := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.DataBox/jobs"+azureDataBoxTestAPIVersion, ""))
	if items := list["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one job in the subscription, got %#v", items)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, job+azureDataBoxTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, job+azureDataBoxTestAPIVersion, ""), http.StatusNoContent)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, job+"/listCredentials"+azureDataBoxTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
}

func TestAzureDataBoxCredentials(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-databox")
	jobs := sub + "/resourceGroups/rg-databox/providers/Microsoft.DataBox/jobs/"
	azureResourcesExpectStatus(t, do(http.MethodPut, jobs+"set"+azureDataBoxTestAPIVersion, azureDataBoxTestJob(t, nil)), http.StatusOK)
	secrets := providerContractJSONMap(t, do(http.MethodPost, jobs+"set/listCredentials"+azureDataBoxTestAPIVersion, ""))["value"].([]any)[0].(map[string]any)
	jobSecrets := secrets["jobSecrets"].(map[string]any)
	pod := jobSecrets["podSecrets"].([]any)[0].(map[string]any)
	account := pod["accountCredentialDetails"].([]any)[0].(map[string]any)
	shares := account["shareCredentialDetails"].([]any)
	if secrets["jobName"] != "set" || jobSecrets["jobSecretsType"] != "DataBox" || pod["devicePassword"] != "Secure#Pa55wrdX" ||
		len(pod["networkConfigurations"].([]any)) != 3 || account["accountName"] != "ingest" || len(shares) != 3 ||
		shares[0].(map[string]any)["shareName"] != "ingest_BlockBlob" || shares[0].(map[string]any)["password"] != "Share#Pa55wrdXY" {
		t.Fatalf("credentials should carry the passwords the job set: %#v", secrets)
	}

	// Without passwords, the service generates ones that meet the rules.
	heavy := azureDataBoxTestJob(t, func(job, details map[string]any) {
		job["sku"] = map[string]any{"name": "DataBoxHeavy"}
		details["jobDetailsType"] = "DataBoxHeavy"
		delete(details, "devicePassword")
		details["dataImportDetails"] = []any{map[string]any{"accountDetails": map[string]any{"dataAccountType": "ManagedDisk",
			"resourceGroupId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-disks", "stagingStorageAccountId": azureDataBoxTestAccount}}}
	})
	azureResourcesExpectStatus(t, do(http.MethodPut, jobs+"heavy"+azureDataBoxTestAPIVersion, heavy), http.StatusOK)
	heavySecrets := providerContractJSONMap(t, do(http.MethodPost, jobs+"heavy/listCredentials"+azureDataBoxTestAPIVersion, ""))["value"].([]any)[0].(map[string]any)["jobSecrets"].(map[string]any)
	cabinet := heavySecrets["cabinetPodSecrets"].([]any)
	node := cabinet[0].(map[string]any)
	disk := node["accountCredentialDetails"].([]any)[0].(map[string]any)
	if len(cabinet) != 2 || !azureDataBoxPasswordValid(node["devicePassword"].(string)) || disk["accountName"] != "rg-disks" ||
		disk["shareCredentialDetails"].([]any)[0].(map[string]any)["shareType"] != "ManagedDisk" || node["deviceSerialNumber"] == cabinet[1].(map[string]any)["deviceSerialNumber"] {
		t.Fatalf("unexpected heavy credentials: %#v", heavySecrets)
	}

	disk2 := azureDataBoxTestJob(t, func(job, details map[string]any) {
		job["sku"] = map[string]any{"name": "DataBoxDisk"}
		details["jobDetailsType"] = "DataBoxDisk"
		details["passkey"] = "my-own-passkey"
		delete(details, "devicePassword")
	})
	azureResourcesExpectStatus(t, do(http.MethodPut, jobs+"disk"+azureDataBoxTestAPIVersion, disk2), http.StatusOK)
	diskSecrets := providerContractJSONMap(t, do(http.MethodPost, jobs+"disk/listCredentials"+azureDataBoxTestAPIVersion, ""))["value"].([]any)[0].(map[string]any)["jobSecrets"].(map[string]any)
	if diskSecrets["passKey"] != "my-own-passkey" || diskSecrets["isPasskeyUserDefined"] != true || len(diskSecrets["diskSecrets"].([]any)) != 1 {
		t.Fatalf("unexpected disk credentials: %#v", diskSecrets)
	}
}

func TestAzureDataBoxJobValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-databox")
	job := sub + "/resourceGroups/rg-databox/providers/Microsoft.DataBox/jobs/"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, job+"ab"+azureDataBoxTestAPIVersion, azureDataBoxTestJob(t, nil)), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, job+"has%20space"+azureDataBoxTestAPIVersion, azureDataBoxTestJob(t, nil)), http.StatusBadRequest, "InvalidResourceName")
	cases := map[string]struct {
		change func(job, details map[string]any)
		want   string
	}{
		"sku": {func(job, _ map[string]any) { job["sku"] = map[string]any{"name": "DataBoxMini"} }, "sku.name"},
		"model": {func(job, _ map[string]any) {
			job["sku"] = map[string]any{"name": "DataBoxDisk", "model": "AzureDataBox120"}
		}, "sku.model"},
		"new model ok": {func(job, _ map[string]any) {
			job["sku"] = map[string]any{"name": "DataBox", "model": "AzureDataBox525"}
		}, ""},
		"transfer":     {func(job, _ map[string]any) { job["properties"].(map[string]any)["transferType"] = "Both" }, "transferType"},
		"delivery":     {func(job, _ map[string]any) { job["properties"].(map[string]any)["deliveryType"] = "Express" }, "deliveryType"},
		"scheduled":    {func(job, _ map[string]any) { job["properties"].(map[string]any)["deliveryType"] = "Scheduled" }, "scheduledDateTime"},
		"details type": {func(_, d map[string]any) { d["jobDetailsType"] = "DataBoxDisk" }, "jobDetailsType"},
		"contact":      {func(_, d map[string]any) { delete(d["contactDetails"].(map[string]any), "phone") }, "needs contactName, phone and emailList"},
		"blank name":   {func(_, d map[string]any) { d["contactDetails"].(map[string]any)["contactName"] = " " }, "contactName can't be empty"},
		"email":        {func(_, d map[string]any) { d["contactDetails"].(map[string]any)["emailList"] = []any{"ops"} }, "email addresses"},
		"address":      {func(_, d map[string]any) { delete(d["shippingAddress"].(map[string]any), "country") }, "streetAddress1 and country"},
		"address type": {func(_, d map[string]any) { d["shippingAddress"].(map[string]any)["addressType"] = "Office" }, "addressType"},
		"no imports":   {func(_, d map[string]any) { delete(d, "dataImportDetails") }, "takes properties.details.dataImportDetails"},
		"account id": {func(_, d map[string]any) {
			d["dataImportDetails"] = []any{map[string]any{"accountDetails": map[string]any{"storageAccountId": "/subscriptions/x/resourceGroups/y"}}}
		}, "storageAccountId"},
		"managed disk": {func(_, d map[string]any) {
			d["dataImportDetails"] = []any{map[string]any{"accountDetails": map[string]any{"dataAccountType": "ManagedDisk", "resourceGroupId": "rg"}}}
		}, "resourceGroupId and a stagingStorageAccountId"},
		"account type": {func(_, d map[string]any) {
			d["dataImportDetails"] = []any{map[string]any{"accountDetails": map[string]any{"dataAccountType": "Blob"}}}
		}, "dataAccountType"},
		"null password":   {func(_, d map[string]any) { d["devicePassword"] = nil }, ""},
		"passkey on box":  {func(_, d map[string]any) { d["passkey"] = "my-own-passkey" }, "passkey"},
		"short password":  {func(_, d map[string]any) { d["devicePassword"] = "Sh0rt#" }, "devicePassword"},
		"barred password": {func(_, d map[string]any) { d["devicePassword"] = "Secure#Pa55wrdL" }, "devicePassword"},
		"share password": {func(_, d map[string]any) {
			d["dataImportDetails"].([]any)[0].(map[string]any)["accountDetails"].(map[string]any)["sharePassword"] = "nuppercase#55xx"
		}, "sharePassword"},
		"shipment": {func(_, d map[string]any) {
			d["preferences"] = map[string]any{"transportPreferences": map[string]any{"preferredShipmentType": "Drone"}}
		}, "preferredShipmentType"},
		"encryption": {func(_, d map[string]any) {
			d["preferences"] = map[string]any{"encryptionPreferences": map[string]any{"doubleEncryption": "On"}}
		}, "doubleEncryption"},
		"kek":                 {func(_, d map[string]any) { d["keyEncryptionKey"] = map[string]any{"kekType": "Vault"} }, "kekType"},
		"export with imports": {func(job, _ map[string]any) { job["properties"].(map[string]any)["transferType"] = "ExportFromAzure" }, "takes properties.details.dataExportDetails"},
		"export password": {func(job, d map[string]any) {
			job["properties"].(map[string]any)["transferType"] = "ExportFromAzure"
			d["dataExportDetails"], d["dataImportDetails"] = azureDataBoxTestExport("TransferAll"), nil
		}, "devicePassword can't be set"},
		"export config": {func(job, d map[string]any) {
			job["properties"].(map[string]any)["transferType"] = "ExportFromAzure"
			d["dataExportDetails"], d["dataImportDetails"] = azureDataBoxTestExport("TransferSome"), nil
			delete(d, "devicePassword")
		}, "transferConfigurationType"},
		"both directions": {func(_, d map[string]any) { d["dataExportDetails"] = azureDataBoxTestExport("TransferAll") }, "takes properties.details.dataImportDetails and no dataExportDetails"},
		"export both": {func(job, d map[string]any) {
			job["properties"].(map[string]any)["transferType"] = "ExportFromAzure"
			d["dataExportDetails"] = azureDataBoxTestExport("TransferAll")
			delete(d, "devicePassword")
		}, "takes properties.details.dataExportDetails and no dataImportDetails"},
		"export disks": {func(job, d map[string]any) {
			job["properties"].(map[string]any)["transferType"] = "ExportFromAzure"
			export := azureDataBoxTestExport("TransferAll")
			export[0].(map[string]any)["accountDetails"] = map[string]any{"dataAccountType": "ManagedDisk",
				"resourceGroupId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-disks", "stagingStorageAccountId": azureDataBoxTestAccount}
			d["dataExportDetails"], d["dataImportDetails"] = export, nil
			delete(d, "devicePassword")
		}, "ManagedDisk for imports"},
		"export ok": {func(job, d map[string]any) {
			job["properties"].(map[string]any)["transferType"] = "ExportFromAzure"
			d["dataExportDetails"], d["dataImportDetails"] = azureDataBoxTestExport("TransferAll"), nil
			delete(d, "devicePassword")
		}, ""},
	}
	for name, tc := range cases {
		resp := do(http.MethodPut, job+strings.ReplaceAll(name, " ", "-")+azureDataBoxTestAPIVersion, azureDataBoxTestJob(t, tc.change))
		t.Run(name, func(t *testing.T) {
			if tc.want == "" {
				azureResourcesExpectStatus(t, resp, http.StatusOK)
				return
			}
			azureSentinelExpectRejected(t, resp, tc.want)
		})
	}
}

func azureDataBoxTestExport(configType string) []any {
	return []any{map[string]any{
		"accountDetails": map[string]any{"dataAccountType": "StorageAccount", "storageAccountId": azureDataBoxTestAccount},
		"transferConfiguration": map[string]any{"transferConfigurationType": configType,
			"transferAllDetails": map[string]any{"include": map[string]any{"dataAccountType": "StorageAccount", "transferAllBlobs": true}}},
	}}
}

func TestAzureDataBoxShipping(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-databox")
	jobs := sub + "/resourceGroups/rg-databox/providers/Microsoft.DataBox/jobs/"
	azureResourcesExpectStatus(t, do(http.MethodPut, jobs+"microsoft"+azureDataBoxTestAPIVersion, azureDataBoxTestJob(t, nil)), http.StatusOK)
	selfShipped := azureDataBoxTestJob(t, func(_, d map[string]any) {
		d["preferences"] = map[string]any{"transportPreferences": map[string]any{"preferredShipmentType": "CustomerManaged"}}
	})
	azureResourcesExpectStatus(t, do(http.MethodPut, jobs+"self"+azureDataBoxTestAPIVersion, selfShipped), http.StatusOK)

	pickUp := `{"startTime":"2026-11-20T09:00:00Z","endTime":"2026-11-20T17:00:00Z","shipmentLocation":"Front desk"}`
	for payload, want := range map[string]string{
		`{"startTime":"soon","endTime":"2026-11-20T17:00:00Z","shipmentLocation":"Front desk"}`:                 "date-times",
		`{"startTime":"2026-11-20T17:00:00Z","endTime":"2026-11-20T09:00:00Z","shipmentLocation":"Front desk"}`: "after startTime",
		`{"startTime":"2026-11-20T09:00:00Z","endTime":"2026-11-20T17:00:00Z"}`:                                 "shipmentLocation",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPost, jobs+"microsoft/bookShipmentPickUp"+azureDataBoxTestAPIVersion, payload), want)
	}
	azureDataBoxExpectConflict(t, do(http.MethodPost, jobs+"microsoft/bookShipmentPickUp"+azureDataBoxTestAPIVersion, pickUp), "once the device is delivered")
	azureDataBoxExpectConflict(t, do(http.MethodPost, jobs+"self/bookShipmentPickUp"+azureDataBoxTestAPIVersion, pickUp), "jobs that Microsoft ships")

	shipped := `{"deliverToDcPackageDetails":{"carrierName":"Contoso Freight","trackingId":"000123"}}`
	azureSentinelExpectRejected(t, do(http.MethodPost, jobs+"self/markDevicesShipped"+azureDataBoxTestAPIVersion, `{"deliverToDcPackageDetails":{"carrierName":"x"}}`), "trackingId")
	azureDataBoxExpectConflict(t, do(http.MethodPost, jobs+"self/markDevicesShipped"+azureDataBoxTestAPIVersion, shipped), "once they are picked up")
	azureDataBoxExpectConflict(t, do(http.MethodPost, jobs+"microsoft/markDevicesShipped"+azureDataBoxTestAPIVersion, shipped), "customer-managed shipping")
}

func azureDataBoxExpectConflict(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	body := providerContractJSONMap(t, resp)
	errBody, _ := body["error"].(map[string]any)
	if resp.StatusCode != http.StatusConflict || errBody["code"] != "Conflict" || !strings.Contains(azureARMString(errBody["message"]), want) {
		t.Fatalf("expected 409 Conflict mentioning %q, got %d %#v", want, resp.StatusCode, body)
	}
}

func TestAzureDataBoxServiceActions(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-databox")
	group := sub + "/resourceGroups/rg-databox/providers/Microsoft.DataBox/locations/westus/"
	skus := providerContractJSONMap(t, do(http.MethodPost, group+"availableSkus"+azureDataBoxTestAPIVersion,
		`{"transferType":"ImportToAzure","country":"us","location":"westus","skuNames":["DataBox","DataBoxDisk"]}`))["value"].([]any)
	first := skus[0].(map[string]any)
	if len(skus) != 2 || first["enabled"] != true || first["sku"].(map[string]any)["name"] != "DataBox" ||
		first["properties"].(map[string]any)["countriesWithinCommerceBoundary"].([]any)[0] != "US" {
		t.Fatalf("unexpected skus: %#v", skus)
	}
	all := providerContractJSONMap(t, do(http.MethodPost, group+"availableSkus"+azureDataBoxTestAPIVersion, `{"transferType":"ExportFromAzure","country":"US","location":"westus"}`))
	if len(all["value"].([]any)) != 4 {
		t.Fatalf("expected every sku: %#v", all)
	}
	for payload, want := range map[string]string{
		`{"transferType":"Both","country":"US","location":"westus"}`:                                           "transferType",
		`{"transferType":"ImportToAzure","country":"USA","location":"westus"}`:                                 "country",
		`{"transferType":"ImportToAzure","country":"US"}`:                                                      "location is required",
		`{"transferType":"ImportToAzure","country":"US","location":"westus","skuNames":["Mini"]}`:              "skuNames",
		`{"transferType":"ImportToAzure","country":"US","location":"westus","skuNames":"DataBox"}`:             "skuNames must be a list",
		`{"transferType":"ImportToAzure","country":"US","location":"westus","skuNames":["DataBox","DataBox"]}`: "distinct",
		`{"transferType":"ImportToAzure","country":"1!","location":"westus"}`:                                  "country",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPost, group+"availableSkus"+azureDataBoxTestAPIVersion, payload), want)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/rg-missing/providers/Microsoft.DataBox/locations/westus/availableSkus"+azureDataBoxTestAPIVersion,
		`{"transferType":"ImportToAzure","country":"US","location":"westus"}`), http.StatusNotFound, "ResourceGroupNotFound")

	validate := `{"validationCategory":"JobCreationValidation","individualRequestDetails":[
		{"validationType":"ValidateAddress","deviceType":"DataBox","shippingAddress":{"streetAddress1":"1 Main St","country":"US"}},
		{"validationType":"ValidateDataTransferDetails","deviceType":"DataBox","transferType":"ImportToAzure","dataImportDetails":[{"accountDetails":{"storageAccountId":"` + azureDataBoxTestAccount + `"}}]},
		{"validationType":"ValidatePreferences","preference":{"transportPreferences":{"preferredShipmentType":"MicrosoftManaged"}}},
		{"validationType":"ValidateSubscriptionIsAllowedToCreateJob"},{"validationType":"ValidateCreateOrderLimit"},{"validationType":"ValidateSkuAvailability"}]}`
	for _, path := range []string{sub + "/providers/Microsoft.DataBox/locations/westus/validateInputs", group + "validateInputs"} {
		result := providerContractJSONMap(t, do(http.MethodPost, path+azureDataBoxTestAPIVersion, validate))["properties"].(map[string]any)
		responses := result["individualResponseDetails"].([]any)
		address := responses[0].(map[string]any)
		if result["status"] != "AllValidToProceed" || len(responses) != 6 || address["validationStatus"] != "Valid" || len(address["alternateAddresses"].([]any)) != 1 ||
			responses[1].(map[string]any)["status"] != "Valid" {
			t.Fatalf("unexpected validation at %s: %#v", path, result)
		}
	}
	invalid := providerContractJSONMap(t, do(http.MethodPost, group+"validateInputs"+azureDataBoxTestAPIVersion, `{"validationCategory":"JobCreationValidation","individualRequestDetails":[
		{"validationType":"ValidateAddress","shippingAddress":{"streetAddress1":"1 Main St"}},
		{"validationType":"ValidateDataTransferDetails","transferType":"ExportFromAzure","dataImportDetails":[{}]},
		{"validationType":"ValidatePreferences","preference":{"transportPreferences":{"preferredShipmentType":"Drone"}}}]}`))["properties"].(map[string]any)
	invalidResponses := invalid["individualResponseDetails"].([]any)
	if invalid["status"] != "InputsRevisitRequired" || invalidResponses[0].(map[string]any)["validationStatus"] != "Invalid" ||
		invalidResponses[1].(map[string]any)["status"] != "Invalid" || invalidResponses[2].(map[string]any)["status"] != "Invalid" ||
		!strings.Contains(azureARMString(invalidResponses[2].(map[string]any)["error"].(map[string]any)["message"]), "preferredShipmentType") {
		t.Fatalf("unexpected invalid validation: %#v", invalid)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, group+"validateInputs"+azureDataBoxTestAPIVersion, `{"validationCategory":"Other","individualRequestDetails":[{}]}`), "JobCreationValidation")
	azureSentinelExpectRejected(t, do(http.MethodPost, group+"validateInputs"+azureDataBoxTestAPIVersion, `{"validationCategory":"JobCreationValidation","individualRequestDetails":[{"validationType":"ValidateMood"}]}`), "ValidateMood")
	azureSentinelExpectRejected(t, do(http.MethodPost, group+"validateInputs"+azureDataBoxTestAPIVersion, `{"validationCategory":"JobCreationValidation","individualRequestDetails":[{"validationType":"ValidateCreateOrderLimit","deviceType":"Mini"}]}`), "deviceType")
	noTransfer := providerContractJSONMap(t, do(http.MethodPost, group+"validateInputs"+azureDataBoxTestAPIVersion, `{"validationCategory":"JobCreationValidation","individualRequestDetails":[
		{"validationType":"ValidateDataTransferDetails","dataImportDetails":[{"accountDetails":{"storageAccountId":"`+azureDataBoxTestAccount+`"}}]}]}`))["properties"].(map[string]any)
	if msg := noTransfer["individualResponseDetails"].([]any)[0].(map[string]any)["error"].(map[string]any)["message"]; !strings.Contains(azureARMString(msg), "transferType must be") {
		t.Fatalf("a missing transferType should be reported: %#v", noTransfer)
	}

	region := providerContractJSONMap(t, do(http.MethodPost, sub+"/providers/Microsoft.DataBox/locations/westus/regionConfiguration"+azureDataBoxTestAPIVersion,
		`{"transportAvailabilityRequest":{"skuName":"DataBox"},"scheduleAvailabilityRequest":{"skuName":"DataBox","storageLocation":"westus"},"deviceCapabilityRequest":{"skuName":"DataBoxDisk"}}`))
	if len(region["transportAvailabilityResponse"].(map[string]any)["transportAvailabilityDetails"].([]any)) != 2 ||
		len(region["scheduleAvailabilityResponse"].(map[string]any)["availableDates"].([]any)) != 21 ||
		region["deviceCapabilityResponse"].(map[string]any)["deviceCapabilityDetails"].([]any)[0].(map[string]any)["hardwareEncryption"] != "Enabled" {
		t.Fatalf("unexpected region configuration: %#v", region)
	}
	onlySchedule := providerContractJSONMap(t, do(http.MethodPost, group+"regionConfiguration"+azureDataBoxTestAPIVersion,
		`{"scheduleAvailabilityRequest":{"skuName":"DataBox"},"deviceCapabilityRequest":{"skuName":"DataBox"}}`))
	if onlySchedule["transportAvailabilityResponse"] != nil ||
		onlySchedule["deviceCapabilityResponse"].(map[string]any)["deviceCapabilityDetails"].([]any)[0].(map[string]any)["hardwareEncryption"] != "Disabled" {
		t.Fatalf("unexpected schedule-only configuration: %#v", onlySchedule)
	}
	onlyTransport := providerContractJSONMap(t, do(http.MethodPost, group+"regionConfiguration"+azureDataBoxTestAPIVersion, `{"transportAvailabilityRequest":{"skuName":"DataBoxHeavy"}}`))
	if len(onlyTransport) != 1 || onlyTransport["transportAvailabilityResponse"] == nil {
		t.Fatalf("only the requested parts should be answered: %#v", onlyTransport)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, group+"regionConfiguration"+azureDataBoxTestAPIVersion, `{"transportAvailabilityRequest":{"skuName":"Mini"}}`), "transportAvailabilityRequest.skuName")
	azureResourcesExpectStatus(t, do(http.MethodGet, group+"regionConfiguration"+azureDataBoxTestAPIVersion, ""), http.StatusMethodNotAllowed)
}

func TestAzureDataBoxPasswordRules(t *testing.T) {
	t.Parallel()
	for password, want := range map[string]bool{
		"Secure#Pa55wrdX": true, "Secure#Pa55wr": true, "secure#pa55wrdx": false, "Secure#Passwrdx": false,
		"SecurePa55wrdX": false, "Secure#Pa55wrdL": false, "Secure#Pa50wrdX": false, "Secure&Pa55wrdX": false, "Secure#Pa55wrd&X": false, "Sec#Pa55": false,
		"Secure#Pa55wrdX" + strings.Repeat("x", 50): false,
	} {
		if got := azureDataBoxPasswordValid(password); got != want {
			t.Errorf("azureDataBoxPasswordValid(%q) = %v, want %v", password, got, want)
		}
	}
	if generated := azureDataBoxPassword("/subscriptions/x/jobs/y", "device"); !azureDataBoxPasswordValid(generated) {
		t.Errorf("generated password %q breaks the rules", generated)
	}
}
