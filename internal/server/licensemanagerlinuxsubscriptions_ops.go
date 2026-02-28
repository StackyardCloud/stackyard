package server

type licenseManagerLinuxSubscriptionsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS License Manager Linux Subscriptions operations sourced from:
// https://docs.aws.amazon.com/license-manager-linux-subscriptions/latest/APIReference/API_Operations.html
var licenseManagerLinuxSubscriptionsOperations = []licenseManagerLinuxSubscriptionsOperation{
	{Name: "DeregisterSubscriptionProvider", Method: "POST", URI: "/subscription/DeregisterSubscriptionProvider"},
	{Name: "GetRegisteredSubscriptionProvider", Method: "POST", URI: "/subscription/GetRegisteredSubscriptionProvider"},
	{Name: "GetServiceSettings", Method: "POST", URI: "/subscription/GetServiceSettings"},
	{Name: "ListLinuxSubscriptionInstances", Method: "POST", URI: "/subscription/ListLinuxSubscriptionInstances"},
	{Name: "ListLinuxSubscriptions", Method: "POST", URI: "/subscription/ListLinuxSubscriptions"},
	{Name: "ListRegisteredSubscriptionProviders", Method: "POST", URI: "/subscription/ListRegisteredSubscriptionProviders"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "RegisterSubscriptionProvider", Method: "POST", URI: "/subscription/RegisterSubscriptionProvider"},
	{Name: "TagResource", Method: "PUT", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateServiceSettings", Method: "POST", URI: "/subscription/UpdateServiceSettings"},
}

var licenseManagerLinuxSubscriptionsOperationByName = func() map[string]licenseManagerLinuxSubscriptionsOperation {
	out := make(map[string]licenseManagerLinuxSubscriptionsOperation, len(licenseManagerLinuxSubscriptionsOperations))
	for _, op := range licenseManagerLinuxSubscriptionsOperations {
		out[op.Name] = op
	}
	return out
}()
