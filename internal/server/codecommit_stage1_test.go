package server

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func ccB64s(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// ccSeedRepo creates repo with a first commit on main holding README.md and
// src/app.go, and returns that commit id.
func ccSeedRepo(t *testing.T, ts interface{ Close() }, call func(op, body string) map[string]any, repo string) string {
	t.Helper()
	call("CreateRepository", `{"repositoryName":"`+repo+`","repositoryDescription":"demo","tags":{"team":"core"}}`)
	out := call("CreateCommit", fmt.Sprintf(`{"repositoryName":%q,"branchName":"main","authorName":"Ada","email":"ada@example.com","commitMessage":"init",
		"putFiles":[{"filePath":"README.md","fileContent":%q},{"filePath":"src/app.go","fileContent":%q,"fileMode":"EXECUTABLE"}]}`,
		repo, ccB64s("hello\n"), ccB64s("package main\n")))
	return out["commitId"].(string)
}

func TestCodeCommitStage1RepositoriesBranchesAndFiles(t *testing.T) {
	ts := newCodeCommitTestServer(t)
	call := func(op, body string) map[string]any { return ccOK(t, ts, op, body) }
	first := ccSeedRepo(t, ts, call, "demo")

	meta := call("GetRepository", `{"repositoryName":"demo"}`)["repositoryMetadata"].(map[string]any)
	if meta["defaultBranch"] != "main" || !strings.HasSuffix(meta["cloneUrlHttp"].(string), "/v1/repos/demo") || meta["Arn"] != ccRepoARN("demo") {
		t.Fatalf("metadata: %v", meta)
	}
	if code := ccFail(t, ts, "CreateRepository", `{"repositoryName":"demo"}`); code != "RepositoryNameExistsException" {
		t.Fatalf("duplicate repository: %s", code)
	}
	if code := ccFail(t, ts, "GetRepository", `{"repositoryName":"nope"}`); code != "RepositoryDoesNotExistException" {
		t.Fatalf("missing repository: %s", code)
	}

	// A second commit must name the tip as its parent.
	if code := ccFail(t, ts, "PutFile", fmt.Sprintf(`{"repositoryName":"demo","branchName":"main","filePath":"x.txt","fileContent":%q}`, ccB64s("x"))); code != "ParentCommitIdRequiredException" {
		t.Fatalf("missing parent: %s", code)
	}
	put := call("PutFile", fmt.Sprintf(`{"repositoryName":"demo","branchName":"main","parentCommitId":%q,"filePath":"docs/guide.md","fileContent":%q}`, first, ccB64s("guide\n")))
	second := put["commitId"].(string)
	if code := ccFail(t, ts, "PutFile", fmt.Sprintf(`{"repositoryName":"demo","branchName":"main","parentCommitId":%q,"filePath":"y.txt","fileContent":%q}`, first, ccB64s("y"))); code != "ParentCommitIdOutdatedException" {
		t.Fatalf("outdated parent: %s", code)
	}
	if code := ccFail(t, ts, "PutFile", fmt.Sprintf(`{"repositoryName":"demo","branchName":"main","parentCommitId":%q,"filePath":"docs/guide.md","fileContent":%q}`, second, ccB64s("guide\n"))); code != "SameFileContentException" {
		t.Fatalf("same content: %s", code)
	}
	if code := ccFail(t, ts, "PutFile", fmt.Sprintf(`{"repositoryName":"demo","branchName":"main","parentCommitId":%q,"filePath":"README.md/inner","fileContent":%q}`, second, ccB64s("z"))); code != "DirectoryNameConflictsWithFileNameException" {
		t.Fatalf("file used as folder: %s", code)
	}

	file := call("GetFile", `{"repositoryName":"demo","filePath":"docs/guide.md"}`)
	if file["commitId"] != second || file["fileContent"] != ccB64s("guide\n") || file["fileSize"] != float64(6) {
		t.Fatalf("GetFile: %v", file)
	}
	blob := call("GetBlob", fmt.Sprintf(`{"repositoryName":"demo","blobId":%q}`, file["blobId"]))
	if blob["content"] != ccB64s("guide\n") {
		t.Fatalf("GetBlob: %v", blob)
	}
	root := call("GetFolder", `{"repositoryName":"demo","folderPath":"/"}`)
	if len(root["files"].([]any)) != 1 || len(root["subFolders"].([]any)) != 2 {
		t.Fatalf("root folder: %v", root)
	}
	if code := ccFail(t, ts, "GetFolder", `{"repositoryName":"demo","folderPath":"missing"}`); code != "FolderDoesNotExistException" {
		t.Fatalf("missing folder: %s", code)
	}

	diff := call("GetDifferences", fmt.Sprintf(`{"repositoryName":"demo","beforeCommitSpecifier":%q,"afterCommitSpecifier":"main"}`, first))["differences"].([]any)
	if len(diff) != 1 || diff[0].(map[string]any)["changeType"] != "A" {
		t.Fatalf("differences: %v", diff)
	}
	commit := call("GetCommit", fmt.Sprintf(`{"repositoryName":"demo","commitId":%q}`, second))["commit"].(map[string]any)
	if commit["parents"].([]any)[0] != first {
		t.Fatalf("commit parents: %v", commit)
	}
	batch := call("BatchGetCommits", fmt.Sprintf(`{"repositoryName":"demo","commitIds":[%q,"%s"]}`, first, strings.Repeat("0", 40)))
	if len(batch["commits"].([]any)) != 1 || len(batch["errors"].([]any)) != 1 {
		t.Fatalf("BatchGetCommits: %v", batch)
	}
	history := call("ListFileCommitHistory", `{"repositoryName":"demo","filePath":"README.md"}`)["revisionDag"].([]any)
	if len(history) != 1 {
		t.Fatalf("README history: %v", history)
	}

	// Blob differences between two versions of a file.
	upd := call("PutFile", fmt.Sprintf(`{"repositoryName":"demo","branchName":"main","parentCommitId":%q,"filePath":"README.md","fileContent":%q}`, second, ccB64s("hello\nworld\n")))
	newBlob := call("GetFile", fmt.Sprintf(`{"repositoryName":"demo","filePath":"README.md","commitSpecifier":%q}`, upd["commitId"]))["blobId"]
	oldBlob := call("GetFile", fmt.Sprintf(`{"repositoryName":"demo","filePath":"README.md","commitSpecifier":%q}`, first))["blobId"]
	bd := call("GetBlobDifferences", fmt.Sprintf(`{"repositoryName":"demo","beforeBlobId":%q,"afterBlobId":%q}`, oldBlob, newBlob))
	hunks := bd["hunks"].([]any)
	if bd["isBinary"] != false || len(hunks) != 1 || !strings.Contains(fmt.Sprint(hunks[0]), "world") {
		t.Fatalf("GetBlobDifferences: %v", bd)
	}

	// Branches.
	call("CreateBranch", fmt.Sprintf(`{"repositoryName":"demo","branchName":"feature","commitId":%q}`, first))
	if code := ccFail(t, ts, "CreateBranch", fmt.Sprintf(`{"repositoryName":"demo","branchName":"feature","commitId":%q}`, first)); code != "BranchNameExistsException" {
		t.Fatalf("duplicate branch: %s", code)
	}
	if got := call("ListBranches", `{"repositoryName":"demo"}`)["branches"].([]any); len(got) != 2 {
		t.Fatalf("branches: %v", got)
	}
	if code := ccFail(t, ts, "DeleteBranch", `{"repositoryName":"demo","branchName":"main"}`); code != "DefaultBranchCannotBeDeletedException" {
		t.Fatalf("delete default branch: %s", code)
	}
	call("UpdateDefaultBranch", `{"repositoryName":"demo","defaultBranchName":"feature"}`)
	if got := call("DeleteBranch", `{"repositoryName":"demo","branchName":"main"}`)["deletedBranch"].(map[string]any); got["branchName"] != "main" {
		t.Fatalf("DeleteBranch: %v", got)
	}

	// Tags, triggers, rename.
	arn := ccRepoARN("demo")
	call("TagResource", fmt.Sprintf(`{"resourceArn":%q,"tags":{"env":"dev"}}`, arn))
	if tags := call("ListTagsForResource", fmt.Sprintf(`{"resourceArn":%q}`, arn))["tags"].(map[string]any); len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	if code := ccFail(t, ts, "TagResource", fmt.Sprintf(`{"resourceArn":%q,"tags":{"aws:x":"y"}}`, arn)); code != "InvalidSystemTagUsageException" {
		t.Fatalf("aws: tag: %s", code)
	}
	trig := `[{"name":"t1","destinationArn":"arn:aws:sns:us-east-1:123456789012:topic","events":["all"]}]`
	call("PutRepositoryTriggers", `{"repositoryName":"demo","triggers":`+trig+`}`)
	if got := call("GetRepositoryTriggers", `{"repositoryName":"demo"}`)["triggers"].([]any); len(got) != 1 {
		t.Fatalf("triggers: %v", got)
	}
	if code := ccFail(t, ts, "PutRepositoryTriggers", `{"repositoryName":"demo","triggers":[{"name":"t","destinationArn":"arn:aws:s3:::b","events":["all"]}]}`); code != "InvalidRepositoryTriggerDestinationArnException" {
		t.Fatalf("bad trigger destination: %s", code)
	}
	call("UpdateRepositoryName", `{"oldName":"demo","newName":"renamed"}`)
	if code := ccFail(t, ts, "GetRepository", `{"repositoryName":"demo"}`); code != "RepositoryDoesNotExistException" {
		t.Fatalf("old name after rename: %s", code)
	}
	if got := call("DeleteRepository", `{"repositoryName":"renamed"}`); got["repositoryId"] != meta["repositoryId"] {
		t.Fatalf("DeleteRepository: %v", got)
	}
}

func TestCodeCommitStage1ContentIsBoundedAcrossRepositories(t *testing.T) {
	st := newCodeCommitStore()
	if _, err := st.createRepository(map[string]any{"repositoryName": "a"}); err != nil {
		t.Fatal(err.msg)
	}
	r := st.repos["a"]
	st.blobBytes = ccMaxStoreBytes - 1 // other repositories already hold the rest
	if _, err := r.putBlob([]byte("xy")); err == nil || err.code != "RepositoryLimitExceededException" {
		t.Fatalf("store-wide cap: %v", err)
	}
	st.blobBytes = 0
	if _, err := r.putBlob([]byte("xy")); err != nil {
		t.Fatalf("small blob: %v", err.msg)
	}
	if _, err := st.deleteRepository(map[string]any{"repositoryName": "a"}); err != nil || st.blobBytes != 0 {
		t.Fatalf("delete should release content bytes: %d", st.blobBytes)
	}
}

// Paths are stored per commit, so their length is capped and their bytes count
// against the repository's content budget along with messages and authors.
func TestCodeCommitStage1CapsPathLength(t *testing.T) {
	ts := newCodeCommitTestServer(t)
	ccOK(t, ts, "CreateRepository", `{"repositoryName":"paths"}`)
	long := strings.Repeat("d/", 2100) + "f.txt"
	if code := ccFail(t, ts, "PutFile", fmt.Sprintf(`{"repositoryName":"paths","branchName":"main","filePath":%q,"fileContent":%q}`, long, ccB64s("x"))); code != "InvalidPathException" {
		t.Fatalf("over-long path: %s", code)
	}
	ccOK(t, ts, "PutFile", fmt.Sprintf(`{"repositoryName":"paths","branchName":"main","filePath":"ok.txt","fileContent":%q}`, ccB64s("x")))
}
