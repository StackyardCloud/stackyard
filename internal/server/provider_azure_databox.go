package server

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Microsoft.DataBox jobs: ordering a Data Box device to import data into
// Azure storage or export it. The emulator never prepares or ships a
// device, so a job stays DeviceOrdered until it is cancelled: cancel,
// update and delete follow the real rules, while pick-ups and shipping
// report that the device isn't with the customer yet.
// ponytail: no stage ever advances on its own; a way to drive the device
// through its stages is the upgrade if tests need the shipping flow.
const (
	azureDataBoxNamespace = "Microsoft.DataBox"
	azureDataBoxOrdered   = "DeviceOrdered"
	azureDataBoxCancelled = "Cancelled"
)

var (
	azureDataBoxJobName = regexp.MustCompile(`^[-\w.]{3,24}$`)
	// Stages each transfer type goes through, with their display names.
	azureDataBoxStages = map[string][]string{
		"ImportToAzure":   {azureDataBoxOrdered, "DevicePrepared", "Dispatched", "Delivered", "PickedUp", "AtAzureDC", "DataCopy", "Completed"},
		"ExportFromAzure": {azureDataBoxOrdered, "DevicePrepared", "DataCopy", "Dispatched", "Delivered", "PickedUp", "AtAzureDC", "Completed"},
	}
	azureDataBoxStageNames = map[string]string{
		azureDataBoxOrdered: "Ordered", "DevicePrepared": "Processed", "Dispatched": "Dispatched", "Delivered": "Delivered",
		"PickedUp": "Picked up", "AtAzureDC": "Received", "DataCopy": "Data copy", "Completed": "Completed", azureDataBoxCancelled: "Canceled",
	}
	// Statuses a job can be deleted in.
	azureDataBoxFinished = []string{azureDataBoxCancelled, "Completed", "CompletedWithErrors", "CompletedWithWarnings", "Aborted"}
)

func init() {
	var jobs azureARMResourceType
	jobs = azureARMResourceType{
		Namespace: azureDataBoxNamespace, Type: "jobs",
		// Create answers 200 (or 202), never 201.
		CreateStatus: http.StatusOK,
		ValidateName: func(name string) string {
			if !azureDataBoxJobName.MatchString(name) {
				return "Job names are 3 to 24 letters, digits, underscores, hyphens or periods."
			}
			return ""
		},
		Validate: azureDataBoxJobValidate,
		WriteOnlyProperties: []string{"details.devicePassword", "details.passkey",
			"details.dataImportDetails[].accountDetails.sharePassword"},
		StateProperties:    map[string]any{"status": azureDataBoxOrdered, "cancellationReason": nil},
		ReadOnlyProperties: azureDataBoxJobProperties,
		Linked:             azureDataBoxJobLinked,
		PatchBody:          azureDataBoxJobPatch,
		ValidatePatch: func(_ *Server, existing *azureARMResource, payload map[string]any) string {
			if _, present := azureARMObject(payload["properties"])["details"]; present && azureDataBoxStatus(existing) != azureDataBoxOrdered {
				return "Job " + existing.Name + " is " + azureDataBoxStatus(existing) + "; only its tags can change now."
			}
			return ""
		},
		ValidateDelete: func(_ *Server, existing *azureARMResource) string {
			if !slices.Contains(azureDataBoxFinished, azureDataBoxStatus(existing)) {
				return "Only completed or cancelled jobs can be deleted; cancel job " + existing.Name + " first."
			}
			return ""
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"cancel": func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
				return s.azureDataBoxCancel(rec, payload, jobs)
			},
			"listcredentials":    (*Server).azureDataBoxListCredentials,
			"bookshipmentpickup": azureDataBoxBookPickUp,
			"markdevicesshipped": azureDataBoxMarkShipped,
		},
	}
	registerAzureARMResourceType(jobs)
	azureDataBoxRegisterServiceActions()
}

func azureDataBoxStatus(rec *azureARMResource) string {
	if status := azureARMString(rec.Properties["status"]); status != "" {
		return status
	}
	return azureDataBoxOrdered
}

func azureDataBoxShipment(props map[string]any) string {
	preferences := azureARMObject(azureARMObject(props["details"])["preferences"])
	if shipment := azureARMString(azureARMObject(preferences["transportPreferences"])["preferredShipmentType"]); shipment != "" {
		return shipment
	}
	return "MicrosoftManaged"
}

// azureDataBoxJobProperties derives what the job's status allows and its
// stages: the order stage succeeded, a cancellation ends the job, and the
// stages after it haven't started.
func azureDataBoxJobProperties(rec *azureARMResource) map[string]any {
	status := azureDataBoxStatus(rec)
	ordered := status == azureDataBoxOrdered
	stages := []any{}
	for _, name := range azureDataBoxStages[azureARMString(rec.Properties["transferType"])] {
		stage := map[string]any{"stageName": name, "displayName": azureDataBoxStageNames[name], "stageStatus": "None"}
		if name == azureDataBoxOrdered {
			stage["stageStatus"], stage["stageTime"] = "Succeeded", rec.CreatedAt.Format(time.RFC3339)
		}
		stages = append(stages, stage)
		if name == azureDataBoxOrdered && status == azureDataBoxCancelled {
			stages = append(stages, map[string]any{"stageName": status, "displayName": azureDataBoxStageNames[status],
				"stageStatus": "Succeeded", "stageTime": azureDataBoxCancelTime(rec)})
			break
		}
	}
	out := map[string]any{
		"startTime": rec.CreatedAt.Format(time.RFC3339), "isCancellable": ordered, "isCancellableWithoutFee": ordered,
		"isDeletable": slices.Contains(azureDataBoxFinished, status), "isShippingAddressEditable": ordered, "isPrepareToShipEnabled": false,
		"details": map[string]any{"jobStages": stages, "copyProgress": []any{}, "actions": []any{}},
	}
	if rec.Properties["deliveryType"] == nil {
		out["deliveryType"] = "NonScheduled"
	}
	return out
}

// azureDataBoxCancelTime keeps the time a stored cancelled stage has, so
// later tag updates don't move it; a job just cancelled takes UpdatedAt.
func azureDataBoxCancelTime(rec *azureARMResource) any {
	for _, raw := range azureARMSlice(azureARMObject(rec.Properties["details"])["jobStages"]) {
		if stage := azureARMObject(raw); stage["stageName"] == azureDataBoxCancelled && stage["stageTime"] != nil {
			return stage["stageTime"]
		}
	}
	return rec.UpdatedAt.Format(time.RFC3339)
}

// azureDataBoxJobLinked rejects a PUT to an existing job, which only PATCH
// updates, and keeps the passwords the client set for listCredentials.
// Caller holds providerStorageMu.
func azureDataBoxJobLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	if s.azureARMResources[azureARMKey(rec.ID)] != nil {
		return nil, nil, "Job " + rec.Name + " already exists; update it with PATCH."
	}
	details := azureARMObject(rec.Properties["details"])
	shares := []any{}
	for _, item := range azureARMSlice(details["dataImportDetails"]) {
		shares = append(shares, azureARMObject(azureARMObject(item)["accountDetails"])["sharePassword"])
	}
	secrets := &azureARMResource{
		ID: azureDataBoxSecretsID(rec.ID), Type: azureDataBoxNamespace + "/jobSecrets", Name: "default",
		Properties: map[string]any{"devicePassword": details["devicePassword"], "passkey": details["passkey"], "sharePasswords": shares},
		CreatedAt:  rec.CreatedAt, UpdatedAt: rec.CreatedAt,
	}
	return nil, []*azureARMResource{secrets}, ""
}

func azureDataBoxSecretsID(jobID string) string { return jobID + "/jobSecrets/default" }

// azureDataBoxJobPatch takes the update model: tags, identity and the
// contact, shipping, preference and key details.
func azureDataBoxJobPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties", "tags", "identity"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureDataBoxObjects(props, "properties.", "details"); msg != "" {
		return nil, nil, msg
	}
	for key := range props {
		if key != "details" {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	details := azureARMObject(props["details"])
	for key := range details {
		if !slices.Contains([]string{"contactDetails", "shippingAddress", "preferences", "keyEncryptionKey", "reverseShippingDetails", "returnToCustomerPackageDetails"}, key) {
			return nil, nil, "properties.details." + key + " can't be updated."
		}
	}
	if msg := azureDataBoxObjects(details, "properties.details.", "contactDetails", "shippingAddress", "preferences", "keyEncryptionKey",
		"reverseShippingDetails", "returnToCustomerPackageDetails"); msg != "" {
		return nil, nil, msg
	}
	if raw, present := details["contactDetails"]; present {
		if msg := azureDataBoxContactValidate(azureARMObject(raw)); msg != "" {
			return nil, nil, msg
		}
	}
	if raw, present := details["shippingAddress"]; present {
		if msg := azureDataBoxAddressValidate("properties.details.shippingAddress", azureARMObject(raw)); msg != "" {
			return nil, nil, msg
		}
	}
	if msg := azureDataBoxPreferencesValidate(azureARMObject(details["preferences"])); msg != "" {
		return nil, nil, msg
	}
	if msg := azureDataBoxKeyValidate(details["keyEncryptionKey"]); msg != "" {
		return nil, nil, msg
	}
	body := map[string]any{"properties": props}
	for _, key := range []string{"tags", "identity"} {
		if raw, present := payload[key]; present {
			body[key] = raw
		}
	}
	return body, nil, ""
}

// azureDataBoxObjects rejects any of keys that is present in m but not an
// object, since merging one would replace the stored object wholesale.
func azureDataBoxObjects(m map[string]any, prefix string, keys ...string) string {
	for _, key := range keys {
		if raw, present := m[key]; present {
			if _, ok := raw.(map[string]any); !ok {
				return prefix + key + " must be an object."
			}
		}
	}
	return ""
}

// azureDataBoxCancel cancels a job that hasn't been processed yet.
func (s *Server) azureDataBoxCancel(rec *azureARMResource, payload map[string]any, typ azureARMResourceType) (int, any) {
	reason := strings.TrimSpace(azureARMString(payload["reason"]))
	if reason == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "reason is required.")
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	if status := azureDataBoxStatus(current); status != azureDataBoxOrdered {
		return http.StatusConflict, azureARMErrorBody("Conflict", "Job "+current.Name+" is "+status+"; only jobs that haven't been processed can be cancelled.")
	}
	updated := *current
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(current.Properties, map[string]any{"status": azureDataBoxCancelled, "cancellationReason": reason})
	updated.Properties = azureARMProperties(typ, &updated)
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	return http.StatusNoContent, nil
}

// azureDataBoxListCredentials returns the device and share passwords: the
// ones the client set, or ones derived from the job ID.
func (s *Server) azureDataBoxListCredentials(rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	var secrets map[string]any
	if stored := s.azureARMResources[azureARMKey(azureDataBoxSecretsID(rec.ID))]; stored != nil {
		secrets = stored.Properties
	}
	s.providerStorageMu.Unlock()
	sku := azureARMString(rec.SKU["name"])
	details := azureARMObject(rec.Properties["details"])
	jobSecrets := map[string]any{"jobSecretsType": sku, "dcAccessSecurityCode": map[string]any{}}
	serial := func(n int) string {
		return fmt.Sprintf("%s%07d", strings.ToUpper(sku[:3]), azureDataBoxNumber(rec.ID, fmt.Sprint("serial", n)))
	}
	switch sku {
	case "DataBoxDisk", "DataBoxCustomerDisk":
		jobSecrets["diskSecrets"] = []any{map[string]any{"diskSerialNumber": serial(0),
			"bitLockerKey": fmt.Sprintf("%06d-%06d-%06d-%06d", azureDataBoxNumber(rec.ID, "bl0")%1e6, azureDataBoxNumber(rec.ID, "bl1")%1e6,
				azureDataBoxNumber(rec.ID, "bl2")%1e6, azureDataBoxNumber(rec.ID, "bl3")%1e6)}}
		if sku == "DataBoxDisk" {
			passkey := azureARMString(secrets["passkey"])
			jobSecrets["isPasskeyUserDefined"] = passkey != ""
			if passkey == "" {
				passkey = azureDataBoxPassword(rec.ID, "passkey")
			}
			jobSecrets["passKey"] = passkey
		}
	default:
		devicePassword := azureARMString(secrets["devicePassword"])
		if devicePassword == "" {
			devicePassword = azureDataBoxPassword(rec.ID, "device")
		}
		accounts := azureDataBoxAccountCredentials(rec, details, azureARMSlice(secrets["sharePasswords"]))
		nodes := 1
		key := "podSecrets"
		if sku == "DataBoxHeavy" {
			nodes, key = 2, "cabinetPodSecrets"
		}
		pods := []any{}
		for n := range nodes {
			pods = append(pods, map[string]any{
				"deviceSerialNumber": serial(n), "devicePassword": devicePassword,
				"encodedValidationCertPubKey": azureARMDerivedKey(rec.ID, fmt.Sprint("cert", n)),
				"networkConfigurations": []any{
					map[string]any{"name": "DataPort1", "macAddress": azureDataBoxMAC(rec.ID, n, 1)},
					map[string]any{"name": "DataPort2", "macAddress": azureDataBoxMAC(rec.ID, n, 2)},
					map[string]any{"name": "DataPort3", "macAddress": azureDataBoxMAC(rec.ID, n, 3)},
				},
				"accountCredentialDetails": accounts,
			})
		}
		jobSecrets[key] = pods
	}
	return http.StatusOK, map[string]any{"value": []any{map[string]any{"jobName": rec.Name, "jobSecrets": jobSecrets}}}
}

// azureDataBoxAccountCredentials lists the device shares for each account:
// block blob, page blob and file shares for a storage account, and one
// managed disk share for a resource group.
func azureDataBoxAccountCredentials(rec *azureARMResource, details map[string]any, sharePasswords []any) []any {
	items := azureARMSlice(details["dataImportDetails"])
	if len(items) == 0 {
		items = azureARMSlice(details["dataExportDetails"])
	}
	out := []any{}
	for i, item := range items {
		account := azureARMObject(azureARMObject(item)["accountDetails"])
		password := ""
		if i < len(sharePasswords) {
			password = azureARMString(sharePasswords[i])
		}
		if password == "" {
			password = azureDataBoxPassword(rec.ID, fmt.Sprint("share", i))
		}
		kind := azureARMString(account["dataAccountType"])
		if kind == "" {
			kind = "StorageAccount"
		}
		share := func(name, shareType string) map[string]any {
			return map[string]any{"shareName": name, "shareType": shareType, "userName": name, "password": password,
				"supportedAccessProtocols": []any{"SMB", "NFS"}}
		}
		var name string
		var shares []any
		if kind == "ManagedDisk" {
			name = azureARMLastSegment(azureARMString(account["resourceGroupId"]))
			shares = []any{share(name+"_MDisk", "ManagedDisk")}
		} else {
			name = azureARMLastSegment(azureARMString(account["storageAccountId"]))
			shares = []any{share(name+"_BlockBlob", "BlockBlob"), share(name+"_PageBlob", "PageBlob"), share(name+"_AzFile", "AzureFile")}
		}
		out = append(out, map[string]any{"accountName": name, "dataAccountType": kind, "accountConnectionString": "", "shareCredentialDetails": shares})
	}
	return out
}

func azureARMLastSegment(id string) string { return id[strings.LastIndex(id, "/")+1:] }

// azureDataBoxPasswordAlphabet leaves out the characters Data Box
// passwords can't have (IilLoO0).
const azureDataBoxPasswordAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz123456789"

// azureDataBoxPassword derives a stable password that meets the device
// password rules.
func azureDataBoxPassword(id, label string) string {
	sum := azureARMDerivedKey(id, label)
	var b strings.Builder
	for i := 0; i < 12; i++ {
		b.WriteByte(azureDataBoxPasswordAlphabet[int(sum[i])%len(azureDataBoxPasswordAlphabet)])
	}
	return b.String() + "Q7#"
}

func azureDataBoxNumber(id, label string) int {
	n := 0
	for _, c := range azureARMDerivedKey(id, label)[:8] {
		n = (n*31 + int(c)) % 10000000
	}
	return n
}

func azureDataBoxMAC(id string, node, port int) string {
	key := azureARMDerivedKey(id, fmt.Sprint("mac", node, port))
	return strings.ToUpper(fmt.Sprintf("%02x%02x%02x%02x%02x%02x", 0x00, 0x15, key[0], key[1], key[2], port))
}

// azureDataBoxBookPickUp books the return pick-up of a device with the
// customer. Devices never reach the customer in the emulator.
func azureDataBoxBookPickUp(_ *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	start, startErr := time.Parse(time.RFC3339, azureARMString(payload["startTime"]))
	end, endErr := time.Parse(time.RFC3339, azureARMString(payload["endTime"]))
	var msg string
	switch {
	case startErr != nil || endErr != nil:
		msg = "startTime and endTime are required date-times."
	case !end.After(start):
		msg = "endTime must be after startTime."
	case strings.TrimSpace(azureARMString(payload["shipmentLocation"])) == "":
		msg = "shipmentLocation is required."
	}
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	if azureDataBoxShipment(rec.Properties) != "MicrosoftManaged" {
		return http.StatusConflict, azureARMErrorBody("Conflict", "Pick-ups are booked only for jobs that Microsoft ships.")
	}
	return http.StatusConflict, azureARMErrorBody("Conflict", "Job "+rec.Name+" is "+azureDataBoxStatus(rec)+"; a pick-up can be booked once the device is delivered.")
}

// azureDataBoxMarkShipped records that the customer shipped a device to
// the datacenter themselves. Devices never reach the customer in the
// emulator.
func azureDataBoxMarkShipped(_ *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	pkg := azureARMObject(payload["deliverToDcPackageDetails"])
	if strings.TrimSpace(azureARMString(pkg["carrierName"])) == "" || strings.TrimSpace(azureARMString(pkg["trackingId"])) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "deliverToDcPackageDetails.carrierName and trackingId are required.")
	}
	if azureDataBoxShipment(rec.Properties) != "CustomerManaged" {
		return http.StatusConflict, azureARMErrorBody("Conflict", "Only jobs with customer-managed shipping mark their devices shipped.")
	}
	return http.StatusConflict, azureARMErrorBody("Conflict", "Job "+rec.Name+" is "+azureDataBoxStatus(rec)+"; devices can be marked shipped once they are picked up.")
}
