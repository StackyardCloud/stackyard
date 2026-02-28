package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCognitoUserPoolsStage89ManagedBrandingTermsAndCompatibilityActions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-stage89-pool", "stackyard-stage89-client")
	cognitoUserPoolsMustSignUpAndConfirm(t, ts, clientID, "stage89-user", "Pass#Stage89")
	authBody := cognitoUserPoolsMustInitiatePasswordAuth(t, ts, clientID, "stage89-user", "Pass#Stage89")
	authResultObj, _ := authBody["AuthenticationResult"].(map[string]any)
	accessToken, _ := authResultObj["AccessToken"].(string)
	if strings.TrimSpace(accessToken) == "" {
		t.Fatalf("expected AccessToken in auth result")
	}

	domain := "stackyard-stage89-domain"
	createDomainResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPoolDomain", map[string]any{
		"UserPoolId": userPoolID,
		"Domain":     domain,
	})
	assertStatus(t, createDomainResp, http.StatusOK)

	addCustomAttributesResp := cognitoUserPoolsRequestPayload(t, ts, "AddCustomAttributes", map[string]any{
		"UserPoolId": userPoolID,
		"CustomAttributes": []map[string]any{
			{
				"Name":                   "department",
				"AttributeDataType":      "String",
				"Mutable":                true,
				"Required":               false,
				"DeveloperOnlyAttribute": false,
				"StringAttributeConstraints": map[string]any{
					"MinLength": "1",
					"MaxLength": "64",
				},
			},
		},
	})
	assertStatus(t, addCustomAttributesResp, http.StatusOK)

	updateDomainResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateUserPoolDomain", map[string]any{
		"UserPoolId":          userPoolID,
		"Domain":              domain,
		"ManagedLoginVersion": 2,
		"CustomDomainConfig": map[string]any{
			"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/stackyard-stage89",
		},
	})
	assertStatus(t, updateDomainResp, http.StatusOK)

	setUICustomizationResp := cognitoUserPoolsRequestPayload(t, ts, "SetUICustomization", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"CSS":        ".banner{color:#112233;}",
		"ImageFile":  "c3RhY2t5YXJk",
	})
	assertStatus(t, setUICustomizationResp, http.StatusOK)

	getUICustomizationResp := cognitoUserPoolsRequestPayload(t, ts, "GetUICustomization", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	})
	assertStatus(t, getUICustomizationResp, http.StatusOK)
	getUICustomizationBody := decodeCognitoUserPoolsBody(t, getUICustomizationResp)
	uiCustomizationObj, ok := getUICustomizationBody["UICustomization"].(map[string]any)
	if !ok {
		t.Fatalf("expected UICustomization object, got %#v", getUICustomizationBody["UICustomization"])
	}
	if got, _ := uiCustomizationObj["ClientId"].(string); got != clientID {
		t.Fatalf("expected UICustomization.ClientId %q, got %#v", clientID, uiCustomizationObj["ClientId"])
	}

	createManagedLoginBrandingResp := cognitoUserPoolsRequestPayload(t, ts, "CreateManagedLoginBranding", map[string]any{
		"UserPoolId":               userPoolID,
		"ClientId":                 clientID,
		"UseCognitoProvidedValues": true,
		"Settings":                 map[string]any{"theme": "minimal"},
		"Assets": []map[string]any{
			{
				"Category":   "FAVICON_ICO",
				"ColorMode":  "LIGHT",
				"Extension":  "ICO",
				"Bytes":      "c3RhY2t5YXJk",
				"ResourceId": "favicon",
			},
		},
	})
	assertStatus(t, createManagedLoginBrandingResp, http.StatusOK)
	createManagedLoginBrandingBody := decodeCognitoUserPoolsBody(t, createManagedLoginBrandingResp)
	managedLoginBrandingObj, ok := createManagedLoginBrandingBody["ManagedLoginBranding"].(map[string]any)
	if !ok {
		t.Fatalf("expected ManagedLoginBranding object, got %#v", createManagedLoginBrandingBody["ManagedLoginBranding"])
	}
	managedLoginBrandingID, _ := managedLoginBrandingObj["ManagedLoginBrandingId"].(string)
	if strings.TrimSpace(managedLoginBrandingID) == "" {
		t.Fatalf("expected ManagedLoginBrandingId in create response: %#v", managedLoginBrandingObj)
	}

	describeManagedLoginBrandingResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeManagedLoginBranding", map[string]any{
		"UserPoolId":             userPoolID,
		"ManagedLoginBrandingId": managedLoginBrandingID,
	})
	assertStatus(t, describeManagedLoginBrandingResp, http.StatusOK)

	describeManagedLoginBrandingByClientResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeManagedLoginBrandingByClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	})
	assertStatus(t, describeManagedLoginBrandingByClientResp, http.StatusOK)

	updateManagedLoginBrandingResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateManagedLoginBranding", map[string]any{
		"UserPoolId":               userPoolID,
		"ManagedLoginBrandingId":   managedLoginBrandingID,
		"UseCognitoProvidedValues": false,
		"Settings": map[string]any{
			"theme": "contrast",
		},
	})
	assertStatus(t, updateManagedLoginBrandingResp, http.StatusOK)

	getSigningCertificateResp := cognitoUserPoolsRequestPayload(t, ts, "GetSigningCertificate", map[string]any{
		"UserPoolId": userPoolID,
	})
	assertStatus(t, getSigningCertificateResp, http.StatusOK)
	getSigningCertificateBody := decodeCognitoUserPoolsBody(t, getSigningCertificateResp)
	certificate, _ := getSigningCertificateBody["Certificate"].(string)
	if !strings.Contains(certificate, "BEGIN CERTIFICATE") {
		t.Fatalf("expected certificate PEM in GetSigningCertificate response, got %#v", getSigningCertificateBody["Certificate"])
	}

	createTermsResp := cognitoUserPoolsRequestPayload(t, ts, "CreateTerms", map[string]any{
		"UserPoolId": userPoolID,
		"TermsName":  "stage89-terms",
		"Content": map[string]any{
			"Text": "stackyard terms v1",
		},
	})
	assertStatus(t, createTermsResp, http.StatusOK)
	createTermsBody := decodeCognitoUserPoolsBody(t, createTermsResp)
	termsObj, ok := createTermsBody["Terms"].(map[string]any)
	if !ok {
		t.Fatalf("expected Terms object in create response, got %#v", createTermsBody["Terms"])
	}
	termsID, _ := termsObj["TermsId"].(string)
	if strings.TrimSpace(termsID) == "" {
		t.Fatalf("expected TermsId in create response: %#v", termsObj)
	}

	updateTermsResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateTerms", map[string]any{
		"UserPoolId": userPoolID,
		"TermsId":    termsID,
		"TermsName":  "stage89-terms-v2",
		"Content": map[string]any{
			"Text": "stackyard terms v2",
		},
	})
	assertStatus(t, updateTermsResp, http.StatusOK)

	describeTermsResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeTerms", map[string]any{
		"UserPoolId": userPoolID,
		"TermsId":    termsID,
	})
	assertStatus(t, describeTermsResp, http.StatusOK)

	listTermsResp := cognitoUserPoolsRequestPayload(t, ts, "ListTerms", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 10,
	})
	assertStatus(t, listTermsResp, http.StatusOK)

	adminLinkProviderForUserResp := cognitoUserPoolsRequestPayload(t, ts, "AdminLinkProviderForUser", map[string]any{
		"UserPoolId": userPoolID,
		"DestinationUser": map[string]any{
			"ProviderName":           "Cognito",
			"ProviderAttributeName":  "Cognito_Subject",
			"ProviderAttributeValue": "stage89-user",
		},
		"SourceUser": map[string]any{
			"ProviderName":           "Google",
			"ProviderAttributeName":  "Cognito_Subject",
			"ProviderAttributeValue": "google-stage89-user",
		},
	})
	assertStatus(t, adminLinkProviderForUserResp, http.StatusOK)

	adminDisableProviderForUserResp := cognitoUserPoolsRequestPayload(t, ts, "AdminDisableProviderForUser", map[string]any{
		"UserPoolId": userPoolID,
		"User": map[string]any{
			"ProviderName":           "Google",
			"ProviderAttributeName":  "Cognito_Subject",
			"ProviderAttributeValue": "google-stage89-user",
		},
	})
	assertStatus(t, adminDisableProviderForUserResp, http.StatusOK)

	adminSetUserSettingsResp := cognitoUserPoolsRequestPayload(t, ts, "AdminSetUserSettings", map[string]any{
		"UserPoolId": userPoolID,
		"Username":   "stage89-user",
		"MFAOptions": []map[string]any{
			{
				"DeliveryMedium": "SMS",
				"AttributeName":  "phone_number",
			},
		},
	})
	assertStatus(t, adminSetUserSettingsResp, http.StatusOK)

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

	deleteManagedLoginBrandingResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteManagedLoginBranding", map[string]any{
		"UserPoolId":             userPoolID,
		"ManagedLoginBrandingId": managedLoginBrandingID,
	})
	assertStatus(t, deleteManagedLoginBrandingResp, http.StatusOK)

	deleteTermsResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteTerms", map[string]any{
		"UserPoolId": userPoolID,
		"TermsId":    termsID,
	})
	assertStatus(t, deleteTermsResp, http.StatusOK)
}
