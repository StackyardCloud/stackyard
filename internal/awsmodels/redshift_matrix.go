package awsmodels

type RedshiftTestMatrixEntry struct {
	Operation        string
	Types            []string
	ValidationTests  []string
	IntegrationTests []string
}

func RedshiftTestMatrix() []RedshiftTestMatrixEntry {
	ops := RedshiftOperations()
	entries := make([]RedshiftTestMatrixEntry, 0, len(ops))

	defaultEntry := RedshiftTestMatrixEntry{
		ValidationTests:  []string{"not-implemented"},
		IntegrationTests: []string{"not-implemented"},
	}

	typeMap := map[string][]string{
		"AcceptReservedNodeExchange":                  {"ReservedNodeExchangeStatus"},
		"AddPartner":                                  {"PartnerIntegrationInfo"},
		"AuthorizeClusterSecurityGroupIngress":        {"ClusterSecurityGroup", "IPRange", "EC2SecurityGroup"},
		"AuthorizeEndpointAccess":                     {"EndpointAuthorization"},
		"AuthorizeSnapshotAccess":                     {"Snapshot", "AccountWithRestoreAccess"},
		"AuthorizeDataShare":                          {"DataShare", "DataShareAssociation"},
		"AssociateDataShareConsumer":                  {"DataShare", "DataShareAssociation"},
		"DisassociateDataShareConsumer":               {"DataShare", "DataShareAssociation"},
		"CancelResize":                                {"Cluster"},
		"CopyClusterSnapshot":                         {"Snapshot"},
		"CreateAuthenticationProfile":                 {"AuthenticationProfile"},
		"CreateCluster":                               {"Cluster"},
		"CreateClusterParameterGroup":                 {"ClusterParameterGroup"},
		"CreateClusterSecurityGroup":                  {"ClusterSecurityGroup"},
		"CreateClusterSnapshot":                       {"Snapshot"},
		"CreateClusterSubnetGroup":                    {"ClusterSubnetGroup"},
		"CreateCustomDomainAssociation":               {"CertificateAssociation"},
		"CreateEndpointAccess":                        {"EndpointAccess"},
		"CreateEventSubscription":                     {"EventSubscription"},
		"CreateHsmClientCertificate":                  {"HsmClientCertificate"},
		"CreateHsmConfiguration":                      {"HsmConfiguration"},
		"CreateIntegration":                           {"Integration"},
		"CreateRedshiftIdcApplication":                {"RedshiftIdcApplication"},
		"CreateScheduledAction":                       {"ScheduledAction"},
		"CreateSnapshotCopyGrant":                     {"SnapshotCopyGrant"},
		"CreateSnapshotSchedule":                      {"SnapshotSchedule"},
		"CreateTags":                                  {"Tag"},
		"CreateUsageLimit":                            {"UsageLimit"},
		"DeauthorizeDataShare":                        {"DataShare", "DataShareAssociation"},
		"DeleteAuthenticationProfile":                 {"AuthenticationProfile"},
		"DeleteCluster":                               {"Cluster"},
		"DeleteClusterParameterGroup":                 {"ClusterParameterGroup"},
		"DeleteClusterSecurityGroup":                  {"ClusterSecurityGroup"},
		"DeleteClusterSnapshot":                       {"Snapshot"},
		"DeleteClusterSubnetGroup":                    {"ClusterSubnetGroup"},
		"DeleteCustomDomainAssociation":               {"CertificateAssociation"},
		"DeleteEndpointAccess":                        {"EndpointAccess"},
		"DeleteEventSubscription":                     {"EventSubscription"},
		"DeleteHsmClientCertificate":                  {"HsmClientCertificate"},
		"DeleteHsmConfiguration":                      {"HsmConfiguration"},
		"DeleteIntegration":                           {"Integration"},
		"DeletePartner":                               {"PartnerIntegrationInfo"},
		"DeleteRedshiftIdcApplication":                {"RedshiftIdcApplication"},
		"DeleteResourcePolicy":                        {"ResourcePolicy"},
		"DeleteScheduledAction":                       {"ScheduledAction"},
		"DeleteSnapshotCopyGrant":                     {"SnapshotCopyGrant"},
		"DeleteSnapshotSchedule":                      {"SnapshotSchedule"},
		"DeleteTags":                                  {"Tag"},
		"DeleteUsageLimit":                            {"UsageLimit"},
		"DeregisterNamespace":                         {"NamespaceIdentifierUnion"},
		"DescribeClusters":                            {"Cluster"},
		"DescribeAccountAttributes":                   {"AccountAttribute"},
		"DescribeAuthenticationProfiles":              {"AuthenticationProfile"},
		"DescribeClusterDbRevisions":                  {"ClusterDbRevision"},
		"DescribeClusterParameterGroups":              {"ClusterParameterGroup"},
		"DescribeClusterParameters":                   {"ClusterParameterStatus"},
		"DescribeClusterSecurityGroups":               {"ClusterSecurityGroup"},
		"DescribeClusterSnapshots":                    {"Snapshot"},
		"DescribeClusterSubnetGroups":                 {"ClusterSubnetGroup"},
		"DescribeClusterTracks":                       {"MaintenanceTrack"},
		"DescribeClusterVersions":                     {"ClusterVersion"},
		"DescribeCustomDomainAssociations":            {"CertificateAssociation"},
		"DescribeDefaultClusterParameters":            {"DefaultClusterParameters"},
		"DescribeEndpointAccess":                      {"EndpointAccess"},
		"DescribeEndpointAuthorization":               {"EndpointAuthorization"},
		"DescribeEventCategories":                     {"EventCategoriesMap"},
		"DescribeEvents":                              {"Event"},
		"DescribeEventSubscriptions":                  {"EventSubscription"},
		"DescribeHsmClientCertificates":               {"HsmClientCertificate"},
		"DescribeHsmConfigurations":                   {"HsmConfiguration"},
		"DescribeInboundIntegrations":                 {"Integration"},
		"DescribeIntegrations":                        {"Integration"},
		"DescribeLoggingStatus":                       {"Cluster"},
		"DescribeNodeConfigurationOptions":            {"NodeConfigurationOption"},
		"DescribeOrderableClusterOptions":             {"OrderableClusterOption"},
		"DescribePartners":                            {"PartnerIntegrationInfo"},
		"DescribeRedshiftIdcApplications":             {"RedshiftIdcApplication"},
		"DescribeReservedNodeExchangeStatus":          {"ReservedNodeExchangeStatus"},
		"DescribeReservedNodeOfferings":               {"ReservedNodeOffering"},
		"DescribeReservedNodes":                       {"ReservedNode"},
		"DescribeResize":                              {"Cluster"},
		"DescribeScheduledActions":                    {"ScheduledAction"},
		"DescribeSnapshotCopyGrants":                  {"SnapshotCopyGrant"},
		"DescribeSnapshotSchedules":                   {"SnapshotSchedule"},
		"DescribeStorage":                             {"Cluster"},
		"DescribeTableRestoreStatus":                  {"TableRestoreStatus"},
		"DescribeTags":                                {"Tag"},
		"DescribeUsageLimits":                         {"UsageLimit"},
		"DescribeDataShares":                          {"DataShare"},
		"DescribeDataSharesForConsumer":               {"DataShare"},
		"DescribeDataSharesForProducer":               {"DataShare"},
		"DisableLogging":                              {"Cluster"},
		"DisableSnapshotCopy":                         {"ClusterSnapshotCopyStatus"},
		"EnableLogging":                               {"Cluster"},
		"EnableSnapshotCopy":                          {"ClusterSnapshotCopyStatus"},
		"FailoverPrimaryCompute":                      {"Cluster"},
		"GetClusterCredentials":                       {"Cluster"},
		"GetClusterCredentialsWithIAM":                {"Cluster"},
		"GetIdentityCenterAuthToken":                  {"RedshiftIdcApplication"},
		"GetReservedNodeExchangeConfigurationOptions": {"ReservedNodeConfigurationOption"},
		"GetReservedNodeExchangeOfferings":            {"ReservedNodeOffering"},
		"GetResourcePolicy":                           {"ResourcePolicy"},
		"ListRecommendations":                         {"Recommendation"},
		"ModifyAquaConfiguration":                     {"AquaConfiguration"},
		"ModifyAuthenticationProfile":                 {"AuthenticationProfile"},
		"ModifyCluster":                               {"Cluster"},
		"ModifyClusterDbRevision":                     {"ClusterDbRevision"},
		"ModifyClusterIamRoles":                       {"ClusterIamRole"},
		"ModifyClusterMaintenance":                    {"Cluster"},
		"ModifyEndpointAccess":                        {"EndpointAccess"},
		"ModifyClusterSnapshot":                       {"Snapshot"},
		"ModifyClusterSnapshotSchedule":               {"ClusterAssociatedToSchedule"},
		"ModifyClusterParameterGroup":                 {"ClusterParameterStatus"},
		"ModifyClusterSubnetGroup":                    {"ClusterSubnetGroup"},
		"ModifyCustomDomainAssociation":               {"CertificateAssociation"},
		"ModifyEventSubscription":                     {"EventSubscription"},
		"ModifyIntegration":                           {"Integration"},
		"ModifyLakehouseConfiguration":                {"Cluster"},
		"ModifyRedshiftIdcApplication":                {"RedshiftIdcApplication"},
		"ModifyScheduledAction":                       {"ScheduledAction"},
		"ModifySnapshotCopyRetentionPeriod":           {"ClusterSnapshotCopyStatus"},
		"ModifySnapshotSchedule":                      {"SnapshotSchedule"},
		"ModifyUsageLimit":                            {"UsageLimit"},
		"PauseCluster":                                {"Cluster"},
		"PurchaseReservedNodeOffering":                {"ReservedNode"},
		"PutResourcePolicy":                           {"ResourcePolicy"},
		"RebootCluster":                               {"Cluster"},
		"RegisterNamespace":                           {"NamespaceIdentifierUnion"},
		"RevokeClusterSecurityGroupIngress":           {"ClusterSecurityGroup", "IPRange", "EC2SecurityGroup"},
		"RevokeEndpointAccess":                        {"EndpointAuthorization"},
		"RevokeSnapshotAccess":                        {"Snapshot", "AccountWithRestoreAccess"},
		"RejectDataShare":                             {"DataShare", "DataShareAssociation"},
		"ResetClusterParameterGroup":                  {"ClusterParameterStatus"},
		"ResizeCluster":                               {"Cluster"},
		"RestoreFromClusterSnapshot":                  {"Snapshot"},
		"RestoreTableFromClusterSnapshot":             {"TableRestoreStatus"},
		"ResumeCluster":                               {"Cluster"},
		"RotateEncryptionKey":                         {"Cluster"},
		"UpdatePartnerStatus":                         {"PartnerIntegrationInfo"},
		"BatchDeleteClusterSnapshots":                 {"Snapshot"},
		"BatchModifyClusterSnapshots":                 {"Snapshot"},
	}

	testMap := map[string]RedshiftTestMatrixEntry{
		"AcceptReservedNodeExchange": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"reserved-node-exchange-accept"},
		},
		"AddPartner": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"partner-add"},
		},
		"AuthorizeClusterSecurityGroupIngress": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"security-group-ingress-authorize"},
		},
		"AuthorizeEndpointAccess": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"endpoint-access-authorize"},
		},
		"AuthorizeSnapshotAccess": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-access-authorize"},
		},
		"AuthorizeDataShare": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"datashare-authorize"},
		},
		"AssociateDataShareConsumer": {
			ValidationTests:  []string{"missing-datashare"},
			IntegrationTests: []string{"datashare-associate"},
		},
		"DisassociateDataShareConsumer": {
			ValidationTests:  []string{"missing-datashare"},
			IntegrationTests: []string{"datashare-disassociate"},
		},
		"CancelResize": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"resize-cancel"},
		},
		"CopyClusterSnapshot": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-copy"},
		},
		"CreateAuthenticationProfile": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"authentication-profile-create"},
		},
		"CreateCluster": {
			ValidationTests:  []string{"required-fields", "enum-validation"},
			IntegrationTests: []string{"cluster-create"},
		},
		"CreateClusterParameterGroup": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"param-group-create"},
		},
		"CreateClusterSecurityGroup": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"security-group-create"},
		},
		"CreateClusterSnapshot": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-create"},
		},
		"CreateClusterSubnetGroup": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"subnet-group-create"},
		},
		"CreateCustomDomainAssociation": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"custom-domain-create"},
		},
		"CreateEndpointAccess": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"endpoint-access-create"},
		},
		"CreateEventSubscription": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"event-subscription-create"},
		},
		"CreateHsmClientCertificate": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"hsm-cert-create"},
		},
		"CreateHsmConfiguration": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"hsm-config-create"},
		},
		"CreateIntegration": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"integration-create"},
		},
		"CreateRedshiftIdcApplication": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"idc-application-create"},
		},
		"CreateScheduledAction": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"scheduled-action-create"},
		},
		"CreateSnapshotCopyGrant": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-copy-grant-create"},
		},
		"CreateSnapshotSchedule": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-schedule-create"},
		},
		"CreateTags": {
			ValidationTests:  []string{"tag-limits"},
			IntegrationTests: []string{"tags-create"},
		},
		"CreateUsageLimit": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"usage-limit-create"},
		},
		"DeauthorizeDataShare": {
			ValidationTests:  []string{"missing-datashare"},
			IntegrationTests: []string{"datashare-deauthorize"},
		},
		"DeleteAuthenticationProfile": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"authentication-profile-delete"},
		},
		"DeleteCluster": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"cluster-delete"},
		},
		"DeleteClusterParameterGroup": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"param-group-delete"},
		},
		"DeleteClusterSecurityGroup": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"security-group-delete"},
		},
		"DeleteClusterSnapshot": {
			ValidationTests:  []string{"missing-snapshot"},
			IntegrationTests: []string{"snapshot-delete"},
		},
		"DeleteClusterSubnetGroup": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"subnet-group-delete"},
		},
		"DeleteCustomDomainAssociation": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"custom-domain-delete"},
		},
		"DeleteEndpointAccess": {
			ValidationTests:  []string{"missing-endpoint"},
			IntegrationTests: []string{"endpoint-access-delete"},
		},
		"DeleteEventSubscription": {
			ValidationTests:  []string{"missing-subscription"},
			IntegrationTests: []string{"event-subscription-delete"},
		},
		"DeleteHsmClientCertificate": {
			ValidationTests:  []string{"missing-cert"},
			IntegrationTests: []string{"hsm-cert-delete"},
		},
		"DeleteHsmConfiguration": {
			ValidationTests:  []string{"missing-config"},
			IntegrationTests: []string{"hsm-config-delete"},
		},
		"DeleteIntegration": {
			ValidationTests:  []string{"missing-integration"},
			IntegrationTests: []string{"integration-delete"},
		},
		"DeletePartner": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"partner-delete"},
		},
		"DeleteRedshiftIdcApplication": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"idc-application-delete"},
		},
		"DeleteResourcePolicy": {
			ValidationTests:  []string{"missing-policy"},
			IntegrationTests: []string{"resource-policy-delete"},
		},
		"DeleteScheduledAction": {
			ValidationTests:  []string{"missing-action"},
			IntegrationTests: []string{"scheduled-action-delete"},
		},
		"DeleteSnapshotCopyGrant": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-copy-grant-delete"},
		},
		"DeleteSnapshotSchedule": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-schedule-delete"},
		},
		"DeleteTags": {
			ValidationTests:  []string{"tag-limits"},
			IntegrationTests: []string{"tags-delete"},
		},
		"DeleteUsageLimit": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"usage-limit-delete"},
		},
		"DeregisterNamespace": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"namespace-deregister"},
		},
		"DescribeClusters": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"cluster-describe"},
		},
		"DescribeAccountAttributes": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"account-attributes-describe"},
		},
		"DescribeAuthenticationProfiles": {
			ValidationTests:  []string{"missing-profile"},
			IntegrationTests: []string{"authentication-profiles-describe"},
		},
		"DescribeClusterDbRevisions": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"db-revisions-describe"},
		},
		"DescribeClusterParameterGroups": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"param-group-describe"},
		},
		"DescribeClusterParameters": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"param-describe"},
		},
		"DescribeClusterTracks": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"cluster-tracks-describe"},
		},
		"DescribeClusterVersions": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"cluster-versions-describe"},
		},
		"DescribeClusterSecurityGroups": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"security-group-describe"},
		},
		"DescribeClusterSnapshots": {
			ValidationTests:  []string{"missing-snapshot"},
			IntegrationTests: []string{"snapshot-describe"},
		},
		"DescribeClusterSubnetGroups": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"subnet-group-describe"},
		},
		"DescribeCustomDomainAssociations": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"custom-domain-describe"},
		},
		"DescribeDefaultClusterParameters": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"default-params-describe"},
		},
		"DescribeEndpointAccess": {
			ValidationTests:  []string{"missing-endpoint"},
			IntegrationTests: []string{"endpoint-access-describe"},
		},
		"DescribeEndpointAuthorization": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"endpoint-authorization-describe"},
		},
		"DescribeEventCategories": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"event-categories-describe"},
		},
		"DescribeEvents": {
			ValidationTests:  []string{"invalid-duration"},
			IntegrationTests: []string{"events-describe"},
		},
		"DescribeEventSubscriptions": {
			ValidationTests:  []string{"missing-subscription"},
			IntegrationTests: []string{"event-subscription-describe"},
		},
		"DescribeHsmClientCertificates": {
			ValidationTests:  []string{"missing-cert"},
			IntegrationTests: []string{"hsm-cert-describe"},
		},
		"DescribeHsmConfigurations": {
			ValidationTests:  []string{"missing-config"},
			IntegrationTests: []string{"hsm-config-describe"},
		},
		"DescribeInboundIntegrations": {
			ValidationTests:  []string{"missing-integration"},
			IntegrationTests: []string{"integration-inbound"},
		},
		"DescribeIntegrations": {
			ValidationTests:  []string{"missing-integration"},
			IntegrationTests: []string{"integration-describe"},
		},
		"DescribeLoggingStatus": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"logging-describe"},
		},
		"DescribeNodeConfigurationOptions": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"node-config-options-describe"},
		},
		"DescribeOrderableClusterOptions": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"orderable-options-describe"},
		},
		"DescribePartners": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"partners-describe"},
		},
		"DescribeRedshiftIdcApplications": {
			ValidationTests:  []string{"missing-application"},
			IntegrationTests: []string{"idc-applications-describe"},
		},
		"DescribeReservedNodeExchangeStatus": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"reserved-node-exchange-status"},
		},
		"DescribeReservedNodeOfferings": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"reserved-node-offerings-describe"},
		},
		"DescribeReservedNodes": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"reserved-nodes-describe"},
		},
		"DescribeResize": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"resize-describe"},
		},
		"DescribeScheduledActions": {
			ValidationTests:  []string{"missing-action"},
			IntegrationTests: []string{"scheduled-action-describe"},
		},
		"DescribeSnapshotCopyGrants": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"snapshot-copy-grants-describe"},
		},
		"DescribeSnapshotSchedules": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"snapshot-schedules-describe"},
		},
		"DescribeStorage": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"storage-describe"},
		},
		"DescribeTableRestoreStatus": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"table-restore-status-describe"},
		},
		"DescribeTags": {
			ValidationTests:  []string{"tag-limits"},
			IntegrationTests: []string{"tags-describe"},
		},
		"DescribeUsageLimits": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"usage-limits-describe"},
		},
		"DescribeDataShares": {
			ValidationTests:  []string{"missing-datashare"},
			IntegrationTests: []string{"datashare-describe"},
		},
		"DescribeDataSharesForConsumer": {
			ValidationTests:  []string{"missing-datashare"},
			IntegrationTests: []string{"datashare-consumer"},
		},
		"DescribeDataSharesForProducer": {
			ValidationTests:  []string{"missing-datashare"},
			IntegrationTests: []string{"datashare-producer"},
		},
		"DisableLogging": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"logging-disable"},
		},
		"DisableSnapshotCopy": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"snapshot-copy-disable"},
		},
		"EnableLogging": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"logging-enable"},
		},
		"EnableSnapshotCopy": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-copy-enable"},
		},
		"FailoverPrimaryCompute": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"primary-compute-failover"},
		},
		"GetClusterCredentials": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"credentials-get"},
		},
		"GetClusterCredentialsWithIAM": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"credentials-iam"},
		},
		"GetIdentityCenterAuthToken": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"idc-auth-token-get"},
		},
		"GetReservedNodeExchangeConfigurationOptions": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"reserved-node-exchange-options"},
		},
		"GetReservedNodeExchangeOfferings": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"reserved-node-exchange-offerings"},
		},
		"GetResourcePolicy": {
			ValidationTests:  []string{"missing-policy"},
			IntegrationTests: []string{"resource-policy-get"},
		},
		"ListRecommendations": {
			ValidationTests:  []string{"pagination"},
			IntegrationTests: []string{"recommendations-list"},
		},
		"ModifyAquaConfiguration": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"aqua-configuration-modify"},
		},
		"ModifyAuthenticationProfile": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"authentication-profile-modify"},
		},
		"ModifyCluster": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"cluster-modify"},
		},
		"ModifyClusterDbRevision": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"cluster-db-revision-modify"},
		},
		"ModifyClusterIamRoles": {
			ValidationTests:  []string{"missing-role"},
			IntegrationTests: []string{"iam-roles-modify"},
		},
		"ModifyClusterMaintenance": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"maintenance-modify"},
		},
		"ModifyEndpointAccess": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"endpoint-access-modify"},
		},
		"ModifyClusterSnapshot": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"cluster-snapshot-modify"},
		},
		"ModifyClusterSnapshotSchedule": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"cluster-snapshot-schedule-modify"},
		},
		"ModifyClusterParameterGroup": {
			ValidationTests:  []string{"missing-parameter"},
			IntegrationTests: []string{"param-modify"},
		},
		"ModifyClusterSubnetGroup": {
			ValidationTests:  []string{"missing-group"},
			IntegrationTests: []string{"subnet-group-modify"},
		},
		"ModifyCustomDomainAssociation": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"custom-domain-modify"},
		},
		"ModifyEventSubscription": {
			ValidationTests:  []string{"missing-subscription"},
			IntegrationTests: []string{"event-subscription-modify"},
		},
		"ModifyIntegration": {
			ValidationTests:  []string{"missing-integration"},
			IntegrationTests: []string{"integration-modify"},
		},
		"ModifyLakehouseConfiguration": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"lakehouse-configuration-modify"},
		},
		"ModifyRedshiftIdcApplication": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"idc-application-modify"},
		},
		"ModifyScheduledAction": {
			ValidationTests:  []string{"missing-action"},
			IntegrationTests: []string{"scheduled-action-modify"},
		},
		"ModifySnapshotCopyRetentionPeriod": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-copy-retention-modify"},
		},
		"ModifySnapshotSchedule": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-schedule-modify"},
		},
		"ModifyUsageLimit": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"usage-limit-modify"},
		},
		"PauseCluster": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"cluster-pause"},
		},
		"PurchaseReservedNodeOffering": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"reserved-node-purchase"},
		},
		"PutResourcePolicy": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"resource-policy-put"},
		},
		"RebootCluster": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"cluster-reboot"},
		},
		"RegisterNamespace": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"namespace-register"},
		},
		"RevokeClusterSecurityGroupIngress": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"security-group-ingress-revoke"},
		},
		"RevokeEndpointAccess": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"endpoint-access-revoke"},
		},
		"RevokeSnapshotAccess": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-access-revoke"},
		},
		"RejectDataShare": {
			ValidationTests:  []string{"missing-datashare"},
			IntegrationTests: []string{"datashare-reject"},
		},
		"ResetClusterParameterGroup": {
			ValidationTests:  []string{"missing-parameter"},
			IntegrationTests: []string{"param-reset"},
		},
		"ResizeCluster": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"resize"},
		},
		"RestoreFromClusterSnapshot": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-restore"},
		},
		"RestoreTableFromClusterSnapshot": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"table-restore"},
		},
		"ResumeCluster": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"cluster-resume"},
		},
		"RotateEncryptionKey": {
			ValidationTests:  []string{"missing-cluster"},
			IntegrationTests: []string{"encryption-key-rotate"},
		},
		"UpdatePartnerStatus": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"partner-status-update"},
		},
		"BatchDeleteClusterSnapshots": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-batch-delete"},
		},
		"BatchModifyClusterSnapshots": {
			ValidationTests:  []string{"required-fields"},
			IntegrationTests: []string{"snapshot-batch-modify"},
		},
	}

	for _, op := range ops {
		entry := defaultEntry
		entry.Operation = op
		if types, ok := typeMap[op]; ok {
			entry.Types = types
		}
		if override, ok := testMap[op]; ok {
			entry.ValidationTests = override.ValidationTests
			entry.IntegrationTests = override.IntegrationTests
		}
		entries = append(entries, entry)
	}
	return entries
}
