package server

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	azureWorkloadsEnvironments = []string{"NonProd", "Prod"}
	azureWorkloadsProducts     = []string{"ECC", "S4HANA", "Other"}
	azureWorkloadsDatabases    = []string{"HANA", "DB2"}
	azureWorkloadsDeployments  = []string{"SingleServer", "ThreeTier"}
	azureWorkloadsHATypes      = []string{"AvailabilitySet", "AvailabilityZone"}
	azureWorkloadsDiskSKUs     = []string{"Standard_LRS", "Premium_LRS", "StandardSSD_LRS", "UltraSSD_LRS", "Premium_ZRS", "StandardSSD_ZRS", "PremiumV2_LRS"}
	// azureWorkloadsHANAVolumes are the database volumes Learn lists for
	// HANA; all but backup are required.
	azureWorkloadsHANAVolumes = []string{"hana/data", "hana/log", "hana/shared", "usr/sap", "os", "backup"}
	// Learn's OSProfile disallowed administrator names and passwords.
	azureWorkloadsBadUsers = []string{"administrator", "admin", "user", "user1", "test", "user2", "test1", "user3", "admin1", "1", "123", "a",
		"actuser", "adm", "admin2", "aspnet", "backup", "console", "david", "guest", "john", "owner", "root", "server", "sql", "support",
		"support_388945a0", "sys", "test2", "test3", "user4", "user5"}
	azureWorkloadsBadPasswords = []string{"abc@123", "P@$$w0rd", "P@ssw0rd", "P@ssword123", "Pa$$word", "pass@word1", "Password!", "Password1", "Password22", "iloveyou!"}

	azureWorkloadsSID          = regexp.MustCompile(`^[A-Za-z0-9]{3}$`)
	azureWorkloadsVMID         = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.Compute/virtualMachines/[^/]+$`)
	azureWorkloadsSubnetID     = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.Network/virtualNetworks/[^/]+/subnets/[^/]+$`)
	azureWorkloadsWorkspaceID  = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.OperationalInsights/workspaces/[^/]+$`)
	azureWorkloadsStorageID    = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.Storage/storageAccounts/[^/]+$`)
	azureWorkloadsStorageName  = regexp.MustCompile(`^[a-z0-9]{3,24}$`)
	azureWorkloadsInstanceNo   = regexp.MustCompile(`^[0-9]{2}$`)
	azureWorkloadsPrometheusRe = regexp.MustCompile(`^https?://[^/\s]+`)
)

// azureWorkloadsRequired returns a message for the first key that isn't a
// non-empty string in m.
func azureWorkloadsRequired(m map[string]any, path string, keys ...string) string {
	for _, key := range keys {
		if s, ok := m[key].(string); !ok || strings.TrimSpace(s) == "" {
			return path + key + " is required."
		}
	}
	return ""
}

// azureWorkloadsEnums checks each present key against its allowed values
// and requires the keys in required.
func azureWorkloadsEnums(m map[string]any, path string, required []string, enums map[string][]string) string {
	if msg := azureWorkloadsRequired(m, path, required...); msg != "" {
		return msg
	}
	for _, key := range slices.Sorted(maps.Keys(enums)) {
		if msg := azureLabEnum(m, path, key, enums[key]); msg != "" {
			return msg
		}
	}
	return ""
}

// azureWorkloadsCount returns m[key] as a positive integer, 0 when absent,
// or -1 when it isn't one.
func azureWorkloadsCount(m map[string]any, key string) int {
	raw, present := m[key]
	if !present {
		return 0
	}
	n, ok := azureFileShareInt(raw)
	if !ok || n < 1 {
		return -1
	}
	return n
}

func azureWorkloadsIdentity(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "identity"); msg != "" {
		return msg
	}
	identity := azureARMObject(payload["identity"])
	if len(identity) == 0 {
		return ""
	}
	if msg := azureLabEnum(identity, "identity.", "type", []string{"None", "UserAssigned"}); msg != "" {
		return msg
	}
	if identity["type"] != "UserAssigned" {
		return ""
	}
	ids, _ := identity["userAssignedIdentities"].(map[string]any)
	if len(ids) == 0 {
		return "identity.userAssignedIdentities is required for a user-assigned identity."
	}
	for id := range ids {
		if !strings.Contains(strings.ToLower(id), "/providers/microsoft.managedidentity/userassignedidentities/") {
			return "identity.userAssignedIdentities keys must be user-assigned identity IDs."
		}
	}
	return ""
}

// azureWorkloadsTagsPatch takes the update model of Learn's Update
// operations that change only tags and identity.
func azureWorkloadsTagsPatch(identity bool) func(map[string]any) (map[string]any, map[string]any, string) {
	allowed := []string{"tags"}
	if identity {
		allowed = append(allowed, "identity")
	}
	return func(payload map[string]any) (map[string]any, map[string]any, string) {
		for key := range payload {
			if !slices.Contains(allowed, key) {
				return nil, nil, key + " can't be updated."
			}
		}
		if msg := azureDataBoxObjects(payload, "", "tags"); msg != "" {
			return nil, nil, msg
		}
		if identity {
			if msg := azureWorkloadsIdentity(payload); msg != "" {
				return nil, nil, msg
			}
		}
		return payload, nil, ""
	}
}

func azureWorkloadsVISName(name string) string {
	if !azureWorkloadsSID.MatchString(name) {
		return "The Virtual Instance for SAP solutions name is the SAP system ID (SID): three letters or digits."
	}
	return ""
}

// azureWorkloadsVISValidate checks a Virtual Instance for SAP solutions.
func azureWorkloadsVISValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	if msg := azureWorkloadsIdentity(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureWorkloadsEnums(props, "properties.", []string{"environment", "sapProduct"},
		map[string][]string{"environment": azureWorkloadsEnvironments, "sapProduct": azureWorkloadsProducts}); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(props, "properties.", "configuration", "managedResourceGroupConfiguration"); msg != "" {
		return msg
	}
	if msg := azureWorkloadsMRGName(props); msg != "" {
		return msg
	}
	config, ok := props["configuration"].(map[string]any)
	if !ok {
		return "properties.configuration is required."
	}
	const path = "properties.configuration."
	// Registering a system or detecting externally installed software
	// needs a user-assigned identity with access to the system's resources.
	external := config["configurationType"] == "Discovery" || azureARMObject(config["softwareConfiguration"])["softwareInstallationType"] == "External"
	if external && azureARMObject(payload["identity"])["type"] != "UserAssigned" {
		return "identity must be a user-assigned identity to register or detect an SAP system."
	}
	switch config["configurationType"] {
	case "Discovery":
		if msg := azureWorkloadsVMRef(config, path, "centralServerVmId"); msg != "" {
			return msg
		}
		if raw, present := config["managedRgStorageAccountName"]; present && !azureWorkloadsStorageName.MatchString(azureARMString(raw)) {
			return path + "managedRgStorageAccountName must be 3 to 24 lowercase letters and numbers."
		}
		return ""
	case "Deployment", "DeploymentWithOSConfig":
		return azureWorkloadsDeployment(config, path)
	}
	return path + "configurationType must be one of Discovery, Deployment, DeploymentWithOSConfig."
}

func azureWorkloadsVMRef(m map[string]any, path, key string) string {
	if !azureWorkloadsVMID.MatchString(azureARMString(m[key])) {
		return path + key + " must be a virtual machine resource ID."
	}
	return ""
}

func azureWorkloadsDeployment(config map[string]any, path string) string {
	if msg := azureWorkloadsRequired(config, path, "appLocation"); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(config, path, "infrastructureConfiguration", "softwareConfiguration", "osSapConfiguration"); msg != "" {
		return msg
	}
	infra, ok := config["infrastructureConfiguration"].(map[string]any)
	if !ok {
		return path + "infrastructureConfiguration is required."
	}
	if msg := azureWorkloadsInfrastructure(infra, path+"infrastructureConfiguration."); msg != "" {
		return msg
	}
	sw, present := config["softwareConfiguration"].(map[string]any)
	if !present {
		return ""
	}
	if msg := azureWorkloadsSoftware(sw, path+"softwareConfiguration."); msg != "" {
		return msg
	}
	// Installing a highly available system needs the fencing agent's
	// service principal.
	if infra["highAvailabilityConfig"] != nil && sw["softwareInstallationType"] != "External" {
		return azureWorkloadsRequired(azureARMObject(sw["highAvailabilitySoftwareConfiguration"]), path+"softwareConfiguration.highAvailabilitySoftwareConfiguration.", "fencingClientId", "fencingClientPassword")
	}
	return ""
}

func azureWorkloadsSoftware(sw map[string]any, path string) string {
	switch sw["softwareInstallationType"] {
	case "ServiceInitiated", "SAPInstallWithoutOSConfig":
		required := []string{"bomUrl", "softwareVersion", "sapBitsStorageAccountId"}
		if sw["softwareInstallationType"] == "ServiceInitiated" {
			required = append(required, "sapFqdn")
		}
		if msg := azureWorkloadsRequired(sw, path, required...); msg != "" {
			return msg
		}
		if !azureWorkloadsStorageID.MatchString(azureARMString(sw["sapBitsStorageAccountId"])) {
			return path + "sapBitsStorageAccountId must be a storage account resource ID."
		}
		return ""
	case "External":
		return azureWorkloadsVMRef(sw, path, "centralServerVmId")
	}
	return path + "softwareInstallationType must be one of ServiceInitiated, SAPInstallWithoutOSConfig, External."
}

func azureWorkloadsInfrastructure(infra map[string]any, path string) string {
	if msg := azureWorkloadsEnums(infra, path, []string{"deploymentType", "appResourceGroup"},
		map[string][]string{"deploymentType": azureWorkloadsDeployments, "databaseType": azureWorkloadsDatabases}); msg != "" {
		return msg
	}
	if len(azureARMString(infra["appResourceGroup"])) > 90 {
		return path + "appResourceGroup must be at most 90 characters."
	}
	if msg := azureDataBoxObjects(infra, path, "customResourceNames", "highAvailabilityConfig", "storageConfiguration", "networkConfiguration"); msg != "" {
		return msg
	}
	if infra["deploymentType"] == "SingleServer" {
		if msg := azureWorkloadsServer(infra, path, false); msg != "" {
			return msg
		}
		if infra["highAvailabilityConfig"] != nil || infra["storageConfiguration"] != nil {
			return path + "highAvailabilityConfig and storageConfiguration need a ThreeTier deployment."
		}
		if msg := azureWorkloadsDisks(infra, path, "dbDiskConfiguration", infra["databaseType"]); msg != "" {
			return msg
		}
		names := azureARMObject(infra["customResourceNames"])
		if raw, present := names["virtualMachine"]; present {
			return azureWorkloadsVMNames(azureARMObject(raw), path+"customResourceNames.virtualMachine.", true)
		}
		return ""
	}
	return azureWorkloadsThreeTier(infra, path)
}

// azureWorkloadsThreeTier checks the central, application and database
// layers. A highly available system has two central VMs (ASCS and ERS) and
// two database VMs; otherwise each has one.
func azureWorkloadsThreeTier(infra map[string]any, path string) string {
	ha := azureARMObject(infra["highAvailabilityConfig"])
	if infra["highAvailabilityConfig"] != nil {
		if msg := azureWorkloadsEnums(ha, path+"highAvailabilityConfig.", []string{"highAvailabilityType"},
			map[string][]string{"highAvailabilityType": azureWorkloadsHATypes}); msg != "" {
			return msg
		}
	}
	pair := 1
	if len(ha) != 0 {
		pair = 2
	}
	names := azureARMObject(infra["customResourceNames"])
	for _, layer := range []struct {
		key, names string
		want       int
	}{{"centralServer", "centralServer", pair}, {"applicationServer", "applicationServer", 0}, {"databaseServer", "databaseServer", pair}} {
		lp := path + layer.key + "."
		server, ok := infra[layer.key].(map[string]any)
		if !ok {
			return path + layer.key + " is required."
		}
		if msg := azureWorkloadsServer(server, lp, true); msg != "" {
			return msg
		}
		count := azureWorkloadsCount(server, "instanceCount")
		if layer.want != 0 && count != layer.want {
			system := "a system without high availability"
			if pair == 2 {
				system = "a highly available system"
			}
			return lp + "instanceCount must be " + strconv.Itoa(layer.want) + " for " + system + "."
		}
		if layer.key == "databaseServer" {
			if msg := azureLabEnum(server, lp, "databaseType", azureWorkloadsDatabases); msg != "" {
				return msg
			}
			if msg := azureWorkloadsDisks(server, lp, "diskConfiguration", server["databaseType"]); msg != "" {
				return msg
			}
		}
		if raw, present := names[layer.names]; present {
			if msg := azureWorkloadsLayerNames(azureARMObject(raw), path+"customResourceNames."+layer.names+".", count, layer.key == "databaseServer"); msg != "" {
				return msg
			}
		}
	}
	return azureWorkloadsFileShare(azureARMObject(azureARMObject(infra["storageConfiguration"])["transportFileShareConfiguration"]), path+"storageConfiguration.transportFileShareConfiguration.")
}

func azureWorkloadsFileShare(share map[string]any, path string) string {
	switch share["configurationType"] {
	case nil, "Skip", "CreateAndMount":
		if raw, present := share["storageAccountName"]; present && !azureWorkloadsStorageName.MatchString(azureARMString(raw)) {
			return path + "storageAccountName must be 3 to 24 lowercase letters and numbers."
		}
		return ""
	case "Mount":
		return azureWorkloadsRequired(share, path, "id", "privateEndpointId")
	}
	return path + "configurationType must be one of CreateAndMount, Mount, Skip."
}

// azureWorkloadsServer checks a layer's subnet, instance count and VM.
func azureWorkloadsServer(server map[string]any, path string, layer bool) string {
	if !azureWorkloadsSubnetID.MatchString(azureARMString(server["subnetId"])) {
		return path + "subnetId must be a subnet resource ID."
	}
	if layer && azureWorkloadsCount(server, "instanceCount") < 1 {
		return path + "instanceCount must be a positive integer."
	}
	vm, ok := server["virtualMachineConfiguration"].(map[string]any)
	if !ok {
		return path + "virtualMachineConfiguration is required."
	}
	return azureWorkloadsVM(vm, path+"virtualMachineConfiguration.")
}

func azureWorkloadsVM(vm map[string]any, path string) string {
	if msg := azureWorkloadsRequired(vm, path, "vmSize"); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(vm, path, "imageReference", "osProfile"); msg != "" {
		return msg
	}
	if msg := azureWorkloadsRequired(azureARMObject(vm["imageReference"]), path+"imageReference.", "publisher", "offer", "sku", "version"); msg != "" {
		return msg
	}
	os, ok := vm["osProfile"].(map[string]any)
	if !ok {
		return path + "osProfile is required."
	}
	return azureWorkloadsOS(os, path+"osProfile.")
}

// azureWorkloadsOS applies Learn's OSProfile rules: administrator name and
// password limits, and an SSH key for Linux without password sign-in.
func azureWorkloadsOS(os map[string]any, path string) string {
	config := azureARMObject(os["osConfiguration"])
	linux := config["osType"] == "Linux"
	if !linux && config["osType"] != "Windows" {
		return path + "osConfiguration.osType must be one of Linux, Windows."
	}
	user := azureARMString(os["adminUsername"])
	most := map[bool]int{true: 64, false: 20}[linux]
	switch {
	case user == "" || len(user) > most:
		return path + "adminUsername must be 1 to " + strconv.Itoa(most) + " characters."
	case slices.Contains(azureWorkloadsBadUsers, strings.ToLower(user)):
		return path + "adminUsername '" + user + "' isn't allowed."
	case !linux && strings.HasSuffix(user, "."):
		return path + "adminUsername can't end in '.'."
	}
	password, hasPassword := os["adminPassword"].(string)
	if hasPassword {
		if msg := azureWorkloadsPassword(password, linux, path+"adminPassword"); msg != "" {
			return msg
		}
	}
	if !linux || config["disablePasswordAuthentication"] != true {
		if !hasPassword {
			return path + "adminPassword is required unless a Linux VM disables password authentication."
		}
		return ""
	}
	keys := []string{azureARMString(azureARMObject(config["sshKeyPair"])["publicKey"])}
	for _, key := range azureARMSlice(azureARMObject(config["ssh"])["publicKeys"]) {
		keys = append(keys, azureARMString(azureARMObject(key)["keyData"]))
	}
	found := false
	for _, key := range keys {
		if key == "" {
			continue
		}
		if !strings.HasPrefix(key, "ssh-rsa ") {
			return path + "osConfiguration SSH public keys must be in ssh-rsa format."
		}
		found = true
	}
	if !found {
		return path + "osConfiguration needs an SSH public key when password authentication is disabled."
	}
	return ""
}

func azureWorkloadsPassword(password string, linux bool, path string) string {
	least, most := 8, 123
	if linux {
		least, most = 6, 72
	}
	if len(password) < least || len(password) > most {
		return path + " must be " + strconv.Itoa(least) + " to " + strconv.Itoa(most) + " characters."
	}
	if slices.Contains(azureWorkloadsBadPasswords, password) {
		return path + " is a disallowed value."
	}
	var lower, upper, digit, special int
	for _, r := range password {
		switch {
		case r >= 'a' && r <= 'z':
			lower = 1
		case r >= 'A' && r <= 'Z':
			upper = 1
		case r >= '0' && r <= '9':
			digit = 1
		default:
			special = 1
		}
	}
	if lower+upper+digit+special < 3 {
		return path + " must have three of: lowercase letters, uppercase letters, digits and special characters."
	}
	return ""
}

// azureWorkloadsDisks checks a database layer's disk volumes: known volumes
// with a positive count and size and a known SKU, and for HANA every
// required volume.
func azureWorkloadsDisks(m map[string]any, path, key string, database any) string {
	raw, present := m[key]
	if !present {
		return ""
	}
	disk, ok := raw.(map[string]any)
	if !ok {
		return path + key + " must be an object."
	}
	volumes := azureARMObject(disk["diskVolumeConfigurations"])
	path += key + ".diskVolumeConfigurations"
	for _, name := range slices.Sorted(maps.Keys(volumes)) {
		if !slices.Contains(azureWorkloadsHANAVolumes, name) {
			return path + " can't contain '" + name + "'; volumes are " + strings.Join(azureWorkloadsHANAVolumes, ", ") + "."
		}
		volume := azureARMObject(volumes[name])
		if azureWorkloadsCount(volume, "count") < 1 || azureWorkloadsCount(volume, "sizeGB") < 1 {
			return path + "." + name + " needs a positive count and sizeGB."
		}
		if msg := azureLabEnum(azureARMObject(volume["sku"]), path+"."+name+".sku.", "name", azureWorkloadsDiskSKUs); msg != "" {
			return msg
		}
	}
	if database == "HANA" || database == nil {
		for _, name := range azureWorkloadsHANAVolumes[:5] {
			if _, ok := volumes[name]; !ok {
				return path + " needs the " + name + " volume for HANA."
			}
		}
	}
	return ""
}

// azureWorkloadsLayerNames checks a layer's custom names: one VM name
// entry per VM, and at most two central VMs (ASCS and ERS).
func azureWorkloadsLayerNames(names map[string]any, path string, count int, database bool) string {
	vms := azureARMSlice(names["virtualMachines"])
	if _, present := names["virtualMachines"]; present && len(vms) != count {
		return path + "virtualMachines must have one entry per VM (" + strconv.Itoa(count) + ")."
	}
	for _, vm := range vms {
		if msg := azureWorkloadsVMNames(azureARMObject(vm), path+"virtualMachines[].", database); msg != "" {
			return msg
		}
	}
	return ""
}

// azureWorkloadsVMNames checks Learn's VM naming limits: VM names up to 64
// characters, host names up to 13, one network interface, and data disk
// volumes of the layer (database volumes, or only "default").
func azureWorkloadsVMNames(vm map[string]any, path string, database bool) string {
	if len(azureARMString(vm["vmName"])) > 64 {
		return path + "vmName must be at most 64 characters."
	}
	if len(azureARMString(vm["hostName"])) > 13 {
		return path + "hostName must be at most 13 characters."
	}
	if len(azureARMSlice(vm["networkInterfaces"])) > 1 {
		return path + "networkInterfaces supports one network interface per VM."
	}
	for volume := range azureARMObject(vm["dataDiskNames"]) {
		if database && !slices.Contains(azureWorkloadsHANAVolumes, volume) || !database && volume != "default" {
			return path + "dataDiskNames can't name the '" + volume + "' volume."
		}
	}
	return ""
}

// azureWorkloadsMonitorValidate checks an Azure Monitor for SAP solutions
// resource.
func azureWorkloadsMonitorValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	if msg := azureWorkloadsIdentity(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureLabEnum(props, "properties.", "routingPreference", []string{"Default", "RouteAll"}); msg != "" {
		return msg
	}
	for _, id := range []struct {
		key, kind string
		re        *regexp.Regexp
	}{{"monitorSubnet", "subnet", azureWorkloadsSubnetID}, {"logAnalyticsWorkspaceArmId", "Log Analytics workspace", azureWorkloadsWorkspaceID}} {
		if raw, present := props[id.key]; present && !id.re.MatchString(azureARMString(raw)) {
			return "properties." + id.key + " must be a " + id.kind + " resource ID."
		}
	}
	if msg := azureDataBoxObjects(props, "properties.", "managedResourceGroupConfiguration"); msg != "" {
		return msg
	}
	if msg := azureWorkloadsMRGName(props); msg != "" {
		return msg
	}
	for _, key := range []string{"appLocation", "zoneRedundancyPreference"} {
		if raw, present := props[key]; present {
			if _, ok := raw.(string); !ok {
				return "properties." + key + " must be a string."
			}
		}
	}
	return ""
}

// azureWorkloadsProviderRequired are the settings each provider type needs,
// following Learn's provider how-to guides.
var azureWorkloadsProviderRequired = map[string][]string{
	"SapHana":             {"hostname", "dbUsername", "sapSid", "instanceNumber"},
	"SapNetWeaver":        {"sapSid", "sapHostname", "sapInstanceNr"},
	"PrometheusOS":        {"prometheusUrl"},
	"PrometheusHaCluster": {"prometheusUrl", "hostname", "sid", "clusterName"},
	"Db2":                 {"hostname", "dbName", "dbPort", "dbUsername", "sapSid"},
	"MsSqlServer":         {"hostname", "dbPort", "dbUsername", "sapSid"},
}

// azureWorkloadsProviderValidate checks a provider instance's settings.
func azureWorkloadsProviderValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	if msg := azureWorkloadsIdentity(payload); msg != "" {
		return msg
	}
	settings, ok := azureARMObject(payload["properties"])["providerSettings"].(map[string]any)
	if !ok {
		return "properties.providerSettings is required."
	}
	const path = "properties.providerSettings."
	kind := azureARMString(settings["providerType"])
	required, known := azureWorkloadsProviderRequired[kind]
	if !known {
		return path + "providerType must be one of Db2, MsSqlServer, PrometheusHaCluster, PrometheusOS, SapHana, SapNetWeaver."
	}
	if msg := azureWorkloadsRequired(settings, path, required...); msg != "" {
		return msg
	}
	for _, key := range []string{"instanceNumber", "sapInstanceNr"} {
		if raw, present := settings[key]; present && !azureWorkloadsInstanceNo.MatchString(azureARMString(raw)) {
			return path + key + " must be an instance number from 00 to 99."
		}
	}
	for _, key := range []string{"sqlPort", "dbPort", "sapPortNumber"} {
		if raw, present := settings[key]; present {
			if port, err := strconv.Atoi(azureARMString(raw)); err != nil || port < 1 || port > 65535 {
				return path + key + " must be a port from 1 to 65535."
			}
		}
	}
	if raw, present := settings["prometheusUrl"]; present && !azureWorkloadsPrometheusRe.MatchString(azureARMString(raw)) {
		return path + "prometheusUrl must be an http or https URL."
	}
	if msg := azureLabEnum(settings, path, "sslPreference", []string{"Disabled", "RootCertificate", "ServerCertificate"}); msg != "" {
		return msg
	}
	if settings["sslPreference"] == "ServerCertificate" && azureARMString(settings["sslCertificateUri"]) == "" {
		return path + "sslCertificateUri is required for a server certificate."
	}
	if password := map[string]string{"SapHana": "db", "Db2": "db", "MsSqlServer": "db"}[kind]; password != "" &&
		azureARMString(settings[password+"Password"]) == "" && azureARMString(settings[password+"PasswordUri"]) == "" {
		return path + "dbPassword or dbPasswordUri (a Key Vault secret) is required."
	}
	return ""
}

// azureWorkloadsLandscapeValidate checks a landscape monitor dashboard.
func azureWorkloadsLandscapeValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureDataBoxObjects(props, "properties.", "grouping"); msg != "" {
		return msg
	}
	grouping := azureARMObject(props["grouping"])
	for _, key := range []string{"landscape", "sapApplication"} {
		if msg := azureRecommendationsItems(grouping, key, func(item map[string]any) string {
			if msg := azureWorkloadsRequired(item, "properties.grouping."+key+"[].", "name"); msg != "" {
				return msg
			}
			return azureImpactStrings(item, "properties.grouping."+key+"[].", "topSid")
		}); msg != "" {
			return strings.Replace(msg, "properties."+key, "properties.grouping."+key, 1)
		}
	}
	return azureRecommendationsItems(props, "topMetricsThresholds", func(item map[string]any) string {
		if msg := azureWorkloadsRequired(item, "properties.topMetricsThresholds[].", "name"); msg != "" {
			return msg
		}
		for _, key := range []string{"green", "yellow", "red"} {
			if raw, present := item[key]; present {
				if _, ok := raw.(float64); !ok {
					return "properties.topMetricsThresholds[]." + key + " must be a number."
				}
			}
		}
		return ""
	})
}

// azureWorkloadsMRGName checks the optional managed resource group name.
func azureWorkloadsMRGName(props map[string]any) string {
	if mrg, present := azureARMObject(props["managedResourceGroupConfiguration"])["name"]; present {
		if name, ok := mrg.(string); !ok || strings.TrimSpace(name) == "" || len(name) > 90 {
			return "properties.managedResourceGroupConfiguration.name must be 1 to 90 characters."
		}
	}
	return ""
}
