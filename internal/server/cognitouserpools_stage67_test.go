package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCognitoUserPoolsStage6IdentityProviderRiskLogAndAuthEventFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-stage67-stage6-pool", "stackyard-stage67-stage6-client")
	cognitoUserPoolsMustSignUpAndConfirm(t, ts, clientID, "stage67-user", "Pass#123456")

	badAuthResp := cognitoUserPoolsRequestPayload(t, ts, "InitiateAuth", map[string]any{
		"ClientId": clientID,
		"AuthFlow": "USER_PASSWORD_AUTH",
		"AuthParameters": map[string]any{
			"USERNAME": "stage67-user",
			"PASSWORD": "Pass#WrongPassword",
		},
	})
	assertStatus(t, badAuthResp, http.StatusBadRequest)

	authBody := cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage67-user", "Pass#123456")
	authResultObj, _ := authBody["AuthenticationResult"].(map[string]any)
	accessToken, _ := authResultObj["AccessToken"].(string)
	if strings.TrimSpace(accessToken) == "" {
		t.Fatalf("expected AccessToken in auth result")
	}

	createIDPResp := cognitoUserPoolsRequestPayload(t, ts, "CreateIdentityProvider", map[string]any{
		"UserPoolId":       userPoolID,
		"ProviderName":     "google-oauth",
		"ProviderType":     "Google",
		"ProviderDetails":  map[string]any{"client_id": "google-client-id", "client_secret": "google-client-secret"},
		"AttributeMapping": map[string]any{"email": "email"},
		"IdpIdentifiers":   []string{"accounts.google.com"},
	})
	assertStatus(t, createIDPResp, http.StatusOK)

	describeIDPResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": "google-oauth",
	})
	assertStatus(t, describeIDPResp, http.StatusOK)

	getByIdentifierResp := cognitoUserPoolsRequestPayload(t, ts, "GetIdentityProviderByIdentifier", map[string]any{
		"UserPoolId":    userPoolID,
		"IdpIdentifier": "accounts.google.com",
	})
	assertStatus(t, getByIdentifierResp, http.StatusOK)

	listIDPResp := cognitoUserPoolsRequestPayload(t, ts, "ListIdentityProviders", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 10,
	})
	assertStatus(t, listIDPResp, http.StatusOK)

	updateIDPResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": "google-oauth",
		"ProviderDetails": map[string]any{
			"authorize_scopes": "openid email profile",
		},
		"IdpIdentifiers": []string{"accounts.google.com", "google.com"},
	})
	assertStatus(t, updateIDPResp, http.StatusOK)

	setRiskResp := cognitoUserPoolsRequestPayload(t, ts, "SetRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"CompromisedCredentialsRiskConfiguration": map[string]any{
			"Actions":     map[string]any{"EventAction": "BLOCK"},
			"EventFilter": []string{"SIGN_IN"},
		},
		"AccountTakeoverRiskConfiguration": map[string]any{
			"Actions": map[string]any{
				"HighAction": map[string]any{"EventAction": "BLOCK"},
			},
		},
		"RiskExceptionConfiguration": map[string]any{
			"BlockedIPRangeList": []string{"198.51.100.0/24"},
		},
	})
	assertStatus(t, setRiskResp, http.StatusOK)

	describeRiskResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	})
	assertStatus(t, describeRiskResp, http.StatusOK)
	describeRiskBody := decodeCognitoUserPoolsBody(t, describeRiskResp)
	riskConfig, _ := describeRiskBody["RiskConfiguration"].(map[string]any)
	if got, _ := riskConfig["ClientId"].(string); got != clientID {
		t.Fatalf("expected ClientId %q in risk config, got %#v", clientID, riskConfig["ClientId"])
	}

	setLogDeliveryResp := cognitoUserPoolsRequestPayload(t, ts, "SetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"LogConfigurations": []map[string]any{
			{
				"LogLevel":    "ERROR",
				"EventSource": "userAuthEvents",
				"CloudWatchLogsConfiguration": map[string]any{
					"LogGroupArn": "arn:aws:logs:us-east-1:123456789012:log-group:/stackyard/cognito",
				},
			},
		},
	})
	assertStatus(t, setLogDeliveryResp, http.StatusOK)

	getLogDeliveryResp := cognitoUserPoolsRequestPayload(t, ts, "GetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
	})
	assertStatus(t, getLogDeliveryResp, http.StatusOK)
	getLogDeliveryBody := decodeCognitoUserPoolsBody(t, getLogDeliveryResp)
	logConfigurations, ok := getLogDeliveryBody["LogConfigurations"].([]any)
	if !ok || len(logConfigurations) == 0 {
		t.Fatalf("expected LogConfigurations in response, got %#v", getLogDeliveryBody["LogConfigurations"])
	}

	listAuthEventsResp := cognitoUserPoolsRequestPayload(t, ts, "AdminListUserAuthEvents", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "stage67-user",
		"MaxResults": 1,
	})
	assertStatus(t, listAuthEventsResp, http.StatusOK)
	listAuthEventsBody := decodeCognitoUserPoolsBody(t, listAuthEventsResp)
	authEvents, ok := listAuthEventsBody["AuthEvents"].([]any)
	if !ok || len(authEvents) == 0 {
		t.Fatalf("expected at least one auth event, got %#v", listAuthEventsBody["AuthEvents"])
	}
	firstEvent, _ := authEvents[0].(map[string]any)
	eventID, _ := firstEvent["EventId"].(string)
	if strings.TrimSpace(eventID) == "" {
		t.Fatalf("expected EventId in first auth event")
	}

	adminUpdateFeedbackResp := cognitoUserPoolsRequestPayload(t, ts, "AdminUpdateAuthEventFeedback", map[string]any{
		"UserPoolId":    userPoolID,
		"Username":      "stage67-user",
		"EventId":       eventID,
		"FeedbackValue": "Valid",
	})
	assertStatus(t, adminUpdateFeedbackResp, http.StatusOK)

	updateFeedbackResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateAuthEventFeedback", map[string]any{
		"AccessToken":   accessToken,
		"EventId":       eventID,
		"FeedbackValue": "Invalid",
	})
	assertStatus(t, updateFeedbackResp, http.StatusOK)

	deleteIDPResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": "google-oauth",
	})
	assertStatus(t, deleteIDPResp, http.StatusOK)

	describeDeletedIDPResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": "google-oauth",
	})
	assertStatus(t, describeDeletedIDPResp, http.StatusBadRequest)
	describeDeletedIDPBody := string(mustBody(t, describeDeletedIDPResp))
	if !strings.Contains(describeDeletedIDPBody, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException for deleted identity provider, got %q", describeDeletedIDPBody)
	}
}

func TestCognitoUserPoolsStage7CompatibilityHardening(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-stage67-stage7-pool", "stackyard-stage67-stage7-client")
	cognitoUserPoolsMustSignUpAndConfirm(t, ts, clientID, "stage67-hardening-user", "Pass#Hardening1")
	authBody := cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage67-hardening-user", "Pass#Hardening1")
	_ = cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage67-hardening-user", "Pass#Hardening1")
	authResultObj, _ := authBody["AuthenticationResult"].(map[string]any)
	accessToken, _ := authResultObj["AccessToken"].(string)
	if strings.TrimSpace(accessToken) == "" {
		t.Fatalf("expected AccessToken in auth response")
	}

	invalidMaxResultsResp := cognitoUserPoolsRequestPayload(t, ts, "ListIdentityProviders", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 61,
	})
	assertStatus(t, invalidMaxResultsResp, http.StatusBadRequest)
	invalidMaxResultsBody := string(mustBody(t, invalidMaxResultsResp))
	if !strings.Contains(invalidMaxResultsBody, "InvalidParameterException") {
		t.Fatalf("expected InvalidParameterException for oversized MaxResults, got %q", invalidMaxResultsBody)
	}

	for _, name := range []string{"idp-a", "idp-b"} {
		createResp := cognitoUserPoolsRequestPayload(t, ts, "CreateIdentityProvider", map[string]any{
			"UserPoolId":   userPoolID,
			"ProviderName": name,
			"ProviderType": "OIDC",
			"ProviderDetails": map[string]any{
				"oidc_issuer": "https://issuer.example.com/" + name,
			},
		})
		assertStatus(t, createResp, http.StatusOK)
	}

	page1Resp := cognitoUserPoolsRequestPayload(t, ts, "ListIdentityProviders", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 1,
	})
	assertStatus(t, page1Resp, http.StatusOK)
	page1Body := decodeCognitoUserPoolsBody(t, page1Resp)
	providersPage1, ok := page1Body["Providers"].([]any)
	if !ok || len(providersPage1) != 1 {
		t.Fatalf("expected one provider on page1, got %#v", page1Body["Providers"])
	}
	nextToken, _ := page1Body["NextToken"].(string)
	if strings.TrimSpace(nextToken) == "" {
		t.Fatalf("expected NextToken on first identity provider page")
	}

	page2Resp := cognitoUserPoolsRequestPayload(t, ts, "ListIdentityProviders", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 1,
		"NextToken":  nextToken,
	})
	assertStatus(t, page2Resp, http.StatusOK)

	badNextTokenResp := cognitoUserPoolsRequestPayload(t, ts, "ListIdentityProviders", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 1,
		"NextToken":  "bad-token",
	})
	assertStatus(t, badNextTokenResp, http.StatusBadRequest)
	badNextTokenBody := string(mustBody(t, badNextTokenResp))
	if !strings.Contains(badNextTokenBody, "InvalidParameterException") {
		t.Fatalf("expected InvalidParameterException for invalid next token, got %q", badNextTokenBody)
	}

	badAuthEventsTokenResp := cognitoUserPoolsRequestPayload(t, ts, "AdminListUserAuthEvents", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "stage67-hardening-user",
		"MaxResults": 1,
		"NextToken":  "bad-token",
	})
	assertStatus(t, badAuthEventsTokenResp, http.StatusBadRequest)
	badAuthEventsTokenBody := string(mustBody(t, badAuthEventsTokenResp))
	if !strings.Contains(badAuthEventsTokenBody, "InvalidParameterException") {
		t.Fatalf("expected InvalidParameterException for bad auth events next token, got %q", badAuthEventsTokenBody)
	}

	overLimitAuthEventsResp := cognitoUserPoolsRequestPayload(t, ts, "AdminListUserAuthEvents", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "stage67-hardening-user",
		"MaxResults": 100,
	})
	assertStatus(t, overLimitAuthEventsResp, http.StatusBadRequest)
	overLimitAuthEventsBody := string(mustBody(t, overLimitAuthEventsResp))
	if !strings.Contains(overLimitAuthEventsBody, "InvalidParameterException") {
		t.Fatalf("expected InvalidParameterException for oversized auth events max results, got %q", overLimitAuthEventsBody)
	}

	setRiskResp := cognitoUserPoolsRequestPayload(t, ts, "SetRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"CompromisedCredentialsRiskConfiguration": map[string]any{
			"Actions": map[string]any{"EventAction": "NO_ACTION"},
		},
	})
	assertStatus(t, setRiskResp, http.StatusOK)

	setRiskUpdateResp := cognitoUserPoolsRequestPayload(t, ts, "SetRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"CompromisedCredentialsRiskConfiguration": map[string]any{
			"Actions": map[string]any{"EventAction": "BLOCK"},
		},
	})
	assertStatus(t, setRiskUpdateResp, http.StatusOK)

	describeRiskResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	})
	assertStatus(t, describeRiskResp, http.StatusOK)
	describeRiskBody := decodeCognitoUserPoolsBody(t, describeRiskResp)
	riskConfig, _ := describeRiskBody["RiskConfiguration"].(map[string]any)
	compromisedConfig, _ := riskConfig["CompromisedCredentialsRiskConfiguration"].(map[string]any)
	actions, _ := compromisedConfig["Actions"].(map[string]any)
	if got, _ := actions["EventAction"].(string); got != "BLOCK" {
		t.Fatalf("expected EventAction=BLOCK after idempotent update, got %#v", actions["EventAction"])
	}

	setLogResp := cognitoUserPoolsRequestPayload(t, ts, "SetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"LogConfigurations": []map[string]any{{
			"LogLevel":    "ERROR",
			"EventSource": "userAuthEvents",
		}},
	})
	assertStatus(t, setLogResp, http.StatusOK)

	setLogUpdateResp := cognitoUserPoolsRequestPayload(t, ts, "SetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"LogConfigurations": []map[string]any{{
			"LogLevel":    "INFO",
			"EventSource": "userAuthEvents",
		}},
	})
	assertStatus(t, setLogUpdateResp, http.StatusOK)

	getLogResp := cognitoUserPoolsRequestPayload(t, ts, "GetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
	})
	assertStatus(t, getLogResp, http.StatusOK)
	getLogBody := decodeCognitoUserPoolsBody(t, getLogResp)
	logConfigs, _ := getLogBody["LogConfigurations"].([]any)
	if len(logConfigs) != 1 {
		t.Fatalf("expected exactly one log configuration after idempotent update, got %#v", getLogBody["LogConfigurations"])
	}
	logConfig, _ := logConfigs[0].(map[string]any)
	if got, _ := logConfig["LogLevel"].(string); got != "INFO" {
		t.Fatalf("expected log level to be INFO after update, got %#v", logConfig["LogLevel"])
	}

	setUserSettingsResp := cognitoUserPoolsRequestPayload(t, ts, "SetUserSettings", map[string]any{
		"AccessToken": accessToken,
		"MFAOptions": []map[string]any{
			{
				"DeliveryMedium": "SMS",
				"AttributeName":  "phone_number",
			},
		},
	})
	assertStatus(t, setUserSettingsResp, http.StatusOK)
}

func TestCognitoUserPoolsStage6ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-stage67-implemented-pool", "stackyard-stage67-implemented-client")
	cognitoUserPoolsMustSignUpAndConfirm(t, ts, clientID, "stage67-implemented-user", "Pass#Implemented1")
	authBody := cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage67-implemented-user", "Pass#Implemented1")
	authResultObj, _ := authBody["AuthenticationResult"].(map[string]any)
	accessToken, _ := authResultObj["AccessToken"].(string)
	if strings.TrimSpace(accessToken) == "" {
		t.Fatalf("expected AccessToken in auth response")
	}

	createIDPResp := cognitoUserPoolsRequestPayload(t, ts, "CreateIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": "implemented-idp",
		"ProviderType": "OIDC",
		"ProviderDetails": map[string]any{
			"oidc_issuer": "https://issuer.example.com/implemented",
		},
		"IdpIdentifiers": []string{"implemented.example.com"},
	})
	assertStatus(t, createIDPResp, http.StatusOK)

	adminListEventsResp := cognitoUserPoolsRequestPayload(t, ts, "AdminListUserAuthEvents", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "stage67-implemented-user",
		"MaxResults": 10,
	})
	assertStatus(t, adminListEventsResp, http.StatusOK)
	adminListEventsBody := decodeCognitoUserPoolsBody(t, adminListEventsResp)
	events, _ := adminListEventsBody["AuthEvents"].([]any)
	eventID := ""
	if len(events) > 0 {
		if first, ok := events[0].(map[string]any); ok {
			eventID, _ = first["EventId"].(string)
		}
	}

	if strings.TrimSpace(eventID) == "" {
		t.Fatalf("expected auth event id for feedback update tests")
	}

	actions := []struct {
		action  string
		payload map[string]any
	}{
		{action: "DescribeIdentityProvider", payload: map[string]any{"UserPoolId": userPoolID, "ProviderName": "implemented-idp"}},
		{action: "GetIdentityProviderByIdentifier", payload: map[string]any{"UserPoolId": userPoolID, "IdpIdentifier": "implemented.example.com"}},
		{action: "ListIdentityProviders", payload: map[string]any{"UserPoolId": userPoolID, "MaxResults": 10}},
		{action: "UpdateIdentityProvider", payload: map[string]any{"UserPoolId": userPoolID, "ProviderName": "implemented-idp", "ProviderDetails": map[string]any{"oidc_issuer": "https://issuer.example.com/v2"}}},
		{action: "SetRiskConfiguration", payload: map[string]any{"UserPoolId": userPoolID, "ClientId": clientID, "CompromisedCredentialsRiskConfiguration": map[string]any{"Actions": map[string]any{"EventAction": "BLOCK"}}}},
		{action: "DescribeRiskConfiguration", payload: map[string]any{"UserPoolId": userPoolID, "ClientId": clientID}},
		{action: "SetLogDeliveryConfiguration", payload: map[string]any{"UserPoolId": userPoolID, "LogConfigurations": []map[string]any{{"LogLevel": "ERROR", "EventSource": "userAuthEvents"}}}},
		{action: "GetLogDeliveryConfiguration", payload: map[string]any{"UserPoolId": userPoolID}},
		{action: "AdminUpdateAuthEventFeedback", payload: map[string]any{"UserPoolId": userPoolID, "Username": "stage67-implemented-user", "EventId": eventID, "FeedbackValue": "Valid"}},
		{action: "UpdateAuthEventFeedback", payload: map[string]any{"AccessToken": accessToken, "EventId": eventID, "FeedbackValue": "Invalid"}},
		{action: "DeleteIdentityProvider", payload: map[string]any{"UserPoolId": userPoolID, "ProviderName": "implemented-idp"}},
	}

	for _, tc := range actions {
		resp := cognitoUserPoolsRequestPayload(t, ts, tc.action, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", tc.action, resp.StatusCode, body)
		}
	}
}
