package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000)
	poolName := fmt.Sprintf("%s-%s", getenv("STACKYARD_USER_POOL_NAME", "stackyard-cognito"), suffix)
	domainName := fmt.Sprintf("stackyard-cognito-%s", suffix)
	clientName := "stackyard-client"
	resourceServerIdentifier := "https://api.stackyard.local/" + suffix
	identityProviderName := "google-" + suffix

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Cognito User Pools advanced client using %s\n", endpoint)

	createPoolOut, err := runCognitoAction(ctx, endpoint, region, creds, "CreateUserPool", map[string]any{
		"PoolName":         poolName,
		"MfaConfiguration": "OFF",
		"UserPoolTags": map[string]string{
			"env":  "advanced",
			"team": "platform",
		},
	})
	if err != nil {
		exitf("CreateUserPool failed: %v", err)
	}

	userPoolID, err := nestedString(createPoolOut, "UserPool", "Id")
	if err != nil {
		exitf("CreateUserPool response missing UserPool.Id: %v", err)
	}
	userPoolARN, err := nestedString(createPoolOut, "UserPool", "Arn")
	if err != nil {
		exitf("CreateUserPool response missing UserPool.Arn: %v", err)
	}
	logf("created user pool: %s", userPoolID)

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "UpdateUserPool", map[string]any{
		"UserPoolId":       userPoolID,
		"MfaConfiguration": "OPTIONAL",
		"UserPoolTags": map[string]string{
			"env":      "advanced",
			"workload": "identity",
		},
	}); err != nil {
		exitf("UpdateUserPool failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "CreateUserPoolDomain", map[string]any{
		"UserPoolId": userPoolID,
		"Domain":     domainName,
	}); err != nil {
		exitf("CreateUserPoolDomain failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DescribeUserPoolDomain", map[string]any{
		"Domain": domainName,
	}); err != nil {
		exitf("DescribeUserPoolDomain failed: %v", err)
	}

	createClientOut, err := runCognitoAction(ctx, endpoint, region, creds, "CreateUserPoolClient", map[string]any{
		"UserPoolId":     userPoolID,
		"ClientName":     clientName,
		"GenerateSecret": true,
		"ExplicitAuthFlows": []string{
			"ALLOW_USER_PASSWORD_AUTH",
			"ALLOW_REFRESH_TOKEN_AUTH",
		},
		"RefreshTokenValidity": 10,
	})
	if err != nil {
		exitf("CreateUserPoolClient failed: %v", err)
	}

	clientID, err := nestedString(createClientOut, "UserPoolClient", "ClientId")
	if err != nil {
		exitf("CreateUserPoolClient response missing UserPoolClient.ClientId: %v", err)
	}
	logf("created user pool client: %s", clientID)

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DescribeUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	}); err != nil {
		exitf("DescribeUserPoolClient failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "ListUserPoolClients", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 10,
	}); err != nil {
		exitf("ListUserPoolClients failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "UpdateUserPoolClient", map[string]any{
		"UserPoolId":           userPoolID,
		"ClientId":             clientID,
		"ClientName":           clientName + "-updated",
		"RefreshTokenValidity": 14,
	}); err != nil {
		exitf("UpdateUserPoolClient failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "AdminCreateUser", map[string]any{
		"UserPoolId":        userPoolID,
		"Username":          "advanced-admin-user",
		"TemporaryPassword": "Temp#Password1",
		"UserAttributes": []map[string]string{
			{"Name": "email", "Value": "advanced-admin-user@stackyard.local"},
		},
	}); err != nil {
		exitf("AdminCreateUser failed: %v", err)
	}

	adminAuthOut, err := runCognitoAction(ctx, endpoint, region, creds, "AdminInitiateAuth", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"AuthFlow":   "ADMIN_USER_PASSWORD_AUTH",
		"AuthParameters": map[string]string{
			"USERNAME": "advanced-admin-user",
			"PASSWORD": "Temp#Password1",
		},
	})
	if err != nil {
		exitf("AdminInitiateAuth failed: %v", err)
	}
	adminSession, err := nestedString(adminAuthOut, "Session")
	if err != nil {
		exitf("AdminInitiateAuth response missing Session: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "AdminRespondToAuthChallenge", map[string]any{
		"UserPoolId":    userPoolID,
		"ClientId":      clientID,
		"ChallengeName": "NEW_PASSWORD_REQUIRED",
		"Session":       adminSession,
		"ChallengeResponses": map[string]string{
			"USERNAME":     "advanced-admin-user",
			"NEW_PASSWORD": "Advanced#Password2",
		},
	}); err != nil {
		exitf("AdminRespondToAuthChallenge failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "CreateGroup", map[string]any{
		"UserPoolId":  userPoolID,
		"GroupName":   "advanced-engineering",
		"Description": "Advanced example engineering group",
	}); err != nil {
		exitf("CreateGroup failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "AdminAddUserToGroup", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "advanced-admin-user",
		"GroupName":  "advanced-engineering",
	}); err != nil {
		exitf("AdminAddUserToGroup failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "AdminListGroupsForUser", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "advanced-admin-user",
		"Limit":      10,
	}); err != nil {
		exitf("AdminListGroupsForUser failed: %v", err)
	}

	importJobOut, err := runCognitoAction(ctx, endpoint, region, creds, "CreateUserImportJob", map[string]any{
		"UserPoolId":            userPoolID,
		"JobName":               "advanced-import-job",
		"CloudWatchLogsRoleArn": "arn:aws:iam::123456789012:role/cognito-import",
	})
	if err != nil {
		exitf("CreateUserImportJob failed: %v", err)
	}
	importJobID, err := nestedString(importJobOut, "UserImportJob", "JobId")
	if err != nil {
		exitf("CreateUserImportJob response missing UserImportJob.JobId: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "StartUserImportJob", map[string]any{
		"UserPoolId": userPoolID,
		"JobId":      importJobID,
	}); err != nil {
		exitf("StartUserImportJob failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "StopUserImportJob", map[string]any{
		"UserPoolId": userPoolID,
		"JobId":      importJobID,
	}); err != nil {
		exitf("StopUserImportJob failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": "advanced-enduser",
		"Password": "Advanced#Password3",
		"UserAttributes": []map[string]string{
			{"Name": "email", "Value": "advanced-enduser@stackyard.local"},
		},
	}); err != nil {
		exitf("SignUp failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "ConfirmSignUp", map[string]any{
		"ClientId":         clientID,
		"Username":         "advanced-enduser",
		"ConfirmationCode": "",
	}); err != nil {
		exitf("ConfirmSignUp failed: %v", err)
	}

	initAuthOut, err := runCognitoAction(ctx, endpoint, region, creds, "InitiateAuth", map[string]any{
		"ClientId": clientID,
		"AuthFlow": "USER_PASSWORD_AUTH",
		"AuthParameters": map[string]string{
			"USERNAME": "advanced-enduser",
			"PASSWORD": "Advanced#Password3",
		},
	})
	if err != nil {
		exitf("InitiateAuth failed: %v", err)
	}
	accessToken, err := nestedString(initAuthOut, "AuthenticationResult", "AccessToken")
	if err != nil {
		exitf("InitiateAuth response missing AuthenticationResult.AccessToken: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "CreateIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": identityProviderName,
		"ProviderType": "Google",
		"ProviderDetails": map[string]string{
			"client_id":     "advanced-google-client-id",
			"client_secret": "advanced-google-client-secret",
		},
		"IdpIdentifiers": []string{"accounts.google.com"},
	}); err != nil {
		exitf("CreateIdentityProvider failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DescribeIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": identityProviderName,
	}); err != nil {
		exitf("DescribeIdentityProvider failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "SetRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"CompromisedCredentialsRiskConfiguration": map[string]any{
			"Actions": map[string]any{"EventAction": "BLOCK"},
		},
	}); err != nil {
		exitf("SetRiskConfiguration failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DescribeRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	}); err != nil {
		exitf("DescribeRiskConfiguration failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "SetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"LogConfigurations": []map[string]any{
			{
				"LogLevel":    "ERROR",
				"EventSource": "userAuthEvents",
			},
		},
	}); err != nil {
		exitf("SetLogDeliveryConfiguration failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "GetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
	}); err != nil {
		exitf("GetLogDeliveryConfiguration failed: %v", err)
	}

	adminAuthEventsOut, err := runCognitoAction(ctx, endpoint, region, creds, "AdminListUserAuthEvents", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "advanced-enduser",
		"MaxResults": 10,
	})
	if err != nil {
		exitf("AdminListUserAuthEvents failed: %v", err)
	}
	if authEvents, ok := adminAuthEventsOut["AuthEvents"].([]any); ok && len(authEvents) > 0 {
		if firstEvent, ok := authEvents[0].(map[string]any); ok {
			if eventID, ok := firstEvent["EventId"].(string); ok && strings.TrimSpace(eventID) != "" {
				if _, err := runCognitoAction(ctx, endpoint, region, creds, "AdminUpdateAuthEventFeedback", map[string]any{
					"UserPoolId":    userPoolID,
					"Username":      "advanced-enduser",
					"EventId":       eventID,
					"FeedbackValue": "Valid",
				}); err != nil {
					exitf("AdminUpdateAuthEventFeedback failed: %v", err)
				}
				if _, err := runCognitoAction(ctx, endpoint, region, creds, "UpdateAuthEventFeedback", map[string]any{
					"AccessToken":   accessToken,
					"EventId":       eventID,
					"FeedbackValue": "Invalid",
				}); err != nil {
					exitf("UpdateAuthEventFeedback failed: %v", err)
				}
			}
		}
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "SetUserPoolMfaConfig", map[string]any{
		"UserPoolId":       userPoolID,
		"MfaConfiguration": "OPTIONAL",
		"SoftwareTokenMfaConfiguration": map[string]bool{
			"Enabled": true,
		},
		"WebAuthnConfiguration": map[string]string{
			"RelyingPartyId":   "stackyard.local",
			"UserVerification": "required",
		},
	}); err != nil {
		exitf("SetUserPoolMfaConfig failed: %v", err)
	}

	softwareTokenOut, err := runCognitoAction(ctx, endpoint, region, creds, "AssociateSoftwareToken", map[string]any{
		"AccessToken": accessToken,
	})
	if err != nil {
		exitf("AssociateSoftwareToken failed: %v", err)
	}
	softwareTokenSession, err := nestedString(softwareTokenOut, "Session")
	if err != nil {
		exitf("AssociateSoftwareToken response missing Session: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "VerifySoftwareToken", map[string]any{
		"Session":  softwareTokenSession,
		"UserCode": "123456",
	}); err != nil {
		exitf("VerifySoftwareToken failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "SetUserMFAPreference", map[string]any{
		"AccessToken": accessToken,
		"SoftwareTokenMfaSettings": map[string]bool{
			"Enabled":      true,
			"PreferredMfa": true,
		},
	}); err != nil {
		exitf("SetUserMFAPreference failed: %v", err)
	}

	webAuthnStartOut, err := runCognitoAction(ctx, endpoint, region, creds, "StartWebAuthnRegistration", map[string]any{
		"AccessToken": accessToken,
	})
	if err != nil {
		exitf("StartWebAuthnRegistration failed: %v", err)
	}
	webAuthnSession, err := nestedString(webAuthnStartOut, "Session")
	if err != nil {
		exitf("StartWebAuthnRegistration response missing Session: %v", err)
	}

	webAuthnCompleteOut, err := runCognitoAction(ctx, endpoint, region, creds, "CompleteWebAuthnRegistration", map[string]any{
		"AccessToken":  accessToken,
		"Session":      webAuthnSession,
		"Credential":   "{\"id\":\"advanced-credential\"}",
		"FriendlyName": "Advanced credential",
	})
	if err != nil {
		exitf("CompleteWebAuthnRegistration failed: %v", err)
	}
	webAuthnCredentialID, err := nestedString(webAuthnCompleteOut, "Credential", "CredentialId")
	if err != nil {
		exitf("CompleteWebAuthnRegistration response missing Credential.CredentialId: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "GetUserAuthFactors", map[string]any{
		"AccessToken": accessToken,
	}); err != nil {
		exitf("GetUserAuthFactors failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteWebAuthnCredential", map[string]any{
		"AccessToken":  accessToken,
		"CredentialId": webAuthnCredentialID,
	}); err != nil {
		exitf("DeleteWebAuthnCredential failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "CreateResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": resourceServerIdentifier,
		"Name":       "stackyard-api",
		"Scopes": []map[string]string{
			{"ScopeName": "read", "ScopeDescription": "Read access"},
		},
	}); err != nil {
		exitf("CreateResourceServer failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "ListResourceServers", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 10,
	}); err != nil {
		exitf("ListResourceServers failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "UpdateResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": resourceServerIdentifier,
		"Name":       "stackyard-api-v2",
		"Scopes": []map[string]string{
			{"ScopeName": "read", "ScopeDescription": "Read access"},
			{"ScopeName": "write", "ScopeDescription": "Write access"},
		},
	}); err != nil {
		exitf("UpdateResourceServer failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"ResourceArn": userPoolARN,
		"Tags": map[string]string{
			"team": "platform",
			"env":  "advanced",
		},
	}); err != nil {
		exitf("TagResource failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "ListTagsForResource", map[string]any{
		"ResourceArn": userPoolARN,
	}); err != nil {
		exitf("ListTagsForResource failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "UntagResource", map[string]any{
		"ResourceArn": userPoolARN,
		"TagKeys":     []string{"team"},
	}); err != nil {
		exitf("UntagResource failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": resourceServerIdentifier,
	}); err != nil {
		exitf("DeleteResourceServer failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteIdentityProvider", map[string]any{
		"UserPoolId":   userPoolID,
		"ProviderName": identityProviderName,
	}); err != nil {
		exitf("DeleteIdentityProvider failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	}); err != nil {
		exitf("DeleteUserPoolClient failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteUserPoolDomain", map[string]any{
		"UserPoolId": userPoolID,
		"Domain":     domainName,
	}); err != nil {
		exitf("DeleteUserPoolDomain failed: %v", err)
	}

	if _, err := runCognitoAction(ctx, endpoint, region, creds, "DeleteUserPool", map[string]any{
		"UserPoolId": userPoolID,
	}); err != nil {
		exitf("DeleteUserPool failed: %v", err)
	}

	fmt.Println("Done.")
}

func runCognitoAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (map[string]any, error) {
	status, body, err := cognitoRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		return nil, err
	}
	if err := expectOK(action, status, body); err != nil {
		return nil, err
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", action, err)
	}
	return out, nil
}

func cognitoRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "cognito-idp", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func nestedString(m map[string]any, keys ...string) (string, error) {
	var cur any = m
	for _, key := range keys {
		asMap, ok := cur.(map[string]any)
		if !ok {
			return "", fmt.Errorf("missing path %s", strings.Join(keys, "."))
		}
		cur, ok = asMap[key]
		if !ok {
			return "", fmt.Errorf("missing path %s", strings.Join(keys, "."))
		}
	}
	value, ok := cur.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("invalid path %s", strings.Join(keys, "."))
	}
	return value, nil
}

func expectOK(action string, status int, body []byte) error {
	if status >= 200 && status < 300 {
		logf("%s returned %d", action, status)
		return nil
	}
	return fmt.Errorf("expected %s to return 2xx, got %d: %s", action, status, strings.TrimSpace(string(body)))
}

func hashSHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
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
