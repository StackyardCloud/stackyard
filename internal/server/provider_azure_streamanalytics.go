package server

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.StreamAnalytics streaming jobs with their inputs, outputs and
// transformation, served by the shared ARM handler. Jobs start and stop but
// never process events.
func init() {
	const ns = "Microsoft.StreamAnalytics"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "streamingjobs",
		ReadOnlyProperties: azureStreamAnalyticsJobProperties,
		Validate:           azureStreamAnalyticsValidateJob,
		StateProperties:    map[string]any{"jobState": "Created"},
		Transitions:        map[string]map[string]any{"start": {"jobState": "Running"}, "stop": {"jobState": "Stopped"}},
		// Storage account keys are never returned.
		WriteOnlyProperties: []string{"jobStorageAccount.accountKey", "externals.storageAccount.accountKey"},
	})
	// Data source credentials are never returned.
	secrets := []string{
		"datasource.properties.sharedAccessPolicyKey",
		"datasource.properties.password",
		"datasource.properties.accountKey",
		"datasource.properties.refreshToken",
		"datasource.properties.apiKey",
		"datasource.properties.storageAccounts[].accountKey",
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "streamingjobs/inputs", Proxy: true,
		Validate: func(payload map[string]any) string {
			if kind := azureARMString(azureARMObject(payload["properties"])["type"]); kind != "Stream" && kind != "Reference" {
				return "properties.type must be Stream or Reference."
			}
			return azureStreamAnalyticsValidateDatasource(payload)
		},
		WriteOnlyProperties: secrets,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "streamingjobs/outputs", Proxy: true,
		Validate:            azureStreamAnalyticsValidateDatasource,
		WriteOnlyProperties: secrets,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "streamingjobs/transformations", Proxy: true,
		Validate: azureStreamAnalyticsValidateTransformation,
	})
}

var azureStreamAnalyticsJobEnums = map[string][]string{
	"eventsOutOfOrderPolicy": {"Adjust", "Drop"},
	"outputErrorPolicy":      {"Stop", "Drop"},
	"compatibilityLevel":     {"1.0", "1.1", "1.2"},
	"jobType":                {"Cloud", "Edge"},
}

func azureStreamAnalyticsValidateJob(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for key, values := range azureStreamAnalyticsJobEnums {
		if msg := azureEnumProperty(key, values...)(payload); msg != "" {
			return msg
		}
	}
	if sku, present := props["sku"]; present && !slices.Contains([]string{"Standard", "StandardV2"}, azureARMString(azureARMObject(sku)["name"])) {
		return "properties.sku.name must be Standard or StandardV2."
	}
	return ""
}

func azureStreamAnalyticsValidateDatasource(payload map[string]any) string {
	if strings.TrimSpace(azureARMString(azureARMObject(azureARMObject(payload["properties"])["datasource"])["type"])) == "" {
		return "properties.datasource.type is required."
	}
	return ""
}

// azureStreamAnalyticsValidateTransformation requires a query and, when set,
// a valid streaming unit count: 1, 3 or a multiple of 6 up to 396.
func azureStreamAnalyticsValidateTransformation(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["query"])) == "" {
		return "properties.query is required."
	}
	if raw, present := props["streamingUnits"]; present {
		n, ok := raw.(float64)
		valid := ok && n == float64(int(n)) && (n == 1 || n == 3 || (n >= 6 && n <= 396 && int(n)%6 == 0))
		if !valid {
			return "properties.streamingUnits must be 1, 3 or a multiple of 6 up to 396."
		}
	}
	return ""
}

// azureStreamAnalyticsJobProperties sets the job ID and creation date, and
// fills in the policy defaults.
func azureStreamAnalyticsJobProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"jobId":       uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"createdDate": rec.CreatedAt.Format(time.RFC3339),
	}
	for key, value := range map[string]string{"eventsOutOfOrderPolicy": "Adjust", "outputErrorPolicy": "Stop", "compatibilityLevel": "1.2", "jobType": "Cloud"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}
