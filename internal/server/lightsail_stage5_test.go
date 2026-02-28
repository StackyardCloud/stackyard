package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage5InstanceAccessAndNetworking(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage5-instance"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "OpenInstancePublicPorts", []byte(`{"instanceName":"stage5-instance","portInfo":{"fromPort":443,"toPort":443,"protocol":"tcp","cidrs":["0.0.0.0/0"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetInstancePortStates", []byte(`{"instanceName":"stage5-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	var portStatesOut struct {
		PortStates []struct {
			FromPort int32  `json:"fromPort"`
			ToPort   int32  `json:"toPort"`
			Protocol string `json:"protocol"`
			State    string `json:"state"`
		} `json:"portStates"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &portStatesOut); err != nil {
		t.Fatalf("unmarshal GetInstancePortStates: %v", err)
	}
	if len(portStatesOut.PortStates) < 2 {
		t.Fatalf("expected at least two port states")
	}

	resp = lightsailRequest(t, ts, "CloseInstancePublicPorts", []byte(`{"instanceName":"stage5-instance","portInfo":{"fromPort":443,"toPort":443,"protocol":"tcp"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "PutInstancePublicPorts", []byte(`{"instanceName":"stage5-instance","portInfos":[{"fromPort":80,"toPort":80,"protocol":"tcp","cidrs":["0.0.0.0/0"]}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetInstancePortStates", []byte(`{"instanceName":"stage5-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &portStatesOut); err != nil {
		t.Fatalf("unmarshal GetInstancePortStates after put: %v", err)
	}
	if len(portStatesOut.PortStates) != 1 || portStatesOut.PortStates[0].FromPort != 80 {
		t.Fatalf("expected only port 80 after PutInstancePublicPorts")
	}

	resp = lightsailRequest(t, ts, "GetInstanceAccessDetails", []byte(`{"instanceName":"stage5-instance","protocol":"ssh"}`))
	assertStatus(t, resp, http.StatusOK)
	var accessOut struct {
		AccessDetails struct {
			InstanceName string `json:"instanceName"`
			PrivateKey   string `json:"privateKey"`
			Username     string `json:"username"`
		} `json:"accessDetails"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &accessOut); err != nil {
		t.Fatalf("unmarshal GetInstanceAccessDetails: %v", err)
	}
	if accessOut.AccessDetails.InstanceName != "stage5-instance" || accessOut.AccessDetails.PrivateKey == "" {
		t.Fatalf("unexpected access details: %+v", accessOut.AccessDetails)
	}

	resp = lightsailRequest(t, ts, "UpdateInstanceMetadataOptions", []byte(`{"instanceName":"stage5-instance","httpEndpoint":"enabled","httpProtocolIpv6":"disabled","httpTokens":"required","httpPutResponseHopLimit":3}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetInstance", []byte(`{"instanceName":"stage5-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	var instanceOut struct {
		Instance struct {
			MetadataOptions struct {
				HttpTokens              string `json:"httpTokens"`
				HttpPutResponseHopLimit int32  `json:"httpPutResponseHopLimit"`
			} `json:"metadataOptions"`
		} `json:"instance"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &instanceOut); err != nil {
		t.Fatalf("unmarshal GetInstance: %v", err)
	}
	if instanceOut.Instance.MetadataOptions.HttpTokens != "required" || instanceOut.Instance.MetadataOptions.HttpPutResponseHopLimit != 3 {
		t.Fatalf("metadata options were not updated")
	}

	resp = lightsailRequest(t, ts, "CreateInstanceSnapshot", []byte(`{"instanceName":"stage5-instance","instanceSnapshotName":"stage5-snapshot"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateInstancesFromSnapshot", []byte(`{"availabilityZone":"us-east-1a","bundleId":"micro_2_0","instanceNames":["stage5-from-snapshot"],"instanceSnapshotName":"stage5-snapshot"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteKnownHostKeys", []byte(`{"instanceName":"stage5-instance"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage5SDKClientInstanceAccessAndNetworking(t *testing.T) {
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

	client := awslightsail.NewFromConfig(cfg, func(o *awslightsail.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	if _, err := client.CreateInstances(ctx, &awslightsail.CreateInstancesInput{
		AvailabilityZone: aws.String("us-east-1a"),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{"sdk-stage5-instance"},
	}); err != nil {
		t.Fatalf("create instances: %v", err)
	}

	if _, err := client.OpenInstancePublicPorts(ctx, &awslightsail.OpenInstancePublicPortsInput{
		InstanceName: aws.String("sdk-stage5-instance"),
		PortInfo: &awslightsailtypes.PortInfo{
			FromPort: 443,
			ToPort:   443,
			Protocol: awslightsailtypes.NetworkProtocolTcp,
			Cidrs:    []string{"0.0.0.0/0"},
		},
	}); err != nil {
		t.Fatalf("open instance public ports: %v", err)
	}

	portStatesOut, err := client.GetInstancePortStates(ctx, &awslightsail.GetInstancePortStatesInput{InstanceName: aws.String("sdk-stage5-instance")})
	if err != nil {
		t.Fatalf("get instance port states: %v", err)
	}
	if len(portStatesOut.PortStates) < 2 {
		t.Fatalf("expected at least two port states")
	}

	if _, err := client.CloseInstancePublicPorts(ctx, &awslightsail.CloseInstancePublicPortsInput{
		InstanceName: aws.String("sdk-stage5-instance"),
		PortInfo: &awslightsailtypes.PortInfo{
			FromPort: 443,
			ToPort:   443,
			Protocol: awslightsailtypes.NetworkProtocolTcp,
		},
	}); err != nil {
		t.Fatalf("close instance public ports: %v", err)
	}

	if _, err := client.PutInstancePublicPorts(ctx, &awslightsail.PutInstancePublicPortsInput{
		InstanceName: aws.String("sdk-stage5-instance"),
		PortInfos: []awslightsailtypes.PortInfo{{
			FromPort: 80,
			ToPort:   80,
			Protocol: awslightsailtypes.NetworkProtocolTcp,
			Cidrs:    []string{"0.0.0.0/0"},
		}},
	}); err != nil {
		t.Fatalf("put instance public ports: %v", err)
	}

	portStatesOut, err = client.GetInstancePortStates(ctx, &awslightsail.GetInstancePortStatesInput{InstanceName: aws.String("sdk-stage5-instance")})
	if err != nil {
		t.Fatalf("get instance port states after put: %v", err)
	}
	if len(portStatesOut.PortStates) != 1 || portStatesOut.PortStates[0].FromPort != 80 {
		t.Fatalf("expected single port 80 state after put")
	}

	accessOut, err := client.GetInstanceAccessDetails(ctx, &awslightsail.GetInstanceAccessDetailsInput{
		InstanceName: aws.String("sdk-stage5-instance"),
		Protocol:     awslightsailtypes.InstanceAccessProtocolSsh,
	})
	if err != nil {
		t.Fatalf("get instance access details: %v", err)
	}
	if accessOut.AccessDetails == nil || aws.ToString(accessOut.AccessDetails.PrivateKey) == "" {
		t.Fatalf("expected private key in access details")
	}

	if _, err := client.UpdateInstanceMetadataOptions(ctx, &awslightsail.UpdateInstanceMetadataOptionsInput{
		InstanceName:            aws.String("sdk-stage5-instance"),
		HttpEndpoint:            awslightsailtypes.HttpEndpointEnabled,
		HttpProtocolIpv6:        awslightsailtypes.HttpProtocolIpv6Disabled,
		HttpTokens:              awslightsailtypes.HttpTokensRequired,
		HttpPutResponseHopLimit: aws.Int32(4),
	}); err != nil {
		t.Fatalf("update instance metadata options: %v", err)
	}

	instanceOut, err := client.GetInstance(ctx, &awslightsail.GetInstanceInput{InstanceName: aws.String("sdk-stage5-instance")})
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if instanceOut.Instance == nil || instanceOut.Instance.MetadataOptions == nil || instanceOut.Instance.MetadataOptions.HttpTokens != awslightsailtypes.HttpTokensRequired {
		t.Fatalf("expected updated metadata options on instance")
	}

	if _, err := client.CreateInstanceSnapshot(ctx, &awslightsail.CreateInstanceSnapshotInput{
		InstanceName:         aws.String("sdk-stage5-instance"),
		InstanceSnapshotName: aws.String("sdk-stage5-snapshot"),
	}); err != nil {
		t.Fatalf("create instance snapshot: %v", err)
	}

	createdFromSnap, err := client.CreateInstancesFromSnapshot(ctx, &awslightsail.CreateInstancesFromSnapshotInput{
		AvailabilityZone:     aws.String("us-east-1a"),
		BundleId:             aws.String("micro_2_0"),
		InstanceNames:        []string{"sdk-stage5-from-snapshot"},
		InstanceSnapshotName: aws.String("sdk-stage5-snapshot"),
	})
	if err != nil {
		t.Fatalf("create instances from snapshot: %v", err)
	}
	if len(createdFromSnap.Operations) != 1 {
		t.Fatalf("expected one create-from-snapshot operation")
	}

	if _, err := client.DeleteKnownHostKeys(ctx, &awslightsail.DeleteKnownHostKeysInput{InstanceName: aws.String("sdk-stage5-instance")}); err != nil {
		t.Fatalf("delete known host keys: %v", err)
	}
}
