package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func readRedshiftFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", "redshift", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

func assertXMLFixtureEqual(t *testing.T, got []byte, fixture string) {
	t.Helper()
	want := bytes.TrimSpace(readRedshiftFixture(t, fixture))
	if !bytes.Equal(bytes.TrimSpace(got), want) {
		t.Fatalf("unexpected XML response: got=%s want=%s", string(bytes.TrimSpace(got)), string(want))
	}
}

func TestRedshiftStage0OperationAndTypeLists(t *testing.T) {
	expectedOps := []string{
		"AcceptReservedNodeExchange",
		"AddPartner",
		"AssociateDataShareConsumer",
		"AuthorizeClusterSecurityGroupIngress",
		"AuthorizeDataShare",
		"AuthorizeEndpointAccess",
		"AuthorizeSnapshotAccess",
		"BatchDeleteClusterSnapshots",
		"BatchModifyClusterSnapshots",
		"CancelResize",
		"CopyClusterSnapshot",
		"CreateAuthenticationProfile",
		"CreateCluster",
		"CreateClusterParameterGroup",
		"CreateClusterSecurityGroup",
		"CreateClusterSnapshot",
		"CreateClusterSubnetGroup",
		"CreateCustomDomainAssociation",
		"CreateEndpointAccess",
		"CreateEventSubscription",
		"CreateHsmClientCertificate",
		"CreateHsmConfiguration",
		"CreateIntegration",
		"CreateRedshiftIdcApplication",
		"CreateScheduledAction",
		"CreateSnapshotCopyGrant",
		"CreateSnapshotSchedule",
		"CreateTags",
		"CreateUsageLimit",
		"DeauthorizeDataShare",
		"DeleteAuthenticationProfile",
		"DeleteCluster",
		"DeleteClusterParameterGroup",
		"DeleteClusterSecurityGroup",
		"DeleteClusterSnapshot",
		"DeleteClusterSubnetGroup",
		"DeleteCustomDomainAssociation",
		"DeleteEndpointAccess",
		"DeleteEventSubscription",
		"DeleteHsmClientCertificate",
		"DeleteHsmConfiguration",
		"DeleteIntegration",
		"DeletePartner",
		"DeleteRedshiftIdcApplication",
		"DeleteResourcePolicy",
		"DeleteScheduledAction",
		"DeleteSnapshotCopyGrant",
		"DeleteSnapshotSchedule",
		"DeleteTags",
		"DeleteUsageLimit",
		"DeregisterNamespace",
		"DescribeAccountAttributes",
		"DescribeAuthenticationProfiles",
		"DescribeClusterDbRevisions",
		"DescribeClusterParameterGroups",
		"DescribeClusterParameters",
		"DescribeClusters",
		"DescribeClusterSecurityGroups",
		"DescribeClusterSnapshots",
		"DescribeClusterSubnetGroups",
		"DescribeClusterTracks",
		"DescribeClusterVersions",
		"DescribeCustomDomainAssociations",
		"DescribeDataShares",
		"DescribeDataSharesForConsumer",
		"DescribeDataSharesForProducer",
		"DescribeDefaultClusterParameters",
		"DescribeEndpointAccess",
		"DescribeEndpointAuthorization",
		"DescribeEventCategories",
		"DescribeEvents",
		"DescribeEventSubscriptions",
		"DescribeHsmClientCertificates",
		"DescribeHsmConfigurations",
		"DescribeInboundIntegrations",
		"DescribeIntegrations",
		"DescribeLoggingStatus",
		"DescribeNodeConfigurationOptions",
		"DescribeOrderableClusterOptions",
		"DescribePartners",
		"DescribeRedshiftIdcApplications",
		"DescribeReservedNodeExchangeStatus",
		"DescribeReservedNodeOfferings",
		"DescribeReservedNodes",
		"DescribeResize",
		"DescribeScheduledActions",
		"DescribeSnapshotCopyGrants",
		"DescribeSnapshotSchedules",
		"DescribeStorage",
		"DescribeTableRestoreStatus",
		"DescribeTags",
		"DescribeUsageLimits",
		"DisableLogging",
		"DisableSnapshotCopy",
		"DisassociateDataShareConsumer",
		"EnableLogging",
		"EnableSnapshotCopy",
		"FailoverPrimaryCompute",
		"GetClusterCredentials",
		"GetClusterCredentialsWithIAM",
		"GetIdentityCenterAuthToken",
		"GetReservedNodeExchangeConfigurationOptions",
		"GetReservedNodeExchangeOfferings",
		"GetResourcePolicy",
		"ListRecommendations",
		"ModifyAquaConfiguration",
		"ModifyAuthenticationProfile",
		"ModifyCluster",
		"ModifyClusterDbRevision",
		"ModifyClusterIamRoles",
		"ModifyClusterMaintenance",
		"ModifyClusterParameterGroup",
		"ModifyClusterSnapshot",
		"ModifyClusterSnapshotSchedule",
		"ModifyClusterSubnetGroup",
		"ModifyCustomDomainAssociation",
		"ModifyEndpointAccess",
		"ModifyEventSubscription",
		"ModifyIntegration",
		"ModifyLakehouseConfiguration",
		"ModifyRedshiftIdcApplication",
		"ModifyScheduledAction",
		"ModifySnapshotCopyRetentionPeriod",
		"ModifySnapshotSchedule",
		"ModifyUsageLimit",
		"PauseCluster",
		"PurchaseReservedNodeOffering",
		"PutResourcePolicy",
		"RebootCluster",
		"RegisterNamespace",
		"RejectDataShare",
		"ResetClusterParameterGroup",
		"ResizeCluster",
		"RestoreFromClusterSnapshot",
		"RestoreTableFromClusterSnapshot",
		"ResumeCluster",
		"RevokeClusterSecurityGroupIngress",
		"RevokeEndpointAccess",
		"RevokeSnapshotAccess",
		"RotateEncryptionKey",
		"UpdatePartnerStatus",
	}
	if got := awsmodels.RedshiftOperations(); !reflect.DeepEqual(got, expectedOps) {
		t.Fatalf("redshift operations mismatch\nexpected: %v\nactual:   %v", expectedOps, got)
	}

	expectedTypes := []string{
		"AccountAttribute",
		"AccountWithRestoreAccess",
		"AquaConfiguration",
		"Association",
		"AttributeValueTarget",
		"AuthenticationProfile",
		"AuthorizedTokenIssuer",
		"AvailabilityZone",
		"CertificateAssociation",
		"Cluster",
		"ClusterAssociatedToSchedule",
		"ClusterDbRevision",
		"ClusterIamRole",
		"ClusterNode",
		"ClusterParameterGroup",
		"ClusterParameterGroupStatus",
		"ClusterParameterStatus",
		"ClusterSecurityGroup",
		"ClusterSecurityGroupMembership",
		"ClusterSnapshotCopyStatus",
		"ClusterSubnetGroup",
		"ClusterVersion",
		"Connect",
		"DataShare",
		"DataShareAssociation",
		"DataTransferProgress",
		"DefaultClusterParameters",
		"DeferredMaintenanceWindow",
		"DeleteClusterSnapshotMessage",
		"DescribeIntegrationsFilter",
		"EC2SecurityGroup",
		"ElasticIpStatus",
		"Endpoint",
		"EndpointAccess",
		"EndpointAuthorization",
		"Event",
		"EventCategoriesMap",
		"EventInfoMap",
		"EventSubscription",
		"HsmClientCertificate",
		"HsmConfiguration",
		"HsmStatus",
		"InboundIntegration",
		"Integration",
		"IntegrationError",
		"IPRange",
		"LakeFormationQuery",
		"LakeFormationScopeUnion",
		"MaintenanceTrack",
		"NamespaceIdentifierUnion",
		"NetworkInterface",
		"NodeConfigurationOption",
		"NodeConfigurationOptionsFilter",
		"OrderableClusterOption",
		"Parameter",
		"PartnerIntegrationInfo",
		"PauseClusterMessage",
		"PendingModifiedValues",
		"ProvisionedIdentifier",
		"ReadWriteAccess",
		"Recommendation",
		"RecommendedAction",
		"RecurringCharge",
		"RedshiftIdcApplication",
		"RedshiftScopeUnion",
		"ReferenceLink",
		"ReservedNode",
		"ReservedNodeConfigurationOption",
		"ReservedNodeExchangeStatus",
		"ReservedNodeOffering",
		"ResizeClusterMessage",
		"ResizeInfo",
		"ResourcePolicy",
		"RestoreStatus",
		"ResumeClusterMessage",
		"RevisionTarget",
		"S3AccessGrantsScopeUnion",
		"ScheduledAction",
		"ScheduledActionFilter",
		"ScheduledActionType",
		"SecondaryClusterInfo",
		"ServerlessIdentifier",
		"ServiceIntegrationsUnion",
		"Snapshot",
		"SnapshotCopyGrant",
		"SnapshotErrorMessage",
		"SnapshotSchedule",
		"SnapshotSortingEntity",
		"Subnet",
		"SupportedOperation",
		"SupportedPlatform",
		"TableRestoreStatus",
		"Tag",
		"TaggedResource",
		"UpdateTarget",
		"UsageLimit",
		"VpcEndpoint",
		"VpcSecurityGroupMembership",
	}
	if got := awsmodels.RedshiftTypes(); !reflect.DeepEqual(got, expectedTypes) {
		t.Fatalf("redshift types mismatch\nexpected: %v\nactual:   %v", expectedTypes, got)
	}
}

func TestRedshiftStage0MissingAction(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/redshift", []byte("Version=2012-12-01"), headers, "redshift")
	assertStatus(t, resp, http.StatusBadRequest)
	assertXMLFixtureEqual(t, mustBody(t, resp), "error-missing-action.xml")
}

func TestRedshiftStage0InvalidAction(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/redshift", []byte("Action=Nope&Version=2012-12-01"), headers, "redshift")
	assertStatus(t, resp, http.StatusBadRequest)
	assertXMLFixtureEqual(t, mustBody(t, resp), "error-invalid-action.xml")
}

func TestRedshiftStage0ModeledOperationRoutesToConcreteHandler(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/redshift", []byte("Action=AddPartner&Version=2012-12-01"), headers, "redshift")
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "<Code>InvalidParameterValue</Code>") {
		t.Fatalf("expected InvalidParameterValue, got %s", body)
	}
}

func TestRedshiftStage0RootPathRoutesBySigV4ServiceHint(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte("Action=AddPartner&Version=2012-12-01"),
		headers,
		"redshift",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "<Code>InvalidParameterValue</Code>") {
		t.Fatalf("expected InvalidParameterValue, got %s", body)
	}
}
