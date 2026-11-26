// Command mturk runs the Amazon Mechanical Turk requester API (sandbox
// semantics) against Stackyard: create a qualification type and a HIT that
// requires it, let simulated workers request the qualification, accept and
// submit the HIT through Stackyard's worker links, then approve, reject, pay
// a bonus and dispose of the HIT.
//
// The SDK has no MTurk module in this repository's go.mod, so requests are
// signed by hand.
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

const (
	signingName  = "mturk-requester"
	targetPrefix = "MTurkRequesterServiceV20170117."
	question     = `<HTMLQuestion xmlns="http://mechanicalturk.amazonaws.com/AWSMechanicalTurkDataSchemas/2011-11-11/HTMLQuestion.xsd"><HTMLContent><![CDATA[<p>Which animal is in the photo?</p>]]></HTMLContent><FrameHeight>400</FrameHeight></HTMLQuestion>`
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard MTurk client using %s\n", c.endpoint)
	fmt.Printf("sandbox balance: $%v\n", c.call("GetAccountBalance", map[string]any{})["AvailableBalance"])
	run := time.Now().UnixNano()

	qual := c.call("CreateQualificationType", map[string]any{
		"Name": fmt.Sprintf("Animal tagging %d", run), "Description": "Can tell animals apart",
		"QualificationTypeStatus": "Active", "RetryDelayInSeconds": 3600,
	})["QualificationType"].(map[string]any)["QualificationTypeId"].(string)
	hit := c.call("CreateHIT", map[string]any{
		"Title": "Tag the animal", "Description": "Name the animal in one photo", "Reward": "0.15",
		"MaxAssignments": 2, "LifetimeInSeconds": 3600, "AssignmentDurationInSeconds": 600, "AutoApprovalDelayInSeconds": 86400,
		"Question": question, "QualificationRequirements": []any{map[string]any{
			"QualificationTypeId": qual, "Comparator": "GreaterThanOrEqualTo", "IntegerValues": []int{70},
		}},
	})["HIT"].(map[string]any)
	hitID := hit["HITId"].(string)
	fmt.Printf("HIT %s is %v\n", hitID, hit["HITStatus"])

	// Two workers ask for the qualification; the requester grants both.
	workers := []string{fmt.Sprintf("AALICE%d", run%100000), fmt.Sprintf("ABOB%d", run%100000)}
	requests := c.call("ListQualificationRequests", map[string]any{"QualificationTypeId": qual})
	fmt.Printf("%v open qualification requests before workers ask\n", requests["NumResults"])
	for _, w := range workers {
		c.worker("request-qualification", map[string]any{"QualificationTypeId": qual, "WorkerId": w, "Answer": "<answer>cat</answer>"})
	}
	for _, r := range c.call("ListQualificationRequests", map[string]any{"QualificationTypeId": qual})["QualificationRequests"].([]any) {
		req := r.(map[string]any)
		c.call("AcceptQualificationRequest", map[string]any{"QualificationRequestId": req["QualificationRequestId"], "IntegerValue": 90})
		fmt.Printf("granted %s to %v\n", qual, req["WorkerId"])
	}

	// Both accept and submit; the HIT goes Unassignable, then Reviewable.
	var assignments []string
	for _, w := range workers {
		assignments = append(assignments, c.worker("accept", map[string]any{"HITId": hitID, "WorkerId": w})["AssignmentId"].(string))
	}
	fmt.Printf("after accepts: %v\n", c.hitStatus(hitID))
	for i, a := range assignments {
		answer := fmt.Sprintf("<QuestionFormAnswers><Answer><QuestionIdentifier>animal</QuestionIdentifier><FreeText>%s</FreeText></Answer></QuestionFormAnswers>", []string{"cat", "dog"}[i])
		c.worker("submit", map[string]any{"AssignmentId": a, "Answer": answer})
	}
	fmt.Printf("after submits: %v\n", c.hitStatus(hitID))

	c.call("ApproveAssignment", map[string]any{"AssignmentId": assignments[0], "RequesterFeedback": "Correct, thanks!"})
	c.call("RejectAssignment", map[string]any{"AssignmentId": assignments[1], "RequesterFeedback": "The photo shows a cat."})
	c.call("SendBonus", map[string]any{"WorkerId": workers[0], "AssignmentId": assignments[0], "BonusAmount": "0.25",
		"Reason": "Fast and accurate", "UniqueRequestToken": fmt.Sprintf("bonus-%d", run)})
	for _, a := range c.call("ListAssignmentsForHIT", map[string]any{"HITId": hitID})["Assignments"].([]any) {
		as := a.(map[string]any)
		fmt.Printf("assignment by %v: %v\n", as["WorkerId"], as["AssignmentStatus"])
	}
	fmt.Printf("%v bonus payment(s)\n", c.call("ListBonusPayments", map[string]any{"HITId": hitID})["NumResults"])

	c.call("DeleteHIT", map[string]any{"HITId": hitID})
	c.call("DeleteQualificationType", map[string]any{"QualificationTypeId": qual})
	fmt.Println("HIT disposed and qualification type deleted")
}

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

func (c client) hitStatus(id string) any {
	return c.call("GetHIT", map[string]any{"HITId": id})["HIT"].(map[string]any)["HITStatus"]
}

// call sends one signed JSON 1.1 operation and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", targetPrefix+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hash, signingName, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	return do(op, req)
}

// worker posts a simulated worker action to Stackyard's MTurk worker links.
func (c client) worker(action string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/_stackyard/mturk/"+action, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req.Header.Set("Content-Type", "application/json")
	return do("worker "+action, req)
}

func do(op string, req *http.Request) map[string]any {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
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
