package server

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	azurePGClusterNamePattern = regexp.MustCompile(`^[a-z0-9-]{3,40}$`)
	azurePGCitusPattern       = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
)

// azurePGRetirement is when each PostgreSQL major version retires on Azure,
// from the Learn supported versions table. New clusters can't use a retired
// version; restores and read replicas of existing clusters still can.
var azurePGRetirement = map[string]time.Time{
	"11": time.Date(2023, 11, 9, 0, 0, 0, 0, time.UTC),
	"12": time.Date(2024, 11, 21, 0, 0, 0, 0, time.UTC),
	"13": time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC),
	"14": time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC),
	"15": time.Date(2027, 11, 11, 0, 0, 0, 0, time.UTC),
	"16": time.Date(2028, 11, 9, 0, 0, 0, 0, time.UTC),
}

// azurePGCitusMajor is the Citus major version each PostgreSQL version
// comes with, and azurePGCitusDefault the version a new cluster gets.
var (
	azurePGCitusMajor   = map[string]string{"11": "9", "12": "10", "13": "11", "14": "12", "15": "12", "16": "12"}
	azurePGCitusDefault = map[string]string{"13": "11.3", "14": "12.1", "15": "12.1", "16": "12.1"}
)

const azurePGDefaultVersion = "16"

// Compute and storage options from the Learn compute and storage page;
// storage is in MiB.
var (
	azurePGBurstableStorage   = []int{32768, 65536, 131072}
	azurePGSingleNodeStorage  = []int{131072, 262144, 524288, 1048576, 2097152}
	azurePGSingleNodeVCores   = []int{2, 4, 8, 16, 32, 64}
	azurePGCoordinatorVCores  = []int{4, 8, 16, 32, 64, 96}
	azurePGCoordinatorStorage = []int{131072, 262144, 524288, 1048576, 2097152, 4194304, 8388608, 16777216, 33554432}
	azurePGWorkerVCores       = []int{4, 8, 16, 32, 64, 96, 104}
	azurePGWorkerStorage      = []int{524288, 1048576, 2097152, 4194304, 8388608, 16777216, 33554432}
)

const azurePGMaxWorkers = 20

var (
	// azurePGClusterFixed can only be set at creation.
	azurePGClusterFixed = []string{"databaseName", "enableGeoBackup", "dataEncryption", "pointInTimeUTC", "sourceResourceId", "sourceLocation"}
	// azurePGClusterSettings are kept by a PUT that omits them.
	azurePGClusterSettings = []string{
		"authConfig", "citusVersion", "coordinatorEnablePublicIpAccess", "coordinatorServerEdition", "coordinatorStorageQuotaInMb",
		"coordinatorVCores", "enableHa", "enableShardsOnCoordinator", "maintenanceWindow", "nodeCount", "nodeEnablePublicIpAccess",
		"nodeServerEdition", "nodeStorageQuotaInMb", "nodeVCores", "postgresqlVersion", "preferredPrimaryZone",
	}
	// azurePGInherited are copied from the source of a restore or read
	// replica.
	azurePGInherited = []string{
		"postgresqlVersion", "citusVersion", "coordinatorServerEdition", "coordinatorStorageQuotaInMb", "coordinatorVCores",
		"nodeCount", "nodeServerEdition", "nodeStorageQuotaInMb", "nodeVCores", "databaseName", "enableShardsOnCoordinator",
	}
	// azurePGClusterUpdatable are ClusterForUpdate's properties.
	azurePGClusterUpdatable = []string{
		"administratorLoginPassword", "citusVersion", "coordinatorEnablePublicIpAccess", "coordinatorServerEdition",
		"coordinatorStorageQuotaInMb", "coordinatorVCores", "enableHa", "enableShardsOnCoordinator", "maintenanceWindow",
		"nodeCount", "nodeEnablePublicIpAccess", "nodeServerEdition", "nodeStorageQuotaInMb", "nodeVCores",
		"postgresqlVersion", "preferredPrimaryZone",
	}
)

// azurePGClusterValidateName applies Learn's cluster name rules: 3 to 40
// lowercase letters, digits and hyphens, not all digits and not starting
// or ending with a hyphen.
func azurePGClusterValidateName(name string) string {
	if !azurePGClusterNamePattern.MatchString(name) || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") ||
		strings.Trim(name, "0123456789") == "" {
		return "Cluster names must be 3 to 40 lowercase letters, numbers and hyphens, can't be only numbers and can't start or end with a hyphen."
	}
	return ""
}

func azurePGClusterValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties", "identity"); msg != "" {
		return msg
	}
	props, ok := payload["properties"].(map[string]any)
	if !ok {
		return "properties is required."
	}
	if msg := azurePGClusterFields(props); msg != "" {
		return msg
	}
	if _, _, msg := azureImpactTime(props, "properties.", "pointInTimeUTC"); msg != "" {
		return msg
	}
	if _, ok := props["administratorLoginPassword"]; !ok && azureARMString(props["sourceResourceId"]) == "" &&
		azureARMString(azureARMObject(props["authConfig"])["passwordAuth"]) != "disabled" {
		return "properties.administratorLoginPassword is required."
	}
	encryption := azureARMObject(props["dataEncryption"])
	if azureARMString(encryption["type"]) != "AzureKeyVault" {
		return ""
	}
	if props["enableGeoBackup"] == true {
		return "Customer-managed keys aren't supported with geo-redundant backup."
	}
	identity := azureARMString(encryption["primaryUserAssignedIdentityId"])
	for id := range azureARMObject(azureARMObject(payload["identity"])["userAssignedIdentities"]) {
		if strings.EqualFold(id, identity) {
			return ""
		}
	}
	return "properties.dataEncryption.primaryUserAssignedIdentityId must be one of identity.userAssignedIdentities."
}

// azurePGClusterPatch takes ClusterForUpdate: tags, identity and the
// properties that can change.
func azurePGClusterPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties", "identity"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	for key := range props {
		if !slices.Contains(azurePGClusterUpdatable, key) {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	return payload, nil, azurePGClusterFields(props)
}

// azurePGClusterFields checks the shapes and values of cluster properties.
func azurePGClusterFields(props map[string]any) string {
	for _, key := range []string{"coordinatorStorageQuotaInMb", "coordinatorVCores", "nodeCount", "nodeStorageQuotaInMb", "nodeVCores"} {
		if raw, present := props[key]; present && !azureWholeNumber(raw, 0, 1<<31-1) {
			return "properties." + key + " must be a whole number of at least 0."
		}
	}
	for _, key := range []string{"coordinatorEnablePublicIpAccess", "enableGeoBackup", "enableHa", "enableShardsOnCoordinator", "nodeEnablePublicIpAccess"} {
		if raw, present := props[key]; present {
			if _, ok := raw.(bool); !ok {
				return "properties." + key + " must be a boolean."
			}
		}
	}
	for _, key := range []string{"administratorLoginPassword", "citusVersion", "coordinatorServerEdition", "databaseName", "nodeServerEdition", "postgresqlVersion", "preferredPrimaryZone", "sourceLocation", "sourceResourceId"} {
		if raw, present := props[key]; present {
			if _, ok := raw.(string); !ok {
				return "properties." + key + " must be a string."
			}
		}
	}
	if raw, present := props["administratorLoginPassword"]; present {
		if msg := azurePGPassword(raw.(string), azurePGClusterAdmin, "properties.administratorLoginPassword"); msg != "" {
			return msg
		}
	}
	if version, present := props["postgresqlVersion"]; present {
		if _, ok := azurePGRetirement[version.(string)]; !ok {
			return "properties.postgresqlVersion must be a PostgreSQL major version from 11 to 16."
		}
	}
	if citus, present := props["citusVersion"]; present && !azurePGCitusPattern.MatchString(citus.(string)) {
		return "properties.citusVersion must be a Citus version such as 12.1."
	}
	if id, present := props["sourceResourceId"]; present && !strings.Contains(strings.ToLower(id.(string)), "/providers/microsoft.dbforpostgresql/servergroupsv2/") {
		return "properties.sourceResourceId must be a cluster ID."
	}
	if msg := azureDataBoxObjects(props, "properties.", "authConfig", "maintenanceWindow", "dataEncryption"); msg != "" {
		return msg
	}
	for _, key := range []string{"activeDirectoryAuth", "passwordAuth"} {
		if msg := azureLabEnum(azureARMObject(props["authConfig"]), "properties.authConfig.", key, []string{"enabled", "disabled"}); msg != "" {
			return msg
		}
	}
	if msg := azurePGMaintenanceWindow(azureARMObject(props["maintenanceWindow"])); msg != "" {
		return msg
	}
	encryption := azureARMObject(props["dataEncryption"])
	if msg := azureLabEnum(encryption, "properties.dataEncryption.", "type", []string{"AzureKeyVault", "SystemAssigned"}); msg != "" {
		return msg
	}
	if azureARMString(encryption["type"]) == "AzureKeyVault" &&
		(azureARMString(encryption["primaryKeyUri"]) == "" || azureARMString(encryption["primaryUserAssignedIdentityId"]) == "") {
		return "properties.dataEncryption.primaryKeyUri and primaryUserAssignedIdentityId are required for AzureKeyVault encryption."
	}
	return ""
}

func azurePGMaintenanceWindow(window map[string]any) string {
	if msg := azureLabEnum(window, "properties.maintenanceWindow.", "customWindow", []string{"Enabled", "Disabled"}); msg != "" {
		return msg
	}
	for key, most := range map[string]float64{"dayOfWeek": 6, "startHour": 23, "startMinute": 59} {
		if raw, present := window[key]; present && !azureWholeNumber(raw, 0, most) {
			return fmt.Sprintf("properties.maintenanceWindow.%s must be a whole number from 0 to %v.", key, most)
		}
	}
	return ""
}

// azurePGPassword applies Learn's password rules: 8 to 128 characters,
// with uppercase and lowercase letters, numbers and non-alphanumeric
// characters, and no three or more consecutive alphanumeric characters of
// the role name.
func azurePGPassword(password, role, path string) string {
	if n := utf8.RuneCountInString(password); n < 8 || n > 128 {
		return path + " must be 8 to 128 characters long."
	}
	var upper, lower, digit, other bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		default:
			other = true
		}
	}
	if !upper || !lower || !digit || !other {
		return path + " must contain uppercase letters, lowercase letters, numbers and non-alphanumeric characters."
	}
	name, folded := []rune(strings.ToLower(role)), strings.ToLower(password)
	for i := 0; i+3 <= len(name); i++ {
		part := string(name[i : i+3])
		if strings.IndexFunc(part, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) < 0 && strings.Contains(folded, part) {
			return path + " can't contain all or part of the role name '" + role + "'."
		}
	}
	return ""
}

// azurePGVersionLess compares dotted numeric versions such as 11.3.
func azurePGVersionLess(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(as), len(bs)) {
		x, _ := strconv.Atoi(as[i])
		y, _ := strconv.Atoi(bs[i])
		if x != y {
			return x < y
		}
	}
	return len(as) < len(bs)
}

func azurePGRetired(version string) bool {
	retires, ok := azurePGRetirement[version]
	return ok && !time.Now().Before(retires)
}

// azurePGClusterReconcile creates clusters, restores and read replicas,
// enforces the update rules and derives the server-owned properties.
func azurePGClusterReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := azureARMMerge(rec.Properties, nil)
	if existing == nil {
		if msg := s.azurePGClusterCreateLocked(rec, props); msg != "" {
			return nil, msg
		}
	} else if msg := azurePGClusterChange(existing.Properties, props); msg != "" {
		return nil, msg
	}
	nodes := azurePGInt(props, "nodeCount")
	for key, value := range map[string]any{
		"databaseName": "citus", "postgresqlVersion": azurePGDefaultVersion, "coordinatorServerEdition": "GeneralPurpose",
		"nodeServerEdition": "MemoryOptimized", "coordinatorEnablePublicIpAccess": true, "nodeEnablePublicIpAccess": false,
		"enableHa": false, "enableGeoBackup": false, "enableShardsOnCoordinator": nodes == 0,
		"maintenanceWindow": map[string]any{"customWindow": "Disabled", "dayOfWeek": 0, "startHour": 0, "startMinute": 0},
	} {
		if _, ok := props[key]; !ok {
			props[key] = value
		}
	}
	version := azureARMString(props["postgresqlVersion"])
	if _, ok := props["citusVersion"]; !ok {
		props["citusVersion"] = azurePGCitusDefault[version]
	}
	if citus := azureARMString(props["citusVersion"]); strings.Split(citus, ".")[0] != azurePGCitusMajor[version] {
		return nil, "properties.citusVersion " + citus + " doesn't match PostgreSQL " + version + ", which comes with Citus " + azurePGCitusMajor[version] + "."
	}
	if msg := azurePGClusterCompute(props); msg != "" {
		return nil, msg
	}
	auth := azureARMObject(props["authConfig"])
	aad, password := azureARMString(auth["activeDirectoryAuth"]), azureARMString(auth["passwordAuth"])
	if aad == "" {
		aad = "disabled"
	}
	if password == "" {
		password = "enabled"
	}
	switch {
	case aad == "disabled" && password == "disabled":
		return nil, "properties.authConfig must enable password or Microsoft Entra ID authentication."
	case aad == "enabled" && azurePGVersionLess(version, "14"):
		return nil, "Microsoft Entra ID authentication needs PostgreSQL 14 or later."
	}
	props["aadAuthEnabled"], props["passwordEnabled"] = aad, password
	servers := []any{}
	for _, name := range azurePGServerNames(rec.Name, nodes) {
		servers = append(servers, map[string]any{"name": name, "fullyQualifiedDomainName": name + azurePGClusterDNSSuffix})
	}
	props["serverNames"] = servers
	props["earliestRestoreTime"] = azurePGEarliestRestore(rec).Format(time.RFC3339Nano)
	return props, ""
}

// azurePGEarliestRestore is the oldest point a cluster can be restored to:
// its creation, or the start of the retention period.
func azurePGEarliestRestore(rec *azureARMResource) time.Time {
	created := rec.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	if window := time.Now().UTC().Add(-azurePGRestoreWindow); created.Before(window) {
		return window
	}
	return created
}

// azurePGClusterCreateLocked checks a new cluster's name and source, and
// copies a restore's or replica's settings from its source. Caller holds
// providerStorageMu.
func (s *Server) azurePGClusterCreateLocked(rec *azureARMResource, props map[string]any) string {
	if len(s.azureARMResourcesMatchingLocked(func(other *azureARMResource) bool {
		return strings.EqualFold(other.Type, azurePGClusterType) && strings.EqualFold(other.Name, rec.Name)
	})) > 0 {
		return "A cluster named '" + rec.Name + "' already exists."
	}
	sourceID := azureARMString(props["sourceResourceId"])
	if sourceID == "" {
		if azureARMString(props["pointInTimeUTC"]) != "" || azureARMString(props["sourceLocation"]) != "" {
			return "properties.sourceResourceId is required to restore a cluster or create a read replica."
		}
		for _, key := range []string{"coordinatorServerEdition", "coordinatorStorageQuotaInMb", "coordinatorVCores", "nodeCount"} {
			if _, ok := props[key]; !ok {
				return "properties." + key + " is required to create a cluster."
			}
		}
		version := azureARMString(props["postgresqlVersion"])
		if version == "" {
			version = azurePGDefaultVersion
		}
		if azurePGRetired(version) {
			return "PostgreSQL " + version + " is retired, so new clusters can't use it."
		}
		return ""
	}
	source := s.azurePGClusterLocked(sourceID)
	if source == nil {
		return "Source cluster " + sourceID + " was not found."
	}
	if loc := azureARMString(props["sourceLocation"]); loc != "" && !strings.EqualFold(strings.ReplaceAll(loc, " ", ""), strings.ReplaceAll(source.Location, " ", "")) {
		return "properties.sourceLocation must be the source cluster's region, " + source.Location + "."
	}
	props["sourceResourceId"], props["sourceLocation"] = source.ID, source.Location
	sp := source.Properties
	if at := azureARMString(props["pointInTimeUTC"]); at != "" {
		point, _ := time.Parse(time.RFC3339, at)
		earliest, now := azurePGEarliestRestore(source), time.Now().UTC()
		if point.Before(earliest) || point.After(now) {
			return "properties.pointInTimeUTC must be between " + earliest.Format(time.RFC3339) + " and now."
		}
	} else {
		if msg := s.azurePGReplicaSourceLocked(source); msg != "" {
			return msg
		}
		for _, key := range []string{"nodeCount", "coordinatorStorageQuotaInMb", "nodeStorageQuotaInMb"} {
			if _, sent := props[key]; sent && azurePGInt(props, key) != azurePGInt(sp, key) {
				return "properties." + key + " of a read replica must match its source cluster's."
			}
		}
	}
	for _, key := range azurePGInherited {
		if _, ok := props[key]; !ok && sp[key] != nil {
			props[key] = sp[key]
		}
	}
	return ""
}

// azurePGReplicaSourceLocked checks that a cluster can have another read
// replica. Caller holds providerStorageMu.
func (s *Server) azurePGReplicaSourceLocked(source *azureARMResource) string {
	switch {
	case azurePGIsReplica(source.Properties):
		return "A read replica can't be created from another read replica."
	case strings.HasPrefix(azureARMString(source.Properties["coordinatorServerEdition"]), "Burstable"):
		return "Clusters with burstable compute can't have read replicas."
	}
	replicas := s.azureARMResourcesMatchingLocked(func(other *azureARMResource) bool {
		return strings.EqualFold(other.Type, azurePGClusterType) && azurePGIsReplica(other.Properties) &&
			strings.EqualFold(azureARMString(other.Properties["sourceResourceId"]), source.ID)
	})
	if len(replicas) >= azurePGMaxReplicas {
		return fmt.Sprintf("Cluster %s already has %d read replicas, the most it can have.", source.Name, azurePGMaxReplicas)
	}
	return ""
}

// azurePGClusterChange keeps the properties set only at creation and those
// a PUT omits, and enforces the scaling and upgrade rules.
func azurePGClusterChange(old, props map[string]any) string {
	for _, key := range azurePGClusterFixed {
		stored, was := old[key]
		sent, is := props[key]
		switch {
		case is && !azurePGSame(sent, stored):
			return "properties." + key + " can only be set at creation."
		case was:
			props[key] = stored
		}
	}
	for _, key := range azurePGClusterSettings {
		if _, ok := props[key]; !ok && old[key] != nil {
			props[key] = old[key]
		}
	}
	if azurePGInt(props, "nodeCount") < azurePGInt(old, "nodeCount") {
		return "Worker nodes can't be removed, so properties.nodeCount can't decrease."
	}
	for _, key := range []string{"coordinatorStorageQuotaInMb", "nodeStorageQuotaInMb"} {
		if azurePGInt(props, key) < azurePGInt(old, key) {
			return "Storage can't be scaled down, so properties." + key + " can't decrease."
		}
	}
	if azurePGIsReplica(old) {
		for _, key := range []string{"nodeCount", "coordinatorStorageQuotaInMb", "nodeStorageQuotaInMb"} {
			if azurePGInt(props, key) != azurePGInt(old, key) {
				return "A read replica's storage and worker node count can't change."
			}
		}
	}
	oldVersion, version := azureARMString(old["postgresqlVersion"]), azureARMString(props["postgresqlVersion"])
	oldCitus, citus := azureARMString(old["citusVersion"]), azureARMString(props["citusVersion"])
	// An upgrade that leaves the Citus version as it was takes the new
	// PostgreSQL version's Citus; PATCH always carries the old one.
	if version != oldVersion && citus == oldCitus {
		citus = azurePGCitusDefault[version]
		props["citusVersion"] = citus
	}
	upgraded := version != oldVersion || citus != oldCitus
	switch {
	case azurePGVersionLess(version, oldVersion) || azurePGVersionLess(citus, oldCitus):
		return "PostgreSQL and Citus versions can only be upgraded."
	case upgraded && strings.HasPrefix(azureARMString(props["coordinatorServerEdition"]), "Burstable"):
		return "Clusters with burstable compute can't upgrade PostgreSQL or Citus."
	case version != oldVersion && azurePGRetired(version):
		return "PostgreSQL " + version + " is retired, so clusters can't upgrade to it."
	}
	return ""
}

// azurePGSame compares stored values; IDs and regions compare without
// regard to case, as ARM does.
func azurePGSame(a, b any) bool {
	if x, ok := a.(string); ok {
		y, ok := b.(string)
		return ok && strings.EqualFold(strings.ReplaceAll(x, " ", ""), strings.ReplaceAll(y, " ", ""))
	}
	return reflect.DeepEqual(a, b)
}

// azurePGClusterCompute checks the coordinator and worker compute and
// storage against the options for single node and multi-node clusters.
func azurePGClusterCompute(p map[string]any) string {
	nodes := azurePGInt(p, "nodeCount")
	if nodes == 1 || nodes > azurePGMaxWorkers {
		return fmt.Sprintf("properties.nodeCount must be 0 for a single node cluster or 2 to %d worker nodes.", azurePGMaxWorkers)
	}
	edition := azureARMString(p["coordinatorServerEdition"])
	cores, storage := azurePGInt(p, "coordinatorVCores"), azurePGInt(p, "coordinatorStorageQuotaInMb")
	if nodes == 0 {
		var allowedCores, allowedStorage []int
		switch edition {
		case "BurstableMemoryOptimized":
			allowedCores, allowedStorage = []int{1}, azurePGBurstableStorage
		case "BurstableGeneralPurpose":
			allowedCores, allowedStorage = []int{2}, azurePGBurstableStorage
		case "GeneralPurpose":
			allowedCores, allowedStorage = azurePGSingleNodeVCores, azurePGSingleNodeStorage
		default:
			return "properties.coordinatorServerEdition of a single node cluster must be BurstableMemoryOptimized, BurstableGeneralPurpose or GeneralPurpose."
		}
		if msg := azurePGCompute("coordinator", edition, cores, storage, allowedCores, allowedStorage); msg != "" {
			return msg
		}
		if edition == "GeneralPurpose" && storage >= 1048576 && cores < 8 {
			return "1024 GiB and 2048 GiB of storage need 8 or more vCores."
		}
		if strings.HasPrefix(edition, "Burstable") && (p["enableHa"] == true || p["enableGeoBackup"] == true) {
			return "Clusters with burstable compute don't support high availability or geo-redundant backup."
		}
		return ""
	}
	if edition != "GeneralPurpose" {
		return "properties.coordinatorServerEdition of a multi-node cluster must be GeneralPurpose."
	}
	if msg := azurePGCompute("coordinator", edition, cores, storage, azurePGCoordinatorVCores, azurePGCoordinatorStorage); msg != "" {
		return msg
	}
	if azureARMString(p["nodeServerEdition"]) != "MemoryOptimized" {
		return "properties.nodeServerEdition must be MemoryOptimized."
	}
	return azurePGCompute("node", "MemoryOptimized", azurePGInt(p, "nodeVCores"), azurePGInt(p, "nodeStorageQuotaInMb"), azurePGWorkerVCores, azurePGWorkerStorage)
}

func azurePGCompute(prefix, edition string, cores, storage int, allowedCores, allowedStorage []int) string {
	if !slices.Contains(allowedCores, cores) {
		return fmt.Sprintf("properties.%sVCores must be one of %v for %s.", prefix, allowedCores, edition)
	}
	if !slices.Contains(allowedStorage, storage) {
		return fmt.Sprintf("properties.%sStorageQuotaInMb must be one of %v for %s.", prefix, allowedStorage, edition)
	}
	return ""
}

func azurePGRoleValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props, ok := payload["properties"].(map[string]any)
	if !ok {
		return "properties is required."
	}
	if msg := azureLabEnum(props, "properties.", "roleType", []string{"user", "admin"}); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(props, "properties.", "externalIdentity"); msg != "" {
		return msg
	}
	if raw, present := props["password"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.password must be a string."
		}
	}
	identity, present := props["externalIdentity"].(map[string]any)
	if !present {
		if _, ok := props["password"]; !ok {
			return "properties.password is required unless the role uses properties.externalIdentity."
		}
		return ""
	}
	if azureARMString(identity["objectId"]) == "" {
		return "properties.externalIdentity.objectId is required."
	}
	if raw, present := identity["tenantId"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.externalIdentity.tenantId must be a string."
		}
	}
	if !slices.Contains([]string{"user", "servicePrincipal", "group"}, azureARMString(identity["principalType"])) {
		return "properties.externalIdentity.principalType must be user, servicePrincipal or group."
	}
	return ""
}

// azurePGRoleInCluster checks a role against its name and the cluster's
// authentication methods.
func azurePGRoleInCluster(cluster map[string]any, name string, payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if password, ok := props["password"].(string); ok {
		if msg := azurePGPassword(password, name, "properties.password"); msg != "" {
			return msg
		}
	}
	identity, external := props["externalIdentity"].(map[string]any)
	switch {
	case external && azureARMString(cluster["aadAuthEnabled"]) != "enabled":
		return "The cluster doesn't have Microsoft Entra ID authentication enabled."
	case external && identity["principalType"] == "group":
		return "Microsoft Entra groups aren't supported."
	case !external && azureARMString(cluster["passwordEnabled"]) == "disabled":
		return "The cluster doesn't allow password authentication."
	}
	return ""
}
