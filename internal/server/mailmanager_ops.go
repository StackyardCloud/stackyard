package server

type mailManagerOperation struct{ Name string }

// Amazon SES Mail Manager operations sourced from:
// https://docs.aws.amazon.com/sesmailmanager/latest/APIReference/API_Operations.html
var mailManagerOperations = []mailManagerOperation{
	{Name: "CreateAddonInstance"},
	{Name: "CreateAddonSubscription"},
	{Name: "CreateAddressList"},
	{Name: "CreateAddressListImportJob"},
	{Name: "CreateArchive"},
	{Name: "CreateIngressPoint"},
	{Name: "CreateRelay"},
	{Name: "CreateRuleSet"},
	{Name: "CreateTrafficPolicy"},
	{Name: "DeleteAddonInstance"},
	{Name: "DeleteAddonSubscription"},
	{Name: "DeleteAddressList"},
	{Name: "DeleteArchive"},
	{Name: "DeleteIngressPoint"},
	{Name: "DeleteRelay"},
	{Name: "DeleteRuleSet"},
	{Name: "DeleteTrafficPolicy"},
	{Name: "DeregisterMemberFromAddressList"},
	{Name: "GetAddonInstance"},
	{Name: "GetAddonSubscription"},
	{Name: "GetAddressList"},
	{Name: "GetAddressListImportJob"},
	{Name: "GetArchive"},
	{Name: "GetArchiveExport"},
	{Name: "GetArchiveMessage"},
	{Name: "GetArchiveMessageContent"},
	{Name: "GetArchiveSearch"},
	{Name: "GetArchiveSearchResults"},
	{Name: "GetIngressPoint"},
	{Name: "GetMemberOfAddressList"},
	{Name: "GetRelay"},
	{Name: "GetRuleSet"},
	{Name: "GetTrafficPolicy"},
	{Name: "ListAddonInstances"},
	{Name: "ListAddonSubscriptions"},
	{Name: "ListAddressListImportJobs"},
	{Name: "ListAddressLists"},
	{Name: "ListArchiveExports"},
	{Name: "ListArchiveSearches"},
	{Name: "ListArchives"},
	{Name: "ListIngressPoints"},
	{Name: "ListMembersOfAddressList"},
	{Name: "ListRelays"},
	{Name: "ListRuleSets"},
	{Name: "ListTagsForResource"},
	{Name: "ListTrafficPolicies"},
	{Name: "RegisterMemberToAddressList"},
	{Name: "StartAddressListImportJob"},
	{Name: "StartArchiveExport"},
	{Name: "StartArchiveSearch"},
	{Name: "StopAddressListImportJob"},
	{Name: "StopArchiveExport"},
	{Name: "StopArchiveSearch"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateArchive"},
	{Name: "UpdateIngressPoint"},
	{Name: "UpdateRelay"},
	{Name: "UpdateRuleSet"},
	{Name: "UpdateTrafficPolicy"},
}

var mailManagerOperationByName = func() map[string]bool {
	out := make(map[string]bool, len(mailManagerOperations))
	for _, op := range mailManagerOperations {
		out[op.Name] = true
	}
	return out
}()
