// Command codecommit walks a CodeCommit workflow against Stackyard: a
// repository, commits on two branches, a conflict-free line-level merge,
// and a pull request with an approval rule, an approval, a comment and a
// merge.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		creds:    credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	repo := fmt.Sprintf("stackyard-repo-%d", time.Now().UnixNano()%1_000_000)
	fmt.Printf("Stackyard CodeCommit client using %s\n", c.endpoint)

	c.must("CreateRepository", map[string]any{"repositoryName": repo, "repositoryDescription": "demo"})
	first := c.must("CreateCommit", map[string]any{
		"repositoryName": repo, "branchName": "main", "commitMessage": "init",
		"putFiles": []any{map[string]any{"filePath": "notes.txt", "fileContent": b64("one\ntwo\nthree\n")}},
	})["commitId"].(string)

	// Change different lines on two branches; the three-way merge combines them.
	left := c.must("PutFile", map[string]any{"repositoryName": repo, "branchName": "left", "parentCommitId": first, "filePath": "notes.txt", "fileContent": b64("ONE\ntwo\nthree\n")})["commitId"].(string)
	c.must("PutFile", map[string]any{"repositoryName": repo, "branchName": "right", "parentCommitId": first, "filePath": "notes.txt", "fileContent": b64("one\ntwo\nTHREE\n")})
	c.must("MergeBranchesByThreeWay", map[string]any{"repositoryName": repo, "sourceCommitSpecifier": "left", "destinationCommitSpecifier": "right", "targetBranch": "right"})
	merged := c.must("GetFile", map[string]any{"repositoryName": repo, "commitSpecifier": "right", "filePath": "notes.txt"})
	content, _ := base64.StdEncoding.DecodeString(merged["fileContent"].(string))
	if string(content) != "ONE\ntwo\nTHREE\n" {
		fail(fmt.Errorf("merged content = %q", content))
	}
	fmt.Printf("merged notes.txt: %q\n", content)

	// A pull request with an approval rule, approval, comment and merge.
	pr := c.must("CreatePullRequest", map[string]any{"title": "Uppercase the first line", "targets": []any{map[string]any{"repositoryName": repo, "sourceReference": "left", "destinationReference": "main"}}})["pullRequest"].(map[string]any)
	id, rev := pr["pullRequestId"].(string), pr["revisionId"].(string)
	c.must("CreatePullRequestApprovalRule", map[string]any{"pullRequestId": id, "approvalRuleName": "one-approval",
		"approvalRuleContent": `{"Version":"2018-11-08","Statements":[{"Type":"Approvers","NumberOfApprovalsNeeded":1}]}`})
	c.must("PostCommentForPullRequest", map[string]any{"pullRequestId": id, "repositoryName": repo, "beforeCommitId": first, "afterCommitId": left, "content": "Looks good"})
	c.must("UpdatePullRequestApprovalState", map[string]any{"pullRequestId": id, "revisionId": rev, "approvalState": "APPROVE"})
	done := c.must("MergePullRequestByFastForward", map[string]any{"pullRequestId": id, "repositoryName": repo})["pullRequest"].(map[string]any)
	fmt.Printf("pull request %s is %s\n", id, done["pullRequestStatus"])

	c.must("DeleteRepository", map[string]any{"repositoryName": repo})
	fmt.Println("Done.")
}

func (c client) must(action string, payload map[string]any) map[string]any {
	out, err := c.call(action, payload)
	if err != nil {
		fail(fmt.Errorf("%s: %w", action, err))
	}
	fmt.Printf("%s ok\n", action)
	return out
}

func (c client) call(action string, payload map[string]any) (map[string]any, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "CodeCommit_20150413."+action)
	sum := sha256.Sum256(body)
	creds, err := c.creds.Retrieve(context.Background())
	if err != nil {
		return nil, err
	}
	if err := v4.NewSigner().SignHTTP(context.Background(), creds, req, hex.EncodeToString(sum[:]), "codecommit", c.region, time.Now()); err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
