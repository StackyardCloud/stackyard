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

type requestCall struct {
	Name    string
	Method  string
	Path    string
	Query   map[string]string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Amazon End User Messaging Social advanced client using %s\n", endpoint)

	wabaID := "waba-000001"
	resourceArn := "arn:aws:social-messaging:us-east-1:123456789012:whatsapp-business-account/" + wabaID
	mediaID := "media-000001"
	templateID := "template-000001"

	calls := []requestCall{
		{Name: "AssociateWhatsAppBusinessAccount", Method: http.MethodPost, Path: "/v1/whatsapp/signup", Payload: map[string]any{"id": wabaID}},
		{Name: "GetLinkedWhatsAppBusinessAccount", Method: http.MethodGet, Path: "/v1/whatsapp/waba/details", Query: map[string]string{"id": wabaID}},
		{Name: "PutWhatsAppBusinessAccountEventDestinations", Method: http.MethodPut, Path: "/v1/whatsapp/waba/eventdestinations", Payload: map[string]any{"id": wabaID, "eventDestinations": []any{map[string]any{"name": "stackyard-default", "enabled": true}}}},
		{Name: "PostWhatsAppMessageMedia", Method: http.MethodPost, Path: "/v1/whatsapp/media", Payload: map[string]any{"resourceArn": resourceArn, "mediaId": mediaID, "fileName": "logo.png", "mimeType": "image/png"}},
		{Name: "GetWhatsAppMessageMedia", Method: http.MethodPost, Path: "/v1/whatsapp/media/get", Payload: map[string]any{"mediaId": mediaID}},
		{Name: "CreateWhatsAppMessageTemplate", Method: http.MethodPost, Path: "/v1/whatsapp/template/put", Payload: map[string]any{"name": "stackyard-template", "language": "en_US", "category": "UTILITY"}},
		{Name: "GetWhatsAppMessageTemplate", Method: http.MethodGet, Path: "/v1/whatsapp/template", Query: map[string]string{"id": templateID}},
		{Name: "ListWhatsAppMessageTemplates", Method: http.MethodGet, Path: "/v1/whatsapp/template/list", Query: map[string]string{"id": wabaID}},
		{Name: "SendWhatsAppMessage", Method: http.MethodPost, Path: "/v1/whatsapp/send", Payload: map[string]any{"to": "+12065550101", "from": "+12065550100", "message": "hello from stackyard"}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/v1/tags/list", Query: map[string]string{"resourceArn": resourceArn}},
	}

	for _, call := range calls {
		status, body, err := socialMessagingRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Query, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", call.Name, status)
	}

	fmt.Println("Done.")
}

func socialMessagingRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method string,
	path string,
	query map[string]string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil && method != http.MethodGet && method != http.MethodDelete {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	u, err := url.Parse(strings.TrimRight(endpoint, "/") + path)
	if err != nil {
		return 0, nil, err
	}
	if len(query) > 0 {
		q := u.Query()
		for key, value := range query {
			if strings.TrimSpace(value) == "" {
				continue
			}
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "social-messaging", region, time.Now()); err != nil {
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
