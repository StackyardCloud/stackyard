package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func FuzzRedshiftInvalidForm(f *testing.F) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	seeds := [][]byte{
		[]byte(""),
		[]byte("Action=CreateCluster"),
		[]byte("Action=CreateCluster&ClusterIdentifier=%ZZ"),
		[]byte("Action=%00%00"),
		[]byte("Action=PutResourcePolicy&ResourceArn=arn:aws:redshift:us-east-1:123456789012:cluster/demo&Policy=%7B%7D"),
		[]byte("Action=ModifyEventSubscription&SubscriptionName=sub-1&Enabled=notabool"),
		[]byte("Action=CreateCluster&ClusterIdentifier=a&NodeType=ra3.xlplus&MasterUsername=admin&MasterUserPassword=short"),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/redshift", body, map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		}, "redshift")
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusInternalServerError {
			t.Fatalf("unexpected %d response", resp.StatusCode)
		}
	})
}
