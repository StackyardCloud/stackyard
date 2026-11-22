package server

type route53domainsOperation struct{ Name string }

// Amazon Route 53 Domains operations sourced from:
// https://docs.aws.amazon.com/Route53/latest/APIReference/API_Operations_Amazon_Route_53_Domains.html
var route53domainsOperations = []route53domainsOperation{
	{Name: "AcceptDomainTransferFromAnotherAwsAccount"},
	{Name: "AssociateDelegationSignerToDomain"},
	{Name: "CancelDomainTransferToAnotherAwsAccount"},
	{Name: "CheckDomainAvailability"},
	{Name: "CheckDomainTransferability"},
	{Name: "DeleteDomain"},
	{Name: "DeleteTagsForDomain"},
	{Name: "DisableDomainAutoRenew"},
	{Name: "DisableDomainTransferLock"},
	{Name: "DisassociateDelegationSignerFromDomain"},
	{Name: "EnableDomainAutoRenew"},
	{Name: "EnableDomainTransferLock"},
	{Name: "GetContactReachabilityStatus"},
	{Name: "GetDomainDetail"},
	{Name: "GetDomainPrices"},
	{Name: "GetDomainSuggestions"},
	{Name: "GetOperationDetail"},
	{Name: "ListDomains"},
	{Name: "ListOperations"},
	{Name: "ListPrices"},
	{Name: "ListTagsForDomain"},
	{Name: "PushDomain"},
	{Name: "RegisterDomain"},
	{Name: "RejectDomainTransferFromAnotherAwsAccount"},
	{Name: "RenewDomain"},
	{Name: "ResendContactReachabilityEmail"},
	{Name: "ResendOperationAuthorization"},
	{Name: "RestoreDomain"},
	{Name: "RetrieveDomainAuthCode"},
	{Name: "TransferDomain"},
	{Name: "TransferDomainToAnotherAwsAccount"},
	{Name: "UpdateDomainContact"},
	{Name: "UpdateDomainContactPrivacy"},
	{Name: "UpdateDomainNameservers"},
	{Name: "UpdateTagsForDomain"},
	{Name: "ViewBilling"},
}

var route53domainsOperationByName = func() map[string]route53domainsOperation {
	out := make(map[string]route53domainsOperation, len(route53domainsOperations))
	for _, op := range route53domainsOperations {
		out[op.Name] = op
	}
	return out
}()
