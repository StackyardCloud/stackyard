package server

import (
	"net/http"
	"slices"
)

// azureWorkloadsSKU is a VM size Azure Center for SAP solutions offers,
// with its memory in GiB from Learn's VM size pages. Database sizes carry
// the Premium SSD tiers Learn's HANA storage tables recommend for
// /hana/data (four disks), /hana/log (three), /hana/shared and the root
// volume (one each).
type azureWorkloadsSKU struct {
	name                    string
	memory                  int
	data, log, shared, root string
}

func (sku azureWorkloadsSKU) database() bool { return sku.data != "" }

// ponytail: a fixed catalog of E-series application and M-series HANA
// database sizes; add sizes when a scenario needs them. Learn lists no
// /hana/log row for M128ms; it takes the 3 x P15 of every larger size.
var azureWorkloadsSKUs = []azureWorkloadsSKU{
	{name: "Standard_E4ds_v4", memory: 32}, {name: "Standard_E8ds_v4", memory: 64}, {name: "Standard_E16ds_v4", memory: 128},
	{name: "Standard_E20ds_v4", memory: 160}, {name: "Standard_E32ds_v4", memory: 256}, {name: "Standard_E48ds_v4", memory: 384},
	{name: "Standard_E64ds_v4", memory: 504},
	{"Standard_M32ts", 192, "P6", "P10", "P15", "P6"}, {"Standard_M32ls", 256, "P6", "P10", "P15", "P6"},
	{"Standard_M64ls", 512, "P10", "P10", "P20", "P6"}, {"Standard_M64s", 1024, "P15", "P15", "P30", "P6"},
	{"Standard_M64ms", 1792, "P20", "P15", "P30", "P6"}, {"Standard_M128s", 2048, "P20", "P15", "P30", "P6"},
	{"Standard_M128ms", 3892, "P30", "P15", "P30", "P10"},
}

// azureWorkloadsPremiumDisks are Premium SSD tiers: size in GiB,
// provisioned IOPS and MBps.
var azureWorkloadsPremiumDisks = map[string][3]int{
	"P6": {64, 240, 50}, "P10": {128, 500, 100}, "P15": {256, 1100, 125}, "P20": {512, 2300, 150}, "P30": {1024, 5000, 200},
}

const azureWorkloadsMetadata = "sapVirtualInstanceMetadata/default/"

func azureWorkloadsRegisterMetadata() {
	for action, fn := range map[string]azureARMLocationAction{
		"getSizingRecommendations":   azureWorkloadsSizing,
		"getSapSupportedSku":         azureWorkloadsSupportedSKUs,
		"getDiskConfigurations":      azureWorkloadsDiskConfigurations,
		"getAvailabilityZoneDetails": azureWorkloadsZones,
	} {
		registerAzureARMLocationAction(azureWorkloadsNamespace, azureWorkloadsMetadata+action, fn)
	}
}

func azureWorkloadsBadRequest(msg string) (int, any) {
	return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
}

// azureWorkloadsSystem checks the system description the metadata actions
// share; deployment says whether deploymentType is part of it.
func azureWorkloadsSystem(payload map[string]any, deployment bool) string {
	required := []string{"appLocation", "databaseType", "sapProduct"}
	enums := map[string][]string{"databaseType": azureWorkloadsDatabases, "sapProduct": azureWorkloadsProducts}
	if deployment {
		required = append(required, "deploymentType", "environment")
		enums["deploymentType"], enums["environment"], enums["highAvailabilityType"] = azureWorkloadsDeployments, azureWorkloadsEnvironments, azureWorkloadsHATypes
	}
	if msg := azureWorkloadsEnums(payload, "", required, enums); msg != "" {
		return msg
	}
	if payload["highAvailabilityType"] != nil && payload["deploymentType"] == "SingleServer" {
		return "highAvailabilityType needs a ThreeTier deployment."
	}
	return ""
}

// azureWorkloadsSizing recommends VM sizes: the smallest database size
// with dbMemory GiB, and application servers for the SAPS.
// ponytail: Learn doesn't publish the sizing model; one E8ds_v4 application
// server per 20,000 SAPS is a stand-in.
func azureWorkloadsSizing(_ *Server, _, _ string, payload map[string]any) (int, any) {
	if msg := azureWorkloadsSystem(payload, true); msg != "" {
		return azureWorkloadsBadRequest(msg)
	}
	memory, saps := azureWorkloadsCount(payload, "dbMemory"), azureWorkloadsCount(payload, "saps")
	if memory < 1 || saps < 1 {
		return azureWorkloadsBadRequest("dbMemory and saps must be positive integers.")
	}
	if msg := azureLabEnum(payload, "", "dbScaleMethod", []string{"ScaleUp"}); msg != "" {
		return azureWorkloadsBadRequest(msg)
	}
	i := slices.IndexFunc(azureWorkloadsSKUs, func(sku azureWorkloadsSKU) bool { return sku.database() && sku.memory >= memory })
	if i < 0 {
		return azureWorkloadsBadRequest("No supported database VM size has dbMemory GiB of memory.")
	}
	if payload["deploymentType"] == "SingleServer" {
		return http.StatusOK, map[string]any{"deploymentType": "SingleServer", "vmSku": azureWorkloadsSKUs[i].name}
	}
	pair := 1
	if payload["highAvailabilityType"] != nil {
		pair = 2
	}
	return http.StatusOK, map[string]any{
		"deploymentType":         "ThreeTier",
		"applicationServerVmSku": "Standard_E8ds_v4", "applicationServerInstanceCount": max(pair, (saps+19999)/20000),
		"centralServerVmSku": "Standard_E4ds_v4", "centralServerInstanceCount": pair,
		"dbVmSku": azureWorkloadsSKUs[i].name, "databaseInstanceCount": pair,
	}
}

// azureWorkloadsSupportedSKUs lists the catalog, marking which sizes are
// certified for application servers and which for the database.
func azureWorkloadsSupportedSKUs(_ *Server, _, _ string, payload map[string]any) (int, any) {
	if msg := azureWorkloadsSystem(payload, true); msg != "" {
		return azureWorkloadsBadRequest(msg)
	}
	skus := make([]any, 0, len(azureWorkloadsSKUs))
	for _, sku := range azureWorkloadsSKUs {
		skus = append(skus, map[string]any{"vmSku": sku.name, "isAppServerCertified": !sku.database(), "isDatabaseCertified": sku.database()})
	}
	return http.StatusOK, map[string]any{"supportedSkus": skus}
}

// azureWorkloadsDiskConfigurations recommends the Premium SSD volumes of
// Learn's HANA storage tables for a database VM size, with a P6 for
// /usr/sap.
// ponytail: DB2 gets only the os and usr/sap volumes.
func azureWorkloadsDiskConfigurations(_ *Server, _, _ string, payload map[string]any) (int, any) {
	if msg := azureWorkloadsSystem(payload, true); msg != "" {
		return azureWorkloadsBadRequest(msg)
	}
	name := azureARMString(payload["dbVmSku"])
	i := slices.IndexFunc(azureWorkloadsSKUs, func(sku azureWorkloadsSKU) bool { return sku.database() && sku.name == name })
	if i < 0 {
		return azureWorkloadsBadRequest("dbVmSku must be a supported database VM size, such as Standard_M64s.")
	}
	sku := azureWorkloadsSKUs[i]
	volumes := map[string]any{"os": azureWorkloadsVolume(sku.root, 1), "usr/sap": azureWorkloadsVolume("P6", 1)}
	if payload["databaseType"] == "HANA" {
		volumes["hana/data"] = azureWorkloadsVolume(sku.data, 4)
		volumes["hana/log"] = azureWorkloadsVolume(sku.log, 3)
		volumes["hana/shared"] = azureWorkloadsVolume(sku.shared, 1)
	}
	return http.StatusOK, map[string]any{"volumeConfigurations": volumes}
}

// azureWorkloadsVolume is count Premium SSD disks of a tier.
func azureWorkloadsVolume(tier string, count int) map[string]any {
	disk := azureWorkloadsPremiumDisks[tier]
	sku := map[string]any{"name": "Premium_LRS"}
	return map[string]any{
		"recommendedConfiguration": map[string]any{"sku": sku, "count": count, "sizeGB": disk[0]},
		"supportedConfigurations": []any{map[string]any{
			"sku": sku, "sizeGB": disk[0], "minimumSupportedDiskCount": count, "maximumSupportedDiskCount": count,
			"iopsReadWrite": disk[1], "mbpsReadWrite": disk[2], "diskTier": tier,
		}},
	}
}

// azureWorkloadsZones returns the availability zone pair to deploy a
// zonal system across.
// ponytail: every region offers zones 1 and 2; model regional zone
// latency when a test needs other pairs.
func azureWorkloadsZones(_ *Server, _, _ string, payload map[string]any) (int, any) {
	if msg := azureWorkloadsSystem(payload, false); msg != "" {
		return azureWorkloadsBadRequest(msg)
	}
	return http.StatusOK, map[string]any{"availabilityZonePairs": []any{map[string]any{"zoneA": 1, "zoneB": 2}}}
}
