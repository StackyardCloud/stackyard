package server

type redshiftServerlessDataType struct {
	Name string
}

// Amazon Redshift Serverless data types sourced from:
// https://docs.aws.amazon.com/redshift-serverless/latest/APIReference/API_Types.html
var redshiftServerlessDataTypes = []redshiftServerlessDataType{
	{Name: "Association"},
	{Name: "ConfigParameter"},
	{Name: "CreateSnapshotScheduleActionParameters"},
	{Name: "Endpoint"},
	{Name: "EndpointAccess"},
	{Name: "ManagedWorkgroupListItem"},
	{Name: "Namespace"},
	{Name: "NetworkInterface"},
	{Name: "PerformanceTarget"},
	{Name: "RecoveryPoint"},
	{Name: "Reservation"},
	{Name: "ReservationOffering"},
	{Name: "ResourcePolicy"},
	{Name: "Schedule"},
	{Name: "ScheduledActionAssociation"},
	{Name: "ScheduledActionResponse"},
	{Name: "ServerlessTrack"},
	{Name: "Snapshot"},
	{Name: "SnapshotCopyConfiguration"},
	{Name: "TableRestoreStatus"},
	{Name: "Tag"},
	{Name: "TargetAction"},
	{Name: "UpdateTarget"},
	{Name: "UsageLimit"},
	{Name: "VpcEndpoint"},
	{Name: "VpcSecurityGroupMembership"},
	{Name: "Workgroup"},
}

var redshiftServerlessDataTypeByName = func() map[string]redshiftServerlessDataType {
	out := make(map[string]redshiftServerlessDataType, len(redshiftServerlessDataTypes))
	for _, dt := range redshiftServerlessDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
