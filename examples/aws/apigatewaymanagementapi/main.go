// Command apigatewaymanagementapi shows a WebSocket backend pushing messages to
// a connected client through the API Gateway Management API.
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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: getenv("STACKYARD_ENDPOINT", "http://localhost:4566"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard API Gateway Management API client using %s\n", c.endpoint)

	// 1. A WebSocket API (control plane, signed "apigateway").
	status, body := c.signed(http.MethodPost, "/apigatewayv2/create-api", []byte(`{"Name":"chat","ProtocolType":"WEBSOCKET"}`), "apigateway")
	expect("CreateApi", status, http.StatusOK, body)
	var api struct{ ApiId string }
	mustJSON(body, &api)

	// 2. A client connects. Stackyard has no WebSocket runtime, so the control
	// endpoint stands in for the $connect handshake.
	status, body = c.plain(http.MethodPost, "/_stackyard/apigatewaymanagementapi/connections",
		[]byte(fmt.Sprintf(`{"apiId":%q,"stage":"prod"}`, api.ApiId)))
	expect("connect", status, http.StatusCreated, body)
	var conn struct{ ConnectionId string }
	mustJSON(body, &conn)
	path := "/prod/@connections/" + url.PathEscape(conn.ConnectionId)
	fmt.Printf("connected %s to API %s\n", conn.ConnectionId, api.ApiId)

	// 3. The backend pushes a message and inspects the connection.
	status, body = c.signed(http.MethodPost, path, []byte(`{"event":"welcome"}`), "execute-api")
	expect("PostToConnection", status, http.StatusOK, body)
	status, body = c.signed(http.MethodGet, path, nil, "execute-api")
	expect("GetConnection", status, http.StatusOK, body)
	fmt.Printf("GetConnection: %s\n", bytes.TrimSpace(body))

	// 4. The client sees what was pushed.
	status, body = c.plain(http.MethodGet, "/_stackyard/apigatewaymanagementapi/connections/"+url.PathEscape(conn.ConnectionId)+"/messages", nil)
	expect("messages", status, http.StatusOK, body)
	var inbox struct{ Messages []struct{ Data []byte } }
	mustJSON(body, &inbox)
	if len(inbox.Messages) != 1 || string(inbox.Messages[0].Data) != `{"event":"welcome"}` {
		exitf("unexpected inbox: %s", body)
	}

	// 5. The backend disconnects the client; the connection is then gone.
	status, body = c.signed(http.MethodDelete, path, nil, "execute-api")
	expect("DeleteConnection", status, http.StatusNoContent, body)
	status, body = c.signed(http.MethodGet, path, nil, "execute-api")
	expect("GetConnection after delete", status, http.StatusGone, body)

	fmt.Println("Done.")
}

func (c client) signed(method, path string, payload []byte, service string) (int, []byte) {
	req := newRequest(method, c.endpoint+path, payload)
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])
	creds, err := c.creds.Retrieve(context.Background())
	if err != nil {
		exitf("credentials: %v", err)
	}
	if err := v4.NewSigner().SignHTTP(context.Background(), creds, req, hash, service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	return do(req)
}

func (c client) plain(method, path string, payload []byte) (int, []byte) {
	return do(newRequest(method, c.endpoint+path, payload))
}

func newRequest(method, target string, payload []byte) *http.Request {
	req, err := http.NewRequest(method, target, bytes.NewReader(payload))
	if err != nil {
		exitf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func do(req *http.Request) (int, []byte) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read body: %v", err)
	}
	return resp.StatusCode, body
}

func expect(name string, got, want int, body []byte) {
	if got != want {
		exitf("%s: HTTP %d, want %d: %s", name, got, want, body)
	}
	fmt.Printf("%s returned %d\n", name, got)
}

func mustJSON(body []byte, out any) {
	if err := json.Unmarshal(body, out); err != nil {
		exitf("decode %s: %v", body, err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
