package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	prefix := getenv("STACKYARD_REDSHIFT_PREFIX", "redshift")

	clusterID := makeName(prefix, "cluster")
	paramGroup := makeName(prefix, "pg")
	subnetGroup := makeName(prefix, "subnet")
	secGroup := makeName(prefix, "sec")
	snapshotID := makeName(prefix, "snap")
	endpointName := makeName(prefix, "endpoint")
	scheduledAction := makeName(prefix, "schedule")
	tagResource := fmt.Sprintf("arn:aws:redshift:%s:123456789012:cluster/%s", region, clusterID)

	ctx := context.Background()
	creds, err := loadCreds()
	if err != nil {
		exitf("load aws config: %v", err)
	}

	fmt.Printf("Stackyard Redshift advanced client using %s\n", endpoint)

	if err := createParameterGroup(ctx, endpoint, region, creds, paramGroup); err != nil {
		exitf("create parameter group: %v", err)
	}
	logf("created parameter group: %s", paramGroup)

	if err := modifyParameterGroup(ctx, endpoint, region, creds, paramGroup); err != nil {
		exitf("modify parameter group: %v", err)
	}
	logf("modified parameter group: %s", paramGroup)

	if err := describeParameterGroup(ctx, endpoint, region, creds, paramGroup); err != nil {
		exitf("describe parameter group: %v", err)
	}

	if err := describeClusterParameters(ctx, endpoint, region, creds, paramGroup); err != nil {
		exitf("describe cluster parameters: %v", err)
	}

	if err := createSubnetGroup(ctx, endpoint, region, creds, subnetGroup); err != nil {
		exitf("create subnet group: %v", err)
	}
	logf("created subnet group: %s", subnetGroup)

	if err := createSecurityGroup(ctx, endpoint, region, creds, secGroup); err != nil {
		exitf("create security group: %v", err)
	}
	logf("created security group: %s", secGroup)

	if err := createCluster(ctx, endpoint, region, creds, clusterID, paramGroup, subnetGroup, secGroup); err != nil {
		exitf("create cluster: %v", err)
	}
	logf("created cluster: %s", clusterID)

	if err := modifyCluster(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("modify cluster: %v", err)
	}
	logf("modified cluster: %s", clusterID)

	if err := modifyClusterIamRoles(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("modify cluster iam roles: %v", err)
	}
	logf("updated iam roles")

	if err := getClusterCredentials(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("get cluster credentials: %v", err)
	}
	logf("fetched cluster credentials")

	if err := enableLogging(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("enable logging: %v", err)
	}
	logf("enabled logging")

	if err := describeLogging(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("describe logging: %v", err)
	}

	if err := createSnapshot(ctx, endpoint, region, creds, clusterID, snapshotID); err != nil {
		exitf("create snapshot: %v", err)
	}
	logf("created snapshot: %s", snapshotID)

	if err := describeSnapshot(ctx, endpoint, region, creds, snapshotID); err != nil {
		exitf("describe snapshot: %v", err)
	}

	if err := createEndpointAccess(ctx, endpoint, region, creds, endpointName, clusterID, subnetGroup, secGroup); err != nil {
		exitf("create endpoint access: %v", err)
	}
	logf("created endpoint access: %s", endpointName)

	if err := describeEndpointAccess(ctx, endpoint, region, creds, endpointName); err != nil {
		exitf("describe endpoint access: %v", err)
	}

	if err := createTags(ctx, endpoint, region, creds, tagResource); err != nil {
		exitf("create tags: %v", err)
	}
	logf("created tags for %s", tagResource)

	if err := describeTags(ctx, endpoint, region, creds, tagResource); err != nil {
		exitf("describe tags: %v", err)
	}

	if err := createScheduledAction(ctx, endpoint, region, creds, scheduledAction); err != nil {
		exitf("create scheduled action: %v", err)
	}
	logf("created scheduled action: %s", scheduledAction)

	if err := describeScheduledActions(ctx, endpoint, region, creds, scheduledAction); err != nil {
		exitf("describe scheduled actions: %v", err)
	}

	if err := modifyScheduledAction(ctx, endpoint, region, creds, scheduledAction); err != nil {
		exitf("modify scheduled action: %v", err)
	}
	logf("modified scheduled action: %s", scheduledAction)

	if err := deleteScheduledAction(ctx, endpoint, region, creds, scheduledAction); err != nil {
		exitf("delete scheduled action: %v", err)
	}
	logf("deleted scheduled action: %s", scheduledAction)

	if err := deleteEndpointAccess(ctx, endpoint, region, creds, endpointName); err != nil {
		exitf("delete endpoint access: %v", err)
	}
	logf("deleted endpoint access: %s", endpointName)

	if err := disableLogging(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("disable logging: %v", err)
	}
	logf("disabled logging")

	if err := deleteSnapshot(ctx, endpoint, region, creds, snapshotID); err != nil {
		exitf("delete snapshot: %v", err)
	}
	logf("deleted snapshot: %s", snapshotID)

	if err := deleteCluster(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("delete cluster: %v", err)
	}
	logf("deleted cluster: %s", clusterID)

	if err := deleteSecurityGroup(ctx, endpoint, region, creds, secGroup); err != nil {
		exitf("delete security group: %v", err)
	}
	logf("deleted security group: %s", secGroup)

	if err := deleteSubnetGroup(ctx, endpoint, region, creds, subnetGroup); err != nil {
		exitf("delete subnet group: %v", err)
	}
	logf("deleted subnet group: %s", subnetGroup)

	if err := deleteParameterGroup(ctx, endpoint, region, creds, paramGroup); err != nil {
		exitf("delete parameter group: %v", err)
	}
	logf("deleted parameter group: %s", paramGroup)

	if err := deleteTags(ctx, endpoint, region, creds, tagResource); err != nil {
		exitf("delete tags: %v", err)
	}
	logf("deleted tags")

	fmt.Println("Done.")
}

func loadCreds() (aws.CredentialsProvider, error) {
	return credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	), nil
}

func redshiftRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, params url.Values) ([]byte, error) {
	if params.Get("Version") == "" {
		params.Set("Version", "2012-12-01")
	}
	body := []byte(params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/redshift", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	credentials, err := creds.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentials, req, hashSHA256(body), "redshift", region, time.Now()); err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func createParameterGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":               []string{"CreateClusterParameterGroup"},
		"ParameterGroupName":   []string{name},
		"ParameterGroupFamily": []string{"redshift-1.0"},
		"Description":          []string{"stackyard demo"},
	})
	return err
}

func modifyParameterGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":                             []string{"ModifyClusterParameterGroup"},
		"ParameterGroupName":                 []string{name},
		"Parameters.member.1.ParameterName":  []string{"max_concurrency_scaling_clusters"},
		"Parameters.member.1.ParameterValue": []string{"1"},
	})
	return err
}

func describeParameterGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":             []string{"DescribeClusterParameterGroups"},
		"ParameterGroupName": []string{name},
	})
	return err
}

func describeClusterParameters(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":             []string{"DescribeClusterParameters"},
		"ParameterGroupName": []string{name},
	})
	return err
}

func deleteParameterGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":             []string{"DeleteClusterParameterGroup"},
		"ParameterGroupName": []string{name},
	})
	return err
}

func createSubnetGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":                 []string{"CreateClusterSubnetGroup"},
		"ClusterSubnetGroupName": []string{name},
		"Description":            []string{"stackyard subnet group"},
		"SubnetIds.member.1":     []string{"subnet-1234"},
	})
	return err
}

func deleteSubnetGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":                 []string{"DeleteClusterSubnetGroup"},
		"ClusterSubnetGroupName": []string{name},
	})
	return err
}

func createSecurityGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":                   []string{"CreateClusterSecurityGroup"},
		"ClusterSecurityGroupName": []string{name},
		"Description":              []string{"stackyard security group"},
	})
	return err
}

func deleteSecurityGroup(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":                   []string{"DeleteClusterSecurityGroup"},
		"ClusterSecurityGroupName": []string{name},
	})
	return err
}

func createCluster(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID, paramGroup, subnetGroup, secGroup string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":                       []string{"CreateCluster"},
		"ClusterIdentifier":            []string{clusterID},
		"NodeType":                     []string{"ra3.xlplus"},
		"MasterUsername":               []string{"admin"},
		"MasterUserPassword":           []string{"Secret1234"},
		"DBName":                       []string{"dev"},
		"ClusterParameterGroupName":    []string{paramGroup},
		"ClusterSubnetGroupName":       []string{subnetGroup},
		"VpcSecurityGroupIds.member.1": []string{secGroup},
	})
	return err
}

func modifyCluster(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"ModifyCluster"},
		"ClusterIdentifier": []string{clusterID},
		"NodeType":          []string{"ra3.large"},
	})
	return err
}

func modifyClusterIamRoles(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":               []string{"ModifyClusterIamRoles"},
		"ClusterIdentifier":    []string{clusterID},
		"AddIamRoles.member.1": []string{"arn:aws:iam::123456789012:role/demo"},
	})
	return err
}

func getClusterCredentials(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"GetClusterCredentials"},
		"ClusterIdentifier": []string{clusterID},
		"DbUser":            []string{"devuser"},
	})
	return err
}

func deleteCluster(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"DeleteCluster"},
		"ClusterIdentifier": []string{clusterID},
	})
	return err
}

func enableLogging(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"EnableLogging"},
		"ClusterIdentifier": []string{clusterID},
		"BucketName":        []string{"redshift-logs"},
		"S3KeyPrefix":       []string{"stackyard"},
	})
	return err
}

func describeLogging(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"DescribeLoggingStatus"},
		"ClusterIdentifier": []string{clusterID},
	})
	return err
}

func disableLogging(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"DisableLogging"},
		"ClusterIdentifier": []string{clusterID},
	})
	return err
}

func createSnapshot(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID, snapshotID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":             []string{"CreateClusterSnapshot"},
		"ClusterIdentifier":  []string{clusterID},
		"SnapshotIdentifier": []string{snapshotID},
	})
	return err
}

func describeSnapshot(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, snapshotID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":             []string{"DescribeClusterSnapshots"},
		"SnapshotIdentifier": []string{snapshotID},
	})
	return err
}

func deleteSnapshot(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, snapshotID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":             []string{"DeleteClusterSnapshot"},
		"SnapshotIdentifier": []string{snapshotID},
	})
	return err
}

func createEndpointAccess(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, endpointName, clusterID, subnetGroup, secGroup string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":                       []string{"CreateEndpointAccess"},
		"EndpointName":                 []string{endpointName},
		"ClusterIdentifier":            []string{clusterID},
		"SubnetGroupName":              []string{subnetGroup},
		"VpcSecurityGroupIds.member.1": []string{secGroup},
	})
	return err
}

func describeEndpointAccess(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, endpointName string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":       []string{"DescribeEndpointAccess"},
		"EndpointName": []string{endpointName},
	})
	return err
}

func deleteEndpointAccess(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, endpointName string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":       []string{"DeleteEndpointAccess"},
		"EndpointName": []string{endpointName},
	})
	return err
}

func createTags(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, resource string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":              []string{"CreateTags"},
		"ResourceName":        []string{resource},
		"Tags.member.1.Key":   []string{"env"},
		"Tags.member.1.Value": []string{"dev"},
		"Tags.member.2.Key":   []string{"team"},
		"Tags.member.2.Value": []string{"stackyard"},
	})
	return err
}

func describeTags(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, resource string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":       []string{"DescribeTags"},
		"ResourceName": []string{resource},
	})
	return err
}

func deleteTags(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, resource string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":           []string{"DeleteTags"},
		"ResourceName":     []string{resource},
		"TagKeys.member.1": []string{"env"},
		"TagKeys.member.2": []string{"team"},
	})
	return err
}

func createScheduledAction(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":              []string{"CreateScheduledAction"},
		"ScheduledActionName": []string{name},
		"TargetAction":        []string{"ResizeCluster"},
		"Schedule":            []string{"cron(0 12 * * ? *)"},
	})
	return err
}

func describeScheduledActions(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":              []string{"DescribeScheduledActions"},
		"ScheduledActionName": []string{name},
	})
	return err
}

func modifyScheduledAction(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":              []string{"ModifyScheduledAction"},
		"ScheduledActionName": []string{name},
		"Schedule":            []string{"cron(5 12 * * ? *)"},
	})
	return err
}

func deleteScheduledAction(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, name string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":              []string{"DeleteScheduledAction"},
		"ScheduledActionName": []string{name},
	})
	return err
}

func hashSHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func makeName(prefix, suffix string) string {
	if suffix == "" {
		return prefix
	}
	return fmt.Sprintf("%s-%s", prefix, suffix)
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func init() {
	_ = time.Now()
}
