// Command docdb walks a DocumentDB cluster through its lifecycle against
// Stackyard: a parameter group, a cluster with two instances, a failover, a
// snapshot and a restore, then cleans everything up.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
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
		creds:    credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	suffix := fmt.Sprint(time.Now().Unix())
	cluster, pg, snapshot, restored := "orders-"+suffix, "orders-pg-"+suffix, "orders-snap-"+suffix, "orders-restored-"+suffix
	fmt.Printf("Stackyard DocumentDB example using %s\n", c.endpoint)

	c.must("CreateDBClusterParameterGroup", "", url.Values{"DBClusterParameterGroupName": {pg}, "DBParameterGroupFamily": {"docdb5.0"}, "Description": {"example"}})
	c.must("ModifyDBClusterParameterGroup", pg, url.Values{"DBClusterParameterGroupName": {pg},
		"Parameters.Parameter.1.ParameterName": {"profiler"}, "Parameters.Parameter.1.ParameterValue": {"enabled"}, "Parameters.Parameter.1.ApplyMethod": {"immediate"}})
	c.must("CreateDBCluster", cluster, url.Values{"DBClusterIdentifier": {cluster}, "Engine": {"docdb"}, "MasterUsername": {"example"},
		"MasterUserPassword": {"Passw0rd-example"}, "DBClusterParameterGroupName": {pg}})
	c.must("CreateDBInstance", cluster+"-a", url.Values{"DBInstanceIdentifier": {cluster + "-a"}, "DBInstanceClass": {"db.r6g.large"}, "Engine": {"docdb"}, "DBClusterIdentifier": {cluster}})
	c.must("CreateDBInstance", cluster+"-b", url.Values{"DBInstanceIdentifier": {cluster + "-b"}, "DBInstanceClass": {"db.r6g.large"}, "Engine": {"docdb"}, "DBClusterIdentifier": {cluster}, "PromotionTier": {"0"}})

	failover := c.must("FailoverDBCluster", "", url.Values{"DBClusterIdentifier": {cluster}})
	if writer := writerOf(failover); writer != cluster+"-b" {
		exitf("FailoverDBCluster: expected %s-b to become the writer, got %q", cluster, writer)
	}
	fmt.Printf("failover promoted %s\n", cluster+"-b")

	c.must("CreateDBClusterSnapshot", snapshot, url.Values{"DBClusterSnapshotIdentifier": {snapshot}, "DBClusterIdentifier": {cluster}})
	c.must("RestoreDBClusterFromSnapshot", restored, url.Values{"DBClusterIdentifier": {restored}, "SnapshotIdentifier": {snapshot}, "Engine": {"docdb"}})

	// A cluster with instances can't be deleted, so the order matters.
	if status, body := c.call("DeleteDBCluster", url.Values{"DBClusterIdentifier": {cluster}, "SkipFinalSnapshot": {"true"}}); status != http.StatusBadRequest || !strings.Contains(body, "InvalidDBClusterStateFault") {
		exitf("DeleteDBCluster with instances: expected InvalidDBClusterStateFault, got %d %s", status, body)
	}
	c.must("DeleteDBInstance", "", url.Values{"DBInstanceIdentifier": {cluster + "-a"}})
	c.must("DeleteDBInstance", "", url.Values{"DBInstanceIdentifier": {cluster + "-b"}})
	c.must("DeleteDBCluster", "", url.Values{"DBClusterIdentifier": {cluster}, "SkipFinalSnapshot": {"true"}})
	c.must("DeleteDBCluster", "", url.Values{"DBClusterIdentifier": {restored}, "SkipFinalSnapshot": {"true"}})
	c.must("DeleteDBClusterSnapshot", "", url.Values{"DBClusterSnapshotIdentifier": {snapshot}})
	c.must("DeleteDBClusterParameterGroup", "", url.Values{"DBClusterParameterGroupName": {pg}})
	fmt.Println("DocumentDB example completed")
}

var memberPattern = regexp.MustCompile(`(?s)<DBClusterMember>.*?<DBInstanceIdentifier>([^<]+)</DBInstanceIdentifier>.*?<IsClusterWriter>(true|false)</IsClusterWriter>.*?</DBClusterMember>`)

func writerOf(body string) string {
	for _, m := range memberPattern.FindAllStringSubmatch(body, -1) {
		if m[2] == "true" {
			return m[1]
		}
	}
	return ""
}

// must calls action, exits on a non-200, and checks that want (if set)
// appears in the response.
func (c client) must(action, want string, params url.Values) string {
	status, body := c.call(action, params)
	if status != http.StatusOK || (want != "" && !strings.Contains(body, want)) {
		exitf("%s: %d %s", action, status, body)
	}
	fmt.Printf("%s ok\n", action)
	return body
}

func (c client) call(action string, params url.Values) (int, string) {
	params.Set("Action", action)
	params.Set("Version", "2014-10-31")
	body := []byte(params.Encode())
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(c.endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	// DocumentDB shares the RDS endpoint and signing name; the SDK's
	// User-Agent is what tells the two apart.
	req.Header.Set("User-Agent", "aws-sdk-go-v2/1.0 api/docdb#1.0")
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	creds, err := c.creds.Retrieve(context.Background())
	if err != nil {
		exitf("credentials: %v", err)
	}
	if err := v4.NewSigner().SignHTTP(context.Background(), creds, req, hash, "rds", c.region, time.Now()); err != nil {
		exitf("%s: sign: %v", action, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
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
