package server

type licenseManagerLinuxSubscriptionsDataType struct {
	Name string
}

// AWS License Manager Linux Subscriptions data types sourced from:
// https://docs.aws.amazon.com/license-manager-linux-subscriptions/latest/APIReference/API_Types.html
var licenseManagerLinuxSubscriptionsDataTypes = []licenseManagerLinuxSubscriptionsDataType{
	{Name: "Filter"},
	{Name: "Instance"},
	{Name: "LinuxSubscriptionsDiscoverySettings"},
	{Name: "RegisteredSubscriptionProvider"},
	{Name: "Subscription"},
	{Name: "UpdateServiceSettings"},
}

var licenseManagerLinuxSubscriptionsDataTypeByName = func() map[string]licenseManagerLinuxSubscriptionsDataType {
	out := make(map[string]licenseManagerLinuxSubscriptionsDataType, len(licenseManagerLinuxSubscriptionsDataTypes))
	for _, dt := range licenseManagerLinuxSubscriptionsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
