// Command personalize-runtime runs Amazon Personalize Runtime against
// Stackyard: stream interactions into a new dataset group, train
// user-personalization and personalized-ranking campaigns, then ask for
// recommendations (plain and through a filter) and for a ranking of a
// candidate list.
//
// Recommendations are deterministic for the same data, so repeated runs
// print the same lists. The SDK has no Personalize Runtime module in this
// repository's go.mod, so requests are signed by hand.
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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// group creates an ACTIVE dataset group with an ACTIVE dataset per type
// (name/type pairs give each schema) and returns the group and datasets.
func (c client) group(run string, datasets map[string][]string) (string, map[string]string) {
	group := c.call("CreateDatasetGroup", map[string]any{"name": "shop-" + run})["datasetGroupArn"].(string)
	c.waitActive("DatasetGroup", group)
	arns := map[string]string{}
	for typ, fields := range datasets {
		name := strings.ToLower(strings.ReplaceAll(typ, "_", "-")) + "-" + run
		schema := c.call("CreateSchema", map[string]any{"name": name, "schema": avro(fields...)})["schemaArn"].(string)
		arns[typ] = c.call("CreateDataset", map[string]any{"name": name, "schemaArn": schema,
			"datasetGroupArn": group, "datasetType": typ})["datasetArn"].(string)
		c.waitActive("Dataset", arns[typ])
	}
	return group, arns
}

func main() {
	c := newClient()
	fmt.Printf("Stackyard Personalize Runtime client using %s\n", c.endpoint)
	run := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	group, ds := c.group(run, map[string][]string{
		"Interactions": {"USER_ID", "string", "ITEM_ID", "string", "EVENT_TYPE", "string", "TIMESTAMP", "long"},
		"Items":        {"ITEM_ID", "string", "GENRE", "string"},
	})
	trackingID := c.call("CreateEventTracker", map[string]any{"name": "web-" + run, "datasetGroupArn": group})["trackingId"].(string)
	now := float64(time.Now().Unix())
	for i, user := range []string{"u1", "u1", "u2", "u2", "u3"} {
		c.rest("/events", map[string]any{"trackingId": trackingID, "userId": user, "sessionId": "s-" + user,
			"eventList": []any{map[string]any{"eventType": "click", "itemId": fmt.Sprintf("i%d", i%4+1), "sentAt": now + float64(i)}}})
	}
	c.rest("/items", map[string]any{"datasetArn": ds["Items"], "items": []any{
		map[string]any{"itemId": "i1", "properties": `{"genre": "drama"}`}, map[string]any{"itemId": "i2", "properties": `{"genre": "comedy"}`},
		map[string]any{"itemId": "i3", "properties": `{"genre": "drama"}`}, map[string]any{"itemId": "i4", "properties": `{"genre": "action"}`},
	}})

	campaign := func(recipe string) string {
		solution := c.call("CreateSolution", map[string]any{"name": recipe + "-" + run, "datasetGroupArn": group,
			"recipeArn": "arn:aws:personalize:::recipe/" + recipe})["solutionArn"].(string)
		c.waitActive("Solution", solution)
		version := c.call("CreateSolutionVersion", map[string]any{"solutionArn": solution})["solutionVersionArn"].(string)
		c.waitActive("SolutionVersion", version)
		arn := c.call("CreateCampaign", map[string]any{"name": recipe + "-" + run, "solutionVersionArn": version})["campaignArn"].(string)
		c.waitActive("Campaign", arn)
		return arn
	}
	picks, ranking := campaign("aws-user-personalization"), campaign("aws-personalized-ranking")

	show := func(label string, list any) {
		fmt.Printf("%s:", label)
		for _, v := range list.([]any) {
			item := v.(map[string]any)
			fmt.Printf(" %v(%.3f)", item["itemId"], item["score"])
		}
		fmt.Println()
	}
	show("recommendations for u1", c.rest("/recommendations", map[string]any{"campaignArn": picks, "userId": "u1", "numResults": 3})["itemList"])

	filter := c.call("CreateFilter", map[string]any{"name": "no-drama-" + run, "datasetGroupArn": group,
		"filterExpression": "EXCLUDE ItemID WHERE Items.GENRE IN ($GENRE)"})["filterArn"].(string)
	c.waitActive("Filter", filter)
	show("without drama", c.rest("/recommendations", map[string]any{"campaignArn": picks, "userId": "u1",
		"filterArn": filter, "filterValues": map[string]string{"GENRE": `"drama"`}})["itemList"])
	show("ranking for u2", c.rest("/personalize-ranking", map[string]any{"campaignArn": ranking, "userId": "u2",
		"inputList": []string{"i1", "i2", "i3", "i4"}})["personalizedRanking"])
}

// avro builds an Avro record schema from name/type pairs.
func avro(pairs ...string) string {
	var fields []map[string]string
	for i := 0; i+1 < len(pairs); i += 2 {
		fields = append(fields, map[string]string{"name": pairs[i], "type": pairs[i+1]})
	}
	raw, _ := json.Marshal(map[string]any{"type": "record", "name": "Interactions",
		"namespace": "com.amazonaws.personalize.schema", "fields": fields, "version": "1.0"})
	return string(raw)
}

// waitActive describes kind until it is ACTIVE and returns it.
func (c client) waitActive(kind, arn string) map[string]any {
	key := strings.ToLower(kind[:1]) + kind[1:]
	for i := 0; i < 20; i++ {
		obj, _ := c.call("Describe"+kind, map[string]any{key + "Arn": arn})[key].(map[string]any)
		switch status, _ := obj["status"].(string); {
		case status == "ACTIVE":
			return obj
		case strings.HasSuffix(status, "FAILED"):
			exitf("%s %s failed: %v", kind, arn, obj["failureReason"])
		}
		time.Sleep(100 * time.Millisecond)
	}
	exitf("%s %s never became ACTIVE", kind, arn)
	return nil
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(http.MethodPost, "/", "AmazonPersonalize."+op, "personalize", body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(method, path, target, service string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	if target != "" {
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
		req.Header.Set("X-Amz-Target", target)
	} else if service == "personalize" {
		req.Header.Set("Content-Type", "application/json")
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s %s %s: HTTP %d: %s", method, path, target, resp.StatusCode, raw)
	}
	return raw
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// rest posts a Personalize Events or Runtime request.
func (c client) rest(path string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", path, err)
	}
	out := map[string]any{}
	if raw := c.send(http.MethodPost, path, "", "personalize", body); len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", path, err)
		}
	}
	return out
}
