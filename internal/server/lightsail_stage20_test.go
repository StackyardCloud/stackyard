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
)

func TestLightsailStage20RelationalSnapshotsRestoreLogging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateRelationalDatabase", []byte(`{"relationalDatabaseName":"stage20-db","availabilityZone":"us-east-1a","masterDatabaseName":"appdb","masterUsername":"admin","masterUserPassword":"Stage20pass!","relationalDatabaseBlueprintId":"mysql_8_0","relationalDatabaseBundleId":"micro_1_0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateRelationalDatabaseSnapshot", []byte(`{"relationalDatabaseName":"stage20-db","relationalDatabaseSnapshotName":"stage20-snap"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseSnapshot", []byte(`{"relationalDatabaseSnapshotName":"stage20-snap"}`))
	assertStatus(t, resp, http.StatusOK)
	var getSnapshotOut struct {
		RelationalDatabaseSnapshot struct {
			Name string `json:"name"`
		} `json:"relationalDatabaseSnapshot"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getSnapshotOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseSnapshot: %v", err)
	}
	if getSnapshotOut.RelationalDatabaseSnapshot.Name != "stage20-snap" {
		t.Fatalf("unexpected GetRelationalDatabaseSnapshot output: %+v", getSnapshotOut)
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseSnapshots", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateRelationalDatabaseFromSnapshot", []byte(`{"relationalDatabaseName":"stage20-db-restore","relationalDatabaseSnapshotName":"stage20-snap","relationalDatabaseBundleId":"small_1_0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseEvents", []byte(`{"relationalDatabaseName":"stage20-db"}`))
	assertStatus(t, resp, http.StatusOK)
	var eventsOut struct {
		RelationalDatabaseEvents []struct {
			Message string `json:"message"`
		} `json:"relationalDatabaseEvents"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &eventsOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseEvents: %v", err)
	}
	if len(eventsOut.RelationalDatabaseEvents) == 0 {
		t.Fatalf("expected relational database events")
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseLogStreams", []byte(`{"relationalDatabaseName":"stage20-db"}`))
	assertStatus(t, resp, http.StatusOK)
	var logStreamsOut struct {
		LogStreams []string `json:"logStreams"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &logStreamsOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseLogStreams: %v", err)
	}
	if len(logStreamsOut.LogStreams) == 0 {
		t.Fatalf("expected log streams")
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseLogEvents", []byte(`{"relationalDatabaseName":"stage20-db","logStreamName":"`+logStreamsOut.LogStreams[0]+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var logEventsOut struct {
		ResourceLogEvents []struct {
			Message string `json:"message"`
		} `json:"resourceLogEvents"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &logEventsOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseLogEvents: %v", err)
	}
	if len(logEventsOut.ResourceLogEvents) == 0 {
		t.Fatalf("expected log events")
	}

	resp = lightsailRequest(t, ts, "DeleteRelationalDatabaseSnapshot", []byte(`{"relationalDatabaseSnapshotName":"stage20-snap"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage20SDKClientRelationalSnapshotsRestoreLogging(t *testing.T) {
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

	if _, err := client.CreateRelationalDatabase(ctx, &awslightsail.CreateRelationalDatabaseInput{
		RelationalDatabaseName:        aws.String("sdk-stage20-db"),
		AvailabilityZone:              aws.String("us-east-1a"),
		MasterDatabaseName:            aws.String("appdb"),
		MasterUsername:                aws.String("admin"),
		MasterUserPassword:            aws.String("Stage20pass!"),
		RelationalDatabaseBlueprintId: aws.String("mysql_8_0"),
		RelationalDatabaseBundleId:    aws.String("micro_1_0"),
	}); err != nil {
		t.Fatalf("create relational database: %v", err)
	}

	createSnapshotOut, err := client.CreateRelationalDatabaseSnapshot(ctx, &awslightsail.CreateRelationalDatabaseSnapshotInput{
		RelationalDatabaseName:         aws.String("sdk-stage20-db"),
		RelationalDatabaseSnapshotName: aws.String("sdk-stage20-snap"),
	})
	if err != nil {
		t.Fatalf("create relational database snapshot: %v", err)
	}
	if len(createSnapshotOut.Operations) != 1 {
		t.Fatalf("expected snapshot create operations")
	}

	getSnapshotOut, err := client.GetRelationalDatabaseSnapshot(ctx, &awslightsail.GetRelationalDatabaseSnapshotInput{
		RelationalDatabaseSnapshotName: aws.String("sdk-stage20-snap"),
	})
	if err != nil {
		t.Fatalf("get relational database snapshot: %v", err)
	}
	if getSnapshotOut.RelationalDatabaseSnapshot == nil || getSnapshotOut.RelationalDatabaseSnapshot.Name == nil || *getSnapshotOut.RelationalDatabaseSnapshot.Name != "sdk-stage20-snap" {
		t.Fatalf("unexpected get snapshot output: %+v", getSnapshotOut.RelationalDatabaseSnapshot)
	}

	listSnapshotsOut, err := client.GetRelationalDatabaseSnapshots(ctx, &awslightsail.GetRelationalDatabaseSnapshotsInput{})
	if err != nil {
		t.Fatalf("get relational database snapshots: %v", err)
	}
	if len(listSnapshotsOut.RelationalDatabaseSnapshots) == 0 {
		t.Fatalf("expected relational database snapshots")
	}

	createFromSnapshotOut, err := client.CreateRelationalDatabaseFromSnapshot(ctx, &awslightsail.CreateRelationalDatabaseFromSnapshotInput{
		RelationalDatabaseName:         aws.String("sdk-stage20-db-restore"),
		RelationalDatabaseSnapshotName: aws.String("sdk-stage20-snap"),
		RelationalDatabaseBundleId:     aws.String("small_1_0"),
	})
	if err != nil {
		t.Fatalf("create relational database from snapshot: %v", err)
	}
	if len(createFromSnapshotOut.Operations) != 1 {
		t.Fatalf("expected create-from-snapshot operations")
	}

	eventsOut, err := client.GetRelationalDatabaseEvents(ctx, &awslightsail.GetRelationalDatabaseEventsInput{
		RelationalDatabaseName: aws.String("sdk-stage20-db"),
	})
	if err != nil {
		t.Fatalf("get relational database events: %v", err)
	}
	if len(eventsOut.RelationalDatabaseEvents) == 0 {
		t.Fatalf("expected relational database events")
	}

	logStreamsOut, err := client.GetRelationalDatabaseLogStreams(ctx, &awslightsail.GetRelationalDatabaseLogStreamsInput{
		RelationalDatabaseName: aws.String("sdk-stage20-db"),
	})
	if err != nil {
		t.Fatalf("get relational database log streams: %v", err)
	}
	if len(logStreamsOut.LogStreams) == 0 {
		t.Fatalf("expected log streams")
	}

	logEventsOut, err := client.GetRelationalDatabaseLogEvents(ctx, &awslightsail.GetRelationalDatabaseLogEventsInput{
		RelationalDatabaseName: aws.String("sdk-stage20-db"),
		LogStreamName:          aws.String(logStreamsOut.LogStreams[0]),
	})
	if err != nil {
		t.Fatalf("get relational database log events: %v", err)
	}
	if len(logEventsOut.ResourceLogEvents) == 0 {
		t.Fatalf("expected log events")
	}

	deleteSnapshotOut, err := client.DeleteRelationalDatabaseSnapshot(ctx, &awslightsail.DeleteRelationalDatabaseSnapshotInput{
		RelationalDatabaseSnapshotName: aws.String("sdk-stage20-snap"),
	})
	if err != nil {
		t.Fatalf("delete relational database snapshot: %v", err)
	}
	if len(deleteSnapshotOut.Operations) != 1 {
		t.Fatalf("expected snapshot delete operations")
	}
}
