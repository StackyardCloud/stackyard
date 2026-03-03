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

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Cognito Federated Identities advanced client using %s\n", endpoint)

	createOut := mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "CreateIdentityPool", map[string]any{
		"IdentityPoolName":               "stackyard-cognito-advanced",
		"AllowUnauthenticatedIdentities": true,
		"DeveloperProviderName":          "login.stackyard.example",
		"SupportedLoginProviders": map[string]any{
			"graph.facebook.com": "app-id",
		},
		"IdentityPoolTags": map[string]any{
			"env": "dev",
		},
	})
	identityPoolID := mustStringField("CreateIdentityPool", createOut, "IdentityPoolId")
	logf("CreateIdentityPool succeeded: %s", identityPoolID)
	resourceARN := "arn:aws:cognito-identity:" + region + ":123456789012:identitypool/" + identityPoolID

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "UpdateIdentityPool", map[string]any{
		"IdentityPoolId":                 identityPoolID,
		"IdentityPoolName":               "stackyard-cognito-advanced-updated",
		"AllowUnauthenticatedIdentities": true,
		"AllowClassicFlow":               true,
		"SupportedLoginProviders": map[string]any{
			"graph.facebook.com": "app-id-updated",
		},
	})
	logf("UpdateIdentityPool succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "SetIdentityPoolRoles", map[string]any{
		"IdentityPoolId": identityPoolID,
		"Roles": map[string]any{
			"authenticated": "arn:aws:iam::123456789012:role/stackyard-authenticated",
		},
		"RoleMappings": map[string]any{
			"login.stackyard.example": map[string]any{
				"Type":                    "Rules",
				"AmbiguousRoleResolution": "AuthenticatedRole",
				"RulesConfiguration": map[string]any{
					"Rules": []map[string]any{
						{
							"Claim":     "tier",
							"MatchType": "Equals",
							"Value":     "admin",
							"RoleARN":   "arn:aws:iam::123456789012:role/stackyard-admin",
						},
					},
				},
			},
		},
	})
	logf("SetIdentityPoolRoles succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "GetIdentityPoolRoles", map[string]any{
		"IdentityPoolId": identityPoolID,
	})
	logf("GetIdentityPoolRoles succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "SetPrincipalTagAttributeMap", map[string]any{
		"IdentityPoolId":       identityPoolID,
		"IdentityProviderName": "login.stackyard.example",
		"UseDefaults":          true,
		"PrincipalTags": map[string]any{
			"team": "platform",
		},
	})
	logf("SetPrincipalTagAttributeMap succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "GetPrincipalTagAttributeMap", map[string]any{
		"IdentityPoolId":       identityPoolID,
		"IdentityProviderName": "login.stackyard.example",
	})
	logf("GetPrincipalTagAttributeMap succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "ListIdentityPools", map[string]any{
		"MaxResults": 25,
	})
	logf("ListIdentityPools succeeded")

	getIDOut := mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "GetId", map[string]any{
		"IdentityPoolId": identityPoolID,
		"Logins": map[string]any{
			"cognito-idp.us-east-1.amazonaws.com/us-east-1_stackyard": "advanced-user",
		},
	})
	identityID := mustStringField("GetId", getIDOut, "IdentityId")
	logf("GetId succeeded: %s", identityID)

	devPrimaryOut := mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "GetOpenIdTokenForDeveloperIdentity", map[string]any{
		"IdentityPoolId": identityPoolID,
		"Logins": map[string]any{
			"login.stackyard.example": "dev-user-primary",
		},
	})
	devPrimaryIdentityID := mustStringField("GetOpenIdTokenForDeveloperIdentity", devPrimaryOut, "IdentityId")
	logf("GetOpenIdTokenForDeveloperIdentity(primary) succeeded: %s", devPrimaryIdentityID)

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "GetOpenIdTokenForDeveloperIdentity", map[string]any{
		"IdentityPoolId": identityPoolID,
		"Logins": map[string]any{
			"login.stackyard.example": "dev-user-secondary",
		},
	})
	logf("GetOpenIdTokenForDeveloperIdentity(secondary) succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "LookupDeveloperIdentity", map[string]any{
		"IdentityPoolId":          identityPoolID,
		"DeveloperUserIdentifier": "dev-user-secondary",
	})
	logf("LookupDeveloperIdentity succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "MergeDeveloperIdentities", map[string]any{
		"IdentityPoolId":                    identityPoolID,
		"DeveloperProviderName":             "login.stackyard.example",
		"DestinationUserIdentifierForMerge": "dev-user-primary",
		"SourceUserIdentifier":              "dev-user-secondary",
	})
	logf("MergeDeveloperIdentities succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "UnlinkDeveloperIdentity", map[string]any{
		"IdentityPoolId":          identityPoolID,
		"IdentityId":              devPrimaryIdentityID,
		"DeveloperProviderName":   "login.stackyard.example",
		"DeveloperUserIdentifier": "dev-user-secondary",
	})
	logf("UnlinkDeveloperIdentity succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"ResourceArn": resourceARN,
		"Tags": map[string]any{
			"owner": "advanced-example",
			"tier":  "dev",
		},
	})
	logf("TagResource succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "ListTagsForResource", map[string]any{
		"ResourceArn": resourceARN,
	})
	logf("ListTagsForResource succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "UntagResource", map[string]any{
		"ResourceArn": resourceARN,
		"TagKeys":     []string{"tier"},
	})
	logf("UntagResource succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "DescribeIdentity", map[string]any{
		"IdentityId": identityID,
	})
	logf("DescribeIdentity succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "ListIdentities", map[string]any{
		"IdentityPoolId": identityPoolID,
		"MaxResults":     25,
	})
	logf("ListIdentities succeeded")

	credentialsOut := mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "GetCredentialsForIdentity", map[string]any{
		"IdentityId": identityID,
	})
	if _, ok := credentialsOut["Credentials"].(map[string]any); !ok {
		exitf("GetCredentialsForIdentity response missing Credentials: %#v", credentialsOut)
	}
	logf("GetCredentialsForIdentity succeeded")

	openIDOut := mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "GetOpenIdToken", map[string]any{
		"IdentityId": identityID,
	})
	_ = mustStringField("GetOpenIdToken", openIDOut, "Token")
	logf("GetOpenIdToken succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "UnlinkIdentity", map[string]any{
		"IdentityId": identityID,
		"Logins": map[string]any{
			"cognito-idp.us-east-1.amazonaws.com/us-east-1_stackyard": "advanced-user",
		},
		"LoginsToRemove": []string{"cognito-idp.us-east-1.amazonaws.com/us-east-1_stackyard"},
	})
	logf("UnlinkIdentity succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "DeleteIdentities", map[string]any{
		"IdentityIdsToDelete": []string{devPrimaryIdentityID},
	})
	logf("DeleteIdentities succeeded")

	mustCognitoIdentitySuccess(ctx, endpoint, region, creds, "DeleteIdentityPool", map[string]any{
		"IdentityPoolId": identityPoolID,
	})
	logf("DeleteIdentityPool succeeded")

	fmt.Println("Done.")
}

func callCognitoIdentityAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(encodedPayload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "com.amazonaws.cognito.identity.model.AWSCognitoIdentityService."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(encodedPayload), "cognito-identity", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

func mustCognitoIdentitySuccess(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) map[string]any {
	status, body, err := callCognitoIdentityAction(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status != http.StatusOK {
		exitf("%s expected %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	out := map[string]any{}
	if len(body) == 0 {
		return out
	}
	if err := json.Unmarshal(body, &out); err != nil {
		exitf("%s returned invalid JSON: %v body=%s", action, err, strings.TrimSpace(string(body)))
	}
	return out
}

func mustStringField(action string, payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok {
		exitf("%s response missing %s", action, key)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		exitf("%s response has invalid %s: %#v", action, key, raw)
	}
	return value
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
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
