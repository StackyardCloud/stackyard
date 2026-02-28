package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cognitoUserPoolsMustCreatePoolAndClient(t *testing.T, ts *httptest.Server, poolName, clientName string) (string, string) {
	t.Helper()
	createPoolResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPool", map[string]any{"PoolName": poolName})
	assertStatus(t, createPoolResp, http.StatusOK)
	createPoolBody := decodeCognitoUserPoolsBody(t, createPoolResp)
	poolObj, _ := createPoolBody["UserPool"].(map[string]any)
	userPoolID, _ := poolObj["Id"].(string)
	if strings.TrimSpace(userPoolID) == "" {
		t.Fatalf("expected UserPool.Id in create response: %#v", createPoolBody)
	}

	createClientResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientName": clientName,
	})
	assertStatus(t, createClientResp, http.StatusOK)
	createClientBody := decodeCognitoUserPoolsBody(t, createClientResp)
	clientObj, _ := createClientBody["UserPoolClient"].(map[string]any)
	clientID, _ := clientObj["ClientId"].(string)
	if strings.TrimSpace(clientID) == "" {
		t.Fatalf("expected UserPoolClient.ClientId in create response: %#v", createClientBody)
	}

	return userPoolID, clientID
}

func cognitoUserPoolsMustAdminCreateUser(t *testing.T, ts *httptest.Server, userPoolID, username, tempPassword string) {
	t.Helper()
	resp := cognitoUserPoolsRequestPayload(t, ts, "AdminCreateUser", map[string]any{
		"UserPoolId":        userPoolID,
		"Username":          username,
		"TemporaryPassword": tempPassword,
		"UserAttributes": []map[string]any{
			{"Name": "email", "Value": username + "@stackyard.local"},
		},
	})
	assertStatus(t, resp, http.StatusOK)
}

func cognitoUserPoolsMustSignUpAndConfirm(t *testing.T, ts *httptest.Server, clientID, username, password string) {
	t.Helper()
	signUpResp := cognitoUserPoolsRequestPayload(t, ts, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": username,
		"Password": password,
		"UserAttributes": []map[string]any{
			{"Name": "email", "Value": username + "@stackyard.local"},
		},
	})
	assertStatus(t, signUpResp, http.StatusOK)

	confirmResp := cognitoUserPoolsRequestPayload(t, ts, "ConfirmSignUp", map[string]any{
		"ClientId":         clientID,
		"Username":         username,
		"ConfirmationCode": "",
	})
	assertStatus(t, confirmResp, http.StatusOK)
}

func cognitoUserPoolsMustInitiatePasswordAuth(t *testing.T, ts *httptest.Server, clientID, username, password string) map[string]any {
	t.Helper()
	resp := cognitoUserPoolsRequestPayload(t, ts, "InitiateAuth", map[string]any{
		"ClientId": clientID,
		"AuthFlow": "USER_PASSWORD_AUTH",
		"AuthParameters": map[string]any{
			"USERNAME": username,
			"PASSWORD": password,
		},
	})
	assertStatus(t, resp, http.StatusOK)
	return decodeCognitoUserPoolsBody(t, resp)
}

func TestCognitoUserPoolsStage3AdminUserGroupDeviceImportFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-stage345-admin-pool", "stackyard-stage345-admin-client")
	_ = clientID

	cognitoUserPoolsMustAdminCreateUser(t, ts, userPoolID, "admin-user", "Temp#123456")

	adminGetUserResp := cognitoUserPoolsRequestPayload(t, ts, "AdminGetUser", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "admin-user",
	})
	assertStatus(t, adminGetUserResp, http.StatusOK)
	adminGetUserBody := decodeCognitoUserPoolsBody(t, adminGetUserResp)
	if got, _ := adminGetUserBody["Username"].(string); got != "admin-user" {
		t.Fatalf("expected Username=admin-user, got %#v", adminGetUserBody["Username"])
	}

	adminSetPasswordResp := cognitoUserPoolsRequestPayload(t, ts, "AdminSetUserPassword", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "admin-user",
		"Password":   "Admin#Password1",
		"Permanent":  true,
	})
	assertStatus(t, adminSetPasswordResp, http.StatusOK)

	createGroupResp := cognitoUserPoolsRequestPayload(t, ts, "CreateGroup", map[string]any{
		"UserPoolId":  userPoolID,
		"GroupName":   "engineers",
		"Description": "Engineering group",
	})
	assertStatus(t, createGroupResp, http.StatusOK)

	addToGroupResp := cognitoUserPoolsRequestPayload(t, ts, "AdminAddUserToGroup", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "admin-user",
		"GroupName":  "engineers",
	})
	assertStatus(t, addToGroupResp, http.StatusOK)

	listGroupsForUserResp := cognitoUserPoolsRequestPayload(t, ts, "AdminListGroupsForUser", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "admin-user",
		"Limit":      10,
	})
	assertStatus(t, listGroupsForUserResp, http.StatusOK)
	listGroupsForUserBody := decodeCognitoUserPoolsBody(t, listGroupsForUserResp)
	groups, ok := listGroupsForUserBody["Groups"].([]any)
	if !ok || len(groups) != 1 {
		t.Fatalf("expected exactly one group for user, got %#v", listGroupsForUserBody["Groups"])
	}

	listUsersInGroupResp := cognitoUserPoolsRequestPayload(t, ts, "ListUsersInGroup", map[string]any{
		"UserPoolId": userPoolID,
		"GroupName":  "engineers",
		"Limit":      10,
	})
	assertStatus(t, listUsersInGroupResp, http.StatusOK)
	listUsersInGroupBody := decodeCognitoUserPoolsBody(t, listUsersInGroupResp)
	users, ok := listUsersInGroupBody["Users"].([]any)
	if !ok || len(users) != 1 {
		t.Fatalf("expected exactly one user in group, got %#v", listUsersInGroupBody["Users"])
	}

	removeFromGroupResp := cognitoUserPoolsRequestPayload(t, ts, "AdminRemoveUserFromGroup", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "admin-user",
		"GroupName":  "engineers",
	})
	assertStatus(t, removeFromGroupResp, http.StatusOK)

	getCSVHeaderResp := cognitoUserPoolsRequestPayload(t, ts, "GetCSVHeader", map[string]any{
		"UserPoolId": userPoolID,
	})
	assertStatus(t, getCSVHeaderResp, http.StatusOK)
	getCSVHeaderBody := decodeCognitoUserPoolsBody(t, getCSVHeaderResp)
	if header, ok := getCSVHeaderBody["CSVHeader"].([]any); !ok || len(header) == 0 {
		t.Fatalf("expected CSVHeader values, got %#v", getCSVHeaderBody["CSVHeader"])
	}

	createImportJobResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserImportJob", map[string]any{
		"UserPoolId":            userPoolID,
		"JobName":               "stage345-import-job",
		"CloudWatchLogsRoleArn": "arn:aws:iam::123456789012:role/cognito-import",
	})
	assertStatus(t, createImportJobResp, http.StatusOK)
	createImportJobBody := decodeCognitoUserPoolsBody(t, createImportJobResp)
	importJobObj, _ := createImportJobBody["UserImportJob"].(map[string]any)
	jobID, _ := importJobObj["JobId"].(string)
	if strings.TrimSpace(jobID) == "" {
		t.Fatalf("expected import JobId, got %#v", createImportJobBody)
	}

	startImportJobResp := cognitoUserPoolsRequestPayload(t, ts, "StartUserImportJob", map[string]any{
		"UserPoolId": userPoolID,
		"JobId":      jobID,
	})
	assertStatus(t, startImportJobResp, http.StatusOK)

	describeImportJobResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeUserImportJob", map[string]any{
		"UserPoolId": userPoolID,
		"JobId":      jobID,
	})
	assertStatus(t, describeImportJobResp, http.StatusOK)

	listImportJobsResp := cognitoUserPoolsRequestPayload(t, ts, "ListUserImportJobs", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 10,
	})
	assertStatus(t, listImportJobsResp, http.StatusOK)

	stopImportJobResp := cognitoUserPoolsRequestPayload(t, ts, "StopUserImportJob", map[string]any{
		"UserPoolId": userPoolID,
		"JobId":      jobID,
	})
	assertStatus(t, stopImportJobResp, http.StatusOK)

	deleteGroupResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteGroup", map[string]any{
		"UserPoolId": userPoolID,
		"GroupName":  "engineers",
	})
	assertStatus(t, deleteGroupResp, http.StatusOK)

	adminDeleteUserResp := cognitoUserPoolsRequestPayload(t, ts, "AdminDeleteUser", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "admin-user",
	})
	assertStatus(t, adminDeleteUserResp, http.StatusOK)
}

func TestCognitoUserPoolsStage4AuthAndDeviceFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-stage345-auth-pool", "stackyard-stage345-auth-client")

	cognitoUserPoolsMustSignUpAndConfirm(t, ts, clientID, "stage4-user", "Pass#123456")

	initAuthBody := cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage4-user", "Pass#123456")
	authResultObj, _ := initAuthBody["AuthenticationResult"].(map[string]any)
	accessToken, _ := authResultObj["AccessToken"].(string)
	refreshToken, _ := authResultObj["RefreshToken"].(string)
	if strings.TrimSpace(accessToken) == "" || strings.TrimSpace(refreshToken) == "" {
		t.Fatalf("expected AccessToken and RefreshToken in auth result: %#v", authResultObj)
	}

	getUserResp := cognitoUserPoolsRequestPayload(t, ts, "GetUser", map[string]any{"AccessToken": accessToken})
	assertStatus(t, getUserResp, http.StatusOK)

	updateAttrsResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateUserAttributes", map[string]any{
		"AccessToken": accessToken,
		"UserAttributes": []map[string]any{
			{"Name": "email", "Value": "updated-stage4-user@stackyard.local"},
			{"Name": "custom:team", "Value": "identity"},
		},
	})
	assertStatus(t, updateAttrsResp, http.StatusOK)

	getAttrCodeResp := cognitoUserPoolsRequestPayload(t, ts, "GetUserAttributeVerificationCode", map[string]any{
		"AccessToken":   accessToken,
		"AttributeName": "email",
	})
	assertStatus(t, getAttrCodeResp, http.StatusOK)

	deleteAttrsResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteUserAttributes", map[string]any{
		"AccessToken":        accessToken,
		"UserAttributeNames": []string{"custom:team"},
	})
	assertStatus(t, deleteAttrsResp, http.StatusOK)

	changePasswordResp := cognitoUserPoolsRequestPayload(t, ts, "ChangePassword", map[string]any{
		"AccessToken":      accessToken,
		"PreviousPassword": "Pass#123456",
		"ProposedPassword": "Pass#654321",
	})
	assertStatus(t, changePasswordResp, http.StatusOK)

	oldPasswordAuthResp := cognitoUserPoolsRequestPayload(t, ts, "InitiateAuth", map[string]any{
		"ClientId": clientID,
		"AuthFlow": "USER_PASSWORD_AUTH",
		"AuthParameters": map[string]any{
			"USERNAME": "stage4-user",
			"PASSWORD": "Pass#123456",
		},
	})
	assertStatus(t, oldPasswordAuthResp, http.StatusBadRequest)

	newAuthBody := cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage4-user", "Pass#654321")
	newAuthResultObj, _ := newAuthBody["AuthenticationResult"].(map[string]any)
	newAccessToken, _ := newAuthResultObj["AccessToken"].(string)
	if strings.TrimSpace(newAccessToken) == "" {
		t.Fatalf("expected new AccessToken after password change")
	}

	confirmDeviceResp := cognitoUserPoolsRequestPayload(t, ts, "ConfirmDevice", map[string]any{
		"AccessToken": newAccessToken,
		"DeviceKey":   "device-stage4-1",
		"DeviceName":  "test-device",
	})
	assertStatus(t, confirmDeviceResp, http.StatusOK)

	listDevicesResp := cognitoUserPoolsRequestPayload(t, ts, "ListDevices", map[string]any{
		"AccessToken": newAccessToken,
		"Limit":       10,
	})
	assertStatus(t, listDevicesResp, http.StatusOK)
	listDevicesBody := decodeCognitoUserPoolsBody(t, listDevicesResp)
	devices, ok := listDevicesBody["Devices"].([]any)
	if !ok || len(devices) != 1 {
		t.Fatalf("expected one device, got %#v", listDevicesBody["Devices"])
	}

	getDeviceResp := cognitoUserPoolsRequestPayload(t, ts, "GetDevice", map[string]any{
		"AccessToken": newAccessToken,
		"DeviceKey":   "device-stage4-1",
	})
	assertStatus(t, getDeviceResp, http.StatusOK)

	updateDeviceStatusResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateDeviceStatus", map[string]any{
		"AccessToken":            newAccessToken,
		"DeviceKey":              "device-stage4-1",
		"DeviceRememberedStatus": "not_remembered",
	})
	assertStatus(t, updateDeviceStatusResp, http.StatusOK)

	forgetDeviceResp := cognitoUserPoolsRequestPayload(t, ts, "ForgetDevice", map[string]any{
		"AccessToken": newAccessToken,
		"DeviceKey":   "device-stage4-1",
	})
	assertStatus(t, forgetDeviceResp, http.StatusOK)

	getTokensFromRefreshResp := cognitoUserPoolsRequestPayload(t, ts, "GetTokensFromRefreshToken", map[string]any{
		"ClientId":     clientID,
		"RefreshToken": refreshToken,
	})
	assertStatus(t, getTokensFromRefreshResp, http.StatusOK)

	revokeTokenResp := cognitoUserPoolsRequestPayload(t, ts, "RevokeToken", map[string]any{
		"ClientId": clientID,
		"Token":    refreshToken,
	})
	assertStatus(t, revokeTokenResp, http.StatusOK)

	globalSignOutResp := cognitoUserPoolsRequestPayload(t, ts, "GlobalSignOut", map[string]any{
		"AccessToken": newAccessToken,
	})
	assertStatus(t, globalSignOutResp, http.StatusOK)

	getUserAfterSignOutResp := cognitoUserPoolsRequestPayload(t, ts, "GetUser", map[string]any{"AccessToken": newAccessToken})
	assertStatus(t, getUserAfterSignOutResp, http.StatusBadRequest)

	cognitoUserPoolsMustAdminCreateUser(t, ts, userPoolID, "admin-auth-user", "Temp#987654")

	adminInitiateAuthResp := cognitoUserPoolsRequestPayload(t, ts, "AdminInitiateAuth", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"AuthFlow":   "ADMIN_USER_PASSWORD_AUTH",
		"AuthParameters": map[string]any{
			"USERNAME": "admin-auth-user",
			"PASSWORD": "Temp#987654",
		},
	})
	assertStatus(t, adminInitiateAuthResp, http.StatusOK)
	adminInitiateAuthBody := decodeCognitoUserPoolsBody(t, adminInitiateAuthResp)
	if got, _ := adminInitiateAuthBody["ChallengeName"].(string); got != "NEW_PASSWORD_REQUIRED" {
		t.Fatalf("expected ChallengeName=NEW_PASSWORD_REQUIRED, got %#v", adminInitiateAuthBody["ChallengeName"])
	}
	adminChallengeSession, _ := adminInitiateAuthBody["Session"].(string)
	if strings.TrimSpace(adminChallengeSession) == "" {
		t.Fatalf("expected Session in AdminInitiateAuth challenge response")
	}

	adminRespondResp := cognitoUserPoolsRequestPayload(t, ts, "AdminRespondToAuthChallenge", map[string]any{
		"UserPoolId":    userPoolID,
		"ClientId":      clientID,
		"ChallengeName": "NEW_PASSWORD_REQUIRED",
		"Session":       adminChallengeSession,
		"ChallengeResponses": map[string]any{
			"USERNAME":     "admin-auth-user",
			"NEW_PASSWORD": "Admin#Updated1",
		},
	})
	assertStatus(t, adminRespondResp, http.StatusOK)
}

func TestCognitoUserPoolsStage5MFAAndWebAuthnFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-stage345-mfa-pool", "stackyard-stage345-mfa-client")
	cognitoUserPoolsMustSignUpAndConfirm(t, ts, clientID, "stage5-user", "Pass#123456")

	authBody := cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage5-user", "Pass#123456")
	authResultObj, _ := authBody["AuthenticationResult"].(map[string]any)
	accessToken, _ := authResultObj["AccessToken"].(string)
	if strings.TrimSpace(accessToken) == "" {
		t.Fatalf("expected AccessToken from InitiateAuth")
	}

	setPoolMFAConfigResp := cognitoUserPoolsRequestPayload(t, ts, "SetUserPoolMfaConfig", map[string]any{
		"UserPoolId":       userPoolID,
		"MfaConfiguration": "OPTIONAL",
		"SoftwareTokenMfaConfiguration": map[string]any{
			"Enabled": true,
		},
		"WebAuthnConfiguration": map[string]any{
			"RelyingPartyId":   "stackyard.local",
			"UserVerification": "required",
		},
	})
	assertStatus(t, setPoolMFAConfigResp, http.StatusOK)

	getPoolMFAConfigResp := cognitoUserPoolsRequestPayload(t, ts, "GetUserPoolMfaConfig", map[string]any{
		"UserPoolId": userPoolID,
	})
	assertStatus(t, getPoolMFAConfigResp, http.StatusOK)

	associateSoftwareTokenResp := cognitoUserPoolsRequestPayload(t, ts, "AssociateSoftwareToken", map[string]any{
		"AccessToken": accessToken,
	})
	assertStatus(t, associateSoftwareTokenResp, http.StatusOK)
	associateSoftwareTokenBody := decodeCognitoUserPoolsBody(t, associateSoftwareTokenResp)
	setupSession, _ := associateSoftwareTokenBody["Session"].(string)
	if strings.TrimSpace(setupSession) == "" {
		t.Fatalf("expected Session in AssociateSoftwareToken response")
	}

	verifySoftwareTokenResp := cognitoUserPoolsRequestPayload(t, ts, "VerifySoftwareToken", map[string]any{
		"Session":  setupSession,
		"UserCode": "123456",
	})
	assertStatus(t, verifySoftwareTokenResp, http.StatusOK)

	setUserMFAPreferenceResp := cognitoUserPoolsRequestPayload(t, ts, "SetUserMFAPreference", map[string]any{
		"AccessToken": accessToken,
		"SoftwareTokenMfaSettings": map[string]any{
			"Enabled":      true,
			"PreferredMfa": true,
		},
	})
	assertStatus(t, setUserMFAPreferenceResp, http.StatusOK)

	adminSetUserMFAPreferenceResp := cognitoUserPoolsRequestPayload(t, ts, "AdminSetUserMFAPreference", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "stage5-user",
		"SoftwareTokenMfaSettings": map[string]any{
			"Enabled":      true,
			"PreferredMfa": true,
		},
	})
	assertStatus(t, adminSetUserMFAPreferenceResp, http.StatusOK)

	getUserAuthFactorsResp := cognitoUserPoolsRequestPayload(t, ts, "GetUserAuthFactors", map[string]any{
		"AccessToken": accessToken,
	})
	assertStatus(t, getUserAuthFactorsResp, http.StatusOK)
	getUserAuthFactorsBody := decodeCognitoUserPoolsBody(t, getUserAuthFactorsResp)
	factors, ok := getUserAuthFactorsBody["ConfiguredUserAuthFactors"].([]any)
	if !ok || len(factors) == 0 {
		t.Fatalf("expected ConfiguredUserAuthFactors list, got %#v", getUserAuthFactorsBody["ConfiguredUserAuthFactors"])
	}

	startWebAuthnRegistrationResp := cognitoUserPoolsRequestPayload(t, ts, "StartWebAuthnRegistration", map[string]any{
		"AccessToken": accessToken,
	})
	assertStatus(t, startWebAuthnRegistrationResp, http.StatusOK)
	startWebAuthnRegistrationBody := decodeCognitoUserPoolsBody(t, startWebAuthnRegistrationResp)
	webauthnSession, _ := startWebAuthnRegistrationBody["Session"].(string)
	if strings.TrimSpace(webauthnSession) == "" {
		t.Fatalf("expected Session in StartWebAuthnRegistration response")
	}

	completeWebAuthnRegistrationResp := cognitoUserPoolsRequestPayload(t, ts, "CompleteWebAuthnRegistration", map[string]any{
		"AccessToken":  accessToken,
		"Session":      webauthnSession,
		"Credential":   "{\"id\":\"credential-stage5\"}",
		"FriendlyName": "Stage5 Credential",
	})
	assertStatus(t, completeWebAuthnRegistrationResp, http.StatusOK)
	completeWebAuthnRegistrationBody := decodeCognitoUserPoolsBody(t, completeWebAuthnRegistrationResp)
	credentialObj, _ := completeWebAuthnRegistrationBody["Credential"].(map[string]any)
	credentialID, _ := credentialObj["CredentialId"].(string)
	if strings.TrimSpace(credentialID) == "" {
		t.Fatalf("expected CredentialId in complete registration response, got %#v", completeWebAuthnRegistrationBody)
	}

	listWebAuthnCredentialsResp := cognitoUserPoolsRequestPayload(t, ts, "ListWebAuthnCredentials", map[string]any{
		"AccessToken": accessToken,
		"MaxResults":  10,
	})
	assertStatus(t, listWebAuthnCredentialsResp, http.StatusOK)
	listWebAuthnCredentialsBody := decodeCognitoUserPoolsBody(t, listWebAuthnCredentialsResp)
	credentials, ok := listWebAuthnCredentialsBody["Credentials"].([]any)
	if !ok || len(credentials) == 0 {
		t.Fatalf("expected WebAuthn credentials list, got %#v", listWebAuthnCredentialsBody["Credentials"])
	}

	deleteWebAuthnCredentialResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteWebAuthnCredential", map[string]any{
		"AccessToken":  accessToken,
		"CredentialId": credentialID,
	})
	assertStatus(t, deleteWebAuthnCredentialResp, http.StatusOK)
}

func TestCognitoUserPoolsStage345ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	actions := []string{
		"AdminCreateUser",
		"AdminGetUser",
		"ListUsers",
		"AdminUpdateUserAttributes",
		"AdminDeleteUserAttributes",
		"AdminDeleteUser",
		"AdminDisableUser",
		"AdminEnableUser",
		"AdminSetUserPassword",
		"AdminResetUserPassword",
		"AdminConfirmSignUp",
		"AdminUserGlobalSignOut",
		"CreateGroup",
		"GetGroup",
		"UpdateGroup",
		"DeleteGroup",
		"ListGroups",
		"AdminAddUserToGroup",
		"AdminRemoveUserFromGroup",
		"AdminListGroupsForUser",
		"ListUsersInGroup",
		"AdminGetDevice",
		"AdminListDevices",
		"AdminUpdateDeviceStatus",
		"AdminForgetDevice",
		"CreateUserImportJob",
		"DescribeUserImportJob",
		"ListUserImportJobs",
		"StartUserImportJob",
		"StopUserImportJob",
		"GetCSVHeader",
		"SignUp",
		"ConfirmSignUp",
		"ResendConfirmationCode",
		"InitiateAuth",
		"AdminInitiateAuth",
		"RespondToAuthChallenge",
		"AdminRespondToAuthChallenge",
		"ForgotPassword",
		"ConfirmForgotPassword",
		"ChangePassword",
		"GetUser",
		"DeleteUser",
		"UpdateUserAttributes",
		"DeleteUserAttributes",
		"GetUserAttributeVerificationCode",
		"VerifyUserAttribute",
		"GetTokensFromRefreshToken",
		"RevokeToken",
		"GlobalSignOut",
		"ConfirmDevice",
		"GetDevice",
		"ListDevices",
		"UpdateDeviceStatus",
		"ForgetDevice",
		"AssociateSoftwareToken",
		"VerifySoftwareToken",
		"SetUserMFAPreference",
		"AdminSetUserMFAPreference",
		"GetUserPoolMfaConfig",
		"SetUserPoolMfaConfig",
		"StartWebAuthnRegistration",
		"CompleteWebAuthnRegistration",
		"ListWebAuthnCredentials",
		"DeleteWebAuthnCredential",
		"GetUserAuthFactors",
	}

	for _, action := range actions {
		resp := cognitoUserPoolsRequestPayload(t, ts, action, map[string]any{})
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", action, resp.StatusCode, body)
		}
	}
}
