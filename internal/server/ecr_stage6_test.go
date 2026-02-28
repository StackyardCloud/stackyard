package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	awsecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func TestECRStage6ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const repositoryName = "stage6-raw"
	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	putImagePayload, err := json.Marshal(map[string]any{
		"repositoryName": repositoryName,
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "latest",
	})
	if err != nil {
		t.Fatalf("marshal put image payload: %v", err)
	}
	resp = ecrRequest(t, ts, "PutImage", putImagePayload)
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeRegistry", body: []byte(`{}`)},
		{name: "PutRegistryPolicy", body: []byte(`{"policyText":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)},
		{name: "GetRegistryPolicy", body: []byte(`{}`)},
		{name: "DeleteRegistryPolicy", body: []byte(`{}`)},
		{name: "PutRegistryScanningConfiguration", body: []byte(`{"scanType":"BASIC","rules":[{"repositoryFilters":[{"filter":"*","filterType":"WILDCARD"}],"scanFrequency":"SCAN_ON_PUSH"}]}`)},
		{name: "GetRegistryScanningConfiguration", body: []byte(`{}`)},
		{name: "PutReplicationConfiguration", body: []byte(`{"replicationConfiguration":{"rules":[{"destinations":[{"region":"us-west-2","registryId":"123456789012"}],"repositoryFilters":[{"filter":"demo","filterType":"PREFIX_MATCH"}]}]}}`)},
		{name: "PutAccountSetting", body: []byte(`{"name":"REGISTRY_POLICY_SCOPE","value":"V2"}`)},
		{name: "GetAccountSetting", body: []byte(`{"name":"REGISTRY_POLICY_SCOPE"}`)},
		{name: "DescribeImageReplicationStatus", body: []byte(`{"repositoryName":"` + repositoryName + `","imageId":{"imageTag":"latest"}}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage6SDKRegistryAndReplicationFlow(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := awsecr.NewFromConfig(cfg, func(o *awsecr.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	const repositoryName = "stage6-sdk"
	if _, err := client.CreateRepository(ctx, &awsecr.CreateRepositoryInput{
		RepositoryName: aws.String(repositoryName),
	}); err != nil {
		t.Fatalf("create repository: %v", err)
	}

	if _, err := client.PutImage(ctx, &awsecr.PutImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageManifest:  aws.String(`{"schemaVersion":2}`),
		ImageTag:       aws.String("v1"),
	}); err != nil {
		t.Fatalf("put image: %v", err)
	}

	describeRegistryOut, err := client.DescribeRegistry(ctx, &awsecr.DescribeRegistryInput{})
	if err != nil {
		t.Fatalf("describe registry: %v", err)
	}
	if aws.ToString(describeRegistryOut.RegistryId) == "" {
		t.Fatalf("expected registry id in describe registry output")
	}

	const policyText = `{"Version":"2012-10-17","Statement":[]}`
	if _, err := client.PutRegistryPolicy(ctx, &awsecr.PutRegistryPolicyInput{
		PolicyText: aws.String(policyText),
	}); err != nil {
		t.Fatalf("put registry policy: %v", err)
	}

	getPolicyOut, err := client.GetRegistryPolicy(ctx, &awsecr.GetRegistryPolicyInput{})
	if err != nil {
		t.Fatalf("get registry policy: %v", err)
	}
	if aws.ToString(getPolicyOut.PolicyText) != policyText {
		t.Fatalf("unexpected registry policy text: %q", aws.ToString(getPolicyOut.PolicyText))
	}

	if _, err := client.DeleteRegistryPolicy(ctx, &awsecr.DeleteRegistryPolicyInput{}); err != nil {
		t.Fatalf("delete registry policy: %v", err)
	}

	_, err = client.GetRegistryPolicy(ctx, &awsecr.GetRegistryPolicyInput{})
	if err == nil {
		t.Fatalf("expected get registry policy after delete to fail")
	}
	var registryPolicyErr *awsecrtypes.RegistryPolicyNotFoundException
	if !errors.As(err, &registryPolicyErr) {
		t.Fatalf("expected RegistryPolicyNotFoundException, got %v", err)
	}

	getScanningOut, err := client.GetRegistryScanningConfiguration(ctx, &awsecr.GetRegistryScanningConfigurationInput{})
	if err != nil {
		t.Fatalf("get registry scanning configuration: %v", err)
	}
	if getScanningOut.ScanningConfiguration == nil {
		t.Fatalf("expected scanning configuration in get output")
	}

	putScanningOut, err := client.PutRegistryScanningConfiguration(ctx, &awsecr.PutRegistryScanningConfigurationInput{
		ScanType: awsecrtypes.ScanTypeBasic,
		Rules: []awsecrtypes.RegistryScanningRule{
			{
				ScanFrequency: awsecrtypes.ScanFrequencyScanOnPush,
				RepositoryFilters: []awsecrtypes.ScanningRepositoryFilter{
					{
						Filter:     aws.String("*"),
						FilterType: awsecrtypes.ScanningRepositoryFilterTypeWildcard,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("put registry scanning configuration: %v", err)
	}
	if putScanningOut.RegistryScanningConfiguration == nil || putScanningOut.RegistryScanningConfiguration.ScanType != awsecrtypes.ScanTypeBasic {
		t.Fatalf("unexpected registry scan type after update")
	}

	putReplicationOut, err := client.PutReplicationConfiguration(ctx, &awsecr.PutReplicationConfigurationInput{
		ReplicationConfiguration: &awsecrtypes.ReplicationConfiguration{
			Rules: []awsecrtypes.ReplicationRule{
				{
					Destinations: []awsecrtypes.ReplicationDestination{
						{
							Region:     aws.String("us-west-2"),
							RegistryId: aws.String("123456789012"),
						},
					},
					RepositoryFilters: []awsecrtypes.RepositoryFilter{
						{
							Filter:     aws.String("stage6"),
							FilterType: awsecrtypes.RepositoryFilterTypePrefixMatch,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("put replication configuration: %v", err)
	}
	if putReplicationOut.ReplicationConfiguration == nil || len(putReplicationOut.ReplicationConfiguration.Rules) != 1 {
		t.Fatalf("expected one replication rule")
	}

	replicationStatusOut, err := client.DescribeImageReplicationStatus(ctx, &awsecr.DescribeImageReplicationStatusInput{
		RepositoryName: aws.String(repositoryName),
		ImageId: &awsecrtypes.ImageIdentifier{
			ImageTag: aws.String("v1"),
		},
	})
	if err != nil {
		t.Fatalf("describe image replication status: %v", err)
	}
	if len(replicationStatusOut.ReplicationStatuses) != 1 {
		t.Fatalf("expected one replication status")
	}
	if replicationStatusOut.ReplicationStatuses[0].Status != awsecrtypes.ReplicationStatusComplete {
		t.Fatalf("unexpected replication status: %s", replicationStatusOut.ReplicationStatuses[0].Status)
	}
}
