package server

type acctOperation struct{ Name string }

// AWS Account operations sourced from:
// https://docs.aws.amazon.com/accounts/latest/reference/API_Operations.html
var acctOperations = []acctOperation{
	{Name: "AcceptPrimaryEmailUpdate"},
	{Name: "DeleteAlternateContact"},
	{Name: "DisableRegion"},
	{Name: "EnableRegion"},
	{Name: "GetAccountInformation"},
	{Name: "GetAlternateContact"},
	{Name: "GetContactInformation"},
	{Name: "GetGovCloudAccountInformation"},
	{Name: "GetPrimaryEmail"},
	{Name: "GetPrimaryEmailUpdateStatus"},
	{Name: "GetRegionOptStatus"},
	{Name: "ListRegions"},
	{Name: "PutAccountName"},
	{Name: "PutAlternateContact"},
	{Name: "PutContactInformation"},
	{Name: "SendPhoneNumberVerification"},
	{Name: "StartPrimaryEmailUpdate"},
	{Name: "VerifyPhoneNumber"},
}
