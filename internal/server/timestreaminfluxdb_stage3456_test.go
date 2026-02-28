package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTimestreamInfluxDBStage3InstanceLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := timestreamInfluxDBRequest(t, ts, "CreateDbInstance", map[string]any{
		"name":                "stage3-instance",
		"password":            "ChangeMe123!",
		"dbInstanceType":      "db.influx.medium",
		"vpcSubnetIds":        []string{"subnet-12345678"},
		"vpcSecurityGroupIds": []string{"sg-12345678"},
		"allocatedStorage":    50,
		"tags": map[string]string{
			"env": "stage3",
		},
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := decodeBodyJSON(t, createResp)
	instanceID, _ := createBody["id"].(string)
	if strings.TrimSpace(instanceID) == "" {
		t.Fatalf("expected created instance id, got %#v", createBody)
	}

	getResp := timestreamInfluxDBRequest(t, ts, "GetDbInstance", map[string]any{
		"identifier": instanceID,
	})
	assertStatus(t, getResp, http.StatusOK)
	getBody := decodeBodyJSON(t, getResp)
	if gotName, _ := getBody["name"].(string); gotName != "stage3-instance" {
		t.Fatalf("expected instance name stage3-instance, got %#v", getBody["name"])
	}

	updateResp := timestreamInfluxDBRequest(t, ts, "UpdateDbInstance", map[string]any{
		"identifier":       instanceID,
		"dbInstanceType":   "db.influx.large",
		"allocatedStorage": 60,
	})
	assertStatus(t, updateResp, http.StatusOK)
	updateBody := decodeBodyJSON(t, updateResp)
	if gotStatus, _ := updateBody["status"].(string); gotStatus != "UPDATING" {
		t.Fatalf("expected status UPDATING, got %#v", updateBody["status"])
	}

	rebootResp := timestreamInfluxDBRequest(t, ts, "RebootDbInstance", map[string]any{
		"identifier": instanceID,
	})
	assertStatus(t, rebootResp, http.StatusOK)

	deleteResp := timestreamInfluxDBRequest(t, ts, "DeleteDbInstance", map[string]any{
		"identifier": instanceID,
	})
	assertStatus(t, deleteResp, http.StatusOK)

	getAfterDeleteResp := timestreamInfluxDBRequest(t, ts, "GetDbInstance", map[string]any{
		"identifier": instanceID,
	})
	assertStatus(t, getAfterDeleteResp, http.StatusNotFound)
	if !strings.Contains(string(mustBody(t, getAfterDeleteResp)), "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException after delete")
	}
}

func TestTimestreamInfluxDBStage45ParameterGroupsAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createPGResp := timestreamInfluxDBRequest(t, ts, "CreateDbParameterGroup", map[string]any{
		"name":        "stage4-params",
		"description": "stage4 parameter group",
		"parameters": map[string]any{
			"influxDBv2": map[string]any{
				"logLevel": "info",
			},
		},
		"tags": map[string]string{
			"owner": "qa",
		},
	})
	assertStatus(t, createPGResp, http.StatusOK)
	createPGBody := decodeBodyJSON(t, createPGResp)
	pgID, _ := createPGBody["id"].(string)
	if strings.TrimSpace(pgID) == "" {
		t.Fatalf("expected parameter group id, got %#v", createPGBody)
	}

	getPGResp := timestreamInfluxDBRequest(t, ts, "GetDbParameterGroup", map[string]any{
		"identifier": pgID,
	})
	assertStatus(t, getPGResp, http.StatusOK)
	getPGBody := decodeBodyJSON(t, getPGResp)
	if gotName, _ := getPGBody["name"].(string); gotName != "stage4-params" {
		t.Fatalf("expected parameter group name stage4-params, got %#v", getPGBody["name"])
	}

	listPGResp := timestreamInfluxDBRequest(t, ts, "ListDbParameterGroups", map[string]any{
		"maxResults": 1,
	})
	assertStatus(t, listPGResp, http.StatusOK)
	listPGBody := decodeBodyJSON(t, listPGResp)
	if _, ok := listPGBody["nextToken"].(string); !ok {
		t.Fatalf("expected paginated nextToken in ListDbParameterGroups response")
	}

	createClusterResp := timestreamInfluxDBRequest(t, ts, "CreateDbCluster", map[string]any{
		"name":                "stage5-cluster",
		"dbInstanceType":      "db.influx.medium",
		"vpcSubnetIds":        []string{"subnet-11111111", "subnet-22222222"},
		"vpcSecurityGroupIds": []string{"sg-12345678"},
	})
	assertStatus(t, createClusterResp, http.StatusOK)
	clusterID, _ := decodeBodyJSON(t, createClusterResp)["dbClusterId"].(string)

	getClusterResp := timestreamInfluxDBRequest(t, ts, "GetDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	assertStatus(t, getClusterResp, http.StatusOK)
	clusterARN, _ := decodeBodyJSON(t, getClusterResp)["arn"].(string)

	tagResp := timestreamInfluxDBRequest(t, ts, "TagResource", map[string]any{
		"resourceArn": clusterARN,
		"tags": map[string]string{
			"env":  "stage5",
			"team": "platform",
		},
	})
	assertStatus(t, tagResp, http.StatusOK)

	listTagsResp := timestreamInfluxDBRequest(t, ts, "ListTagsForResource", map[string]any{
		"resourceArn": clusterARN,
	})
	assertStatus(t, listTagsResp, http.StatusOK)
	listTagsBody := decodeBodyJSON(t, listTagsResp)
	tagsRaw, ok := listTagsBody["tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected tags object, got %#v", listTagsBody["tags"])
	}
	if got, _ := tagsRaw["env"].(string); got != "stage5" {
		t.Fatalf("expected env=stage5 tag, got %#v", tagsRaw["env"])
	}

	untagResp := timestreamInfluxDBRequest(t, ts, "UntagResource", map[string]any{
		"resourceArn": clusterARN,
		"tagKeys":     []string{"team"},
	})
	assertStatus(t, untagResp, http.StatusOK)

	listTagsAfterResp := timestreamInfluxDBRequest(t, ts, "ListTagsForResource", map[string]any{
		"resourceArn": clusterARN,
	})
	assertStatus(t, listTagsAfterResp, http.StatusOK)
	tagsAfter, _ := decodeBodyJSON(t, listTagsAfterResp)["tags"].(map[string]any)
	if _, exists := tagsAfter["team"]; exists {
		t.Fatalf("expected team tag to be removed")
	}
}

func TestTimestreamInfluxDBStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createClusterPayload := map[string]any{
		"name":                "stage6-cluster",
		"dbInstanceType":      "db.influx.medium",
		"vpcSubnetIds":        []string{"subnet-12345678"},
		"vpcSecurityGroupIds": []string{"sg-12345678"},
	}
	createClusterResp := timestreamInfluxDBRequest(t, ts, "CreateDbCluster", createClusterPayload)
	assertStatus(t, createClusterResp, http.StatusOK)

	duplicateClusterResp := timestreamInfluxDBRequest(t, ts, "CreateDbCluster", createClusterPayload)
	assertStatus(t, duplicateClusterResp, http.StatusConflict)
	if !strings.Contains(string(mustBody(t, duplicateClusterResp)), "ConflictException") {
		t.Fatalf("expected ConflictException for duplicate cluster create")
	}

	invalidTokenResp := timestreamInfluxDBRequest(t, ts, "ListDbClusters", map[string]any{
		"nextToken": "bad-token",
	})
	assertStatus(t, invalidTokenResp, http.StatusBadRequest)

	invalidPageResp := timestreamInfluxDBRequest(t, ts, "ListDbClusters", map[string]any{
		"maxResults": 101,
	})
	assertStatus(t, invalidPageResp, http.StatusBadRequest)

	invalidInstanceResp := timestreamInfluxDBRequest(t, ts, "CreateDbInstance", map[string]any{
		"name":                "stage6-instance",
		"password":            "ChangeMe123!",
		"dbInstanceType":      "db.influx.medium",
		"vpcSubnetIds":        []string{"subnet-12345678"},
		"vpcSecurityGroupIds": []string{"sg-12345678"},
		"allocatedStorage":    5,
	})
	assertStatus(t, invalidInstanceResp, http.StatusBadRequest)

	invalidTagResp := timestreamInfluxDBRequest(t, ts, "TagResource", map[string]any{
		"resourceArn": "not-an-arn",
		"tags": map[string]string{
			"env": "stage6",
		},
	})
	assertStatus(t, invalidTagResp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, invalidTagResp)), "ValidationException") {
		t.Fatalf("expected ValidationException for invalid resource ARN")
	}
}
