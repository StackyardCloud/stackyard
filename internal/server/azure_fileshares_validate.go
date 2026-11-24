package server

import (
	"regexp"
	"slices"
	"strings"
)

var (
	// File share names are 3 to 63 lowercase letters, digits and single
	// hyphens, starting and ending with a letter or digit.
	azureFileShareName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
	// azureFileShareUpdatable are the properties FileShareUpdateProperties
	// lists, which PATCH may change.
	azureFileShareUpdatable = []string{"encryption", "nfsProtocolProperties", "provisionedIOPerSec", "provisionedStorageGiB", "provisionedThroughputMiBPerSec", "publicAccessProperties", "publicNetworkAccess"}
	// azureFileShareFixed can't change once the share exists; redundancy
	// changes only by redundancy conversion.
	azureFileShareFixed = []string{"mediaTier", "protocol", "scaleMode", "mountName", "redundancy"}
)

func azureFileShareNameError(name string) string {
	if !azureFileShareName.MatchString(name) || strings.Contains(name, "--") {
		return "File share names are 3 to 63 lowercase letters, digits and single hyphens, starting and ending with a letter or digit."
	}
	return ""
}

func azureFileShareValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties", "identity"); msg != "" {
		return msg
	}
	if msg := azureFileShareIdentity(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if _, present := props["provisionedStorageGiB"]; !present {
		return "properties.provisionedStorageGiB is required."
	}
	if _, present := props["redundancy"]; !present {
		return "properties.redundancy is required."
	}
	if raw, present := props["mountName"]; present && azureFileShareNameError(azureARMString(raw)) != "" {
		return "properties.mountName must follow the file share naming rules."
	}
	return azureFileShareProperties(props)
}

func azureFileSharePatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties", "identity"); msg != "" {
		return nil, nil, msg
	}
	if msg := azureFileShareIdentity(payload); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	for key := range props {
		if !slices.Contains(azureFileShareUpdatable, key) {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	return payload, nil, azureFileShareProperties(props)
}

func azureFileShareIdentity(payload map[string]any) string {
	if raw, present := azureARMObject(payload["identity"])["type"]; present && !slices.Contains([]string{"None", "SystemAssigned", "UserAssigned", "SystemAssigned,UserAssigned"}, azureARMString(raw)) {
		return "identity.type must be None, SystemAssigned, UserAssigned or SystemAssigned,UserAssigned."
	}
	return ""
}

// azureFileShareProperties checks the shapes and enums of the properties a
// client sends; provisioning limits depend on the stored share and are
// checked in azureFileShareReconcile.
func azureFileShareProperties(props map[string]any) string {
	// Microsoft.FileShares offers SSD NFS shares only.
	for key, values := range map[string][]string{
		"mediaTier": {"SSD"}, "protocol": {"NFS"}, "redundancy": {"Local", "Zone"},
		"scaleMode": {"Compatibility", "Scale"}, "publicNetworkAccess": {"Enabled", "Disabled"},
	} {
		if msg := azureLabEnum(props, "properties.", key, values); msg != "" {
			return msg
		}
	}
	for _, key := range []string{"provisionedStorageGiB", "provisionedIOPerSec", "provisionedThroughputMiBPerSec"} {
		if raw, present := props[key]; present && !azureWholeNumber(raw, 0, 1<<31-1) {
			return "properties." + key + " must be a whole number."
		}
	}
	if msg := azureDataBoxObjects(props, "properties.", "nfsProtocolProperties", "publicAccessProperties", "encryption"); msg != "" {
		return msg
	}
	nfs := azureARMObject(props["nfsProtocolProperties"])
	if msg := azureLabEnum(nfs, "properties.nfsProtocolProperties.", "rootSquash", []string{"NoRootSquash", "RootSquash"}); msg != "" {
		return msg
	}
	if msg := azureLabEnum(nfs, "properties.nfsProtocolProperties.", "encryptionInTransitRequired", []string{"Enabled", "Disabled"}); msg != "" {
		return msg
	}
	if raw, present := azureARMObject(props["publicAccessProperties"])["allowedSubnets"]; present {
		subnets, ok := raw.([]any)
		if !ok || slices.ContainsFunc(subnets, func(v any) bool {
			return !strings.Contains(strings.ToLower(azureARMString(v)), "/providers/microsoft.network/virtualnetworks/") || !strings.Contains(strings.ToLower(azureARMString(v)), "/subnets/")
		}) {
			return "properties.publicAccessProperties.allowedSubnets must list subnet IDs."
		}
	}
	return azureFileShareEncryption(azureARMObject(props["encryption"]))
}

func azureFileShareEncryption(enc map[string]any) string {
	if msg := azureLabEnum(enc, "properties.encryption.", "infrastructureEncryption", []string{"enabled", "disabled"}); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(enc, "properties.encryption.", "customerManagedKeyEncryption"); msg != "" {
		return msg
	}
	cmk := azureARMObject(enc["customerManagedKeyEncryption"])
	if raw, present := cmk["keyEncryptionKeyUrl"]; present && !strings.HasPrefix(azureARMString(raw), "https://") {
		return "properties.encryption.customerManagedKeyEncryption.keyEncryptionKeyUrl must be a Key Vault key URL."
	}
	identity := azureARMObject(cmk["keyEncryptionKeyIdentity"])
	if msg := azureLabEnum(identity, "properties.encryption.customerManagedKeyEncryption.keyEncryptionKeyIdentity.", "identityType", []string{"systemAssignedIdentity", "userAssignedIdentity", "delegatedResourceIdentity"}); msg != "" {
		return msg
	}
	if identity["identityType"] == "userAssignedIdentity" && azureARMString(identity["userAssignedIdentityResourceId"]) == "" {
		return "properties.encryption.customerManagedKeyEncryption.keyEncryptionKeyIdentity.userAssignedIdentityResourceId is required for a user-assigned identity."
	}
	return ""
}

// azureFileShareOffer checks the offer dimensions getLimits and
// getProvisioningRecommendation take.
func azureFileShareOffer(payload map[string]any) (map[string]any, string) {
	props, ok := payload["properties"].(map[string]any)
	if !ok {
		return nil, "properties is required."
	}
	for key, values := range map[string][]string{"mediaTier": {"SSD"}, "protocol": {"NFS"}, "scaleMode": {"Compatibility", "Scale"}} {
		if msg := azureLabEnum(props, "properties.", key, values); msg != "" {
			return nil, msg
		}
	}
	return props, ""
}

func azureFileShareSnapshotValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if raw, present := props["initiatorId"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.initiatorId must be a string."
		}
	}
	return azureFileShareMetadata(props)
}

func azureFileShareMetadata(props map[string]any) string {
	raw, present := props["metadata"]
	if !present {
		return ""
	}
	metadata, ok := raw.(map[string]any)
	if !ok {
		return "properties.metadata must be an object of strings."
	}
	for _, v := range metadata {
		if _, ok := v.(string); !ok {
			return "properties.metadata must be an object of strings."
		}
	}
	return ""
}

func azureFileShareConnectionValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	state, ok := props["privateLinkServiceConnectionState"].(map[string]any)
	if !ok {
		return "properties.privateLinkServiceConnectionState is required."
	}
	if !slices.Contains([]string{"Pending", "Approved", "Rejected"}, azureARMString(state["status"])) {
		return "properties.privateLinkServiceConnectionState.status must be Pending, Approved or Rejected."
	}
	if raw, present := props["privateEndpoint"]; present && !azureNetworkIDOf(raw, "privateendpoints") {
		return "properties.privateEndpoint.id must be a private endpoint ID."
	}
	return ""
}
