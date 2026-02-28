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

func TestLightsailStage9AlarmsAddOnsAutoSnapshots(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage9-instance"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "PutAlarm", []byte(`{"alarmName":"stage9-cpu-high","comparisonOperator":"GreaterThanOrEqualToThreshold","evaluationPeriods":1,"metricName":"CPUUtilization","monitoredResourceName":"stage9-instance","threshold":80}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetAlarms", []byte(`{"alarmName":"stage9-cpu-high"}`))
	assertStatus(t, resp, http.StatusOK)
	var getAlarmsOut struct {
		Alarms []struct {
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"alarms"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getAlarmsOut); err != nil {
		t.Fatalf("unmarshal GetAlarms: %v", err)
	}
	if len(getAlarmsOut.Alarms) != 1 || getAlarmsOut.Alarms[0].Name != "stage9-cpu-high" {
		t.Fatalf("unexpected GetAlarms output: %+v", getAlarmsOut)
	}

	resp = lightsailRequest(t, ts, "TestAlarm", []byte(`{"alarmName":"stage9-cpu-high","state":"ALARM"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetAlarms", []byte(`{"alarmName":"stage9-cpu-high"}`))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &getAlarmsOut); err != nil {
		t.Fatalf("unmarshal GetAlarms after test: %v", err)
	}
	if len(getAlarmsOut.Alarms) != 1 || getAlarmsOut.Alarms[0].State != "ALARM" {
		t.Fatalf("expected alarm state ALARM after TestAlarm: %+v", getAlarmsOut)
	}

	resp = lightsailRequest(t, ts, "DeleteAlarm", []byte(`{"alarmName":"stage9-cpu-high"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "EnableAddOn", []byte(`{"resourceName":"stage9-instance","addOnRequest":{"addOnType":"AutoSnapshot","autoSnapshotAddOnRequest":{"snapshotTimeOfDay":"06:00"}}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetAutoSnapshots", []byte(`{"resourceName":"stage9-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	var getAutoOut struct {
		AutoSnapshots []struct {
			Date string `json:"date"`
		} `json:"autoSnapshots"`
		ResourceName string `json:"resourceName"`
		ResourceType string `json:"resourceType"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getAutoOut); err != nil {
		t.Fatalf("unmarshal GetAutoSnapshots: %v", err)
	}
	if getAutoOut.ResourceName != "stage9-instance" || getAutoOut.ResourceType != "Instance" {
		t.Fatalf("unexpected GetAutoSnapshots resource metadata: %+v", getAutoOut)
	}
	if len(getAutoOut.AutoSnapshots) == 0 {
		t.Fatalf("expected at least one auto snapshot")
	}

	resp = lightsailRequest(t, ts, "DeleteAutoSnapshot", []byte(`{"resourceName":"stage9-instance","date":"`+getAutoOut.AutoSnapshots[0].Date+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DisableAddOn", []byte(`{"resourceName":"stage9-instance","addOnType":"AutoSnapshot"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage9SDKClientAlarmsAddOnsAutoSnapshots(t *testing.T) {
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
		InstanceNames:    []string{"sdk-stage9-instance"},
	}); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if _, err := client.PutAlarm(ctx, &awslightsail.PutAlarmInput{
		AlarmName:             aws.String("sdk-stage9-cpu-high"),
		ComparisonOperator:    awslightsailtypes.ComparisonOperatorGreaterThanOrEqualToThreshold,
		EvaluationPeriods:     aws.Int32(1),
		MetricName:            awslightsailtypes.MetricNameCPUUtilization,
		MonitoredResourceName: aws.String("sdk-stage9-instance"),
		Threshold:             aws.Float64(80),
		ContactProtocols:      []awslightsailtypes.ContactProtocol{awslightsailtypes.ContactProtocolEmail},
	}); err != nil {
		t.Fatalf("put alarm: %v", err)
	}

	getAlarmsOut, err := client.GetAlarms(ctx, &awslightsail.GetAlarmsInput{
		AlarmName: aws.String("sdk-stage9-cpu-high"),
	})
	if err != nil {
		t.Fatalf("get alarms: %v", err)
	}
	if len(getAlarmsOut.Alarms) != 1 {
		t.Fatalf("expected one alarm, got %d", len(getAlarmsOut.Alarms))
	}

	if _, err := client.TestAlarm(ctx, &awslightsail.TestAlarmInput{
		AlarmName: aws.String("sdk-stage9-cpu-high"),
		State:     awslightsailtypes.AlarmStateAlarm,
	}); err != nil {
		t.Fatalf("test alarm: %v", err)
	}

	if _, err := client.DeleteAlarm(ctx, &awslightsail.DeleteAlarmInput{
		AlarmName: aws.String("sdk-stage9-cpu-high"),
	}); err != nil {
		t.Fatalf("delete alarm: %v", err)
	}

	if _, err := client.EnableAddOn(ctx, &awslightsail.EnableAddOnInput{
		ResourceName: aws.String("sdk-stage9-instance"),
		AddOnRequest: &awslightsailtypes.AddOnRequest{
			AddOnType: awslightsailtypes.AddOnTypeAutoSnapshot,
			AutoSnapshotAddOnRequest: &awslightsailtypes.AutoSnapshotAddOnRequest{
				SnapshotTimeOfDay: aws.String("06:00"),
			},
		},
	}); err != nil {
		t.Fatalf("enable add-on: %v", err)
	}

	autoOut, err := client.GetAutoSnapshots(ctx, &awslightsail.GetAutoSnapshotsInput{
		ResourceName: aws.String("sdk-stage9-instance"),
	})
	if err != nil {
		t.Fatalf("get auto snapshots: %v", err)
	}
	if len(autoOut.AutoSnapshots) == 0 {
		t.Fatalf("expected at least one auto snapshot")
	}

	if _, err := client.DeleteAutoSnapshot(ctx, &awslightsail.DeleteAutoSnapshotInput{
		ResourceName: aws.String("sdk-stage9-instance"),
		Date:         autoOut.AutoSnapshots[0].Date,
	}); err != nil {
		t.Fatalf("delete auto snapshot: %v", err)
	}

	if _, err := client.DisableAddOn(ctx, &awslightsail.DisableAddOnInput{
		ResourceName: aws.String("sdk-stage9-instance"),
		AddOnType:    awslightsailtypes.AddOnTypeAutoSnapshot,
	}); err != nil {
		t.Fatalf("disable add-on: %v", err)
	}
}
