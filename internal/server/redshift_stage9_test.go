package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func TestRedshiftStage9MatrixCoverage(t *testing.T) {
	ops := awsmodels.RedshiftOperations()
	types := awsmodels.RedshiftTypes()
	typeSet := make(map[string]struct{}, len(types))
	for _, typ := range types {
		typeSet[typ] = struct{}{}
	}
	matrix := awsmodels.RedshiftTestMatrix()

	seen := make(map[string]awsmodels.RedshiftTestMatrixEntry, len(matrix))
	for _, entry := range matrix {
		if entry.Operation == "" {
			t.Fatalf("matrix entry missing operation")
		}
		if _, ok := seen[entry.Operation]; ok {
			t.Fatalf("duplicate matrix entry for %s", entry.Operation)
		}
		if len(entry.ValidationTests) == 0 || len(entry.IntegrationTests) == 0 {
			t.Fatalf("matrix entry %s missing tests", entry.Operation)
		}
		for _, typ := range entry.Types {
			if _, ok := typeSet[typ]; !ok {
				t.Fatalf("matrix entry %s references unknown type %s", entry.Operation, typ)
			}
		}
		seen[entry.Operation] = entry
	}

	for _, op := range ops {
		if _, ok := seen[op]; !ok {
			t.Fatalf("missing matrix entry for %s", op)
		}
	}
}

func TestRedshiftStage9OperationRegression(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	type opCase struct {
		success func(t *testing.T) (int, []byte)
		failure func(t *testing.T) (int, []byte)
	}

	makeName := func(op string, suffix string) string {
		base := strings.ToLower(op)
		var b strings.Builder
		for _, r := range base {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			} else {
				b.WriteByte('-')
			}
		}
		if suffix != "" {
			return fmt.Sprintf("stage9-%s-%s", b.String(), suffix)
		}
		return fmt.Sprintf("stage9-%s", b.String())
	}

	mustOK := func(t *testing.T, params url.Values) {
		t.Helper()
		status, body := redshiftRequest(t, ts, params)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", status, string(body))
		}
	}

	createCluster := func(t *testing.T, id string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":             []string{"CreateCluster"},
			"ClusterIdentifier":  []string{id},
			"NodeType":           []string{"ra3.xlplus"},
			"MasterUsername":     []string{"admin"},
			"MasterUserPassword": []string{"Secret1234"},
			"DBName":             []string{"dev"},
		})
	}

	createSnapshot := func(t *testing.T, clusterID, snapID string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":             []string{"CreateClusterSnapshot"},
			"ClusterIdentifier":  []string{clusterID},
			"SnapshotIdentifier": []string{snapID},
		})
	}

	createSubnetGroup := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                 []string{"CreateClusterSubnetGroup"},
			"ClusterSubnetGroupName": []string{name},
			"Description":            []string{"demo"},
			"SubnetIds.member.1":     []string{"subnet-1234"},
		})
	}

	createSecurityGroup := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                   []string{"CreateClusterSecurityGroup"},
			"ClusterSecurityGroupName": []string{name},
			"Description":              []string{"demo"},
		})
	}

	createEndpoint := func(t *testing.T, endpointName, clusterID, subnetGroup, secGroup string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                       []string{"CreateEndpointAccess"},
			"EndpointName":                 []string{endpointName},
			"ClusterIdentifier":            []string{clusterID},
			"SubnetGroupName":              []string{subnetGroup},
			"VpcSecurityGroupIds.member.1": []string{secGroup},
		})
	}

	authorizeEndpointAccess := func(t *testing.T, clusterID, account string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":            []string{"AuthorizeEndpointAccess"},
			"ClusterIdentifier": []string{clusterID},
			"Account":           []string{account},
		})
	}

	createParamGroup := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":               []string{"CreateClusterParameterGroup"},
			"ParameterGroupName":   []string{name},
			"ParameterGroupFamily": []string{"redshift-1.0"},
			"Description":          []string{"demo"},
		})
	}

	createScheduledAction := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":              []string{"CreateScheduledAction"},
			"ScheduledActionName": []string{name},
			"TargetAction":        []string{"ResizeCluster"},
			"Schedule":            []string{"cron(0 12 * * ? *)"},
		})
	}

	createIntegration := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":          []string{"CreateIntegration"},
			"IntegrationName": []string{name},
			"SourceArn":       []string{"arn:aws:redshift:us-east-1:123456789012:cluster/source"},
			"TargetArn":       []string{"arn:aws:redshift:us-east-1:123456789012:namespace/target"},
			"Description":     []string{"demo"},
		})
	}

	createHsmCert := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                         []string{"CreateHsmClientCertificate"},
			"HsmClientCertificateIdentifier": []string{name},
		})
	}

	createHsmConfig := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                     []string{"CreateHsmConfiguration"},
			"HsmConfigurationIdentifier": []string{name},
			"Description":                []string{"demo"},
			"HsmIpAddress":               []string{"10.0.0.1"},
			"HsmPartitionName":           []string{"partition-1"},
			"HsmPartitionPassword":       []string{"secret"},
			"HsmServerPublicCertificate": []string{"cert-data"},
		})
	}

	createEventSubscription := func(t *testing.T, name string, clusterID string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":             []string{"CreateEventSubscription"},
			"SubscriptionName":   []string{name},
			"SnsTopicArn":        []string{"arn:aws:sns:us-east-1:123456789012:topic-1"},
			"SourceType":         []string{"cluster"},
			"SourceIds.member.1": []string{clusterID},
			"Severity":           []string{"INFO"},
			"Enabled":            []string{"true"},
		})
	}

	enableLogging := func(t *testing.T, clusterID string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":            []string{"EnableLogging"},
			"ClusterIdentifier": []string{clusterID},
			"BucketName":        []string{"demo-bucket"},
			"S3KeyPrefix":       []string{"logs"},
		})
	}

	putResourcePolicy := func(t *testing.T, arn string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":      []string{"PutResourcePolicy"},
			"ResourceArn": []string{arn},
			"Policy":      []string{`{"Version":"2012-10-17","Statement":[]}`},
		})
	}

	createTags := func(t *testing.T, arn string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":              []string{"CreateTags"},
			"ResourceName":        []string{arn},
			"Tags.member.1.Key":   []string{"env"},
			"Tags.member.1.Value": []string{"test"},
		})
	}

	createSnapshotCopyGrant := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                []string{"CreateSnapshotCopyGrant"},
			"SnapshotCopyGrantName": []string{name},
		})
	}

	createSnapshotSchedule := func(t *testing.T, id string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                       []string{"CreateSnapshotSchedule"},
			"SnapshotScheduleIdentifier":   []string{id},
			"ScheduleDefinitions.member.1": []string{"cron(0 1 * * ? *)"},
		})
	}

	authorizeSnapshotAccess := func(t *testing.T, snapshotID, account string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                   []string{"AuthorizeSnapshotAccess"},
			"SnapshotIdentifier":       []string{snapshotID},
			"AccountWithRestoreAccess": []string{account},
		})
	}

	authorizeDataShare := func(t *testing.T, name string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":             []string{"AuthorizeDataShare"},
			"DataShareName":      []string{name},
			"ConsumerIdentifier": []string{"consumer-1"},
		})
	}

	createUsageLimit := func(t *testing.T, clusterID, usageLimitID string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":            []string{"CreateUsageLimit"},
			"ClusterIdentifier": []string{clusterID},
			"UsageLimitId":      []string{usageLimitID},
			"FeatureType":       []string{"concurrency-scaling"},
			"LimitType":         []string{"time"},
			"Amount":            []string{"60"},
			"Period":            []string{"daily"},
			"BreachAction":      []string{"log"},
		})
	}

	createAuthenticationProfile := func(t *testing.T, profileName string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                       []string{"CreateAuthenticationProfile"},
			"AuthenticationProfileName":    []string{profileName},
			"AuthenticationProfileContent": []string{`{"AllowDBUserOverride":"1"}`},
		})
	}

	createRedshiftIdcApplication := func(t *testing.T, appName string) string {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                     []string{"CreateRedshiftIdcApplication"},
			"RedshiftIdcApplicationName": []string{appName},
			"IdcInstanceArn":             []string{"arn:aws:sso:::instance/ssoins-1234567890abcdef"},
		})
		return redshiftIdcApplicationArn(appName)
	}

	registerNamespace := func(t *testing.T, namespaceID string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":              []string{"RegisterNamespace"},
			"NamespaceIdentifier": []string{namespaceID},
		})
	}

	addPartner := func(t *testing.T, clusterID, partnerName string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":            []string{"AddPartner"},
			"AccountId":         []string{"123456789012"},
			"ClusterIdentifier": []string{clusterID},
			"DatabaseName":      []string{"dev"},
			"PartnerName":       []string{partnerName},
		})
	}

	createCustomDomainAssociation := func(t *testing.T, clusterID, domainName string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                     []string{"CreateCustomDomainAssociation"},
			"ClusterIdentifier":          []string{clusterID},
			"CustomDomainName":           []string{domainName},
			"CustomDomainCertificateArn": []string{"arn:aws:acm:us-east-1:123456789012:certificate/1234abcd-12ab-34cd-56ef-1234567890ab"},
		})
	}

	purchaseReservedNodeOffering := func(t *testing.T, reservedNodeID string) {
		t.Helper()
		mustOK(t, url.Values{
			"Action":                 []string{"PurchaseReservedNodeOffering"},
			"ReservedNodeOfferingId": []string{"offering-1"},
			"ReservedNodeId":         []string{reservedNodeID},
			"NodeCount":              []string{"1"},
		})
	}

	ops := map[string]opCase{
		"CreateCluster": {
			success: func(t *testing.T) (int, []byte) {
				params := url.Values{
					"Action":             []string{"CreateCluster"},
					"ClusterIdentifier":  []string{makeName("CreateCluster", "cluster")},
					"NodeType":           []string{"ra3.xlplus"},
					"MasterUsername":     []string{"admin"},
					"MasterUserPassword": []string{"Secret1234"},
				}
				return redshiftRequest(t, ts, params)
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateCluster"}})
			},
		},
		"ModifyCluster": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("ModifyCluster", "cluster")
				createCluster(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"ModifyCluster"},
					"ClusterIdentifier": []string{id},
					"NodeType":          []string{"ra3.xlplus"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyCluster"}})
			},
		},
		"DeleteCluster": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DeleteCluster", "cluster")
				createCluster(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DeleteCluster"},
					"ClusterIdentifier": []string{id},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteCluster"}})
			},
		},
		"DescribeClusters": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DescribeClusters", "cluster")
				createCluster(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeClusters"},
					"ClusterIdentifier": []string{id},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeClusters"},
					"ClusterIdentifier": []string{makeName("DescribeClusters", "missing")},
				})
			},
		},
		"DescribeAccountAttributes": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeAccountAttributes"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeAccountAttributes"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeClusterDbRevisions": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DescribeClusterDbRevisions", "cluster")
				createCluster(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeClusterDbRevisions"},
					"ClusterIdentifier": []string{id},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeClusterDbRevisions"},
					"ClusterIdentifier": []string{makeName("DescribeClusterDbRevisions", "missing")},
				})
			},
		},
		"DescribeClusterTracks": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeClusterTracks"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeClusterTracks"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeClusterVersions": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeClusterVersions"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeClusterVersions"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeDefaultClusterParameters": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":               []string{"DescribeDefaultClusterParameters"},
					"ParameterGroupFamily": []string{"redshift-1.0"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeDefaultClusterParameters"},
				})
			},
		},
		"DescribeEventCategories": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeEventCategories"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeEventCategories"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeEvents": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeEvents"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":   []string{"DescribeEvents"},
					"Duration": []string{"0"},
				})
			},
		},
		"DescribeNodeConfigurationOptions": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeNodeConfigurationOptions"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeNodeConfigurationOptions"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeOrderableClusterOptions": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeOrderableClusterOptions"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeOrderableClusterOptions"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeReservedNodeOfferings": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeReservedNodeOfferings"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeReservedNodeOfferings"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeReservedNodes": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeReservedNodes"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeReservedNodes"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeReservedNodeExchangeStatus": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeReservedNodeExchangeStatus"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeReservedNodeExchangeStatus"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"GetReservedNodeExchangeConfigurationOptions": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"GetReservedNodeExchangeConfigurationOptions"},
					"ActionType": []string{"restore-cluster"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"GetReservedNodeExchangeConfigurationOptions"},
				})
			},
		},
		"GetReservedNodeExchangeOfferings": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":         []string{"GetReservedNodeExchangeOfferings"},
					"ReservedNodeId": []string{"reserved-node-1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"GetReservedNodeExchangeOfferings"},
				})
			},
		},
		"ListRecommendations": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"ListRecommendations"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"ListRecommendations"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"DescribeStorage": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DescribeStorage", "cluster")
				createCluster(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeStorage"},
					"ClusterIdentifier": []string{id},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeStorage"},
					"ClusterIdentifier": []string{makeName("DescribeStorage", "missing")},
				})
			},
		},
		"CreateClusterSnapshot": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("CreateClusterSnapshot", "cluster")
				createCluster(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"CreateClusterSnapshot"},
					"ClusterIdentifier":  []string{id},
					"SnapshotIdentifier": []string{makeName("CreateClusterSnapshot", "snap")},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateClusterSnapshot"}})
			},
		},
		"DescribeClusterSnapshots": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DescribeClusterSnapshots", "cluster")
				snap := makeName("DescribeClusterSnapshots", "snap")
				createCluster(t, id)
				createSnapshot(t, id, snap)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DescribeClusterSnapshots"},
					"SnapshotIdentifier": []string{snap},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DescribeClusterSnapshots"},
					"SnapshotIdentifier": []string{makeName("DescribeClusterSnapshots", "missing")},
				})
			},
		},
		"DeleteClusterSnapshot": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DeleteClusterSnapshot", "cluster")
				snap := makeName("DeleteClusterSnapshot", "snap")
				createCluster(t, id)
				createSnapshot(t, id, snap)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DeleteClusterSnapshot"},
					"SnapshotIdentifier": []string{snap},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteClusterSnapshot"}})
			},
		},
		"CopyClusterSnapshot": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("CopyClusterSnapshot", "cluster")
				source := makeName("CopyClusterSnapshot", "source")
				createCluster(t, id)
				createSnapshot(t, id, source)
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"CopyClusterSnapshot"},
					"SourceSnapshotIdentifier": []string{source},
					"TargetSnapshotIdentifier": []string{makeName("CopyClusterSnapshot", "target")},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CopyClusterSnapshot"}})
			},
		},
		"BatchDeleteClusterSnapshots": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("BatchDeleteClusterSnapshots", "cluster")
				snapA := makeName("BatchDeleteClusterSnapshots", "snap-a")
				snapB := makeName("BatchDeleteClusterSnapshots", "snap-b")
				createCluster(t, id)
				createSnapshot(t, id, snapA)
				createSnapshot(t, id, snapB)
				return redshiftRequest(t, ts, url.Values{
					"Action":                          []string{"BatchDeleteClusterSnapshots"},
					"SnapshotIdentifierList.member.1": []string{snapA},
					"SnapshotIdentifierList.member.2": []string{snapB},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"BatchDeleteClusterSnapshots"}})
			},
		},
		"BatchModifyClusterSnapshots": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("BatchModifyClusterSnapshots", "cluster")
				snapA := makeName("BatchModifyClusterSnapshots", "snap-a")
				createCluster(t, id)
				createSnapshot(t, id, snapA)
				return redshiftRequest(t, ts, url.Values{
					"Action":                          []string{"BatchModifyClusterSnapshots"},
					"SnapshotIdentifierList.member.1": []string{snapA},
					"ManualSnapshotRetentionPeriod":   []string{"7"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"BatchModifyClusterSnapshots"}})
			},
		},
		"ModifyClusterSnapshot": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("ModifyClusterSnapshot", "cluster")
				snap := makeName("ModifyClusterSnapshot", "snap")
				createCluster(t, id)
				createSnapshot(t, id, snap)
				return redshiftRequest(t, ts, url.Values{
					"Action":                        []string{"ModifyClusterSnapshot"},
					"SnapshotIdentifier":            []string{snap},
					"ManualSnapshotRetentionPeriod": []string{"5"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyClusterSnapshot"}})
			},
		},
		"CreateSnapshotCopyGrant": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                []string{"CreateSnapshotCopyGrant"},
					"SnapshotCopyGrantName": []string{makeName("CreateSnapshotCopyGrant", "grant")},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateSnapshotCopyGrant"}})
			},
		},
		"DeleteSnapshotCopyGrant": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteSnapshotCopyGrant", "grant")
				createSnapshotCopyGrant(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                []string{"DeleteSnapshotCopyGrant"},
					"SnapshotCopyGrantName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteSnapshotCopyGrant"}})
			},
		},
		"DescribeSnapshotCopyGrants": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeSnapshotCopyGrants", "grant")
				createSnapshotCopyGrant(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                []string{"DescribeSnapshotCopyGrants"},
					"SnapshotCopyGrantName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeSnapshotCopyGrants"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"EnableSnapshotCopy": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("EnableSnapshotCopy", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"EnableSnapshotCopy"},
					"ClusterIdentifier": []string{clusterID},
					"DestinationRegion": []string{"us-west-2"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"EnableSnapshotCopy"}})
			},
		},
		"DisableSnapshotCopy": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DisableSnapshotCopy", "cluster")
				createCluster(t, clusterID)
				mustOK(t, url.Values{
					"Action":            []string{"EnableSnapshotCopy"},
					"ClusterIdentifier": []string{clusterID},
					"DestinationRegion": []string{"us-west-2"},
				})
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DisableSnapshotCopy"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DisableSnapshotCopy"}})
			},
		},
		"ModifySnapshotCopyRetentionPeriod": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifySnapshotCopyRetentionPeriod", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"ModifySnapshotCopyRetentionPeriod"},
					"ClusterIdentifier": []string{clusterID},
					"RetentionPeriod":   []string{"7"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifySnapshotCopyRetentionPeriod"}})
			},
		},
		"CreateSnapshotSchedule": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                       []string{"CreateSnapshotSchedule"},
					"SnapshotScheduleIdentifier":   []string{makeName("CreateSnapshotSchedule", "schedule")},
					"ScheduleDefinitions.member.1": []string{"cron(0 1 * * ? *)"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateSnapshotSchedule"}})
			},
		},
		"DeleteSnapshotSchedule": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DeleteSnapshotSchedule", "schedule")
				createSnapshotSchedule(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"DeleteSnapshotSchedule"},
					"SnapshotScheduleIdentifier": []string{id},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteSnapshotSchedule"}})
			},
		},
		"DescribeSnapshotSchedules": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("DescribeSnapshotSchedules", "schedule")
				createSnapshotSchedule(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"DescribeSnapshotSchedules"},
					"SnapshotScheduleIdentifier": []string{id},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeSnapshotSchedules"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"ModifySnapshotSchedule": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("ModifySnapshotSchedule", "schedule")
				createSnapshotSchedule(t, id)
				return redshiftRequest(t, ts, url.Values{
					"Action":                       []string{"ModifySnapshotSchedule"},
					"SnapshotScheduleIdentifier":   []string{id},
					"ScheduleDefinitions.member.1": []string{"cron(0 2 * * ? *)"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifySnapshotSchedule"}})
			},
		},
		"ModifyClusterSnapshotSchedule": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyClusterSnapshotSchedule", "cluster")
				scheduleID := makeName("ModifyClusterSnapshotSchedule", "schedule")
				createCluster(t, clusterID)
				createSnapshotSchedule(t, scheduleID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"ModifyClusterSnapshotSchedule"},
					"ClusterIdentifier": []string{clusterID},
					"SnapshotScheduleIdentifierList.member.1": []string{scheduleID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyClusterSnapshotSchedule"}})
			},
		},
		"AuthorizeSnapshotAccess": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("AuthorizeSnapshotAccess", "cluster")
				snap := makeName("AuthorizeSnapshotAccess", "snap")
				createCluster(t, id)
				createSnapshot(t, id, snap)
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"AuthorizeSnapshotAccess"},
					"SnapshotIdentifier":       []string{snap},
					"AccountWithRestoreAccess": []string{"123456789012"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"AuthorizeSnapshotAccess"}})
			},
		},
		"RevokeSnapshotAccess": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("RevokeSnapshotAccess", "cluster")
				snap := makeName("RevokeSnapshotAccess", "snap")
				createCluster(t, id)
				createSnapshot(t, id, snap)
				authorizeSnapshotAccess(t, snap, "123456789012")
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"RevokeSnapshotAccess"},
					"SnapshotIdentifier":       []string{snap},
					"AccountWithRestoreAccess": []string{"123456789012"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RevokeSnapshotAccess"}})
			},
		},
		"RestoreFromClusterSnapshot": {
			success: func(t *testing.T) (int, []byte) {
				id := makeName("RestoreFromClusterSnapshot", "cluster")
				snap := makeName("RestoreFromClusterSnapshot", "snap")
				createCluster(t, id)
				createSnapshot(t, id, snap)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"RestoreFromClusterSnapshot"},
					"ClusterIdentifier":  []string{makeName("RestoreFromClusterSnapshot", "restored")},
					"SnapshotIdentifier": []string{snap},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RestoreFromClusterSnapshot"}})
			},
		},
		"CreateClusterSubnetGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("CreateClusterSubnetGroup", "subnet")
				return redshiftRequest(t, ts, url.Values{
					"Action":                 []string{"CreateClusterSubnetGroup"},
					"ClusterSubnetGroupName": []string{name},
					"Description":            []string{"demo"},
					"SubnetIds.member.1":     []string{"subnet-1234"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateClusterSubnetGroup"}})
			},
		},
		"ModifyClusterSubnetGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("ModifyClusterSubnetGroup", "subnet")
				createSubnetGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                 []string{"ModifyClusterSubnetGroup"},
					"ClusterSubnetGroupName": []string{name},
					"SubnetIds.member.1":     []string{"subnet-5678"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyClusterSubnetGroup"}})
			},
		},
		"DescribeClusterSubnetGroups": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeClusterSubnetGroups", "subnet")
				createSubnetGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                 []string{"DescribeClusterSubnetGroups"},
					"ClusterSubnetGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                 []string{"DescribeClusterSubnetGroups"},
					"ClusterSubnetGroupName": []string{makeName("DescribeClusterSubnetGroups", "missing")},
				})
			},
		},
		"DeleteClusterSubnetGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteClusterSubnetGroup", "subnet")
				createSubnetGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                 []string{"DeleteClusterSubnetGroup"},
					"ClusterSubnetGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteClusterSubnetGroup"}})
			},
		},
		"CreateClusterSecurityGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("CreateClusterSecurityGroup", "sec")
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"CreateClusterSecurityGroup"},
					"ClusterSecurityGroupName": []string{name},
					"Description":              []string{"demo"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateClusterSecurityGroup"}})
			},
		},
		"DescribeClusterSecurityGroups": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeClusterSecurityGroups", "sec")
				createSecurityGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"DescribeClusterSecurityGroups"},
					"ClusterSecurityGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"DescribeClusterSecurityGroups"},
					"ClusterSecurityGroupName": []string{makeName("DescribeClusterSecurityGroups", "missing")},
				})
			},
		},
		"DeleteClusterSecurityGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteClusterSecurityGroup", "sec")
				createSecurityGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"DeleteClusterSecurityGroup"},
					"ClusterSecurityGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteClusterSecurityGroup"}})
			},
		},
		"AuthorizeClusterSecurityGroupIngress": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("AuthorizeClusterSecurityGroupIngress", "sec")
				createSecurityGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"AuthorizeClusterSecurityGroupIngress"},
					"ClusterSecurityGroupName": []string{name},
					"CIDRIP":                   []string{"10.0.0.0/24"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"AuthorizeClusterSecurityGroupIngress"}})
			},
		},
		"RevokeClusterSecurityGroupIngress": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("RevokeClusterSecurityGroupIngress", "sec")
				createSecurityGroup(t, name)
				mustOK(t, url.Values{
					"Action":                   []string{"AuthorizeClusterSecurityGroupIngress"},
					"ClusterSecurityGroupName": []string{name},
					"CIDRIP":                   []string{"10.0.0.0/24"},
				})
				return redshiftRequest(t, ts, url.Values{
					"Action":                   []string{"RevokeClusterSecurityGroupIngress"},
					"ClusterSecurityGroupName": []string{name},
					"CIDRIP":                   []string{"10.0.0.0/24"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RevokeClusterSecurityGroupIngress"}})
			},
		},
		"CreateEndpointAccess": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("CreateEndpointAccess", "cluster")
				subnetName := makeName("CreateEndpointAccess", "subnet")
				secName := makeName("CreateEndpointAccess", "sec")
				createSubnetGroup(t, subnetName)
				createSecurityGroup(t, secName)
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":                       []string{"CreateEndpointAccess"},
					"EndpointName":                 []string{makeName("CreateEndpointAccess", "ep")},
					"ClusterIdentifier":            []string{clusterID},
					"SubnetGroupName":              []string{subnetName},
					"VpcSecurityGroupIds.member.1": []string{secName},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateEndpointAccess"}})
			},
		},
		"DescribeEndpointAccess": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeEndpointAccess", "cluster")
				subnetName := makeName("DescribeEndpointAccess", "subnet")
				secName := makeName("DescribeEndpointAccess", "sec")
				endpoint := makeName("DescribeEndpointAccess", "ep")
				createSubnetGroup(t, subnetName)
				createSecurityGroup(t, secName)
				createCluster(t, clusterID)
				createEndpoint(t, endpoint, clusterID, subnetName, secName)
				return redshiftRequest(t, ts, url.Values{
					"Action":       []string{"DescribeEndpointAccess"},
					"EndpointName": []string{endpoint},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":       []string{"DescribeEndpointAccess"},
					"EndpointName": []string{makeName("DescribeEndpointAccess", "missing")},
				})
			},
		},
		"DeleteEndpointAccess": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DeleteEndpointAccess", "cluster")
				subnetName := makeName("DeleteEndpointAccess", "subnet")
				secName := makeName("DeleteEndpointAccess", "sec")
				endpoint := makeName("DeleteEndpointAccess", "ep")
				createSubnetGroup(t, subnetName)
				createSecurityGroup(t, secName)
				createCluster(t, clusterID)
				createEndpoint(t, endpoint, clusterID, subnetName, secName)
				return redshiftRequest(t, ts, url.Values{
					"Action":       []string{"DeleteEndpointAccess"},
					"EndpointName": []string{endpoint},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteEndpointAccess"}})
			},
		},
		"AuthorizeEndpointAccess": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("AuthorizeEndpointAccess", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"AuthorizeEndpointAccess"},
					"ClusterIdentifier": []string{clusterID},
					"Account":           []string{"123456789012"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"AuthorizeEndpointAccess"}})
			},
		},
		"RevokeEndpointAccess": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("RevokeEndpointAccess", "cluster")
				createCluster(t, clusterID)
				authorizeEndpointAccess(t, clusterID, "123456789012")
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"RevokeEndpointAccess"},
					"ClusterIdentifier": []string{clusterID},
					"Account":           []string{"123456789012"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RevokeEndpointAccess"}})
			},
		},
		"DescribeEndpointAuthorization": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeEndpointAuthorization", "cluster")
				createCluster(t, clusterID)
				authorizeEndpointAccess(t, clusterID, "123456789012")
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeEndpointAuthorization"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeEndpointAuthorization"},
					"ClusterIdentifier": []string{makeName("DescribeEndpointAuthorization", "missing")},
				})
			},
		},
		"ModifyEndpointAccess": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyEndpointAccess", "cluster")
				subnetName := makeName("ModifyEndpointAccess", "subnet")
				secA := makeName("ModifyEndpointAccess", "sec-a")
				secB := makeName("ModifyEndpointAccess", "sec-b")
				endpoint := makeName("ModifyEndpointAccess", "ep")
				createSubnetGroup(t, subnetName)
				createSecurityGroup(t, secA)
				createSecurityGroup(t, secB)
				createCluster(t, clusterID)
				createEndpoint(t, endpoint, clusterID, subnetName, secA)
				return redshiftRequest(t, ts, url.Values{
					"Action":                       []string{"ModifyEndpointAccess"},
					"EndpointName":                 []string{endpoint},
					"VpcSecurityGroupIds.member.1": []string{secB},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyEndpointAccess"}})
			},
		},
		"ModifyClusterIamRoles": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyClusterIamRoles", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":               []string{"ModifyClusterIamRoles"},
					"ClusterIdentifier":    []string{clusterID},
					"AddIamRoles.member.1": []string{"arn:aws:iam::123456789012:role/demo"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyClusterIamRoles"}})
			},
		},
		"CreateClusterParameterGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("CreateClusterParameterGroup", "pg")
				return redshiftRequest(t, ts, url.Values{
					"Action":               []string{"CreateClusterParameterGroup"},
					"ParameterGroupName":   []string{name},
					"ParameterGroupFamily": []string{"redshift-1.0"},
					"Description":          []string{"demo"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateClusterParameterGroup"}})
			},
		},
		"DescribeClusterParameterGroups": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeClusterParameterGroups", "pg")
				createParamGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DescribeClusterParameterGroups"},
					"ParameterGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DescribeClusterParameterGroups"},
					"ParameterGroupName": []string{makeName("DescribeClusterParameterGroups", "missing")},
				})
			},
		},
		"DescribeClusterParameters": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeClusterParameters", "pg")
				createParamGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DescribeClusterParameters"},
					"ParameterGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DescribeClusterParameters"}})
			},
		},
		"ModifyClusterParameterGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("ModifyClusterParameterGroup", "pg")
				createParamGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                             []string{"ModifyClusterParameterGroup"},
					"ParameterGroupName":                 []string{name},
					"Parameters.member.1.ParameterName":  []string{"max_concurrency_scaling_clusters"},
					"Parameters.member.1.ParameterValue": []string{"1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyClusterParameterGroup"}})
			},
		},
		"ResetClusterParameterGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("ResetClusterParameterGroup", "pg")
				createParamGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"ResetClusterParameterGroup"},
					"ParameterGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ResetClusterParameterGroup"}})
			},
		},
		"DeleteClusterParameterGroup": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteClusterParameterGroup", "pg")
				createParamGroup(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DeleteClusterParameterGroup"},
					"ParameterGroupName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteClusterParameterGroup"}})
			},
		},
		"ModifyClusterMaintenance": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyClusterMaintenance", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"ModifyClusterMaintenance"},
					"ClusterIdentifier":          []string{clusterID},
					"PreferredMaintenanceWindow": []string{"sun:05:00-sun:06:00"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyClusterMaintenance"}})
			},
		},
		"ModifyClusterDbRevision": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyClusterDbRevision", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"ModifyClusterDbRevision"},
					"ClusterIdentifier": []string{clusterID},
					"RevisionTarget":    []string{"1.1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyClusterDbRevision"}})
			},
		},
		"RebootCluster": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("RebootCluster", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"RebootCluster"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RebootCluster"}})
			},
		},
		"PauseCluster": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("PauseCluster", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"PauseCluster"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"PauseCluster"}})
			},
		},
		"ResumeCluster": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ResumeCluster", "cluster")
				createCluster(t, clusterID)
				mustOK(t, url.Values{
					"Action":            []string{"PauseCluster"},
					"ClusterIdentifier": []string{clusterID},
				})
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"ResumeCluster"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ResumeCluster"}})
			},
		},
		"FailoverPrimaryCompute": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("FailoverPrimaryCompute", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"FailoverPrimaryCompute"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"FailoverPrimaryCompute"}})
			},
		},
		"RotateEncryptionKey": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("RotateEncryptionKey", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"RotateEncryptionKey"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RotateEncryptionKey"}})
			},
		},
		"RestoreTableFromClusterSnapshot": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("RestoreTableFromClusterSnapshot", "cluster")
				snapID := makeName("RestoreTableFromClusterSnapshot", "snap")
				createCluster(t, clusterID)
				createSnapshot(t, clusterID, snapID)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"RestoreTableFromClusterSnapshot"},
					"ClusterIdentifier":  []string{clusterID},
					"SnapshotIdentifier": []string{snapID},
					"SourceTableName":    []string{"orders"},
					"NewTableName":       []string{"orders_restored"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RestoreTableFromClusterSnapshot"}})
			},
		},
		"DescribeTableRestoreStatus": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeTableRestoreStatus", "cluster")
				snapID := makeName("DescribeTableRestoreStatus", "snap")
				createCluster(t, clusterID)
				createSnapshot(t, clusterID, snapID)
				mustOK(t, url.Values{
					"Action":             []string{"RestoreTableFromClusterSnapshot"},
					"ClusterIdentifier":  []string{clusterID},
					"SnapshotIdentifier": []string{snapID},
					"SourceTableName":    []string{"orders"},
					"NewTableName":       []string{"orders_restored"},
				})
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeTableRestoreStatus"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeTableRestoreStatus"},
					"ClusterIdentifier": []string{makeName("DescribeTableRestoreStatus", "missing")},
				})
			},
		},
		"ResizeCluster": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ResizeCluster", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"ResizeCluster"},
					"ClusterIdentifier": []string{clusterID},
					"NodeType":          []string{"ra3.xlplus"},
					"NumberOfNodes":     []string{"2"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ResizeCluster"}})
			},
		},
		"DescribeResize": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeResize", "cluster")
				createCluster(t, clusterID)
				mustOK(t, url.Values{
					"Action":            []string{"ResizeCluster"},
					"ClusterIdentifier": []string{clusterID},
					"NodeType":          []string{"ra3.xlplus"},
					"NumberOfNodes":     []string{"2"},
				})
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeResize"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DescribeResize"}})
			},
		},
		"CancelResize": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("CancelResize", "cluster")
				createCluster(t, clusterID)
				mustOK(t, url.Values{
					"Action":            []string{"ResizeCluster"},
					"ClusterIdentifier": []string{clusterID},
					"NodeType":          []string{"ra3.xlplus"},
					"NumberOfNodes":     []string{"2"},
				})
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"CancelResize"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CancelResize"}})
			},
		},
		"CreateScheduledAction": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("CreateScheduledAction", "sa")
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"CreateScheduledAction"},
					"ScheduledActionName": []string{name},
					"TargetAction":        []string{"ResizeCluster"},
					"Schedule":            []string{"cron(0 12 * * ? *)"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateScheduledAction"}})
			},
		},
		"DescribeScheduledActions": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeScheduledActions", "sa")
				createScheduledAction(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"DescribeScheduledActions"},
					"ScheduledActionName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"DescribeScheduledActions"},
					"ScheduledActionName": []string{makeName("DescribeScheduledActions", "missing")},
				})
			},
		},
		"ModifyScheduledAction": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("ModifyScheduledAction", "sa")
				createScheduledAction(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"ModifyScheduledAction"},
					"ScheduledActionName": []string{name},
					"Schedule":            []string{"cron(5 12 * * ? *)"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyScheduledAction"}})
			},
		},
		"DeleteScheduledAction": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteScheduledAction", "sa")
				createScheduledAction(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"DeleteScheduledAction"},
					"ScheduledActionName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteScheduledAction"}})
			},
		},
		"CreateIntegration": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":          []string{"CreateIntegration"},
					"IntegrationName": []string{makeName("CreateIntegration", "int")},
					"SourceArn":       []string{"arn:aws:redshift:us-east-1:123456789012:cluster/source"},
					"TargetArn":       []string{"arn:aws:redshift:us-east-1:123456789012:namespace/target"},
					"Description":     []string{"demo"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateIntegration"}})
			},
		},
		"DescribeIntegrations": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeIntegrations", "int")
				createIntegration(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":          []string{"DescribeIntegrations"},
					"IntegrationName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":          []string{"DescribeIntegrations"},
					"IntegrationName": []string{makeName("DescribeIntegrations", "missing")},
				})
			},
		},
		"DescribeInboundIntegrations": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeInboundIntegrations", "int")
				createIntegration(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeInboundIntegrations"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":          []string{"DescribeInboundIntegrations"},
					"IntegrationName": []string{makeName("DescribeInboundIntegrations", "missing")},
				})
			},
		},
		"ModifyIntegration": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("ModifyIntegration", "int")
				createIntegration(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":          []string{"ModifyIntegration"},
					"IntegrationName": []string{name},
					"Description":     []string{"updated"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyIntegration"}})
			},
		},
		"DeleteIntegration": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteIntegration", "int")
				createIntegration(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":          []string{"DeleteIntegration"},
					"IntegrationName": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteIntegration"}})
			},
		},
		"AuthorizeDataShare": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"AuthorizeDataShare"},
					"DataShareName":      []string{makeName("AuthorizeDataShare", "share")},
					"ConsumerIdentifier": []string{"consumer-1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"AuthorizeDataShare"}})
			},
		},
		"AssociateDataShareConsumer": {
			success: func(t *testing.T) (int, []byte) {
				share := makeName("AssociateDataShareConsumer", "share")
				authorizeDataShare(t, share)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"AssociateDataShareConsumer"},
					"DataShareName":      []string{share},
					"ConsumerIdentifier": []string{"consumer-1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"AssociateDataShareConsumer"}})
			},
		},
		"DisassociateDataShareConsumer": {
			success: func(t *testing.T) (int, []byte) {
				share := makeName("DisassociateDataShareConsumer", "share")
				authorizeDataShare(t, share)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DisassociateDataShareConsumer"},
					"DataShareName":      []string{share},
					"ConsumerIdentifier": []string{"consumer-1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DisassociateDataShareConsumer"}})
			},
		},
		"DescribeDataSharesForConsumer": {
			success: func(t *testing.T) (int, []byte) {
				share := makeName("DescribeDataSharesForConsumer", "share")
				authorizeDataShare(t, share)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DescribeDataSharesForConsumer"},
					"ConsumerIdentifier": []string{"consumer-1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DescribeDataSharesForConsumer"}})
			},
		},
		"DescribeDataSharesForProducer": {
			success: func(t *testing.T) (int, []byte) {
				share := makeName("DescribeDataSharesForProducer", "share")
				authorizeDataShare(t, share)
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeDataSharesForProducer"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":      []string{"DescribeDataSharesForProducer"},
					"ProducerArn": []string{"arn:aws:redshift:us-east-1:123456789012:cluster/missing"},
				})
			},
		},
		"DescribeDataShares": {
			success: func(t *testing.T) (int, []byte) {
				share := makeName("DescribeDataShares", "share")
				authorizeDataShare(t, share)
				return redshiftRequest(t, ts, url.Values{
					"Action":        []string{"DescribeDataShares"},
					"DataShareName": []string{share},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":        []string{"DescribeDataShares"},
					"DataShareName": []string{makeName("DescribeDataShares", "missing")},
				})
			},
		},
		"DeauthorizeDataShare": {
			success: func(t *testing.T) (int, []byte) {
				share := makeName("DeauthorizeDataShare", "share")
				authorizeDataShare(t, share)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DeauthorizeDataShare"},
					"DataShareName":      []string{share},
					"ConsumerIdentifier": []string{"consumer-1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeauthorizeDataShare"}})
			},
		},
		"RejectDataShare": {
			success: func(t *testing.T) (int, []byte) {
				share := makeName("RejectDataShare", "share")
				authorizeDataShare(t, share)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"RejectDataShare"},
					"DataShareName":      []string{share},
					"ConsumerIdentifier": []string{"consumer-2"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RejectDataShare"}})
			},
		},
		"CreateHsmClientCertificate": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                         []string{"CreateHsmClientCertificate"},
					"HsmClientCertificateIdentifier": []string{makeName("CreateHsmClientCertificate", "cert")},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateHsmClientCertificate"}})
			},
		},
		"DescribeHsmClientCertificates": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeHsmClientCertificates", "cert")
				createHsmCert(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeHsmClientCertificates"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                         []string{"DescribeHsmClientCertificates"},
					"HsmClientCertificateIdentifier": []string{makeName("DescribeHsmClientCertificates", "missing")},
				})
			},
		},
		"DeleteHsmClientCertificate": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteHsmClientCertificate", "cert")
				createHsmCert(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                         []string{"DeleteHsmClientCertificate"},
					"HsmClientCertificateIdentifier": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteHsmClientCertificate"}})
			},
		},
		"CreateHsmConfiguration": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"CreateHsmConfiguration"},
					"HsmConfigurationIdentifier": []string{makeName("CreateHsmConfiguration", "config")},
					"Description":                []string{"demo"},
					"HsmIpAddress":               []string{"10.0.0.1"},
					"HsmPartitionName":           []string{"partition-1"},
					"HsmPartitionPassword":       []string{"secret"},
					"HsmServerPublicCertificate": []string{"cert-data"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateHsmConfiguration"}})
			},
		},
		"DescribeHsmConfigurations": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DescribeHsmConfigurations", "config")
				createHsmConfig(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeHsmConfigurations"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"DescribeHsmConfigurations"},
					"HsmConfigurationIdentifier": []string{makeName("DescribeHsmConfigurations", "missing")},
				})
			},
		},
		"DeleteHsmConfiguration": {
			success: func(t *testing.T) (int, []byte) {
				name := makeName("DeleteHsmConfiguration", "config")
				createHsmConfig(t, name)
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"DeleteHsmConfiguration"},
					"HsmConfigurationIdentifier": []string{name},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteHsmConfiguration"}})
			},
		},
		"EnableLogging": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("EnableLogging", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"EnableLogging"},
					"ClusterIdentifier": []string{clusterID},
					"BucketName":        []string{"demo-bucket"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"EnableLogging"}})
			},
		},
		"DescribeLoggingStatus": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeLoggingStatus", "cluster")
				createCluster(t, clusterID)
				enableLogging(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeLoggingStatus"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DescribeLoggingStatus"}})
			},
		},
		"DisableLogging": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DisableLogging", "cluster")
				createCluster(t, clusterID)
				enableLogging(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DisableLogging"},
					"ClusterIdentifier": []string{clusterID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DisableLogging"}})
			},
		},
		"CreateEventSubscription": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("CreateEventSubscription", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"CreateEventSubscription"},
					"SubscriptionName":   []string{makeName("CreateEventSubscription", "sub")},
					"SnsTopicArn":        []string{"arn:aws:sns:us-east-1:123456789012:topic-1"},
					"SourceType":         []string{"cluster"},
					"SourceIds.member.1": []string{clusterID},
					"Severity":           []string{"INFO"},
					"Enabled":            []string{"true"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateEventSubscription"}})
			},
		},
		"DescribeEventSubscriptions": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeEventSubscriptions", "cluster")
				sub := makeName("DescribeEventSubscriptions", "sub")
				createCluster(t, clusterID)
				createEventSubscription(t, sub, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":           []string{"DescribeEventSubscriptions"},
					"SubscriptionName": []string{sub},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":           []string{"DescribeEventSubscriptions"},
					"SubscriptionName": []string{makeName("DescribeEventSubscriptions", "missing")},
				})
			},
		},
		"ModifyEventSubscription": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyEventSubscription", "cluster")
				sub := makeName("ModifyEventSubscription", "sub")
				createCluster(t, clusterID)
				createEventSubscription(t, sub, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":           []string{"ModifyEventSubscription"},
					"SubscriptionName": []string{sub},
					"Enabled":          []string{"false"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyEventSubscription"}})
			},
		},
		"DeleteEventSubscription": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DeleteEventSubscription", "cluster")
				sub := makeName("DeleteEventSubscription", "sub")
				createCluster(t, clusterID)
				createEventSubscription(t, sub, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":           []string{"DeleteEventSubscription"},
					"SubscriptionName": []string{sub},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteEventSubscription"}})
			},
		},
		"GetClusterCredentials": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("GetClusterCredentials", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"GetClusterCredentials"},
					"ClusterIdentifier": []string{clusterID},
					"DbUser":            []string{"devuser"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"GetClusterCredentials"}})
			},
		},
		"GetClusterCredentialsWithIAM": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("GetClusterCredentialsWithIAM", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"GetClusterCredentialsWithIAM"},
					"ClusterIdentifier": []string{clusterID},
					"DbUser":            []string{"devuser"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"GetClusterCredentialsWithIAM"}})
			},
		},
		"AddPartner": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("AddPartner", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"AddPartner"},
					"AccountId":         []string{"123456789012"},
					"ClusterIdentifier": []string{clusterID},
					"DatabaseName":      []string{"dev"},
					"PartnerName":       []string{makeName("AddPartner", "partner")},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"AddPartner"}})
			},
		},
		"DescribePartners": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribePartners", "cluster")
				partnerName := makeName("DescribePartners", "partner")
				createCluster(t, clusterID)
				addPartner(t, clusterID, partnerName)
				return redshiftRequest(t, ts, url.Values{
					"Action":      []string{"DescribePartners"},
					"PartnerName": []string{partnerName},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribePartners"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"UpdatePartnerStatus": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("UpdatePartnerStatus", "cluster")
				partnerName := makeName("UpdatePartnerStatus", "partner")
				createCluster(t, clusterID)
				addPartner(t, clusterID, partnerName)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"UpdatePartnerStatus"},
					"AccountId":         []string{"123456789012"},
					"ClusterIdentifier": []string{clusterID},
					"DatabaseName":      []string{"dev"},
					"PartnerName":       []string{partnerName},
					"Status":            []string{"Active"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"UpdatePartnerStatus"}})
			},
		},
		"DeletePartner": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DeletePartner", "cluster")
				partnerName := makeName("DeletePartner", "partner")
				createCluster(t, clusterID)
				addPartner(t, clusterID, partnerName)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DeletePartner"},
					"AccountId":         []string{"123456789012"},
					"ClusterIdentifier": []string{clusterID},
					"DatabaseName":      []string{"dev"},
					"PartnerName":       []string{partnerName},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeletePartner"}})
			},
		},
		"CreateCustomDomainAssociation": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("CreateCustomDomainAssociation", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"CreateCustomDomainAssociation"},
					"ClusterIdentifier":          []string{clusterID},
					"CustomDomainName":           []string{makeName("CreateCustomDomainAssociation", "domain.example.com")},
					"CustomDomainCertificateArn": []string{"arn:aws:acm:us-east-1:123456789012:certificate/1234abcd-12ab-34cd-56ef-1234567890ab"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateCustomDomainAssociation"}})
			},
		},
		"DescribeCustomDomainAssociations": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeCustomDomainAssociations", "cluster")
				domainName := makeName("DescribeCustomDomainAssociations", "domain.example.com")
				createCluster(t, clusterID)
				createCustomDomainAssociation(t, clusterID, domainName)
				return redshiftRequest(t, ts, url.Values{
					"Action":           []string{"DescribeCustomDomainAssociations"},
					"CustomDomainName": []string{domainName},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":     []string{"DescribeCustomDomainAssociations"},
					"MaxRecords": []string{"0"},
				})
			},
		},
		"ModifyCustomDomainAssociation": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyCustomDomainAssociation", "cluster")
				domainName := makeName("ModifyCustomDomainAssociation", "domain.example.com")
				createCluster(t, clusterID)
				createCustomDomainAssociation(t, clusterID, domainName)
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"ModifyCustomDomainAssociation"},
					"CustomDomainName":           []string{domainName},
					"CustomDomainCertificateArn": []string{"arn:aws:acm:us-east-1:123456789012:certificate/ffffffff-12ab-34cd-56ef-1234567890ab"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyCustomDomainAssociation"}})
			},
		},
		"DeleteCustomDomainAssociation": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DeleteCustomDomainAssociation", "cluster")
				domainName := makeName("DeleteCustomDomainAssociation", "domain.example.com")
				createCluster(t, clusterID)
				createCustomDomainAssociation(t, clusterID, domainName)
				return redshiftRequest(t, ts, url.Values{
					"Action":           []string{"DeleteCustomDomainAssociation"},
					"CustomDomainName": []string{domainName},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteCustomDomainAssociation"}})
			},
		},
		"PurchaseReservedNodeOffering": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                 []string{"PurchaseReservedNodeOffering"},
					"ReservedNodeOfferingId": []string{"offering-1"},
					"ReservedNodeId":         []string{makeName("PurchaseReservedNodeOffering", "reserved-node")},
					"NodeCount":              []string{"1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"PurchaseReservedNodeOffering"}})
			},
		},
		"AcceptReservedNodeExchange": {
			success: func(t *testing.T) (int, []byte) {
				reservedNodeID := makeName("AcceptReservedNodeExchange", "reserved-node")
				purchaseReservedNodeOffering(t, reservedNodeID)
				return redshiftRequest(t, ts, url.Values{
					"Action":                       []string{"AcceptReservedNodeExchange"},
					"ReservedNodeId":               []string{reservedNodeID},
					"TargetReservedNodeOfferingId": []string{"exchange-offering-1"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"AcceptReservedNodeExchange"}})
			},
		},
		"CreateUsageLimit": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("CreateUsageLimit", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"CreateUsageLimit"},
					"ClusterIdentifier": []string{clusterID},
					"UsageLimitId":      []string{makeName("CreateUsageLimit", "limit")},
					"FeatureType":       []string{"concurrency-scaling"},
					"LimitType":         []string{"time"},
					"Amount":            []string{"60"},
					"Period":            []string{"daily"},
					"BreachAction":      []string{"log"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateUsageLimit"}})
			},
		},
		"DescribeUsageLimits": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DescribeUsageLimits", "cluster")
				limitID := makeName("DescribeUsageLimits", "limit")
				createCluster(t, clusterID)
				createUsageLimit(t, clusterID, limitID)
				return redshiftRequest(t, ts, url.Values{
					"Action":       []string{"DescribeUsageLimits"},
					"UsageLimitId": []string{limitID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":            []string{"DescribeUsageLimits"},
					"ClusterIdentifier": []string{makeName("DescribeUsageLimits", "missing")},
				})
			},
		},
		"ModifyUsageLimit": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyUsageLimit", "cluster")
				limitID := makeName("ModifyUsageLimit", "limit")
				createCluster(t, clusterID)
				createUsageLimit(t, clusterID, limitID)
				return redshiftRequest(t, ts, url.Values{
					"Action":       []string{"ModifyUsageLimit"},
					"UsageLimitId": []string{limitID},
					"Amount":       []string{"120"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyUsageLimit"}})
			},
		},
		"DeleteUsageLimit": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("DeleteUsageLimit", "cluster")
				limitID := makeName("DeleteUsageLimit", "limit")
				createCluster(t, clusterID)
				createUsageLimit(t, clusterID, limitID)
				return redshiftRequest(t, ts, url.Values{
					"Action":       []string{"DeleteUsageLimit"},
					"UsageLimitId": []string{limitID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteUsageLimit"}})
			},
		},
		"CreateAuthenticationProfile": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                       []string{"CreateAuthenticationProfile"},
					"AuthenticationProfileName":    []string{makeName("CreateAuthenticationProfile", "profile")},
					"AuthenticationProfileContent": []string{`{"AllowDBUserOverride":"1"}`},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateAuthenticationProfile"}})
			},
		},
		"DescribeAuthenticationProfiles": {
			success: func(t *testing.T) (int, []byte) {
				profile := makeName("DescribeAuthenticationProfiles", "profile")
				createAuthenticationProfile(t, profile)
				return redshiftRequest(t, ts, url.Values{
					"Action":                    []string{"DescribeAuthenticationProfiles"},
					"AuthenticationProfileName": []string{profile},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                    []string{"DescribeAuthenticationProfiles"},
					"AuthenticationProfileName": []string{makeName("DescribeAuthenticationProfiles", "missing")},
				})
			},
		},
		"ModifyAuthenticationProfile": {
			success: func(t *testing.T) (int, []byte) {
				profile := makeName("ModifyAuthenticationProfile", "profile")
				createAuthenticationProfile(t, profile)
				return redshiftRequest(t, ts, url.Values{
					"Action":                       []string{"ModifyAuthenticationProfile"},
					"AuthenticationProfileName":    []string{profile},
					"AuthenticationProfileContent": []string{`{"AllowDBUserOverride":"0"}`},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyAuthenticationProfile"}})
			},
		},
		"DeleteAuthenticationProfile": {
			success: func(t *testing.T) (int, []byte) {
				profile := makeName("DeleteAuthenticationProfile", "profile")
				createAuthenticationProfile(t, profile)
				return redshiftRequest(t, ts, url.Values{
					"Action":                    []string{"DeleteAuthenticationProfile"},
					"AuthenticationProfileName": []string{profile},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteAuthenticationProfile"}})
			},
		},
		"CreateRedshiftIdcApplication": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"CreateRedshiftIdcApplication"},
					"RedshiftIdcApplicationName": []string{makeName("CreateRedshiftIdcApplication", "app")},
					"IdcInstanceArn":             []string{"arn:aws:sso:::instance/ssoins-1234567890abcdef"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateRedshiftIdcApplication"}})
			},
		},
		"DescribeRedshiftIdcApplications": {
			success: func(t *testing.T) (int, []byte) {
				createRedshiftIdcApplication(t, makeName("DescribeRedshiftIdcApplications", "app"))
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeRedshiftIdcApplications"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":                    []string{"DescribeRedshiftIdcApplications"},
					"RedshiftIdcApplicationArn": []string{redshiftIdcApplicationArn(makeName("DescribeRedshiftIdcApplications", "missing"))},
				})
			},
		},
		"ModifyRedshiftIdcApplication": {
			success: func(t *testing.T) (int, []byte) {
				appName := makeName("ModifyRedshiftIdcApplication", "app")
				appARN := createRedshiftIdcApplication(t, appName)
				return redshiftRequest(t, ts, url.Values{
					"Action":                     []string{"ModifyRedshiftIdcApplication"},
					"RedshiftIdcApplicationArn":  []string{appARN},
					"RedshiftIdcApplicationName": []string{makeName("ModifyRedshiftIdcApplication", "updated")},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyRedshiftIdcApplication"}})
			},
		},
		"DeleteRedshiftIdcApplication": {
			success: func(t *testing.T) (int, []byte) {
				appARN := createRedshiftIdcApplication(t, makeName("DeleteRedshiftIdcApplication", "app"))
				return redshiftRequest(t, ts, url.Values{
					"Action":                    []string{"DeleteRedshiftIdcApplication"},
					"RedshiftIdcApplicationArn": []string{appARN},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteRedshiftIdcApplication"}})
			},
		},
		"GetIdentityCenterAuthToken": {
			success: func(t *testing.T) (int, []byte) {
				appARN := createRedshiftIdcApplication(t, makeName("GetIdentityCenterAuthToken", "app"))
				return redshiftRequest(t, ts, url.Values{
					"Action":                    []string{"GetIdentityCenterAuthToken"},
					"RedshiftIdcApplicationArn": []string{appARN},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"GetIdentityCenterAuthToken"}})
			},
		},
		"RegisterNamespace": {
			success: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"RegisterNamespace"},
					"NamespaceIdentifier": []string{makeName("RegisterNamespace", "ns")},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"RegisterNamespace"}})
			},
		},
		"DeregisterNamespace": {
			success: func(t *testing.T) (int, []byte) {
				namespaceID := makeName("DeregisterNamespace", "ns")
				registerNamespace(t, namespaceID)
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"DeregisterNamespace"},
					"NamespaceIdentifier": []string{namespaceID},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeregisterNamespace"}})
			},
		},
		"ModifyLakehouseConfiguration": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyLakehouseConfiguration", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":                 []string{"ModifyLakehouseConfiguration"},
					"ClusterIdentifier":      []string{clusterID},
					"LakehouseConfiguration": []string{"enabled"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyLakehouseConfiguration"}})
			},
		},
		"ModifyAquaConfiguration": {
			success: func(t *testing.T) (int, []byte) {
				clusterID := makeName("ModifyAquaConfiguration", "cluster")
				createCluster(t, clusterID)
				return redshiftRequest(t, ts, url.Values{
					"Action":                  []string{"ModifyAquaConfiguration"},
					"ClusterIdentifier":       []string{clusterID},
					"AquaConfigurationStatus": []string{"enabled"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"ModifyAquaConfiguration"}})
			},
		},
		"CreateTags": {
			success: func(t *testing.T) (int, []byte) {
				arn := "arn:aws:redshift:us-east-1:123456789012:cluster/tag-demo"
				return redshiftRequest(t, ts, url.Values{
					"Action":              []string{"CreateTags"},
					"ResourceName":        []string{arn},
					"Tags.member.1.Key":   []string{"env"},
					"Tags.member.1.Value": []string{"test"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"CreateTags"}})
			},
		},
		"DescribeTags": {
			success: func(t *testing.T) (int, []byte) {
				arn := "arn:aws:redshift:us-east-1:123456789012:cluster/tag-describe"
				createTags(t, arn)
				return redshiftRequest(t, ts, url.Values{
					"Action": []string{"DescribeTags"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{
					"Action":             []string{"DescribeTags"},
					"TagKeys.member.1":   []string{"aws:bad"},
					"TagValues.member.1": []string{"x"},
				})
			},
		},
		"DeleteTags": {
			success: func(t *testing.T) (int, []byte) {
				arn := "arn:aws:redshift:us-east-1:123456789012:cluster/tag-delete"
				createTags(t, arn)
				return redshiftRequest(t, ts, url.Values{
					"Action":           []string{"DeleteTags"},
					"ResourceName":     []string{arn},
					"TagKeys.member.1": []string{"env"},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteTags"}})
			},
		},
		"PutResourcePolicy": {
			success: func(t *testing.T) (int, []byte) {
				arn := "arn:aws:redshift:us-east-1:123456789012:cluster/policy-demo"
				return redshiftRequest(t, ts, url.Values{
					"Action":      []string{"PutResourcePolicy"},
					"ResourceArn": []string{arn},
					"Policy":      []string{`{"Version":"2012-10-17","Statement":[]}`},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"PutResourcePolicy"}})
			},
		},
		"GetResourcePolicy": {
			success: func(t *testing.T) (int, []byte) {
				arn := "arn:aws:redshift:us-east-1:123456789012:cluster/policy-get"
				putResourcePolicy(t, arn)
				return redshiftRequest(t, ts, url.Values{
					"Action":      []string{"GetResourcePolicy"},
					"ResourceArn": []string{arn},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"GetResourcePolicy"}})
			},
		},
		"DeleteResourcePolicy": {
			success: func(t *testing.T) (int, []byte) {
				arn := "arn:aws:redshift:us-east-1:123456789012:cluster/policy-delete"
				putResourcePolicy(t, arn)
				return redshiftRequest(t, ts, url.Values{
					"Action":      []string{"DeleteResourcePolicy"},
					"ResourceArn": []string{arn},
				})
			},
			failure: func(t *testing.T) (int, []byte) {
				return redshiftRequest(t, ts, url.Values{"Action": []string{"DeleteResourcePolicy"}})
			},
		},
	}

	for _, op := range awsmodels.RedshiftOperations() {
		op := op
		c, ok := ops[op]
		if !ok {
			t.Run(op+"/success", func(t *testing.T) {
				status, _ := redshiftRequest(t, ts, url.Values{"Action": []string{op}})
				if status != http.StatusNotImplemented {
					t.Fatalf("expected 501 for %s, got %d", op, status)
				}
			})
			t.Run(op+"/failure", func(t *testing.T) {
				status, _ := redshiftRequest(t, ts, url.Values{"Action": []string{op}})
				if status != http.StatusNotImplemented {
					t.Fatalf("expected 501 for %s, got %d", op, status)
				}
			})
			continue
		}

		t.Run(op+"/success", func(t *testing.T) {
			status, body := c.success(t)
			if status != http.StatusOK {
				t.Fatalf("expected 200 for %s, got %d: %s", op, status, string(body))
			}
		})
		t.Run(op+"/failure", func(t *testing.T) {
			status, body := c.failure(t)
			if status < http.StatusBadRequest {
				t.Fatalf("expected error for %s, got %d: %s", op, status, string(body))
			}
		})
	}
}
