package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage21RelationalConfigCatalog(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateRelationalDatabase", []byte(`{"relationalDatabaseName":"stage21-db","availabilityZone":"us-east-1a","masterDatabaseName":"appdb","masterUsername":"admin","masterUserPassword":"Stage21pass!","relationalDatabaseBlueprintId":"mysql_8_0","relationalDatabaseBundleId":"micro_1_0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseBlueprints", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var blueprintsOut struct {
		Blueprints []struct {
			BlueprintID string `json:"blueprintId"`
		} `json:"blueprints"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &blueprintsOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseBlueprints: %v", err)
	}
	if len(blueprintsOut.Blueprints) == 0 {
		t.Fatalf("expected relational database blueprints")
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseBundles", []byte(`{"includeInactive":true}`))
	assertStatus(t, resp, http.StatusOK)
	var bundlesOut struct {
		Bundles []struct {
			BundleID string `json:"bundleId"`
		} `json:"bundles"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &bundlesOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseBundles: %v", err)
	}
	if len(bundlesOut.Bundles) == 0 {
		t.Fatalf("expected relational database bundles")
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseMasterUserPassword", []byte(`{"relationalDatabaseName":"stage21-db","passwordVersion":"CURRENT"}`))
	assertStatus(t, resp, http.StatusOK)
	var passwordOut struct {
		MasterUserPassword string `json:"masterUserPassword"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &passwordOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseMasterUserPassword: %v", err)
	}
	if passwordOut.MasterUserPassword == "" {
		t.Fatalf("expected master user password")
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseMetricData", []byte(`{"relationalDatabaseName":"stage21-db","metricName":"CPUUtilization","startTime":1700000000,"endTime":1700000600,"period":60,"statistics":["Average"],"unit":"Percent"}`))
	assertStatus(t, resp, http.StatusOK)
	var metricOut struct {
		MetricName string `json:"metricName"`
		MetricData []struct {
			Timestamp float64 `json:"timestamp"`
		} `json:"metricData"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &metricOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseMetricData: %v", err)
	}
	if metricOut.MetricName != "CPUUtilization" || len(metricOut.MetricData) == 0 {
		t.Fatalf("unexpected metric output: %+v", metricOut)
	}

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseParameters", []byte(`{"relationalDatabaseName":"stage21-db"}`))
	assertStatus(t, resp, http.StatusOK)
	var parametersOut struct {
		Parameters []struct {
			ParameterName string `json:"parameterName"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &parametersOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseParameters: %v", err)
	}
	if len(parametersOut.Parameters) == 0 {
		t.Fatalf("expected relational database parameters")
	}

	resp = lightsailRequest(t, ts, "UpdateRelationalDatabaseParameters", []byte(`{"relationalDatabaseName":"stage21-db","parameters":[{"parameterName":"max_connections","parameterValue":"200","applyMethod":"pending-reboot"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetRelationalDatabaseParameters", []byte(`{"relationalDatabaseName":"stage21-db"}`))
	assertStatus(t, resp, http.StatusOK)
	var parametersUpdatedOut struct {
		Parameters []struct {
			ParameterName  string `json:"parameterName"`
			ParameterValue string `json:"parameterValue"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &parametersUpdatedOut); err != nil {
		t.Fatalf("unmarshal GetRelationalDatabaseParameters updated: %v", err)
	}
	found := false
	for _, parameter := range parametersUpdatedOut.Parameters {
		if parameter.ParameterName == "max_connections" {
			found = true
			if parameter.ParameterValue != "200" {
				t.Fatalf("expected updated max_connections value")
			}
		}
	}
	if !found {
		t.Fatalf("expected updated max_connections parameter")
	}
}

func TestLightsailStage21SDKClientRelationalConfigCatalog(t *testing.T) {
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
		RelationalDatabaseName:        aws.String("sdk-stage21-db"),
		AvailabilityZone:              aws.String("us-east-1a"),
		MasterDatabaseName:            aws.String("appdb"),
		MasterUsername:                aws.String("admin"),
		MasterUserPassword:            aws.String("Stage21pass!"),
		RelationalDatabaseBlueprintId: aws.String("mysql_8_0"),
		RelationalDatabaseBundleId:    aws.String("micro_1_0"),
	}); err != nil {
		t.Fatalf("create relational database: %v", err)
	}

	blueprintsOut, err := client.GetRelationalDatabaseBlueprints(ctx, &awslightsail.GetRelationalDatabaseBlueprintsInput{})
	if err != nil {
		t.Fatalf("get relational database blueprints: %v", err)
	}
	if len(blueprintsOut.Blueprints) == 0 {
		t.Fatalf("expected relational database blueprints")
	}

	bundlesOut, err := client.GetRelationalDatabaseBundles(ctx, &awslightsail.GetRelationalDatabaseBundlesInput{
		IncludeInactive: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("get relational database bundles: %v", err)
	}
	if len(bundlesOut.Bundles) == 0 {
		t.Fatalf("expected relational database bundles")
	}

	passwordOut, err := client.GetRelationalDatabaseMasterUserPassword(ctx, &awslightsail.GetRelationalDatabaseMasterUserPasswordInput{
		RelationalDatabaseName: aws.String("sdk-stage21-db"),
		PasswordVersion:        awslightsailtypes.RelationalDatabasePasswordVersionCurrent,
	})
	if err != nil {
		t.Fatalf("get relational database master user password: %v", err)
	}
	if passwordOut.MasterUserPassword == nil || *passwordOut.MasterUserPassword == "" {
		t.Fatalf("expected master user password")
	}

	metricOut, err := client.GetRelationalDatabaseMetricData(ctx, &awslightsail.GetRelationalDatabaseMetricDataInput{
		RelationalDatabaseName: aws.String("sdk-stage21-db"),
		MetricName:             awslightsailtypes.RelationalDatabaseMetricNameCPUUtilization,
		StartTime:              aws.Time(time.Now().UTC().Add(-5 * time.Minute)),
		EndTime:                aws.Time(time.Now().UTC()),
		Period:                 aws.Int32(60),
		Statistics:             []awslightsailtypes.MetricStatistic{awslightsailtypes.MetricStatisticAverage},
		Unit:                   awslightsailtypes.MetricUnitPercent,
	})
	if err != nil {
		t.Fatalf("get relational database metric data: %v", err)
	}
	if metricOut.MetricName != awslightsailtypes.RelationalDatabaseMetricNameCPUUtilization || len(metricOut.MetricData) == 0 {
		t.Fatalf("unexpected metric output")
	}

	parametersOut, err := client.GetRelationalDatabaseParameters(ctx, &awslightsail.GetRelationalDatabaseParametersInput{
		RelationalDatabaseName: aws.String("sdk-stage21-db"),
	})
	if err != nil {
		t.Fatalf("get relational database parameters: %v", err)
	}
	if len(parametersOut.Parameters) == 0 {
		t.Fatalf("expected relational database parameters")
	}

	updateOut, err := client.UpdateRelationalDatabaseParameters(ctx, &awslightsail.UpdateRelationalDatabaseParametersInput{
		RelationalDatabaseName: aws.String("sdk-stage21-db"),
		Parameters: []awslightsailtypes.RelationalDatabaseParameter{
			{
				ParameterName:  aws.String("max_connections"),
				ParameterValue: aws.String("200"),
				ApplyMethod:    aws.String("pending-reboot"),
			},
		},
	})
	if err != nil {
		t.Fatalf("update relational database parameters: %v", err)
	}
	if len(updateOut.Operations) != 1 {
		t.Fatalf("expected one update operation")
	}
}
