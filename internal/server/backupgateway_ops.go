package server

type backupGatewayOperation struct{ Name string }

// AWS Backup gateway operations sourced from:
// https://docs.aws.amazon.com/aws-backup/latest/devguide/API_Operations_AWS_Backup_gateway.html
var backupGatewayOperations = []backupGatewayOperation{
	{Name: "AssociateGatewayToServer"},
	{Name: "CreateGateway"},
	{Name: "DeleteGateway"},
	{Name: "DeleteHypervisor"},
	{Name: "DisassociateGatewayFromServer"},
	{Name: "GetBandwidthRateLimitSchedule"},
	{Name: "GetGateway"},
	{Name: "GetHypervisor"},
	{Name: "GetHypervisorPropertyMappings"},
	{Name: "GetVirtualMachine"},
	{Name: "ImportHypervisorConfiguration"},
	{Name: "ListGateways"},
	{Name: "ListHypervisors"},
	{Name: "ListTagsForResource"},
	{Name: "ListVirtualMachines"},
	{Name: "PutBandwidthRateLimitSchedule"},
	{Name: "PutHypervisorPropertyMappings"},
	{Name: "PutMaintenanceStartTime"},
	{Name: "StartVirtualMachinesMetadataSync"},
	{Name: "TagResource"},
	{Name: "TestHypervisorConfiguration"},
	{Name: "UntagResource"},
	{Name: "UpdateGatewayInformation"},
	{Name: "UpdateGatewaySoftwareNow"},
	{Name: "UpdateHypervisor"},
}

var backupGatewayOperationByName = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range backupGatewayOperations {
		out[op.Name] = true
	}
	return out
}()
