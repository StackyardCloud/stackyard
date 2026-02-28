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

func TestLightsailStage19RelationalDatabaseCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateRelationalDatabase", []byte(`{"relationalDatabaseName":"stage19-db","availabilityZone":"us-east-1a","masterDatabaseName":"appdb","masterUsername":"admin","masterUserPassword":"Stage19pass!","relationalDatabaseBlueprintId":"mysql_8_0","relationalDatabaseBundleId":"micro_1_0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetRelationalDatabase", []byte(`{"relationalDatabaseName":"stage19-db"}`))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		RelationalDatabase struct {
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"relationalDatabase"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabase: %v", err)
	}
	if getOut.RelationalDatabase.Name != "stage19-db" || getOut.RelationalDatabase.State == "" {
		t.Fatalf("unexpected GetRelationalDatabase output: %+v", getOut)
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabases", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		RelationalDatabases []struct {
			Name string `json:"name"`
		} `json:"relationalDatabases"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabases: %v", err)
	}
	if len(listOut.RelationalDatabases) != 1 || listOut.RelationalDatabases[0].Name != "stage19-db" {
		t.Fatalf("unexpected GetRelationalDatabases output: %+v", listOut)
	}

	resp = lightsailRequest(t, ts, "UpdateRelationalDatabase", []byte(`{"relationalDatabaseName":"stage19-db","publiclyAccessible":true,"preferredBackupWindow":"04:00-04:30","preferredMaintenanceWindow":"Sun:05:00-Sun:05:30","applyImmediately":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "RebootRelationalDatabase", []byte(`{"relationalDatabaseName":"stage19-db"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "StopRelationalDatabase", []byte(`{"relationalDatabaseName":"stage19-db","relationalDatabaseSnapshotName":"stage19-stop-snap"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "StartRelationalDatabase", []byte(`{"relationalDatabaseName":"stage19-db"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteRelationalDatabase", []byte(`{"relationalDatabaseName":"stage19-db","skipFinalSnapshot":true}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage19SDKClientRelationalDatabaseCore(t *testing.T) {
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

	createOut, err := client.CreateRelationalDatabase(ctx, &awslightsail.CreateRelationalDatabaseInput{
		RelationalDatabaseName:        aws.String("sdk-stage19-db"),
		AvailabilityZone:              aws.String("us-east-1a"),
		MasterDatabaseName:            aws.String("appdb"),
		MasterUsername:                aws.String("admin"),
		MasterUserPassword:            aws.String("Stage19pass!"),
		RelationalDatabaseBlueprintId: aws.String("mysql_8_0"),
		RelationalDatabaseBundleId:    aws.String("micro_1_0"),
	})
	if err != nil {
		t.Fatalf("create relational database: %v", err)
	}
	if len(createOut.Operations) != 1 {
		t.Fatalf("expected create operations")
	}

	getOut, err := client.GetRelationalDatabase(ctx, &awslightsail.GetRelationalDatabaseInput{
		RelationalDatabaseName: aws.String("sdk-stage19-db"),
	})
	if err != nil {
		t.Fatalf("get relational database: %v", err)
	}
	if getOut.RelationalDatabase == nil || getOut.RelationalDatabase.Name == nil || *getOut.RelationalDatabase.Name != "sdk-stage19-db" {
		t.Fatalf("unexpected GetRelationalDatabase output: %+v", getOut.RelationalDatabase)
	}

	listOut, err := client.GetRelationalDatabases(ctx, &awslightsail.GetRelationalDatabasesInput{})
	if err != nil {
		t.Fatalf("get relational databases: %v", err)
	}
	if len(listOut.RelationalDatabases) == 0 {
		t.Fatalf("expected relational databases")
	}

	updateOut, err := client.UpdateRelationalDatabase(ctx, &awslightsail.UpdateRelationalDatabaseInput{
		RelationalDatabaseName: aws.String("sdk-stage19-db"),
		ApplyImmediately:       aws.Bool(true),
		PubliclyAccessible:     aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("update relational database: %v", err)
	}
	if len(updateOut.Operations) != 1 {
		t.Fatalf("expected update operations")
	}

	rebootOut, err := client.RebootRelationalDatabase(ctx, &awslightsail.RebootRelationalDatabaseInput{
		RelationalDatabaseName: aws.String("sdk-stage19-db"),
	})
	if err != nil {
		t.Fatalf("reboot relational database: %v", err)
	}
	if len(rebootOut.Operations) != 1 {
		t.Fatalf("expected reboot operations")
	}

	stopOut, err := client.StopRelationalDatabase(ctx, &awslightsail.StopRelationalDatabaseInput{
		RelationalDatabaseName:         aws.String("sdk-stage19-db"),
		RelationalDatabaseSnapshotName: aws.String("sdk-stage19-stop-snap"),
	})
	if err != nil {
		t.Fatalf("stop relational database: %v", err)
	}
	if len(stopOut.Operations) != 1 {
		t.Fatalf("expected stop operations")
	}

	startOut, err := client.StartRelationalDatabase(ctx, &awslightsail.StartRelationalDatabaseInput{
		RelationalDatabaseName: aws.String("sdk-stage19-db"),
	})
	if err != nil {
		t.Fatalf("start relational database: %v", err)
	}
	if len(startOut.Operations) != 1 {
		t.Fatalf("expected start operations")
	}

	deleteOut, err := client.DeleteRelationalDatabase(ctx, &awslightsail.DeleteRelationalDatabaseInput{
		RelationalDatabaseName: aws.String("sdk-stage19-db"),
		SkipFinalSnapshot:      aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("delete relational database: %v", err)
	}
	if len(deleteOut.Operations) != 1 {
		t.Fatalf("expected delete operations")
	}
}
