package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cognitoIdentityMustCreatePool(t *testing.T, ts *httptest.Server, name string, extras map[string]any) string {
	t.Helper()
	payload := map[string]any{
		"IdentityPoolName":               name,
		"AllowUnauthenticatedIdentities": true,
	}
	for key, value := range extras {
		payload[key] = value
	}
	resp := cognitoIdentityRequestPayload(t, ts, "CreateIdentityPool", payload)
	assertStatus(t, resp, http.StatusOK)
	body := decodeCognitoIdentityBody(t, resp)
	identityPoolID, _ := body["IdentityPoolId"].(string)
	if strings.TrimSpace(identityPoolID) == "" {
		t.Fatalf("expected IdentityPoolId in create response: %#v", body)
	}
	return identityPoolID
}

func cognitoIdentityPoolARN(identityPoolID string) string {
	return "arn:aws:cognito-identity:us-east-1:123456789012:identitypool/" + identityPoolID
}

func TestCognitoIdentityStage34DeveloperAndRolePrincipalMappings(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	poolID := cognitoIdentityMustCreatePool(t, ts, "stackyard-stage34", map[string]any{
		"DeveloperProviderName": "login.stackyard.example",
	})

	devDestResp := cognitoIdentityRequestPayload(t, ts, "GetOpenIdTokenForDeveloperIdentity", map[string]any{
		"IdentityPoolId": poolID,
		"Logins": map[string]any{
			"login.stackyard.example": "dest-user",
		},
	})
	assertStatus(t, devDestResp, http.StatusOK)
	devDestBody := decodeCognitoIdentityBody(t, devDestResp)
	destIdentityID, _ := devDestBody["IdentityId"].(string)
	if strings.TrimSpace(destIdentityID) == "" {
		t.Fatalf("expected identity id in GetOpenIdTokenForDeveloperIdentity response: %#v", devDestBody)
	}

	devSourceResp := cognitoIdentityRequestPayload(t, ts, "GetOpenIdTokenForDeveloperIdentity", map[string]any{
		"IdentityPoolId": poolID,
		"Logins": map[string]any{
			"login.stackyard.example": "source-user",
		},
	})
	assertStatus(t, devSourceResp, http.StatusOK)
	devSourceBody := decodeCognitoIdentityBody(t, devSourceResp)
	sourceIdentityID, _ := devSourceBody["IdentityId"].(string)
	if strings.TrimSpace(sourceIdentityID) == "" || sourceIdentityID == destIdentityID {
		t.Fatalf("expected distinct source identity id in response: %#v", devSourceBody)
	}

	lookupDestResp := cognitoIdentityRequestPayload(t, ts, "LookupDeveloperIdentity", map[string]any{
		"IdentityPoolId":          poolID,
		"DeveloperUserIdentifier": "dest-user",
	})
	assertStatus(t, lookupDestResp, http.StatusOK)
	lookupDestBody := decodeCognitoIdentityBody(t, lookupDestResp)
	if got, _ := lookupDestBody["IdentityId"].(string); got != destIdentityID {
		t.Fatalf("expected lookup destination identity %q, got %#v", destIdentityID, lookupDestBody["IdentityId"])
	}

	mergeResp := cognitoIdentityRequestPayload(t, ts, "MergeDeveloperIdentities", map[string]any{
		"IdentityPoolId":                    poolID,
		"DeveloperProviderName":             "login.stackyard.example",
		"DestinationUserIdentifierForMerge": "dest-user",
		"SourceUserIdentifier":              "source-user",
	})
	assertStatus(t, mergeResp, http.StatusOK)
	mergeBody := decodeCognitoIdentityBody(t, mergeResp)
	if got, _ := mergeBody["IdentityId"].(string); got != destIdentityID {
		t.Fatalf("expected merge identity %q, got %#v", destIdentityID, mergeBody["IdentityId"])
	}

	lookupSourceResp := cognitoIdentityRequestPayload(t, ts, "LookupDeveloperIdentity", map[string]any{
		"IdentityPoolId":          poolID,
		"DeveloperUserIdentifier": "source-user",
	})
	assertStatus(t, lookupSourceResp, http.StatusOK)
	lookupSourceBody := decodeCognitoIdentityBody(t, lookupSourceResp)
	if got, _ := lookupSourceBody["IdentityId"].(string); got != destIdentityID {
		t.Fatalf("expected merged source lookup identity %q, got %#v", destIdentityID, lookupSourceBody["IdentityId"])
	}

	unlinkDeveloperResp := cognitoIdentityRequestPayload(t, ts, "UnlinkDeveloperIdentity", map[string]any{
		"IdentityPoolId":          poolID,
		"IdentityId":              destIdentityID,
		"DeveloperProviderName":   "login.stackyard.example",
		"DeveloperUserIdentifier": "source-user",
	})
	assertStatus(t, unlinkDeveloperResp, http.StatusOK)

	lookupAfterUnlinkResp := cognitoIdentityRequestPayload(t, ts, "LookupDeveloperIdentity", map[string]any{
		"IdentityPoolId":          poolID,
		"DeveloperUserIdentifier": "source-user",
	})
	assertStatus(t, lookupAfterUnlinkResp, http.StatusBadRequest)
	lookupAfterUnlinkBody := string(mustBody(t, lookupAfterUnlinkResp))
	if !strings.Contains(lookupAfterUnlinkBody, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException after unlink developer identity, got %q", lookupAfterUnlinkBody)
	}

	setRolesResp := cognitoIdentityRequestPayload(t, ts, "SetIdentityPoolRoles", map[string]any{
		"IdentityPoolId": poolID,
		"Roles": map[string]any{
			"authenticated": "arn:aws:iam::123456789012:role/authenticated",
		},
		"RoleMappings": map[string]any{
			"login.stackyard.example": map[string]any{
				"Type":                    "Rules",
				"AmbiguousRoleResolution": "AuthenticatedRole",
				"RulesConfiguration": map[string]any{
					"Rules": []map[string]any{
						{
							"Claim":     "isAdmin",
							"MatchType": "Equals",
							"Value":     "true",
							"RoleARN":   "arn:aws:iam::123456789012:role/admin",
						},
					},
				},
			},
		},
	})
	assertStatus(t, setRolesResp, http.StatusOK)

	getRolesResp := cognitoIdentityRequestPayload(t, ts, "GetIdentityPoolRoles", map[string]any{
		"IdentityPoolId": poolID,
	})
	assertStatus(t, getRolesResp, http.StatusOK)
	getRolesBody := decodeCognitoIdentityBody(t, getRolesResp)
	roles, ok := getRolesBody["Roles"].(map[string]any)
	if !ok || len(roles) == 0 {
		t.Fatalf("expected Roles map in get response, got %#v", getRolesBody["Roles"])
	}
	if value, _ := roles["authenticated"].(string); value == "" {
		t.Fatalf("expected authenticated role in response")
	}
	if _, ok := getRolesBody["RoleMappings"].(map[string]any); !ok {
		t.Fatalf("expected RoleMappings in response, got %#v", getRolesBody["RoleMappings"])
	}

	setPrincipalResp := cognitoIdentityRequestPayload(t, ts, "SetPrincipalTagAttributeMap", map[string]any{
		"IdentityPoolId":       poolID,
		"IdentityProviderName": "login.stackyard.example",
		"UseDefaults":          true,
		"PrincipalTags": map[string]any{
			"team": "engineering",
		},
	})
	assertStatus(t, setPrincipalResp, http.StatusOK)

	getPrincipalResp := cognitoIdentityRequestPayload(t, ts, "GetPrincipalTagAttributeMap", map[string]any{
		"IdentityPoolId":       poolID,
		"IdentityProviderName": "login.stackyard.example",
	})
	assertStatus(t, getPrincipalResp, http.StatusOK)
	getPrincipalBody := decodeCognitoIdentityBody(t, getPrincipalResp)
	if useDefaults, _ := getPrincipalBody["UseDefaults"].(bool); !useDefaults {
		t.Fatalf("expected UseDefaults=true, got %#v", getPrincipalBody["UseDefaults"])
	}
	principalTags, ok := getPrincipalBody["PrincipalTags"].(map[string]any)
	if !ok || principalTags["team"] != "engineering" {
		t.Fatalf("expected principal tag team=engineering, got %#v", getPrincipalBody["PrincipalTags"])
	}
}

func TestCognitoIdentityStage5UnlinkIdentityAndResourceTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	poolID := cognitoIdentityMustCreatePool(t, ts, "stackyard-stage5", nil)
	resourceARN := cognitoIdentityPoolARN(poolID)

	getIDResp := cognitoIdentityRequestPayload(t, ts, "GetId", map[string]any{
		"IdentityPoolId": poolID,
		"Logins": map[string]any{
			"accounts.google.com": "google-token",
		},
	})
	assertStatus(t, getIDResp, http.StatusOK)
	getIDBody := decodeCognitoIdentityBody(t, getIDResp)
	identityID, _ := getIDBody["IdentityId"].(string)
	if strings.TrimSpace(identityID) == "" {
		t.Fatalf("expected IdentityId in GetId response: %#v", getIDBody)
	}

	tagResp := cognitoIdentityRequestPayload(t, ts, "TagResource", map[string]any{
		"ResourceArn": resourceARN,
		"Tags": map[string]any{
			"env":  "test",
			"team": "platform",
		},
	})
	assertStatus(t, tagResp, http.StatusOK)

	listTagsResp := cognitoIdentityRequestPayload(t, ts, "ListTagsForResource", map[string]any{
		"ResourceArn": resourceARN,
	})
	assertStatus(t, listTagsResp, http.StatusOK)
	listTagsBody := decodeCognitoIdentityBody(t, listTagsResp)
	tags, ok := listTagsBody["Tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected Tags map, got %#v", listTagsBody["Tags"])
	}
	if tags["env"] != "test" || tags["team"] != "platform" {
		t.Fatalf("expected env/test and team/platform tags, got %#v", tags)
	}

	untagResp := cognitoIdentityRequestPayload(t, ts, "UntagResource", map[string]any{
		"ResourceArn": resourceARN,
		"TagKeys":     []string{"team"},
	})
	assertStatus(t, untagResp, http.StatusOK)

	listAfterUntagResp := cognitoIdentityRequestPayload(t, ts, "ListTagsForResource", map[string]any{
		"ResourceArn": resourceARN,
	})
	assertStatus(t, listAfterUntagResp, http.StatusOK)
	listAfterUntagBody := decodeCognitoIdentityBody(t, listAfterUntagResp)
	tagsAfter, ok := listAfterUntagBody["Tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected Tags map after untag, got %#v", listAfterUntagBody["Tags"])
	}
	if _, exists := tagsAfter["team"]; exists {
		t.Fatalf("expected team tag removed, got %#v", tagsAfter)
	}
	if tagsAfter["env"] != "test" {
		t.Fatalf("expected env tag to remain, got %#v", tagsAfter)
	}

	invalidARNResp := cognitoIdentityRequestPayload(t, ts, "TagResource", map[string]any{
		"ResourceArn": "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_invalid",
		"Tags": map[string]any{
			"env": "test",
		},
	})
	assertStatus(t, invalidARNResp, http.StatusBadRequest)
	invalidARNBody := string(mustBody(t, invalidARNResp))
	if !strings.Contains(invalidARNBody, "InvalidParameterException") {
		t.Fatalf("expected InvalidParameterException for invalid resource arn, got %q", invalidARNBody)
	}

	unlinkResp := cognitoIdentityRequestPayload(t, ts, "UnlinkIdentity", map[string]any{
		"IdentityId": identityID,
		"Logins": map[string]any{
			"accounts.google.com": "google-token",
		},
		"LoginsToRemove": []string{"accounts.google.com"},
	})
	assertStatus(t, unlinkResp, http.StatusOK)

	describeResp := cognitoIdentityRequestPayload(t, ts, "DescribeIdentity", map[string]any{
		"IdentityId": identityID,
	})
	assertStatus(t, describeResp, http.StatusOK)
	describeBody := decodeCognitoIdentityBody(t, describeResp)
	if _, ok := describeBody["Logins"]; ok {
		t.Fatalf("expected no logins after unlink, got %#v", describeBody["Logins"])
	}
}

func TestCognitoIdentityStage6PaginationAndDeleteCompat(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	poolID := cognitoIdentityMustCreatePool(t, ts, "stackyard-stage6", map[string]any{
		"DeveloperProviderName": "login.stackyard.stage6",
	})

	firstResp := cognitoIdentityRequestPayload(t, ts, "GetOpenIdTokenForDeveloperIdentity", map[string]any{
		"IdentityPoolId": poolID,
		"Logins": map[string]any{
			"login.stackyard.stage6": "user-a",
		},
	})
	assertStatus(t, firstResp, http.StatusOK)
	firstBody := decodeCognitoIdentityBody(t, firstResp)
	identityID, _ := firstBody["IdentityId"].(string)
	if strings.TrimSpace(identityID) == "" {
		t.Fatalf("expected identity id in first developer token response")
	}

	for _, developerUser := range []string{"user-b", "user-c"} {
		resp := cognitoIdentityRequestPayload(t, ts, "GetOpenIdTokenForDeveloperIdentity", map[string]any{
			"IdentityPoolId": poolID,
			"IdentityId":     identityID,
			"Logins": map[string]any{
				"login.stackyard.stage6": developerUser,
			},
		})
		assertStatus(t, resp, http.StatusOK)
	}

	lookupPage1Resp := cognitoIdentityRequestPayload(t, ts, "LookupDeveloperIdentity", map[string]any{
		"IdentityPoolId": poolID,
		"IdentityId":     identityID,
		"MaxResults":     1,
	})
	assertStatus(t, lookupPage1Resp, http.StatusOK)
	lookupPage1Body := decodeCognitoIdentityBody(t, lookupPage1Resp)
	page1Users, ok := lookupPage1Body["DeveloperUserIdentifierList"].([]any)
	if !ok || len(page1Users) != 1 {
		t.Fatalf("expected one developer user on first lookup page, got %#v", lookupPage1Body["DeveloperUserIdentifierList"])
	}
	nextToken, _ := lookupPage1Body["NextToken"].(string)
	if strings.TrimSpace(nextToken) == "" {
		t.Fatalf("expected next token for first lookup page")
	}

	lookupPage2Resp := cognitoIdentityRequestPayload(t, ts, "LookupDeveloperIdentity", map[string]any{
		"IdentityPoolId": poolID,
		"IdentityId":     identityID,
		"MaxResults":     1,
		"NextToken":      nextToken,
	})
	assertStatus(t, lookupPage2Resp, http.StatusOK)

	invalidNextTokenResp := cognitoIdentityRequestPayload(t, ts, "LookupDeveloperIdentity", map[string]any{
		"IdentityPoolId": poolID,
		"IdentityId":     identityID,
		"MaxResults":     1,
		"NextToken":      "bad-token",
	})
	assertStatus(t, invalidNextTokenResp, http.StatusBadRequest)
	invalidNextTokenBody := string(mustBody(t, invalidNextTokenResp))
	if !strings.Contains(invalidNextTokenBody, "InvalidParameterException") {
		t.Fatalf("expected invalid parameter error for bad next token, got %q", invalidNextTokenBody)
	}

	listPoolsTooLargeResp := cognitoIdentityRequestPayload(t, ts, "ListIdentityPools", map[string]any{
		"MaxResults": 61,
	})
	assertStatus(t, listPoolsTooLargeResp, http.StatusBadRequest)
	listPoolsTooLargeBody := string(mustBody(t, listPoolsTooLargeResp))
	if !strings.Contains(listPoolsTooLargeBody, "InvalidParameterException") {
		t.Fatalf("expected invalid parameter for oversized MaxResults, got %q", listPoolsTooLargeBody)
	}

	listIdentitiesBadTokenResp := cognitoIdentityRequestPayload(t, ts, "ListIdentities", map[string]any{
		"IdentityPoolId": poolID,
		"MaxResults":     1,
		"NextToken":      "bad-token",
	})
	assertStatus(t, listIdentitiesBadTokenResp, http.StatusBadRequest)
	listIdentitiesBadTokenBody := string(mustBody(t, listIdentitiesBadTokenResp))
	if !strings.Contains(listIdentitiesBadTokenBody, "InvalidParameterException") {
		t.Fatalf("expected invalid parameter for bad list identities token, got %q", listIdentitiesBadTokenBody)
	}

	deleteIdentitiesResp := cognitoIdentityRequestPayload(t, ts, "DeleteIdentities", map[string]any{
		"IdentityIdsToDelete": []string{identityID, fmt.Sprintf("%s:missing", strings.Split(identityID, ":")[0])},
	})
	assertStatus(t, deleteIdentitiesResp, http.StatusOK)
	deleteIdentitiesBody := decodeCognitoIdentityBody(t, deleteIdentitiesResp)
	unprocessed, ok := deleteIdentitiesBody["UnprocessedIdentityIds"].([]any)
	if !ok || len(unprocessed) != 1 {
		t.Fatalf("expected one unprocessed identity after delete, got %#v", deleteIdentitiesBody["UnprocessedIdentityIds"])
	}
}
