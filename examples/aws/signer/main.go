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
	"net/url"
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
	profileName := getenv("STACKYARD_SIGNER_PROFILE", "stackyard-signer")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Signer advanced client using %s\n", endpoint)

	escapedProfile := url.PathEscape(profileName)
	putPath := "/signing-profiles/" + escapedProfile

	putStatus, putBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPut, putPath, map[string]any{
		"platformId": "AWSLambda-SHA384-ECDSA",
		"signatureValidityPeriod": map[string]any{
			"value": 30,
			"type":  "DAYS",
		},
	})
	if err != nil {
		exitf("PutSigningProfile request failed: %v", err)
	}
	if err := expectOK("PutSigningProfile", putStatus, putBody); err != nil {
		exitf("PutSigningProfile response validation failed: %v", err)
	}
	logf("PutSigningProfile succeeded (%d)", putStatus)

	var putOut struct {
		ARN               string `json:"arn"`
		ProfileVersion    string `json:"profileVersion"`
		ProfileVersionARN string `json:"profileVersionArn"`
	}
	if err := json.Unmarshal(putBody, &putOut); err != nil {
		exitf("PutSigningProfile decode failed: %v", err)
	}

	addPermStatus, addPermBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPost, "/signing-profiles/"+escapedProfile+"/permissions", map[string]any{
		"action":      "signer:StartSigningJob",
		"principal":   "123456789012",
		"statementId": "stackyard-statement",
	})
	if err != nil {
		exitf("AddProfilePermission request failed: %v", err)
	}
	if err := expectOK("AddProfilePermission", addPermStatus, addPermBody); err != nil {
		exitf("AddProfilePermission response validation failed: %v", err)
	}
	logf("AddProfilePermission succeeded (%d)", addPermStatus)

	var addPermOut struct {
		RevisionID string `json:"revisionId"`
	}
	if err := json.Unmarshal(addPermBody, &addPermOut); err != nil {
		exitf("AddProfilePermission decode failed: %v", err)
	}

	listPermStatus, listPermBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodGet, "/signing-profiles/"+escapedProfile+"/permissions", nil)
	if err != nil {
		exitf("ListProfilePermissions request failed: %v", err)
	}
	if err := expectOK("ListProfilePermissions", listPermStatus, listPermBody); err != nil {
		exitf("ListProfilePermissions response validation failed: %v", err)
	}
	logf("ListProfilePermissions succeeded (%d)", listPermStatus)

	startStatus, startBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPost, "/signing-jobs", map[string]any{
		"profileName":        profileName,
		"clientRequestToken": "stackyard-token",
		"source": map[string]any{
			"s3": map[string]any{
				"bucketName": "stackyard-source",
				"key":        "artifact.zip",
				"version":    "1",
			},
		},
		"destination": map[string]any{
			"s3": map[string]any{
				"bucketName": "stackyard-destination",
				"prefix":     "signed/",
			},
		},
	})
	if err != nil {
		exitf("StartSigningJob request failed: %v", err)
	}
	if err := expectOK("StartSigningJob", startStatus, startBody); err != nil {
		exitf("StartSigningJob response validation failed: %v", err)
	}
	logf("StartSigningJob succeeded (%d)", startStatus)

	var startOut struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(startBody, &startOut); err != nil {
		exitf("StartSigningJob decode failed: %v", err)
	}
	jobARN := "arn:aws:signer:us-east-1:123456789012:/signing-jobs/" + startOut.JobID

	describeStatus, describeBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodGet, "/signing-jobs/"+url.PathEscape(startOut.JobID), nil)
	if err != nil {
		exitf("DescribeSigningJob request failed: %v", err)
	}
	if err := expectOK("DescribeSigningJob", describeStatus, describeBody); err != nil {
		exitf("DescribeSigningJob response validation failed: %v", err)
	}
	logf("DescribeSigningJob succeeded (%d)", describeStatus)

	tagsPath := "/tags/" + url.PathEscape(putOut.ARN)
	tagStatus, tagBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPost, tagsPath, map[string]any{
		"tags": map[string]string{
			"env":  "dev",
			"team": "stackyard",
		},
	})
	if err != nil {
		exitf("TagResource request failed: %v", err)
	}
	if err := expectOK("TagResource", tagStatus, tagBody); err != nil {
		exitf("TagResource response validation failed: %v", err)
	}
	logf("TagResource succeeded (%d)", tagStatus)

	listTagsStatus, listTagsBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodGet, tagsPath, nil)
	if err != nil {
		exitf("ListTagsForResource request failed: %v", err)
	}
	if err := expectOK("ListTagsForResource", listTagsStatus, listTagsBody); err != nil {
		exitf("ListTagsForResource response validation failed: %v", err)
	}
	logf("ListTagsForResource succeeded (%d)", listTagsStatus)

	untagStatus, untagBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodDelete, tagsPath+"?tagKeys=team", nil)
	if err != nil {
		exitf("UntagResource request failed: %v", err)
	}
	if err := expectOK("UntagResource", untagStatus, untagBody); err != nil {
		exitf("UntagResource response validation failed: %v", err)
	}
	logf("UntagResource succeeded (%d)", untagStatus)

	signPayloadStatus, signPayloadBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPost, "/signing-jobs/with-payload", map[string]any{
		"profileName":   profileName,
		"payload":       "c3RhY2t5YXJk",
		"payloadFormat": "RAW",
	})
	if err != nil {
		exitf("SignPayload request failed: %v", err)
	}
	if err := expectOK("SignPayload", signPayloadStatus, signPayloadBody); err != nil {
		exitf("SignPayload response validation failed: %v", err)
	}
	logf("SignPayload succeeded (%d)", signPayloadStatus)

	revokeSignatureStatus, revokeSignatureBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPut, "/signing-jobs/"+url.PathEscape(startOut.JobID)+"/revoke", map[string]any{
		"reason": "Compromised",
	})
	if err != nil {
		exitf("RevokeSignature request failed: %v", err)
	}
	if err := expectOK("RevokeSignature", revokeSignatureStatus, revokeSignatureBody); err != nil {
		exitf("RevokeSignature response validation failed: %v", err)
	}
	logf("RevokeSignature succeeded (%d)", revokeSignatureStatus)

	revokeProfileStatus, revokeProfileBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPut, "/signing-profiles/"+escapedProfile+"/revoke", map[string]any{
		"profileVersion": putOut.ProfileVersion,
		"reason":         "Compromised",
		"effectiveTime":  "2024-01-01T00:00:00",
	})
	if err != nil {
		exitf("RevokeSigningProfile request failed: %v", err)
	}
	if err := expectOK("RevokeSigningProfile", revokeProfileStatus, revokeProfileBody); err != nil {
		exitf("RevokeSigningProfile response validation failed: %v", err)
	}
	logf("RevokeSigningProfile succeeded (%d)", revokeProfileStatus)

	revocationQuery := url.Values{}
	revocationQuery.Set("signatureTimestamp", "2024-01-01T00:00:00")
	revocationQuery.Set("platformId", "AWSLambda-SHA384-ECDSA")
	revocationQuery.Set("profileVersionArn", putOut.ProfileVersionARN)
	revocationQuery.Set("jobArn", jobARN)
	revocationQuery.Add("certificateHashes", strings.Repeat("0", 64))
	revocationStatus, revocationBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodGet, "/revocations?"+revocationQuery.Encode(), nil)
	if err != nil {
		exitf("GetRevocationStatus request failed: %v", err)
	}
	if err := expectOK("GetRevocationStatus", revocationStatus, revocationBody); err != nil {
		exitf("GetRevocationStatus response validation failed: %v", err)
	}
	logf("GetRevocationStatus succeeded (%d)", revocationStatus)

	if addPermOut.RevisionID != "" {
		removePath := "/signing-profiles/" + escapedProfile + "/permissions/stackyard-statement?revisionId=" + url.QueryEscape(addPermOut.RevisionID)
		removePermStatus, removePermBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodDelete, removePath, nil)
		if err != nil {
			exitf("RemoveProfilePermission request failed: %v", err)
		}
		if err := expectOK("RemoveProfilePermission", removePermStatus, removePermBody); err != nil {
			exitf("RemoveProfilePermission response validation failed: %v", err)
		}
		logf("RemoveProfilePermission succeeded (%d)", removePermStatus)
	}

	cancelStatus, cancelBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodDelete, putPath, nil)
	if err != nil {
		exitf("CancelSigningProfile request failed: %v", err)
	}
	if err := expectOK("CancelSigningProfile", cancelStatus, cancelBody); err != nil {
		exitf("CancelSigningProfile response validation failed: %v", err)
	}
	logf("CancelSigningProfile succeeded (%d)", cancelStatus)

	fmt.Println("Done.")
}

func signerRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	requestURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "signer", region, time.Now()); err != nil {
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

func expectOK(action string, status int, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	return nil
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
